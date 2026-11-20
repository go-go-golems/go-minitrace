package minitracedb

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestToolArchiveOrdinalAndOriginalStatus(t *testing.T) {
	ctx := context.Background()
	db, err := OpenSQLiteMemory(ctx, "provenance")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err = CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	zero := 0
	yes := true
	s := &minitrace.Session{ID: "order", ToolCalls: []minitrace.ToolCall{
		{ID: "z-first", EmittingTurnIndex: &zero, Input: minitrace.ToolCallInput{Arguments: map[string]any{"q": "é"}}},
		{ID: "a-second", EmittingTurnIndex: &zero, Output: minitrace.ToolCallOutput{Success: &yes}},
	}}
	if err = MaterializeSession(ctx, db, s, MaterializeOptions{}); err != nil {
		t.Fatal(err)
	}
	rows, err := db.Query(`SELECT tool_call_id, source_ordinal, arguments_json, success, output_status, outcome_status FROM tool_calls ORDER BY source_ordinal`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	for i := 0; rows.Next(); i++ {
		var id, outcome string
		var ordinal int
		var args, status sql.NullString
		var success sql.NullInt64
		if err = rows.Scan(&id, &ordinal, &args, &success, &status, &outcome); err != nil {
			t.Fatal(err)
		}
		if ordinal != i || id != s.ToolCalls[i].ID {
			t.Fatalf("archive order lost: %d/%s", ordinal, id)
		}
		if status.Valid {
			t.Fatalf("absent original status synthesized: %v", status)
		}
		if i == 0 && (success.Valid || args.String != "{\"q\":\"é\"}") {
			t.Fatalf("nullable/arguments projection: %v %v", success, args)
		}
		if i == 1 && (!success.Valid || success.Int64 != 1 || outcome != "succeeded") {
			t.Fatalf("derived success status: %v %q", success, outcome)
		}
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	old := DefaultCacheKeyOptions()
	old.SchemaVersion = "normalized-sqlite-v5"
	a, _ := ComputeCacheKey(nil, old)
	b, _ := ComputeCacheKey(nil, DefaultCacheKeyOptions())
	if a.Key == b.Key {
		t.Fatal("schema change must invalidate caches")
	}
}

func TestCorpusManifestPreservesCopiesAndConflicts(t *testing.T) {
	dir := t.TempDir()
	paths := []string{}
	for i, body := range []string{`{"id":"same","provenance":{"original_session_id":"native"}}`, `{"id":"same","provenance":{"original_session_id":"native"}}`, `{"id":"same","title":"changed"}`, `{"id":"other"}`} {
		path := filepath.Join(dir, string(rune('a'+i))+".json")
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	m, err := BuildCorpusManifest(paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Entries) != 4 || m.DuplicateCount != 1 || m.ConflictCount != 1 {
		t.Fatalf("counts: %+v", m)
	}
	if m.Entries[1].DuplicateOf == nil || *m.Entries[1].DuplicateOf != 0 {
		t.Fatal("byte duplicate not linked")
	}
	if m.Entries[0].OriginalSessionID == nil || *m.Entries[0].OriginalSessionID != "native" {
		t.Fatal("native identity lost")
	}
	for i := 0; i < 3; i++ {
		if !m.Entries[i].Conflict {
			t.Fatalf("conflict omitted on variant/copy %d", i)
		}
	}
	if m.Entries[3].Conflict {
		t.Fatal("unrelated identity marked conflict")
	}
	bad := filepath.Join(dir, "bad.json")
	if err = os.WriteFile(bad, []byte(`{"title":"no id"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = BuildCorpusManifest([]string{bad}); err == nil {
		t.Fatal("missing identity accepted")
	}
}
