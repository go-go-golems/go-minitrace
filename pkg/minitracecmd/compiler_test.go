package minitracecmd

import (
	"testing"

	fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
)

func TestCompiler_CompileVerb(t *testing.T) {
	flag := fields.New("verbose", fields.TypeBool, fields.WithHelp("Include extra rows"))

	cmd, err := (&Compiler{}).Compile(&MinitraceCommandSpec{
		Name:    "session-list",
		Short:   "List sessions",
		Runtime: CommandRuntimeSQL,
		Query:   "SELECT 1",
		Flags:   []*fields.Definition{flag},
		Tags:    []string{"analysis"},
		Kind:    MinitraceCommandVerb,
	}, CompileOptions{
		Folder:     "queries/core",
		Path:       "queries/core/session-list.sql",
		SourceRoot: "embedded",
		SourcePath: "queries/core/session-list.sql",
		Readonly:   true,
	})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	if cmd.Kind != MinitraceCommandVerb {
		t.Fatalf("Kind = %q, want %q", cmd.Kind, MinitraceCommandVerb)
	}
	if cmd.Path != "queries/core/session-list.sql" {
		t.Fatalf("Path = %q, want normalized path", cmd.Path)
	}
	if cmd.SourceRoot != "embedded" {
		t.Fatalf("SourceRoot = %q, want embedded", cmd.SourceRoot)
	}
	if cmd.Runtime != CommandRuntimeSQL {
		t.Fatalf("Runtime = %q, want %q", cmd.Runtime, CommandRuntimeSQL)
	}
	if !cmd.Readonly {
		t.Fatalf("Readonly = false, want true")
	}
	if len(cmd.Flags) != 1 {
		t.Fatalf("len(Flags) = %d, want 1", len(cmd.Flags))
	}
}

func TestCompiler_CompileAlias(t *testing.T) {
	cmd, err := (&Compiler{}).Compile(&MinitraceCommandSpec{
		Name:       "codex-framework-summary",
		AliasFor:   "framework-summary",
		AliasFlags: map[string]any{"framework": []any{"codex"}},
		Kind:       MinitraceCommandAlias,
	}, CompileOptions{
		Path:       "queries/core/codex-framework-summary.alias.yaml",
		SourceRoot: "embedded",
		SourcePath: "queries/core/codex-framework-summary.alias.yaml",
		Readonly:   true,
	})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	if cmd.Kind != MinitraceCommandAlias {
		t.Fatalf("Kind = %q, want %q", cmd.Kind, MinitraceCommandAlias)
	}
	if cmd.AliasFor != "framework-summary" {
		t.Fatalf("AliasFor = %q, want framework-summary", cmd.AliasFor)
	}
	if cmd.Query != "" {
		t.Fatalf("Query = %q, want empty string", cmd.Query)
	}
}

func TestCompiler_CompileJSVerb(t *testing.T) {
	cmd, err := (&Compiler{}).Compile(&MinitraceCommandSpec{
		Name:    "session-list",
		Short:   "List sessions",
		Runtime: CommandRuntimeJS,
		JS: &JSCommandSpec{
			ModulePath:   "overview/session-list",
			FunctionName: "sessionList",
			OutputMode:   "glaze",
		},
		Kind: MinitraceCommandVerb,
	}, CompileOptions{
		Folder:     "queries/core",
		Path:       "queries/core/session-list.js:session-list",
		SourceRoot: "embedded",
		SourcePath: "queries/core/session-list.js",
		Readonly:   true,
	})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	if cmd.Runtime != CommandRuntimeJS {
		t.Fatalf("Runtime = %q, want %q", cmd.Runtime, CommandRuntimeJS)
	}
	if cmd.JS == nil {
		t.Fatalf("JS = nil, want js metadata")
	}
	if cmd.JS.FunctionName != "sessionList" {
		t.Fatalf("FunctionName = %q, want sessionList", cmd.JS.FunctionName)
	}
}

func TestCompiler_NormalizesOptionalBoolFlags(t *testing.T) {
	flag := fields.New("verbose", fields.TypeBool, fields.WithHelp("Include extra rows"))
	if flag.Default != nil {
		t.Fatalf("test setup error: expected nil default before compile")
	}

	cmd, err := (&Compiler{}).Compile(&MinitraceCommandSpec{
		Name:    "session-list",
		Short:   "List sessions",
		Runtime: CommandRuntimeSQL,
		Query:   "SELECT 1",
		Flags:   []*fields.Definition{flag},
		Kind:    MinitraceCommandVerb,
	}, CompileOptions{})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	if cmd.Flags[0].Default == nil {
		t.Fatalf("compiled bool flag default is nil, want false")
	}
	if got := (*cmd.Flags[0].Default).(bool); got {
		t.Fatalf("compiled bool flag default = %v, want false", got)
	}
	if flag.Default != nil {
		t.Fatalf("original flag default was mutated")
	}
}
