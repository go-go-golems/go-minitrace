package query

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds/runner"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
)

func TestMinitraceCatalogGlazeCommand_RunIntoGlazeProcessorExecutesJSCommand(t *testing.T) {
	catalog := mustJSCatalog(t, `
__section__("filters", {
  fields: {
    limit: { type: "int", default: 10 }
  }
});

function sessionList(filters) {
  const mt = require("minitrace");
  return mt.query(`+"`"+`
    SELECT id, title
    FROM ${mt.tableName}
    LIMIT ${filters.limit}
  `+"`"+`);
}

__verb__("sessionList", {
  name: "session-list",
  short: "List sessions",
  fields: {
    filters: { bind: "filters" }
  }
});
`)

	command := catalog.ByName["session-list"]
	if command == nil {
		t.Fatalf("catalog missing session-list command")
	}
	glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
	if err != nil {
		t.Fatalf("NewMinitraceCatalogGlazeCommand returned error: %v", err)
	}

	archiveGlob := writeFixtureArchive(t)
	parsedValues, err := runner.ParseCommandValues(glazeCommand, runner.WithValuesForSections(map[string]map[string]interface{}{
		"filters": {
			"limit": 1,
		},
		QueryRuntimeSectionSlug: {
			"archive-glob": []string{archiveGlob},
		},
	}))
	if err != nil {
		t.Fatalf("ParseCommandValues returned error: %v", err)
	}

	gp := &captureProcessor{}
	if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err != nil {
		t.Fatalf("RunIntoGlazeProcessor returned error: %v", err)
	}
	if len(gp.rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(gp.rows))
	}
	row := rowToMap(gp.rows[0])
	if row["id"] != "fixture-session" {
		t.Fatalf("row id = %#v, want fixture-session", row["id"])
	}
}

func TestMinitraceCatalogGlazeCommand_RunIntoGlazeProcessorExecutesJSAliasWithSectionDefaults(t *testing.T) {
	catalog, err := minitracecmd.LoadCatalog([]minitracecmd.SourceRoot{{
		Name: "test-root",
		FS: fstest.MapFS{
			"queries/overview/session-list.js": &fstest.MapFile{Data: []byte(`
__section__("filters", {
  fields: {
    limit: { type: "int", default: 10 }
  }
});

function sessionList(filters) {
  const mt = require("minitrace");
  return mt.query(` + "`" + `
    SELECT id
    FROM ${mt.tableName}
    LIMIT ${filters.limit}
  ` + "`" + `);
}

__verb__("sessionList", {
  name: "session-list",
  short: "List sessions",
  fields: {
    filters: { bind: "filters" }
  }
});
`)},
			"queries/overview/aliases/one-session.alias.yaml": &fstest.MapFile{Data: []byte(`name: one-session
short: One session
aliasFor: session-list
flags:
  limit: 1
`)},
		},
		RootDir:  "queries",
		Readonly: true,
	}})
	if err != nil {
		t.Fatalf("LoadCatalog returned error: %v", err)
	}

	alias := catalog.ByName["one-session"]
	if alias == nil {
		t.Fatalf("catalog missing one-session alias")
	}
	glazeCommand, err := NewMinitraceCatalogGlazeCommand(alias, catalog)
	if err != nil {
		t.Fatalf("NewMinitraceCatalogGlazeCommand returned error: %v", err)
	}

	archiveGlob := writeFixtureArchive(t)
	parsedValues, err := runner.ParseCommandValues(glazeCommand, runner.WithValuesForSections(map[string]map[string]interface{}{
		QueryRuntimeSectionSlug: {
			"archive-glob": []string{archiveGlob},
		},
	}))
	if err != nil {
		t.Fatalf("ParseCommandValues returned error: %v", err)
	}

	gp := &captureProcessor{}
	if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err != nil {
		t.Fatalf("RunIntoGlazeProcessor returned error: %v", err)
	}
	if len(gp.rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(gp.rows))
	}
}

