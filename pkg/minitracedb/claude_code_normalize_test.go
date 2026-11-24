package minitracedb

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func claudeFixturePath(t *testing.T, parts ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("locating test file")
	}
	return filepath.Join(append([]string{filepath.Dir(file), "..", "..", "testdata", "claude-code", "projects", "-tmp-demo"}, parts...)...)
}

func TestLoadSessionFileAutoConvertsMetadataOnlyClaudeTranscript(t *testing.T) {
	loaded, err := LoadSessionFileAuto(claudeFixturePath(t, "22222222-2222-4222-8222-222222222222.jsonl"), LoadOptions{AutoConvert: true})
	if err != nil {
		t.Fatalf("metadata-only transcript must convert, got %v", err)
	}
	if loaded.Format != "claude-code-jsonl" || loaded.Session.ID != "22222222-2222-4222-8222-222222222222" {
		t.Fatalf("unexpected load: format=%s id=%s", loaded.Format, loaded.Session.ID)
	}
	if len(loaded.Diagnostics) != 1 || loaded.Diagnostics[0].Severity != "warning" {
		t.Fatalf("expected one warning diagnostic, got %+v", loaded.Diagnostics)
	}
}

func TestLoadSessionFileAutoConvertsSubagentTranscriptAsSubagent(t *testing.T) {
	loaded, err := LoadSessionFileAuto(claudeFixturePath(t, "11111111-1111-4111-8111-111111111111", "subagents", "agent-a1111111111111111.jsonl"), LoadOptions{AutoConvert: true})
	if err != nil {
		t.Fatalf("LoadSessionFileAuto: %v", err)
	}
	session := loaded.Session
	if session.ID != "a1111111111111111" || session.Provenance.SourceFormat != "claude-code-jsonl-v2+subagent" || session.Coordination.PredecessorSession == nil || *session.Coordination.PredecessorSession != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("subagent identity lost: id=%s format=%s predecessor=%v", session.ID, session.Provenance.SourceFormat, session.Coordination.PredecessorSession)
	}
}

func TestMaterializeClaudeFixtureNormalizesSignals(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	for _, parts := range [][]string{
		{"11111111-1111-4111-8111-111111111111.jsonl"},
		{"11111111-1111-4111-8111-111111111111", "subagents", "agent-a1111111111111111.jsonl"},
	} {
		loaded, err := LoadSessionFileAuto(claudeFixturePath(t, parts...), LoadOptions{AutoConvert: true})
		if err != nil {
			t.Fatalf("LoadSessionFileAuto %v: %v", parts, err)
		}
		if err := MaterializeSession(ctx, db, loaded.Session, MaterializeOptions{}); err != nil {
			t.Fatalf("MaterializeSession: %v", err)
		}
	}
	runner, err := NewQueryRunner(db, AllowedObjectNames(), QueryOptions{})
	if err != nil {
		t.Fatalf("NewQueryRunner: %v", err)
	}
	queryOne := func(query string) map[string]any {
		t.Helper()
		row, err := runner.QueryOne(ctx, query)
		if err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		return row
	}

	loads := queryOne(`SELECT count(*) AS n, sum(kind = 'nested_memory') AS nested, sum(tool_call_id = 'toolu_bash1') AS triggered,
		sum(path = '/tmp/demo/.claude/rules/style.md') AS rule FROM context_loads`)
	if loads["n"] != int64(5) || loads["nested"] != int64(2) || loads["triggered"] != int64(2) || loads["rule"] != int64(1) {
		t.Fatalf("context_loads = %+v", loads)
	}
	outcomes := queryOne(`SELECT
		(SELECT outcome_reason FROM tool_calls WHERE tool_call_id = 'toolu_bash1') AS blocked,
		(SELECT success FROM tool_calls WHERE tool_call_id = 'toolu_bash1') AS blocked_success,
		(SELECT outcome_reason FROM tool_calls WHERE tool_call_id = 'toolu_bash2') AS denied,
		(SELECT count(*) FROM tool_calls WHERE outcome_reason IS NULL) AS plain`)
	if outcomes["blocked"] != "hook_blocked" || outcomes["blocked_success"] != int64(0) || outcomes["denied"] != "classifier_denied" {
		t.Fatalf("tool outcome reasons = %+v", outcomes)
	}
	events := queryOne(`SELECT sum(kind = 'hook') AS hooks, sum(kind = 'compaction') AS compactions, sum(kind = 'turn_duration') AS durations,
		(SELECT tool_call_id FROM events WHERE kind = 'hook' AND json_extract(framework_metadata_json, '$.hook_type') = 'hook_success') AS hooked_call
		FROM events`)
	if events["hooks"] != int64(2) || events["compactions"] != int64(1) || events["durations"] != int64(2) || events["hooked_call"] != "toolu_bash1" {
		t.Fatalf("events = %+v", events)
	}
	metrics := queryOne(`SELECT compaction_count, total_turn_duration_ms FROM metrics WHERE session_id = '11111111-1111-4111-8111-111111111111'`)
	if metrics["compaction_count"] != int64(1) || metrics["total_turn_duration_ms"] != int64(151000) {
		t.Fatalf("metrics = %+v", metrics)
	}
	link := queryOne(`SELECT t.spawned_agent_sub_session_id AS sub, s.predecessor_session AS parent
		FROM tool_calls t JOIN sessions s ON s.session_id = t.spawned_agent_sub_session_id
		WHERE t.tool_call_id = 'toolu_agent1'`)
	if link["sub"] != "a1111111111111111" || link["parent"] != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("parent/subagent link = %+v", link)
	}
}
