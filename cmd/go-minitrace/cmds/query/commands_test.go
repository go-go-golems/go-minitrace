package query

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCommandsCommand_IncludesEmbeddedCommandsAsFolderGroups(t *testing.T) {
	setIsolatedConfigHome(t)
	cmd, err := NewCommandsCommand(nil)
	if err != nil {
		t.Fatalf("NewCommandsCommand returned error: %v", err)
	}

	cases := [][]string{
		{"overview", "session-list"},
		{"overview", "framework-summary"},
		{"timing", "timing-analysis"},
		{"nightly", "session-inventory"},
		{"nightly", "workspace-summary"},
		{"nightly", "tool-breakdown"},
		{"nightly", "followup-candidates"},
		{"nightly", "annotation-summary"},
		{"overview", "aliases", "codex-framework-summary"},
	}
	for _, args := range cases {
		found, _, err := cmd.Find(args)
		if err != nil {
			t.Fatalf("Find(%v) returned error: %v", args, err)
		}
		if found == nil || found.Name() != args[len(args)-1] {
			t.Fatalf("Find(%v) resolved to %#v", args, found)
		}
	}
}

func TestNewCommandsCommand_GroupHelpSmokeTests(t *testing.T) {
	setIsolatedConfigHome(t)
	cases := []struct {
		args        []string
		wantContain []string
	}{
		{args: []string{"overview", "--help"}, wantContain: []string{"session-list", "framework-summary", "aliases"}},
		{args: []string{"overview", "aliases", "--help"}, wantContain: []string{"codex-framework-summary"}},
		{args: []string{"timing", "--help"}, wantContain: []string{"timing-analysis"}},
		{args: []string{"tools", "--help"}, wantContain: []string{"tool-failures", "tool-operation-breakdown"}},
		{args: []string{"files", "--help"}, wantContain: []string{"file-operations", "file-timeline"}},
		{args: []string{"nightly", "--help"}, wantContain: []string{"session-inventory", "workspace-summary", "followup-candidates"}},
	}

	for _, tc := range cases {
		cmd, err := NewCommandsCommand(nil)
		if err != nil {
			t.Fatalf("NewCommandsCommand returned error: %v", err)
		}
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs(tc.args)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute(%v) returned error: %v\noutput:\n%s", tc.args, err, buf.String())
		}
		output := buf.String()
		for _, needle := range tc.wantContain {
			if !strings.Contains(output, needle) {
				t.Fatalf("Execute(%v) output missing %q\noutput:\n%s", tc.args, needle, output)
			}
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
	if err := os.MkdirAll(filepath.Join(repo, "overview"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "overview", "session-list.sql"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cmd, err := NewCommandsCommand([]string{repo})
	if err != nil {
		t.Fatalf("NewCommandsCommand returned error: %v", err)
	}

	found, _, err := cmd.Find([]string{"overview", "session-list"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if found == nil || found.Name() != "session-list" {
		t.Fatalf("expected overridden overview/session-list command, got %#v", found)
	}
	if found.Short != "Override session list" {
		t.Fatalf("found.Short = %q, want override short description", found.Short)
	}

	flag := cmd.PersistentFlags().Lookup("query-repository")
	if flag == nil {
		t.Fatalf("commands root missing query-repository flag")
	}
}

func TestNewCommandsCommand_LoadsConfiguredRepositoryFromAppConfig(t *testing.T) {
	setIsolatedConfigHome(t)
	repo := t.TempDir()
	content := `/* sqleton
name: session-list
short: App config override session list
*/
SELECT 42 AS answer FROM {{TABLE_NAME}};`
	if err := os.MkdirAll(filepath.Join(repo, "overview"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "overview", "session-list.sql"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	configDir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "go-minitrace")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	configContent := "queryRepositories:\n  - " + repo + "\n"
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("WriteFile(config) returned error: %v", err)
	}

	cmd, err := NewCommandsCommand(nil)
	if err != nil {
		t.Fatalf("NewCommandsCommand returned error: %v", err)
	}

	found, _, err := cmd.Find([]string{"overview", "session-list"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if found == nil || found.Name() != "session-list" {
		t.Fatalf("expected app-config overridden overview/session-list command, got %#v", found)
	}
	if found.Short != "App config override session list" {
		t.Fatalf("found.Short = %q, want app-config override short description", found.Short)
	}
}

func TestNewCommandsCommand_LoadsJSCommandsAndShowsThemInHelp(t *testing.T) {
	setIsolatedConfigHome(t)
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "overview"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "overview", "framework-summary.sql"), []byte(`/* sqleton
name: framework-summary
short: Summarize frameworks
*/
SELECT 1 AS answer FROM {{TABLE_NAME}};`), 0o644); err != nil {
		t.Fatalf("WriteFile(sql) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "overview", "js-session-list.js"), []byte(`
function sessionList() {
  const mt = require("minitrace");
  return mt.query(`+"`"+`SELECT id FROM ${mt.tableName} LIMIT 1`+"`"+`);
}

__verb__("sessionList", {
  name: "js-session-list",
  short: "List sessions from JS"
});
`), 0o644); err != nil {
		t.Fatalf("WriteFile(js) returned error: %v", err)
	}

	cmd, err := NewCommandsCommand([]string{repo})
	if err != nil {
		t.Fatalf("NewCommandsCommand returned error: %v", err)
	}

	found, _, err := cmd.Find([]string{"overview", "js-session-list"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if found == nil || found.Name() != "js-session-list" {
		t.Fatalf("expected overview/js-session-list command, got %#v", found)
	}

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"overview", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\noutput:\n%s", err, buf.String())
	}
	output := buf.String()
	if !strings.Contains(output, "js-session-list") {
		t.Fatalf("help output missing js-session-list\noutput:\n%s", output)
	}
	if !strings.Contains(output, "framework-summary") {
		t.Fatalf("help output missing framework-summary\noutput:\n%s", output)
	}
}

func TestNewCommandsCommand_XDGConfigOverridesHomeConfig(t *testing.T) {
	setIsolatedConfigHome(t)
	homeRepo := t.TempDir()
	xdgRepo := t.TempDir()
	homeContent := `/* sqleton
name: session-list
short: Home config override session list
*/
SELECT 1 AS answer FROM {{TABLE_NAME}};`
	xdgContent := `/* sqleton
name: session-list
short: XDG config override session list
*/
SELECT 2 AS answer FROM {{TABLE_NAME}};`
	if err := os.MkdirAll(filepath.Join(homeRepo, "overview"), 0o755); err != nil {
		t.Fatalf("MkdirAll(home repo) returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(xdgRepo, "overview"), 0o755); err != nil {
		t.Fatalf("MkdirAll(xdg repo) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeRepo, "overview", "session-list.sql"), []byte(homeContent), 0o644); err != nil {
		t.Fatalf("WriteFile(home repo) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(xdgRepo, "overview", "session-list.sql"), []byte(xdgContent), 0o644); err != nil {
		t.Fatalf("WriteFile(xdg repo) returned error: %v", err)
	}

	homeConfigDir := filepath.Join(os.Getenv("HOME"), ".go-minitrace")
	if err := os.MkdirAll(homeConfigDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(home config) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeConfigDir, "config.yaml"), []byte("queryRepositories:\n  - "+homeRepo+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(home config) returned error: %v", err)
	}

	xdgConfigDir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "go-minitrace")
	if err := os.MkdirAll(xdgConfigDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(xdg config) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(xdgConfigDir, "config.yaml"), []byte("queryRepositories:\n  - "+xdgRepo+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(xdg config) returned error: %v", err)
	}

	cmd, err := NewCommandsCommand(nil)
	if err != nil {
		t.Fatalf("NewCommandsCommand returned error: %v", err)
	}

	found, _, err := cmd.Find([]string{"overview", "session-list"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if found == nil || found.Name() != "session-list" {
		t.Fatalf("expected XDG-config overridden overview/session-list command, got %#v", found)
	}
	if found.Short != "XDG config override session list" {
		t.Fatalf("found.Short = %q, want XDG config override short description", found.Short)
	}
}

func setIsolatedConfigHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
}