func TestMinitraceCatalogGlazeCommand_RunIntoGlazeProcessorReturnsThrownJSError(t *testing.T) {
	catalog := mustJSCatalog(t, `
function sessionList() {
  throw new Error("boom");
}

__verb__("sessionList", {
  name: "session-list",
  short: "List sessions"
});
`)

	command := catalog.ByName["session-list"]
	glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
	if err != nil {
		t.Fatalf("NewMinitraceCatalogGlazeCommand returned error: %v", err)
	}

	archiveGlob := writeFixtureArchive(t)
	parsedValues, err := runner.ParseCommandValues(glazeCommand, runner.WithValuesForSections(map[string]map[string]interface{}{
		QueryRuntimeSectionSlug: {
			"archive-glob": []string{archiveGlob},
		},
	}))
	if err != nil {
		t.Fatalf("ParseCommandValues returned error: %v", err)
	}

	gp := &captureProcessor{}
	if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err == nil {
		t.Fatalf("RunIntoGlazeProcessor returned nil error, want thrown js error")
	}
}

func TestMinitraceCatalogGlazeCommand_RunIntoGlazeProcessorReturnsRejectedPromiseError(t *testing.T) {
	catalog := mustJSCatalog(t, `
function sessionList() {
  return Promise.reject(new Error("promise boom"));
}

__verb__("sessionList", {
  name: "session-list",
  short: "List sessions"
});
`)

	command := catalog.ByName["session-list"]
	glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
	if err != nil {
		t.Fatalf("NewMinitraceCatalogGlazeCommand returned error: %v", err)
	}

	archiveGlob := writeFixtureArchive(t)
	parsedValues, err := runner.ParseCommandValues(glazeCommand, runner.WithValuesForSections(map[string]map[string]interface{}{
		QueryRuntimeSectionSlug: {
			"archive-glob": []string{archiveGlob},
		},
	}))
	if err != nil {
		t.Fatalf("ParseCommandValues returned error: %v", err)
	}

	gp := &captureProcessor{}
	if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err == nil {
		t.Fatalf("RunIntoGlazeProcessor returned nil error, want rejected promise error")
	}
}

func TestMinitraceCatalogGlazeCommand_RunIntoGlazeProcessorRejectsTextModeJSCommand(t *testing.T) {
	catalog := mustJSCatalog(t, `
function sessionList() {
  return "hello";
}

__verb__("sessionList", {
  name: "session-list",
  short: "List sessions",
  output: "text"
});
`)

	command := catalog.ByName["session-list"]
	glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
	if err != nil {
		t.Fatalf("NewMinitraceCatalogGlazeCommand returned error: %v", err)
	}

	archiveGlob := writeFixtureArchive(t)
	parsedValues, err := runner.ParseCommandValues(glazeCommand, runner.WithValuesForSections(map[string]map[string]interface{}{
		QueryRuntimeSectionSlug: {
			"archive-glob": []string{archiveGlob},
		},
	}))
	if err != nil {
		t.Fatalf("ParseCommandValues returned error: %v", err)
	}

	gp := &captureProcessor{}
	if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err == nil {
		t.Fatalf("RunIntoGlazeProcessor returned nil error, want text mode rejection")
	}
}

