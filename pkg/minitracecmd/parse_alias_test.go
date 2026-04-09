package minitracecmd

import (
	"errors"
	"testing"
)

func TestParseAliasSpec_ValidAlias(t *testing.T) {
	contents := []byte(`name: codex-framework-summary
short: Run framework summary for codex
aliasFor: framework-summary
flags:
  framework:
    - codex
  limit: 25
`)

	spec, err := ParseAliasSpec("queries/codex-framework-summary.alias.yaml", contents)
	if err != nil {
		t.Fatalf("ParseAliasSpec returned error: %v", err)
	}

	if spec.Kind != MinitraceCommandAlias {
		t.Fatalf("Kind = %q, want %q", spec.Kind, MinitraceCommandAlias)
	}
	if spec.Name != "codex-framework-summary" {
		t.Fatalf("Name = %q, want codex-framework-summary", spec.Name)
	}
	if spec.Short != "Run framework summary for codex" {
		t.Fatalf("Short = %q, want expected short description", spec.Short)
	}
	if spec.AliasFor != "framework-summary" {
		t.Fatalf("AliasFor = %q, want framework-summary", spec.AliasFor)
	}
	if spec.AliasFlags == nil {
		t.Fatalf("AliasFlags should not be nil")
	}
	if got := spec.AliasFlags["limit"]; got != 25 {
		t.Fatalf("AliasFlags[limit] = %#v, want 25", got)
	}
}

func TestParseAliasSpec_MissingAliasFor(t *testing.T) {
	contents := []byte(`name: codex-framework-summary
flags:
  framework:
    - codex
`)

	_, err := ParseAliasSpec("queries/codex-framework-summary.alias.yaml", contents)
	if !errors.Is(err, ErrMissingAliasTarget) {
		t.Fatalf("error = %v, want ErrMissingAliasTarget", err)
	}
}

func TestParseAliasSpec_MissingName(t *testing.T) {
	contents := []byte(`aliasFor: framework-summary
flags:
  framework:
    - codex
`)

	_, err := ParseAliasSpec("queries/codex-framework-summary.alias.yaml", contents)
	if !errors.Is(err, ErrMissingName) {
		t.Fatalf("error = %v, want ErrMissingName", err)
	}
}
