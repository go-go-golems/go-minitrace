package convert

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/runner"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
)

func claudeFixture(parts ...string) string {
	return filepath.Join(append([]string{"..", "..", "..", "..", "testdata", "claude-code", "projects", "-tmp-demo"}, parts...)...)
}

func runClaudeConvert(t *testing.T, flags map[string]any) []map[string]any {
	t.Helper()
	command, err := NewConvertClaudeCodeGlazeCommand()
	if err != nil {
		t.Fatalf("NewConvertClaudeCodeGlazeCommand: %v", err)
	}
	flags["dry-run"] = true
	values, err := runner.ParseCommandValues(command, runner.WithValuesForSections(map[string]map[string]any{schema.DefaultSlug: flags}))
	if err != nil {
		t.Fatalf("ParseCommandValues: %v", err)
	}
	processor := &phase0CaptureProcessor{}
	if err := command.RunIntoGlazeProcessor(context.Background(), values, processor); err != nil {
		t.Fatalf("RunIntoGlazeProcessor: %v", err)
	}
	rows := make([]map[string]any, 0, len(processor.rows))
	for _, row := range processor.rows {
		rows = append(rows, phase0RowToMap(row))
	}
	return rows
}

func countKinds(rows []map[string]any) map[string]int {
	kinds := map[string]int{}
	for _, row := range rows {
		kind, _ := row["session_kind"].(string)
		kinds[kind]++
	}
	return kinds
}

func TestConvertClaudeCodeExplicitSourcesExpandSubagents(t *testing.T) {
	parent := claudeFixture("11111111-1111-4111-8111-111111111111.jsonl")
	subagent := claudeFixture("11111111-1111-4111-8111-111111111111", "subagents", "agent-a1111111111111111.jsonl")

	rows := runClaudeConvert(t, map[string]any{"source-session": []string{parent, subagent}})
	if kinds := countKinds(rows); kinds["primary"] != 1 || kinds["subagent"] != 2 {
		t.Fatalf("explicit parent should expand to both subagents once, got %v", kinds)
	}
	for _, row := range rows {
		if row["session_kind"] == "subagent" && row["parent_session_id"] != "11111111-1111-4111-8111-111111111111" {
			t.Fatalf("subagent row without parent: %+v", row)
		}
	}

	rows = runClaudeConvert(t, map[string]any{"source-session": []string{parent}, "no-subagents": true})
	if kinds := countKinds(rows); kinds["primary"] != 1 || kinds["subagent"] != 0 {
		t.Fatalf("--no-subagents should convert only the parent, got %v", kinds)
	}

	rows = runClaudeConvert(t, map[string]any{"source-session": []string{subagent}, "no-subagents": true})
	if kinds := countKinds(rows); kinds["primary"] != 0 || kinds["subagent"] != 1 {
		t.Fatalf("a listed subagent path converts as a subagent, got %v", kinds)
	}
}