func TestMinitraceCatalogGlazeCommand_RunIntoGlazeProcessorExecutesPromiseReturningJSCommand(t *testing.T) {
	catalog := mustJSCatalog(t, `
__section__("filters", {
  fields: {
    limit: { type: "int", default: 10 }
  }
});

async function sessionList(filters) {
  const mt = require("minitrace");
  const timer = require("timer");
  await timer.sleep(1);
  return mt.query(`+"`"+`
    SELECT id
    FROM ${mt.tableName}
    LIMIT ${filters.limit}
  `+"`"+`);
}

__verb__("sessionList", {
  name: "session-list",
  short: "List sessions",
  fields: {
    filters: { bind: "filters" }
  }
});
`)

	command := catalog.ByName["session-list"]
	glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
	if err != nil {
		t.Fatalf("NewMinitraceCatalogGlazeCommand returned error: %v", err)
	}

	archiveGlob := writeFixtureArchive(t)
	parsedValues, err := runner.ParseCommandValues(glazeCommand, runner.WithValuesForSections(map[string]map[string]interface{}{
		"filters": {
			"limit": 1,
		},
		QueryRuntimeSectionSlug: {
			"archive-glob": []string{archiveGlob},
		},
	}))
	if err != nil {
		t.Fatalf("ParseCommandValues returned error: %v", err)
	}

	gp := &captureProcessor{}
	if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err != nil {
		t.Fatalf("RunIntoGlazeProcessor returned error: %v", err)
	}
	if len(gp.rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(gp.rows))
	}
}

func TestMinitraceCatalogGlazeCommand_RunIntoGlazeProcessorExecutesJSShowcaseTestdata(t *testing.T) {
	catalog := mustJSShowcaseCatalog(t)
	archiveGlob := writeFixtureArchive(t)

	t.Run("framework share uses helper module post-processing", func(t *testing.T) {
		command := catalog.ByPath["overview/session-tools/framework-share"]
		if command == nil {
			t.Fatalf("catalog missing framework-share command")
		}
		glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
		if err != nil {
			t.Fatalf("NewMinitraceCatalogGlazeCommand returned error: %v", err)
		}

		parsedValues, err := runner.ParseCommandValues(glazeCommand, runner.WithValuesForSections(map[string]map[string]interface{}{
			"filters": {
				"limit": 5,
			},
			QueryRuntimeSectionSlug: {
				"archive-glob": []string{archiveGlob},
			},
		}))
		if err != nil {
			t.Fatalf("ParseCommandValues returned error: %v", err)
		}

		gp := &captureProcessor{}
		if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err != nil {
			t.Fatalf("RunIntoGlazeProcessor returned error: %v", err)
		}
		if len(gp.rows) != 1 {
			t.Fatalf("len(rows) = %d, want 1", len(gp.rows))
		}
		row := rowToMap(gp.rows[0])
		if row["framework"] != "codex" {
			t.Fatalf("framework = %#v, want codex", row["framework"])
		}
		if fmt.Sprint(row["share_percent"]) != "100" {
			t.Fatalf("share_percent = %#v, want 100", row["share_percent"])
		}
	})

	t.Run("runtime playground can emit synthetic rows without querying", func(t *testing.T) {
		command := catalog.ByPath["overview/runtime-playground/build-synthetic-rows"]
		if command == nil {
			t.Fatalf("catalog missing build-synthetic-rows command")
		}
		glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
		if err != nil {
			t.Fatalf("NewMinitraceCatalogGlazeCommand returned error: %v", err)
		}

		parsedValues, err := runner.ParseCommandValues(glazeCommand, runner.WithValuesForSections(map[string]map[string]interface{}{
			"options": {
				"prefix": "ticket",
				"tags":   []string{"review", "urgent"},
			},
			QueryRuntimeSectionSlug: {
				"archive-glob": []string{archiveGlob},
			},
		}))
		if err != nil {
			t.Fatalf("ParseCommandValues returned error: %v", err)
		}

		gp := &captureProcessor{}
		if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err != nil {
			t.Fatalf("RunIntoGlazeProcessor returned error: %v", err)
		}
		if len(gp.rows) != 2 {
			t.Fatalf("len(rows) = %d, want 2", len(gp.rows))
		}
		first := rowToMap(gp.rows[0])
		if first["slug"] != "ticket-review" {
			t.Fatalf("first slug = %#v, want ticket-review", first["slug"])
		}
	})

	t.Run("async tools can await and queryOne", func(t *testing.T) {
		command := catalog.ByPath["overview/async-tools/delayed-summary"]
		if command == nil {
			t.Fatalf("catalog missing delayed-summary command")
		}
		glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
		if err != nil {
			t.Fatalf("NewMinitraceCatalogGlazeCommand returned error: %v", err)
		}

		parsedValues, err := runner.ParseCommandValues(glazeCommand, runner.WithValuesForSections(map[string]map[string]interface{}{
			"filters": {
				"limit": 3,
			},
			QueryRuntimeSectionSlug: {
				"archive-glob": []string{archiveGlob},
			},
		}))
		if err != nil {
			t.Fatalf("ParseCommandValues returned error: %v", err)
		}

		gp := &captureProcessor{}
		if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err != nil {
			t.Fatalf("RunIntoGlazeProcessor returned error: %v", err)
		}
		if len(gp.rows) != 1 {
			t.Fatalf("len(rows) = %d, want 1", len(gp.rows))
		}
		row := rowToMap(gp.rows[0])
		if row["first_id"] != "fixture-session" {
			t.Fatalf("first_id = %#v, want fixture-session", row["first_id"])
		}
		if row["delayed"] != true {
			t.Fatalf("delayed = %#v, want true", row["delayed"])
		}
	})

	t.Run("showcase aliases can target advanced JS commands", func(t *testing.T) {
		command := catalog.ByPath["analysis/aliases/focus-top-workspaces.alias.yaml"]
		if command == nil {
			t.Fatalf("catalog missing focus-top-workspaces alias")
		}
		glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
		if err != nil {
			t.Fatalf("NewMinitraceCatalogGlazeCommand returned error: %v", err)
		}

		parsedValues, err := runner.ParseCommandValues(glazeCommand, runner.WithValuesForSections(map[string]map[string]interface{}{
			QueryRuntimeSectionSlug: {
				"archive-glob": []string{archiveGlob},
			},
		}))
		if err != nil {
			t.Fatalf("ParseCommandValues returned error: %v", err)
		}

		gp := &captureProcessor{}
		if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err != nil {
			t.Fatalf("RunIntoGlazeProcessor returned error: %v", err)
		}
		if len(gp.rows) == 0 {
			t.Fatalf("expected alias to emit rows")
		}
	})
}

