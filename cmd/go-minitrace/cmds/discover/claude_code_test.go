package discover

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/runner"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/types"
)

type captureProcessor struct {
	rows []map[string]any
}

func (p *captureProcessor) AddRow(_ context.Context, row types.Row) error {
	values := map[string]any{}
	for pair := row.Oldest(); pair != nil; pair = pair.Next() {
		values[pair.Key] = pair.Value
	}
	p.rows = append(p.rows, values)
	return nil
}

func (p *captureProcessor) Close(context.Context) error { return nil }

func runClaudeDiscover(t *testing.T, flags map[string]any) []map[string]any {
	t.Helper()
	command, err := NewClaudeCodeGlazeCommand()
	if err != nil {
		t.Fatalf("NewClaudeCodeGlazeCommand: %v", err)
	}
	flags["source-dir"] = filepath.Join("..", "..", "..", "..", "testdata", "claude-code", "projects")
	values, err := runner.ParseCommandValues(command, runner.WithValuesForSections(map[string]map[string]any{schema.DefaultSlug: flags}))
	if err != nil {
		t.Fatalf("ParseCommandValues: %v", err)
	}
	processor := &captureProcessor{}
	if err := command.RunIntoGlazeProcessor(context.Background(), values, processor); err != nil {
		t.Fatalf("RunIntoGlazeProcessor: %v", err)
	}
	return processor.rows
}

func TestDiscoverClaudeCodeIncludeSubagentsFollowsParentSelection(t *testing.T) {
	plain := runClaudeDiscover(t, map[string]any{})
	if len(plain) != 2 {
		t.Fatalf("default discovery lists primaries only, got %d rows", len(plain))
	}
	if _, ok := plain[0]["session_kind"]; ok {
		t.Fatalf("default discovery output must not change shape: %+v", plain[0])
	}

	rows := runClaudeDiscover(t, map[string]any{"include-subagents": true})
	kinds := []string{}
	for _, row := range rows {
		kinds = append(kinds, row["session_kind"].(string)+":"+row["id"].(string))
	}
	want := []string{
		"primary:11111111-1111-4111-8111-111111111111",
		"subagent:a1111111111111111",
		"subagent:a2222222222222222",
		"primary:22222222-2222-4222-8222-222222222222",
	}
	if len(kinds) != len(want) {
		t.Fatalf("rows = %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("rows = %v, want %v", kinds, want)
		}
	}
	if rows[1]["parent_session_id"] != "11111111-1111-4111-8111-111111111111" || rows[1]["agent_type"] != "Explore" || rows[1]["format_hint"] != "jsonl-v2+subagent" {
		t.Fatalf("unexpected subagent row: %+v", rows[1])
	}

	// The parent started on 2026-11-20 and is dropped; its subagents go with it
	// although --since does not look at their own start times.
	since := runClaudeDiscover(t, map[string]any{"include-subagents": true, "since": "2026-11-21"})
	if len(since) != 1 || since[0]["id"] != "22222222-2222-4222-8222-222222222222" {
		t.Fatalf("--since should drop the parent with its subagents, got %+v", since)
	}
	none := runClaudeDiscover(t, map[string]any{"include-subagents": true, "cwd-contains": "elsewhere"})
	if len(none) != 0 {
		t.Fatalf("--cwd-contains should drop parents and subagents, got %+v", none)
	}
}
