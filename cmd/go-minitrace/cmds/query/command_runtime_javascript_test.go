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
  const db = mt.db().RuntimeArchives().Build();
  return db.query(`+"`"+`
    SELECT session_id AS id, title
    FROM sessions
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
  const db = mt.db().RuntimeArchives().Build();
  return db.query(` + "`" + `
    SELECT session_id AS id
    FROM sessions
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

	alias := catalog.ByPath["overview/aliases/one-session.alias.yaml"]
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

func TestMinitraceCatalogGlazeCommand_RunIntoGlazeProcessorSupportsDuplicateSourceRootNames(t *testing.T) {
	catalog, err := minitracecmd.LoadCatalog([]minitracecmd.SourceRoot{
		{
			Name: "embedded",
			FS: fstest.MapFS{
				"queries/overview/session-tools.js": &fstest.MapFile{Data: []byte(`
__section__("filters", {
  fields: {
    limit: { type: "int", default: 10 }
  }
});

function sessionList(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().Build();
  return db.query(` + "`" + `
    SELECT session_id AS id
    FROM sessions
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
			},
			RootDir:  "queries",
			Readonly: true,
		},
		{
			Name: "embedded",
			FS: fstest.MapFS{
				"queries/overview/framework-summary.sql": &fstest.MapFile{Data: []byte(`/* sqleton
name: framework-summary
short: Summarize frameworks
*/
SELECT 1 AS answer FROM {{TABLE_NAME}};`)},
			},
			RootDir:  "queries",
			Readonly: true,
		},
	})
	if err != nil {
		t.Fatalf("LoadCatalog returned error: %v", err)
	}

	command := catalog.ByPath["overview/session-tools/session-list"]
	if command == nil {
		t.Fatalf("catalog missing overview/session-tools/session-list command")
	}
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
  const db = mt.db().RuntimeArchives().Build();
  return db.query(`+"`"+`
    SELECT session_id AS id
    FROM sessions
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

	t.Run("phase3 context inventory joins expanded normalized tables", func(t *testing.T) {
		command := catalog.ByPath["analysis/phase3-cookbook/context-inventory"]
		if command == nil {
			t.Fatalf("catalog missing context-inventory command")
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
		if len(gp.rows) == 0 {
			t.Fatalf("expected context inventory rows")
		}
		first := rowToMap(gp.rows[0])
		if first["id"] != "fixture-alpha-1" {
			t.Fatalf("id = %#v, want fixture-alpha-1; row=%#v", first["id"], first)
		}
		if first["annotation_count"] != int64(1) && first["annotation_count"] != 1 {
			t.Fatalf("annotation_count = %#v, want 1; row=%#v", first["annotation_count"], first)
		}
		if first["handover_count"] != int64(2) && first["handover_count"] != 2 {
			t.Fatalf("handover_count = %#v, want 2; row=%#v", first["handover_count"], first)
		}
		if first["spawned_agent_count"] != int64(1) && first["spawned_agent_count"] != 1 {
			t.Fatalf("spawned_agent_count = %#v, want 1; row=%#v", first["spawned_agent_count"], first)
		}
	})

	t.Run("phase3 annotation and spawned-agent cookbook commands use new tables", func(t *testing.T) {
		for _, tc := range []struct {
			path string
			key  string
			want interface{}
		}{
			{path: "analysis/phase3-cookbook/annotation-risk-matrix", key: "category", want: "risk"},
			{path: "analysis/phase3-cookbook/spawned-agent-audit", key: "spawned_agent_type", want: "codex"},
		} {
			command := catalog.ByPath[tc.path]
			if command == nil {
				t.Fatalf("catalog missing %s command", tc.path)
			}
			glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
			if err != nil {
				t.Fatalf("NewMinitraceCatalogGlazeCommand(%s) returned error: %v", tc.path, err)
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
				t.Fatalf("ParseCommandValues(%s) returned error: %v", tc.path, err)
			}
			gp := &captureProcessor{}
			if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err != nil {
				t.Fatalf("RunIntoGlazeProcessor(%s) returned error: %v", tc.path, err)
			}
			if len(gp.rows) == 0 {
				t.Fatalf("expected rows from %s", tc.path)
			}
			first := rowToMap(gp.rows[0])
			if first[tc.key] != tc.want {
				t.Fatalf("%s row[%s] = %#v, want %#v; row=%#v", tc.path, tc.key, first[tc.key], tc.want, first)
			}
		}
	})

	t.Run("report cookbook commands produce raw SQL analysis rows", func(t *testing.T) {
		for _, tc := range []struct {
			path string
			key  string
			want interface{}
		}{
			{path: "analysis/report-cookbook/session-inventory", key: "id", want: "fixture-beta-1"},
			{path: "analysis/report-cookbook/tool-risk-matrix", key: "risk_label", want: "annotated-risk"},
			{path: "analysis/report-cookbook/file-heatmap", key: "operation_type", want: "modify"},
			{path: "analysis/report-cookbook/prompt-instruction-audit", key: "mentions_tests", want: int64(1)},
			{path: "analysis/report-cookbook/turn-timeline", key: "kind", want: "turn"},
		} {
			command := catalog.ByPath[tc.path]
			if command == nil {
				t.Fatalf("catalog missing %s command", tc.path)
			}
			glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
			if err != nil {
				t.Fatalf("NewMinitraceCatalogGlazeCommand(%s) returned error: %v", tc.path, err)
			}
			parsedValues, err := runner.ParseCommandValues(glazeCommand, runner.WithValuesForSections(map[string]map[string]interface{}{
				"filters": {
					"limit": 10,
				},
				QueryRuntimeSectionSlug: {
					"archive-glob": []string{archiveGlob},
				},
			}))
			if err != nil {
				t.Fatalf("ParseCommandValues(%s) returned error: %v", tc.path, err)
			}
			gp := &captureProcessor{}
			if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err != nil {
				t.Fatalf("RunIntoGlazeProcessor(%s) returned error: %v", tc.path, err)
			}
			if len(gp.rows) == 0 {
				t.Fatalf("expected rows from %s", tc.path)
			}
			first := rowToMap(gp.rows[0])
			if first[tc.key] != tc.want {
				t.Fatalf("%s row[%s] = %#v, want %#v; row=%#v", tc.path, tc.key, first[tc.key], tc.want, first)
			}
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

func stringPtr(value string) *string { return &value }