func mustJSCatalog(t *testing.T, source string) *minitracecmd.Catalog {
	t.Helper()
	catalog, err := minitracecmd.LoadCatalog([]minitracecmd.SourceRoot{{
		Name: "test-root",
		FS: fstest.MapFS{
			"queries/overview/session-list.js": &fstest.MapFile{Data: []byte(source)},
		},
		RootDir:  "queries",
		Readonly: true,
	}})
	if err != nil {
		t.Fatalf("LoadCatalog returned error: %v", err)
	}
	return catalog
}

func mustJSShowcaseCatalog(t *testing.T) *minitracecmd.Catalog {
	t.Helper()
	catalog, err := minitracecmd.LoadCatalog([]minitracecmd.SourceRoot{{
		Name:     "showcase",
		FS:       os.DirFS(jsShowcaseRepoRoot(t)),
		RootDir:  ".",
		Readonly: true,
	}})
	if err != nil {
		t.Fatalf("LoadCatalog returned error: %v", err)
	}
	return catalog
}

func jsShowcaseRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "testdata", "query-repositories", "js-showcase"))
}

func TestMinitraceCatalogGlazeCommand_RunIntoGlazeProcessorExecutesAdvancedJSShowcaseCommands(t *testing.T) {
	catalog := mustJSShowcaseCatalog(t)
	archiveGlob := writeAdvancedFixtureArchive(t)

	t.Run("workspace scoreboard joins multiple queries in JS", func(t *testing.T) {
		command := catalog.ByPath["analysis/workspace-lab/workspace-scoreboard"]
		if command == nil {
			t.Fatalf("catalog missing workspace-scoreboard command")
		}
		glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
		if err != nil {
			t.Fatalf("NewMinitraceCatalogGlazeCommand returned error: %v", err)
		}

		parsedValues, err := runner.ParseCommandValues(glazeCommand, runner.WithValuesForSections(map[string]map[string]interface{}{
			"filters": {
				"limit": 5,
			},
			QueryRuntimeSectionSlug: {
				"archive-glob": []string{archiveGlob},
			},
		}))
		if err != nil {
			t.Fatalf("ParseCommandValues returned error: %v", err)
		}

		gp := &captureProcessor{}
		if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err != nil {
			t.Fatalf("RunIntoGlazeProcessor returned error: %v", err)
		}
		if len(gp.rows) < 2 {
			t.Fatalf("len(rows) = %d, want at least 2", len(gp.rows))
		}
		first := rowToMap(gp.rows[0])
		if first["workspace_slug"] != "alpha/app" {
			t.Fatalf("workspace_slug = %#v, want alpha/app", first["workspace_slug"])
		}
		if first["session_count"] != int64(2) && first["session_count"] != 2 {
			t.Fatalf("session_count = %#v, want 2", first["session_count"])
		}
		if first["top_tool"] != "bash" {
			t.Fatalf("top_tool = %#v, want bash", first["top_tool"])
		}
	})

	t.Run("session shape ranker joins turn and tool aggregates", func(t *testing.T) {
		command := catalog.ByPath["analysis/session-architectures/session-shape-ranker"]
		if command == nil {
			t.Fatalf("catalog missing session-shape-ranker command")
		}
		glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
		if err != nil {
			t.Fatalf("NewMinitraceCatalogGlazeCommand returned error: %v", err)
		}

		parsedValues, err := runner.ParseCommandValues(glazeCommand, runner.WithValuesForSections(map[string]map[string]interface{}{
			"filters": {
				"limit": 5,
			},
			QueryRuntimeSectionSlug: {
				"archive-glob": []string{archiveGlob},
			},
		}))
		if err != nil {
			t.Fatalf("ParseCommandValues returned error: %v", err)
		}

		gp := &captureProcessor{}
		if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err != nil {
			t.Fatalf("RunIntoGlazeProcessor returned error: %v", err)
		}
		if len(gp.rows) < 3 {
			t.Fatalf("len(rows) = %d, want at least 3", len(gp.rows))
		}
		first := rowToMap(gp.rows[0])
		if first["shape_label"] == "" {
			t.Fatalf("shape_label missing from first row: %#v", first)
		}
		if first["unique_tools"] == nil {
			t.Fatalf("unique_tools missing from first row: %#v", first)
		}
	})
}

