package query

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
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

func TestNewCommandsCommand_LoadsJSCommandsAndShowsFileAsGroupInHelp(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(repo, "overview", "session-tools.js"), []byte(`
function sessionList() {
  const mt = require("minitrace");
  return mt.query(`+"`"+`SELECT id FROM ${mt.tableName} LIMIT 1`+"`"+`);
}

__verb__("sessionList", {
  name: "session-list",
  short: "List sessions from JS"
});
`), 0o644); err != nil {
		t.Fatalf("WriteFile(js) returned error: %v", err)
	}

	cmd, err := NewCommandsCommand([]string{repo})
	if err != nil {
		t.Fatalf("NewCommandsCommand returned error: %v", err)
	}

	found, _, err := cmd.Find([]string{"overview", "session-tools", "session-list"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if found == nil || found.Name() != "session-list" {
		t.Fatalf("expected overview/session-tools/session-list command, got %#v", found)
	}

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"overview", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\noutput:\n%s", err, buf.String())
	}
	output := buf.String()
	if !strings.Contains(output, "session-tools") {
		t.Fatalf("help output missing session-tools group\noutput:\n%s", output)
	}
	if !strings.Contains(output, "framework-summary") {
		t.Fatalf("help output missing framework-summary\noutput:\n%s", output)
	}

	buf.Reset()
	cmd.SetArgs([]string{"overview", "session-tools", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute nested help returned error: %v\noutput:\n%s", err, buf.String())
	}
	output = buf.String()
	if !strings.Contains(output, "session-list") {
		t.Fatalf("nested help output missing session-list\noutput:\n%s", output)
	}
}

func TestNewCommandsCommand_SelfNamedSingleVerbJSCommandUsesCollapsedPath(t *testing.T) {
	setIsolatedConfigHome(t)
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "hardware-research"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "hardware-research", "research-summary.js"), []byte(`
__section__("filters", {
  fields: {
    limit: { type: "int", default: 5 }
  }
});

function researchSummary(filters) {
  const mt = require("minitrace");
  return mt.query(`+"`"+`SELECT id FROM ${mt.tableName} LIMIT ${filters.limit}`+"`"+`);
}

__verb__("researchSummary", {
  name: "research-summary",
  short: "Generate summary",
  fields: {
    filters: { bind: "filters" }
  }
});
`), 0o644); err != nil {
		t.Fatalf("WriteFile(js) returned error: %v", err)
	}

	cmd, err := NewCommandsCommand([]string{repo})
	if err != nil {
		t.Fatalf("NewCommandsCommand returned error: %v", err)
	}
	found, _, err := cmd.Find([]string{"hardware-research", "research-summary"})
	if err != nil {
		t.Fatalf("Find collapsed path returned error: %v", err)
	}
	if found == nil || found.Name() != "research-summary" || !found.Runnable() {
		t.Fatalf("expected runnable collapsed command, got %#v", found)
	}

	archiveGlob := writeCommandsFixtureArchive(t)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"hardware-research", "research-summary", "--archive-glob", archiveGlob, "--limit", "1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute collapsed path returned error: %v\noutput:\n%s", err, buf.String())
	}
	// Execution success is the main contract here; glazed's processor output is not guaranteed to honor cmd.SetOut.
}

