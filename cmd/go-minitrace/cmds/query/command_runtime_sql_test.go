package query

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-go-golems/glazed/pkg/cmds/runner"
	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
)

func TestMinitraceCatalogGlazeCommand_RunIntoGlazeProcessorExecutesEmbeddedSQLCommand(t *testing.T) {
	catalog, err := minitracecmd.LoadEmbeddedCatalog()
	if err != nil {
		t.Fatalf("LoadEmbeddedCatalog: %v", err)
	}
	command := catalog.ByName["session-list"]
	if command == nil {
		t.Fatalf("embedded catalog missing session-list command")
	}
	glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
	if err != nil {
		t.Fatalf("NewMinitraceCatalogGlazeCommand: %v", err)
	}

	archiveGlob := writeAdvancedFixtureArchive(t)
	parsedValues, err := runner.ParseCommandValues(glazeCommand, runner.WithValuesForSections(map[string]map[string]interface{}{
		QueryRuntimeSectionSlug: {
			"archive-glob": []string{archiveGlob},
		},
	}))
	if err != nil {
		t.Fatalf("ParseCommandValues: %v", err)
	}

	gp := &captureProcessor{}
	if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err != nil {
		t.Fatalf("RunIntoGlazeProcessor: %v", err)
	}
	if len(gp.rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(gp.rows))
	}
	first := rowToMap(gp.rows[0])
	if first["framework"] != "pi" {
		t.Fatalf("framework = %#v, want pi", first["framework"])
	}
	if id, ok := first["id"].(string); !ok || !strings.HasPrefix(id, "fixture-") {
		t.Fatalf("id = %#v, want fixture-* session id", first["id"])
	}
}

func TestMinitraceCatalogGlazeCommand_SQLCommandUsesSessionsBaseCompatView(t *testing.T) {
	catalog, err := minitracecmd.LoadCatalog([]minitracecmd.SourceRoot{{
		Name: "test-root",
		FS: fstest.MapFS{
			"queries/overview/legacy-session-list.sql": &fstest.MapFile{Data: []byte(`/* sqleton
name: legacy-session-list
short: Legacy sessions_base command
*/
SELECT
  id,
  environment->>'agent_framework' AS framework,
  CAST(metrics->>'tool_call_count' AS INT) AS tools
FROM {{TABLE_NAME}}
ORDER BY id;`)},
		},
		RootDir:  "queries",
		Readonly: true,
	}})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	command := catalog.ByName["legacy-session-list"]
	if command == nil {
		t.Fatalf("catalog missing legacy-session-list command")
	}
	glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
	if err != nil {
		t.Fatalf("NewMinitraceCatalogGlazeCommand: %v", err)
	}

	archiveGlob := writeAdvancedFixtureArchive(t)
	parsedValues, err := runner.ParseCommandValues(glazeCommand, runner.WithValuesForSections(map[string]map[string]interface{}{
		QueryRuntimeSectionSlug: {
			"archive-glob": []string{archiveGlob},
		},
	}))
	if err != nil {
		t.Fatalf("ParseCommandValues: %v", err)
	}

	gp := &captureProcessor{}
	if err := glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err != nil {
		t.Fatalf("RunIntoGlazeProcessor: %v", err)
	}
	if len(gp.rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(gp.rows))
	}
	first := rowToMap(gp.rows[0])
	if first["id"] != "fixture-alpha-1" {
		t.Fatalf("id = %#v, want fixture-alpha-1", first["id"])
	}
	if first["framework"] != "pi" {
		t.Fatalf("framework = %#v, want pi", first["framework"])
	}
}

func TestWarnDeprecatedRuntimeSettings(t *testing.T) {
	cases := []struct {
		name     string
		settings MinitraceQueryRuntimeSettings
		want     bool
	}{
		{name: "defaults stay silent", settings: MinitraceQueryRuntimeSettings{DBPath: ":memory:", TableName: "sessions_base"}, want: false},
		{name: "custom db-path warns", settings: MinitraceQueryRuntimeSettings{DBPath: "/tmp/foo.duckdb", TableName: "sessions_base"}, want: true},
		{name: "custom table-name warns", settings: MinitraceQueryRuntimeSettings{DBPath: ":memory:", TableName: "my_table"}, want: true},
		{name: "persist-loaded warns", settings: MinitraceQueryRuntimeSettings{DBPath: ":memory:", TableName: "sessions_base", PersistLoaded: true}, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := captureStderr(t, func() {
				warnDeprecatedRuntimeSettings(&tc.settings)
			})
			got := strings.Contains(output, "deprecated")
			if got != tc.want {
				t.Fatalf("warning emitted = %v, want %v (output: %q)", got, tc.want, output)
			}
		})
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	fn()
	_ = w.Close()
	os.Stderr = old
	payload, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stderr: %v", err)
	}
	return string(payload)
}
