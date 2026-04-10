package query

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCommandsCommand_IncludesEmbeddedCommands(t *testing.T) {
	setIsolatedConfigHome(t)
	cmd, err := NewCommandsCommand(nil)
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
	setIsolatedConfigHome(t)
	root, err := NewCommand(nil)
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

func TestNewCommandsCommand_LoadsConfiguredRepositoryOverrides(t *testing.T) {
	setIsolatedConfigHome(t)
	repo := t.TempDir()
	content := `/* sqleton
name: session-list
short: Override session list
*/
SELECT 42 AS answer FROM {{TABLE_NAME}};`
	if err := os.WriteFile(filepath.Join(repo, "session-list.sql"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cmd, err := NewCommandsCommand([]string{repo})
	if err != nil {
		t.Fatalf("NewCommandsCommand returned error: %v", err)
	}

	found := false
	for _, sub := range cmd.Commands() {
		if sub.Name() == "session-list" {
			found = true
			if sub.Short != "Override session list" {
				t.Fatalf("sub.Short = %q, want override short description", sub.Short)
			}
		}
	}
	if !found {
		t.Fatalf("expected overridden session-list command in %#v", cmd.Commands())
	}

	flag := cmd.PersistentFlags().Lookup("query-repository")
	if flag == nil {
		t.Fatalf("commands root missing query-repository flag")
	}
}

func setIsolatedConfigHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
}