func TestNewCommandsCommand_SelfNamedSingleVerbJSCommandKeepsExpandedPathWhenSiblingDirectoryHasNestedCommands(t *testing.T) {
	setIsolatedConfigHome(t)
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "hardware-research", "research-summary"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "hardware-research", "research-summary.js"), []byte(`
function researchSummary() {
  const mt = require("minitrace");
  return mt.query(`+"`"+`SELECT id FROM ${mt.tableName} LIMIT 1`+"`"+`);
}

__verb__("researchSummary", {
  name: "research-summary",
  short: "Generate summary"
});
`), 0o644); err != nil {
		t.Fatalf("WriteFile(js) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "hardware-research", "research-summary", "extra.sql"), []byte(`/* sqleton
name: extra
short: Nested extra command
*/
SELECT 1 AS answer FROM {{TABLE_NAME}};`), 0o644); err != nil {
		t.Fatalf("WriteFile(sql) returned error: %v", err)
	}

	cmd, err := NewCommandsCommand([]string{repo})
	if err != nil {
		t.Fatalf("NewCommandsCommand returned error: %v", err)
	}

	// The JS command must fall back to the pre-collapse path so the sibling directory
	// can remain a command group.
	found, _, err := cmd.Find([]string{"hardware-research", "research-summary", "research-summary"})
	if err != nil {
		t.Fatalf("Find expanded JS path returned error: %v", err)
	}
	if found == nil || found.Name() != "research-summary" {
		t.Fatalf("expected expanded JS command under hardware-research/research-summary/research-summary, got %#v", found)
	}

	found, _, err = cmd.Find([]string{"hardware-research", "research-summary", "extra"})
	if err != nil {
		t.Fatalf("Find nested SQL path returned error: %v", err)
	}
	if found == nil || found.Name() != "extra" {
		t.Fatalf("expected nested SQL command under hardware-research/research-summary/extra, got %#v", found)
	}
}

func TestNewCommandsCommand_SelfNamedSingleChildGroupHelpShowsRuntimeFlags(t *testing.T) {
	setIsolatedConfigHome(t)
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "hardware-research"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "hardware-research", "research-summary.js"), []byte(`
function researchSummary() {
  const mt = require("minitrace");
  return mt.query(`+"`"+`SELECT 1 AS ok FROM ${mt.tableName} LIMIT 1`+"`"+`);
}

__verb__("researchSummary", {
  name: "research-summary",
  short: "Generate summary"
});
`), 0o644); err != nil {
		t.Fatalf("WriteFile(js) returned error: %v", err)
	}

	cmd, err := NewCommandsCommand([]string{repo})
	if err != nil {
		t.Fatalf("NewCommandsCommand returned error: %v", err)
	}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"hardware-research", "research-summary", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute shorthand help returned error: %v\noutput:\n%s", err, buf.String())
	}
	output := buf.String()
	for _, needle := range []string{"Generate summary", "archive-glob", "db-path", "table-name", "persist-loaded", "hardware-research research-summary"} {
		if !strings.Contains(output, needle) {
			t.Fatalf("shorthand help missing %q\noutput:\n%s", needle, output)
		}
	}
}

func TestNewCommandsCommand_RejectsLeafGroupNameCollision(t *testing.T) {
	setIsolatedConfigHome(t)
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "overview"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "overview", "session-tools.sql"), []byte(`/* sqleton
name: session-tools
short: Leaf command that collides with a JS group
*/
SELECT 1 AS answer FROM {{TABLE_NAME}};`), 0o644); err != nil {
		t.Fatalf("WriteFile(sql) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "overview", "session-tools.js"), []byte(`
function sessionList() {
  const mt = require("minitrace");
  return mt.query(`+"`"+`SELECT id FROM ${mt.tableName} LIMIT 1`+"`"+`);
}

__verb__("sessionList", {
  name: "session-list",
  short: "List sessions from JS"
});
`), 0o644); err != nil {
		t.Fatalf("WriteFile(js) returned error: %v", err)
	}

	_, err := NewCommandsCommand([]string{repo})
	if err == nil {
		t.Fatalf("NewCommandsCommand returned nil error, want command tree collision")
	}
	if !strings.Contains(err.Error(), "session-tools") {
		t.Fatalf("error = %v, want collision mentioning session-tools", err)
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

func TestNewCommandsCommand_LoadsMixedSQLJSShowcaseRepo(t *testing.T) {
	setIsolatedConfigHome(t)
	repo := mixedShowcaseRepoRoot(t)
	cmd, err := NewCommandsCommand([]string{repo})
	if err != nil {
		t.Fatalf("NewCommandsCommand returned error: %v", err)
	}

	cases := [][]string{
		{"overview", "framework-summary"},
		{"overview", "session-tools", "framework-share"},
		{"overview", "aliases", "codex-framework-summary"},
		{"analysis", "raw-workspace-stats"},
		{"analysis", "workspace-lab", "workspace-scoreboard"},
		{"analysis", "aliases", "top-workspaces"},
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

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"overview", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("overview help returned error: %v\noutput:\n%s", err, buf.String())
	}
	output := buf.String()
	for _, needle := range []string{"framework-summary", "session-tools", "aliases"} {
		if !strings.Contains(output, needle) {
			t.Fatalf("overview help missing %q\noutput:\n%s", needle, output)
		}
	}

	buf.Reset()
	cmd.SetArgs([]string{"analysis", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("analysis help returned error: %v\noutput:\n%s", err, buf.String())
	}
	output = buf.String()
	for _, needle := range []string{"raw-workspace-stats", "workspace-lab", "aliases"} {
		if !strings.Contains(output, needle) {
			t.Fatalf("analysis help missing %q\noutput:\n%s", needle, output)
		}
	}
}

func writeCommandsFixtureArchive(t *testing.T) string {
	t.Helper()
	archiveRoot := t.TempDir()
	session := buildCommandsFixtureSession(t)
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}
	return filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")
}

func buildCommandsFixtureSession(t *testing.T) *minitrace.Session {
	t.Helper()
	ts := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	formatted := minitrace.FormatTimestamp(ts)
	turn := minitrace.BuildTurn(0, &formatted, "user", commandsStringPtr("human"), "hello from fixture")

	session := minitrace.BuildSessionSkeleton("fixture-session", "codex", "fixture", "test")
	session.Title = commandsStringPtr("Fixture Session")
	session.Environment.Model = commandsStringPtr("gpt-5")
	session.Turns = []minitrace.Turn{turn}
	session.ToolCalls = []minitrace.ToolCall{}
	session.Annotations = []minitrace.Annotation{}
	session.Timing = minitrace.ComputeTiming([]time.Time{ts})
	quality := minitrace.AssignQualityTier(session.Turns, session.ToolCalls)
	session.Quality = &quality
	session.Metrics = minitrace.ComputeMetrics(session.Turns, session.ToolCalls, session.Timing, 0, nil)
	return &session
}

func commandsStringPtr(value string) *string { return &value }

func setIsolatedConfigHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
}

func mixedShowcaseRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "testdata", "query-repositories", "mixed-sql-js-showcase"))
}
