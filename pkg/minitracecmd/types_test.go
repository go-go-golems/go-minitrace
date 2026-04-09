package minitracecmd

import "testing"

func TestMinitraceCommandSpecValidateVerb(t *testing.T) {
	spec := &MinitraceCommandSpec{
		Name:  "session-list",
		Short: "List sessions",
		Query: "select 1",
		Kind:  MinitraceCommandVerb,
	}

	if err := spec.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
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