func writeFixtureArchive(t *testing.T) string {
	t.Helper()
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t)
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}
	return filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")
}

func buildFixtureSession(t *testing.T) *minitrace.Session {
	t.Helper()
	ts := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	formatted := minitrace.FormatTimestamp(ts)
	turn := minitrace.BuildTurn(0, &formatted, "user", stringPtr("human"), "hello from fixture")

	session := minitrace.BuildSessionSkeleton("fixture-session", "codex", "fixture", "test")
	session.Title = stringPtr("Fixture Session")
	session.Environment.Model = stringPtr("gpt-5")
	session.Turns = []minitrace.Turn{turn}
	session.ToolCalls = []minitrace.ToolCall{}
	session.Annotations = []minitrace.Annotation{}
	session.Timing = minitrace.ComputeTiming([]time.Time{ts})
	quality := minitrace.AssignQualityTier(session.Turns, session.ToolCalls)
	session.Quality = &quality
	session.Metrics = minitrace.ComputeMetrics(session.Turns, session.ToolCalls, session.Timing, 0, nil)
	return &session
}

func writeAdvancedFixtureArchive(t *testing.T) string {
	t.Helper()
	archiveRoot := t.TempDir()
	for _, session := range buildAdvancedFixtureSessions(t) {
		if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
			t.Fatalf("WriteSession returned error: %v", err)
		}
	}
	return filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")
}

