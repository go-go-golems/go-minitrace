package minitracecmd

import (
	"errors"
	"strings"
	"testing"
)

func TestRenderCommand_RendersFrameworkFilter(t *testing.T) {
	catalog, err := LoadEmbeddedCatalog()
	if err != nil {
		t.Fatalf("LoadEmbeddedCatalog returned error: %v", err)
	}

	cmd := catalog.ByName["framework-summary"]
	if cmd == nil {
		t.Fatalf("embedded catalog missing framework-summary command")
	}

	sqlText, err := RenderCommand(cmd, RenderContext{
		TableName: "sessions_base",
		Values: map[string]any{
			"framework": []string{"codex", "claude"},
		},
	})
	if err != nil {
		t.Fatalf("RenderCommand returned error: %v", err)
	}

	if !strings.Contains(sqlText, "FROM sessions_base") {
		t.Fatalf("rendered SQL missing substituted table name: %s", sqlText)
	}
	if !strings.Contains(sqlText, "IN ('codex', 'claude')") {
		t.Fatalf("rendered SQL missing framework filter: %s", sqlText)
	}
}

func TestRenderCommand_SqlHelpersEscapeStrings(t *testing.T) {
	catalog, err := LoadEmbeddedCatalog()
	if err != nil {
		t.Fatalf("LoadEmbeddedCatalog returned error: %v", err)
	}

	cmd := catalog.ByName["session-list"]
	if cmd == nil {
		t.Fatalf("embedded catalog missing session-list command")
	}

	sqlText, err := RenderCommand(cmd, RenderContext{
		TableName: "sessions_base",
		Values: map[string]any{
			"title_like": "O'Reilly",
			"limit":      5,
		},
	})
	if err != nil {
		t.Fatalf("RenderCommand returned error: %v", err)
	}

	if !strings.Contains(sqlText, "title LIKE '%O''Reilly%'") {
		t.Fatalf("rendered SQL missing escaped LIKE clause: %s", sqlText)
	}
	if !strings.Contains(sqlText, "LIMIT 5;") {
		t.Fatalf("rendered SQL missing limit override: %s", sqlText)
	}
}

func TestResolveAliasCommand_MergesAliasDefaults(t *testing.T) {
	catalog, err := LoadEmbeddedCatalog()
	if err != nil {
		t.Fatalf("LoadEmbeddedCatalog returned error: %v", err)
	}

	alias := catalog.ByPath["overview/aliases/codex-framework-summary.alias.yaml"]
	if alias == nil {
		t.Fatalf("embedded catalog missing alias command")
	}

	target, values, err := ResolveAliasCommand(catalog, alias, map[string]any{"limit": 10})
	if err != nil {
		t.Fatalf("ResolveAliasCommand returned error: %v", err)
	}

	if target.Name != "framework-summary" {
		t.Fatalf("target.Name = %q, want framework-summary", target.Name)
	}
	framework, ok := values["framework"].([]any)
	if !ok || len(framework) != 1 || framework[0] != "codex" {
		t.Fatalf("merged framework defaults = %#v, want []any{\"codex\"}", values["framework"])
	}
	if got := values["limit"]; got != 10 {
		t.Fatalf("merged override limit = %#v, want 10", got)
	}
}

func TestResolveAliasCommand_OverrideWins(t *testing.T) {
	catalog, err := LoadEmbeddedCatalog()
	if err != nil {
		t.Fatalf("LoadEmbeddedCatalog returned error: %v", err)
	}

	alias := catalog.ByPath["overview/aliases/codex-framework-summary.alias.yaml"]
	target, values, err := ResolveAliasCommand(catalog, alias, map[string]any{"framework": []string{"claude"}})
	if err != nil {
		t.Fatalf("ResolveAliasCommand returned error: %v", err)
	}
	if target.Name != "framework-summary" {
		t.Fatalf("target.Name = %q, want framework-summary", target.Name)
	}
	framework, ok := values["framework"].([]string)
	if !ok || len(framework) != 1 || framework[0] != "claude" {
		t.Fatalf("merged framework override = %#v, want []string{\"claude\"}", values["framework"])
	}
}

func TestRenderCommand_RejectsAliasDirectly(t *testing.T) {
	catalog, err := LoadEmbeddedCatalog()
	if err != nil {
		t.Fatalf("LoadEmbeddedCatalog returned error: %v", err)
	}

	alias := catalog.ByPath["overview/aliases/codex-framework-summary.alias.yaml"]
	_, err = RenderCommand(alias, RenderContext{TableName: "sessions_base"})
	if !errors.Is(err, ErrCannotRenderAlias) {
		t.Fatalf("error = %v, want ErrCannotRenderAlias", err)
	}
}

func TestRenderCommand_RejectsInvalidTableName(t *testing.T) {
	catalog, err := LoadEmbeddedCatalog()
	if err != nil {
		t.Fatalf("LoadEmbeddedCatalog returned error: %v", err)
	}

	cmd := catalog.ByName["framework-summary"]
	_, err = RenderCommand(cmd, RenderContext{TableName: "sessions-base"})
	if !errors.Is(err, ErrInvalidTableName) {
		t.Fatalf("error = %v, want ErrInvalidTableName", err)
	}
}

func TestSQLIntIn_AcceptsJSONNumericSlices(t *testing.T) {
	got, err := sqlIntIn([]any{float64(1), float64(2), float64(42)})
	if err != nil {
		t.Fatalf("sqlIntIn returned error: %v", err)
	}
	if got != "1, 2, 42" {
		t.Fatalf("sqlIntIn = %q, want %q", got, "1, 2, 42")
	}
}

func TestSQLIntIn_RejectsNonIntegerFloats(t *testing.T) {
	_, err := sqlIntIn([]any{float64(1.5)})
	if err == nil {
		t.Fatalf("sqlIntIn expected error for non-integer float")
	}
}
