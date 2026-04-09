package minitracecmd

import (
	"errors"
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