func buildAdvancedFixtureSessions(t *testing.T) []*minitrace.Session {
	t.Helper()
	mkSession := func(sessionID, title, workingDir, model string, start time.Time, userTurns, assistantTurns int, toolNames []string) *minitrace.Session {
		session := minitrace.BuildSessionSkeleton(sessionID, "pi", "fixture", "test")
		session.Title = stringPtr(title)
		session.Environment.Model = stringPtr(model)
		session.OperationalContext.WorkingDirectory = stringPtr(workingDir)

		turns := []minitrace.Turn{}
		times := []time.Time{}
		index := 0
		for i := 0; i < userTurns; i++ {
			ts := start.Add(time.Duration(index) * time.Minute)
			formatted := minitrace.FormatTimestamp(ts)
			turns = append(turns, minitrace.BuildTurn(index, &formatted, "user", stringPtr("human"), fmt.Sprintf("user turn %d", i+1)))
			times = append(times, ts)
			index++
		}
		for i := 0; i < assistantTurns; i++ {
			ts := start.Add(time.Duration(index) * time.Minute)
			formatted := minitrace.FormatTimestamp(ts)
			turns = append(turns, minitrace.BuildTurn(index, &formatted, "assistant", stringPtr("model"), fmt.Sprintf("assistant turn %d", i+1)))
			times = append(times, ts)
			index++
		}
		session.Turns = turns

		toolCalls := []minitrace.ToolCall{}
		for i, toolName := range toolNames {
			ts := start.Add(time.Duration(index+i) * time.Minute)
			formatted := minitrace.FormatTimestamp(ts)
			turnIndex := assistantTurns + userTurns - 1
			toolCall := minitrace.BuildToolCall(
				fmt.Sprintf("%s-tool-%02d", sessionID, i+1),
				&turnIndex,
				&formatted,
				toolName,
				"EXECUTE",
				nil,
				nil,
				map[string]any{"example": true},
				true,
				"ok",
				nil,
				nil,
				nil,
				nil,
				stringPtr("fixture"),
				nil,
			)
			toolCalls = append(toolCalls, toolCall)
			times = append(times, ts)
		}
		session.ToolCalls = toolCalls
		session.Annotations = []minitrace.Annotation{}
		session.Timing = minitrace.ComputeTiming(times)
		quality := minitrace.AssignQualityTier(session.Turns, session.ToolCalls)
		session.Quality = &quality
		session.Metrics = minitrace.ComputeMetrics(session.Turns, session.ToolCalls, session.Timing, 0, nil)
		return &session
	}

	return []*minitrace.Session{
		mkSession("fixture-alpha-1", "Alpha planning", "~/projects/alpha/app", "gpt-5", time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC), 2, 6, []string{"bash", "bash", "bash", "read", "read", "edit"}),
		mkSession("fixture-alpha-2", "Alpha follow-up", "~/projects/alpha/app", "gpt-5", time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC), 1, 4, []string{"bash", "bash", "write", "write"}),
		mkSession("fixture-beta-1", "Beta vision", "~/projects/beta/lab", "claude-sonnet-4-6", time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC), 1, 3, []string{"understand_image", "understand_image", "read", "bash"}),
	}
}

func stringPtr(value string) *string { return &value }

type captureProcessor struct {
	rows []types.Row
}

func (c *captureProcessor) AddRow(_ context.Context, row types.Row) error {
	c.rows = append(c.rows, row)
	return nil
}

func (c *captureProcessor) Close(context.Context) error { return nil }

func rowToMap(row types.Row) map[string]interface{} {
	ret := map[string]interface{}{}
	for pair := row.Oldest(); pair != nil; pair = pair.Next() {
		ret[pair.Key] = pair.Value
	}
	return ret
}
