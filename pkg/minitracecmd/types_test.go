package minitracecmd

import "testing"

func TestMinitraceCommandSpecValidateSQLVerb(t *testing.T) {
	spec := &MinitraceCommandSpec{
		Name:    "session-list",
		Short:   "List sessions",
		Runtime: CommandRuntimeSQL,
		Query:   "select 1",
		Kind:    MinitraceCommandVerb,
	}

	if err := spec.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestMinitraceCommandSpecValidateJSVerb(t *testing.T) {
	spec := &MinitraceCommandSpec{
		Name:    "session-list",
		Short:   "List sessions",
		Runtime: CommandRuntimeJS,
		JS: &JSCommandSpec{
			ModulePath:   "overview/session-list",
			FunctionName: "sessionList",
			OutputMode:   "glaze",
		},
		Kind: MinitraceCommandVerb,
	}

	if err := spec.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestMinitraceCommandSpecValidateVerbRejectsMultipleRuntimes(t *testing.T) {
	spec := &MinitraceCommandSpec{
		Name:  "session-list",
		Short: "List sessions",
		Query: "select 1",
		JS: &JSCommandSpec{
			ModulePath:   "overview/session-list",
			FunctionName: "sessionList",
			OutputMode:   "glaze",
		},
		Kind: MinitraceCommandVerb,
	}

	if err := spec.Validate(); err != ErrMultipleRuntimes {
		t.Fatalf("Validate returned %v, want %v", err, ErrMultipleRuntimes)
	}
}

func TestMinitraceCommandSpecValidateVerbRejectsMissingRuntime(t *testing.T) {
	spec := &MinitraceCommandSpec{
		Name:  "session-list",
		Short: "List sessions",
		Kind:  MinitraceCommandVerb,
	}

	if err := spec.Validate(); err != ErrMissingRuntime {
		t.Fatalf("Validate returned %v, want %v", err, ErrMissingRuntime)
	}
}

func TestMinitraceCommandSpecValidateAlias(t *testing.T) {
	spec := &MinitraceCommandSpec{
		Name:     "short",
		AliasFor: "session-list",
		Kind:     MinitraceCommandAlias,
	}

	if err := spec.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestDetectSourceKind(t *testing.T) {
	tests := []struct {
		path string
		want SourceKind
	}{
		{path: "queries/session-list.sql", want: SourceSQLCommand},
		{path: "queries/session-list.js", want: SourceJSCommand},
		{path: "queries/session-list.cjs", want: SourceJSCommand},
		{path: "queries/short.alias.yaml", want: SourceYAMLAlias},
		{path: "queries/short.alias.yml", want: SourceYAMLAlias},
		{path: "queries/readme.md", want: SourceUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := DetectSourceKind(tt.path); got != tt.want {
				t.Fatalf("DetectSourceKind(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
