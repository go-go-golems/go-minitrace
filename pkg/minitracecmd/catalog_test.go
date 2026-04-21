package minitracecmd

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"
)

func TestLoadCatalog_LoadsSQLAndAlias(t *testing.T) {
	catalog, err := LoadCatalog([]SourceRoot{{
		Name: "embedded",
		FS: fstest.MapFS{
			"queries/session-list.sql": &fstest.MapFile{Data: []byte(`/* sqleton
name: session-list
short: List sessions
*/
SELECT * FROM sessions_base
`)},
			"queries/codex-session-list.alias.yaml": &fstest.MapFile{Data: []byte(`name: codex-session-list
aliasFor: session-list
flags:
  framework:
    - codex
`)},
		},
		RootDir:  "queries",
		Readonly: true,
	}})
	if err != nil {
		t.Fatalf("LoadCatalog returned error: %v", err)
	}

	if len(catalog.Commands) != 2 {
		t.Fatalf("len(Commands) = %d, want 2", len(catalog.Commands))
	}
	if _, ok := catalog.ByName["session-list"]; !ok {
		t.Fatalf("ByName missing session-list verb")
	}
	alias := catalog.ByPath["codex-session-list.alias.yaml"]
	if alias == nil {
		t.Fatalf("ByPath missing alias entry")
	}
	if alias.Kind != MinitraceCommandAlias {
		t.Fatalf("alias.Kind = %q, want %q", alias.Kind, MinitraceCommandAlias)
	}
}

func TestLoadCatalog_FirstRootWinsOnDuplicatePath(t *testing.T) {
	catalog, err := LoadCatalog([]SourceRoot{
		{
			Name: "first",
			FS: fstest.MapFS{
				"queries/session-list.sql": &fstest.MapFile{Data: []byte(`/* sqleton
name: session-list
short: First version
*/
SELECT 1
`)},
			},
			RootDir:  "queries",
			Readonly: true,
		},
		{
			Name: "second",
			FS: fstest.MapFS{
				"queries/session-list.sql": &fstest.MapFile{Data: []byte(`/* sqleton
name: session-list
short: Second version
*/
SELECT 2
`)},
			},
			RootDir:  "queries",
			Readonly: true,
		},
	})
	if err != nil {
		t.Fatalf("LoadCatalog returned error: %v", err)
	}

	cmd := catalog.ByPath["session-list.sql"]
	if cmd == nil {
		t.Fatalf("missing command for duplicate path test")
	}
	if cmd.Short != "First version" {
		t.Fatalf("Short = %q, want First version", cmd.Short)
	}
	if cmd.SourceRoot != "first" {
		t.Fatalf("SourceRoot = %q, want first", cmd.SourceRoot)
	}
}

func TestLoadCatalog_AliasTargetMustExist(t *testing.T) {
	_, err := LoadCatalog([]SourceRoot{{
		Name: "embedded",
		FS: fstest.MapFS{
			"queries/missing.alias.yaml": &fstest.MapFile{Data: []byte(`name: missing
aliasFor: does-not-exist
`)},
		},
		RootDir:  "queries",
		Readonly: true,
	}})
	if !errors.Is(err, ErrAliasTargetNotFound) {
		t.Fatalf("error = %v, want ErrAliasTargetNotFound", err)
	}
}

func TestLoadCatalog_DerivesFolderAndPath(t *testing.T) {
	catalog, err := LoadCatalog([]SourceRoot{{
		Name: "embedded",
		FS: fstest.MapFS{
			"queries/core/session-list.sql": &fstest.MapFile{Data: []byte(`/* sqleton
name: session-list
short: List sessions
*/
SELECT * FROM sessions_base
`)},
		},
		RootDir:  "queries",
		Readonly: true,
	}})
	if err != nil {
		t.Fatalf("LoadCatalog returned error: %v", err)
	}

	cmd := catalog.ByPath["core/session-list.sql"]
	if cmd == nil {
		t.Fatalf("missing derived path entry")
	}
	if cmd.Folder != "core" {
		t.Fatalf("Folder = %q, want core", cmd.Folder)
	}
	if cmd.Path != "core/session-list.sql" {
		t.Fatalf("Path = %q, want core/session-list.sql", cmd.Path)
	}
	if cmd.SourcePath != "queries/core/session-list.sql" {
		t.Fatalf("SourcePath = %q, want queries/core/session-list.sql", cmd.SourcePath)
	}
}

