package minitracecmd

import "strings"

type SourceKind int

const (
	SourceUnknown SourceKind = iota
	SourceSQLCommand
	SourceYAMLAlias
)

func DetectSourceKind(path string) SourceKind {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".alias.yaml"), strings.HasSuffix(lower, ".alias.yml"):
		return SourceYAMLAlias
	case strings.HasSuffix(lower, ".sql"):
		return SourceSQLCommand
	default:
		return SourceUnknown
	}
}
