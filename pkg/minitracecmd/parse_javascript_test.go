package minitracecmd

import (
	"errors"
	"testing"
	"testing/fstest"
)

func TestParseJSCommandSpecs(t *testing.T) {
	parsed, err := ParseJSCommandSpecs("overview/session-list.js", []byte(`
__section__("filters", {
  fields: {
    framework: { type: "stringList" },
    limit: { type: "int", default: 20 }
  }
});

function sessionList(filters, ctx) {
  return { ok: true, filters, ctx };
}

__verb__("sessionList", {
  name: "session-list",
  short: "List sessions",
  sections: ["filters"],
  output: "glaze"
});
`))
	if err != nil {
		t.Fatalf("ParseJSCommandSpecs returned error: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("len(parsed) = %d, want 1", len(parsed))
	}
	entry := parsed[0]
	if entry.Path != "overview/session-list.js:session-list" {
		t.Fatalf("Path = %q, want overview/session-list.js:session-list", entry.Path)
	}
	if entry.Spec == nil {
		t.Fatalf("Spec = nil")
	}
	if entry.Spec.Runtime != CommandRuntimeJS {
		t.Fatalf("Runtime = %q, want %q", entry.Spec.Runtime, CommandRuntimeJS)
	}
	if entry.Spec.JS == nil {
		t.Fatalf("JS metadata = nil")
	}
	if entry.Spec.JS.FunctionName != "sessionList" {
		t.Fatalf("FunctionName = %q, want sessionList", entry.Spec.JS.FunctionName)
	}
	if entry.Spec.Schema == nil || entry.Spec.Schema.Len() == 0 {
		t.Fatalf("Schema missing for scanned js command")
	}
}

func TestLoadCatalog_LoadsMultipleJSVerbsFromOneFile(t *testing.T) {
	catalog, err := LoadCatalog([]SourceRoot{{
		Name: "embedded",
		FS: fstest.MapFS{
			"queries/overview/multi.js": &fstest.MapFile{Data: []byte(`
function sessionList(filters) {
  return { ok: true, filters };
}

function frameworkSummary(filters) {
  return { ok: true, filters };
}

__verb__("sessionList", {
  name: "session-list",
  short: "List sessions",
  fields: {
    limit: { type: "int", default: 20 }
  }
});

__verb__("frameworkSummary", {
  name: "framework-summary",
  short: "Summarize frameworks",
  fields: {
    framework: { type: "stringList" }
  }
});
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
	if catalog.ByName["session-list"] == nil {
		t.Fatalf("ByName missing session-list")
	}
	if catalog.ByName["framework-summary"] == nil {
		t.Fatalf("ByName missing framework-summary")
	}
	if catalog.ByPath["overview/multi.js:session-list"] == nil {
		t.Fatalf("ByPath missing overview/multi.js:session-list")
	}
	if catalog.ByPath["overview/multi.js:framework-summary"] == nil {
		t.Fatalf("ByPath missing overview/multi.js:framework-summary")
	}
}

func TestLoadCatalog_RejectsDuplicateLogicalCommandPathAcrossSQLAndJS(t *testing.T) {
	_, err := LoadCatalog([]SourceRoot{{
		Name: "embedded",
		FS: fstest.MapFS{
			"queries/overview/session-list.sql": &fstest.MapFile{Data: []byte(`/* sqleton
name: session-list
short: SQL list
*/
SELECT 1
`)},
			"queries/overview/alternate.js": &fstest.MapFile{Data: []byte(`
function sessionList() {
  return { ok: true };
}

__verb__("sessionList", {
  name: "session-list",
  short: "JS list"
});
`)},
		},
		RootDir:  "queries",
		Readonly: true,
	}})
	if !errors.Is(err, ErrDuplicateCommandPath) {
		t.Fatalf("error = %v, want %v", err, ErrDuplicateCommandPath)
	}
}
