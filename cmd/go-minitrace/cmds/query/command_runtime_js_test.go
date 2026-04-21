package query

import (
	"context"
	"path/filepath"
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
