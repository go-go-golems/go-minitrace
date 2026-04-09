package query

import "testing"

func TestNewCommandsCommand_IncludesEmbeddedCommands(t *testing.T) {
	cmd, err := NewCommandsCommand()
	if err != nil {
		t.Fatalf("NewCommandsCommand returned error: %v", err)
	}

	want := map[string]bool{
		"session-list":            false,
		"framework-summary":       false,
		"timing-analysis":         false,
		"codex-framework-summary": false,
	}
	for _, sub := range cmd.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Fatalf("commands subgroup missing %q", name)
		}
	}
}

func TestNewCommand_AddsCommandsSubgroup(t *testing.T) {
	root, err := NewCommand()
	if err != nil {
		t.Fatalf("NewCommand returned error: %v", err)
	}

	seen := map[string]bool{}
	for _, sub := range root.Commands() {
		seen[sub.Name()] = true
	}
	if !seen["duckdb"] || !seen["commands"] {
		t.Fatalf("query root subcommands missing expected entries: %#v", seen)
	}

	if root.CommandPath() != "query" {
		t.Fatalf("root.CommandPath() = %q, want query", root.CommandPath())
	}
}