func TestLoadCatalog_LoadsJSShowcaseTestdata(t *testing.T) {
	root := jsShowcaseTestdataRoot(t)
	catalog, err := LoadCatalog([]SourceRoot{{
		Name:     "showcase",
		FS:       os.DirFS(root),
		RootDir:  ".",
		Readonly: true,
	}})
	if err != nil {
		t.Fatalf("LoadCatalog returned error: %v", err)
	}

	wantPaths := []string{
		"overview/session-tools/session-list",
		"overview/session-tools/framework-share",
		"overview/runtime-playground/show-context",
		"overview/runtime-playground/build-synthetic-rows",
		"overview/async-tools/delayed-summary",
		"overview/async-tools/top-session-cards",
		"analysis/workspace-lab/workspace-scoreboard",
		"analysis/workspace-lab/workspace-session-highlights",
		"analysis/tool-intelligence/toolbox-overview",
		"analysis/tool-intelligence/tool-pair-matrix",
		"analysis/session-architectures/session-shape-ranker",
		"analysis/session-architectures/session-spotlights",
		"analysis/aliases/focus-top-workspaces.alias.yaml",
		"analysis/aliases/core-tool-pairs.alias.yaml",
		"analysis/aliases/heavy-session-shapes.alias.yaml",
	}
	for _, path := range wantPaths {
		if catalog.ByPath[path] == nil {
			t.Fatalf("catalog missing JS showcase command path %q", path)
		}
	}

	if catalog.ByPath["overview/lib/transforms"] != nil {
		t.Fatalf("overview helper module unexpectedly registered as command")
	}
	if catalog.ByPath["analysis/lib/cookbook"] != nil {
		t.Fatalf("analysis helper module unexpectedly registered as command")
	}
}

func TestLoadCatalog_LoadsMixedSQLJSShowcaseTestdata(t *testing.T) {
	root := mixedShowcaseTestdataRoot(t)
	catalog, err := LoadCatalog([]SourceRoot{{
		Name:     "mixed-showcase",
		FS:       os.DirFS(root),
		RootDir:  ".",
		Readonly: true,
	}})
	if err != nil {
		t.Fatalf("LoadCatalog returned error: %v", err)
	}

	wantPaths := []string{
		"overview/framework-summary.sql",
		"overview/session-tools/session-list",
		"overview/session-tools/framework-share",
		"overview/aliases/codex-framework-summary.alias.yaml",
		"analysis/raw-workspace-stats.sql",
		"analysis/workspace-lab/workspace-scoreboard",
		"analysis/aliases/top-workspaces.alias.yaml",
		"analysis/aliases/pi-raw-workspaces.alias.yaml",
	}
	for _, path := range wantPaths {
		if catalog.ByPath[path] == nil {
			t.Fatalf("catalog missing mixed showcase path %q", path)
		}
	}

	if catalog.ByPath["overview/lib/transforms"] != nil {
		t.Fatalf("mixed overview helper module unexpectedly registered as command")
	}
	if catalog.ByPath["analysis/lib/cookbook"] != nil {
		t.Fatalf("mixed analysis helper module unexpectedly registered as command")
	}
}

func jsShowcaseTestdataRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "testdata", "query-repositories", "js-showcase"))
}

func mixedShowcaseTestdataRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "testdata", "query-repositories", "mixed-sql-js-showcase"))
}
