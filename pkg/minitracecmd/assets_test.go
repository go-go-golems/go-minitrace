package minitracecmd

import "testing"

func TestLoadEmbeddedCatalog(t *testing.T) {
	catalog, err := LoadEmbeddedCatalog()
	if err != nil {
		t.Fatalf("LoadEmbeddedCatalog returned error: %v", err)
	}

	if len(catalog.Commands) < 4 {
		t.Fatalf("len(Commands) = %d, want at least 4", len(catalog.Commands))
	}
	if catalog.ByName["session-list"] == nil {
		t.Fatalf("embedded catalog missing session-list command")
	}
	if catalog.ByName["framework-summary"] == nil {
		t.Fatalf("embedded catalog missing framework-summary command")
	}
	if catalog.ByPath["aliases/codex-framework-summary.alias.yaml"] == nil {
		t.Fatalf("embedded catalog missing codex-framework-summary alias")
	}
}
