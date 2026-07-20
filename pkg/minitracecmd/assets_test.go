package minitracecmd

import "testing"

func TestLoadEmbeddedCatalog(t *testing.T) {
	catalog, err := LoadEmbeddedCatalog()
	if err != nil {
		t.Fatalf("LoadEmbeddedCatalog returned error: %v", err)
	}

	if len(catalog.Commands) < 9 {
		t.Fatalf("len(Commands) = %d, want at least 9", len(catalog.Commands))
	}
	if catalog.ByName["session-list"] == nil {
		t.Fatalf("embedded catalog missing session-list command")
	}
	if catalog.ByName["framework-summary"] == nil {
		t.Fatalf("embedded catalog missing framework-summary command")
	}
	if catalog.ByName["session-inventory"] == nil {
		t.Fatalf("embedded catalog missing nightly/session-inventory command")
	}
	if catalog.ByName["workspace-summary"] == nil {
		t.Fatalf("embedded catalog missing nightly/workspace-summary command")
	}
	if catalog.ByName["tool-breakdown"] == nil {
		t.Fatalf("embedded catalog missing nightly/tool-breakdown command")
	}
	if catalog.ByName["followup-candidates"] == nil {
		t.Fatalf("embedded catalog missing nightly/followup-candidates command")
	}
	if catalog.ByName["annotation-summary"] == nil {
		t.Fatalf("embedded catalog missing nightly/annotation-summary command")
	}
	if catalog.ByPath["nightly/session-inventory.sql"] == nil {
		t.Fatalf("embedded catalog missing nightly/session-inventory path")
	}
	if catalog.ByPath["nightly/workspace-summary.sql"] == nil {
		t.Fatalf("embedded catalog missing nightly/workspace-summary path")
	}
	if catalog.ByPath["nightly/tool-breakdown.sql"] == nil {
		t.Fatalf("embedded catalog missing nightly/tool-breakdown path")
	}
	if catalog.ByPath["nightly/followup-candidates.sql"] == nil {
		t.Fatalf("embedded catalog missing nightly/followup-candidates path")
	}
	if catalog.ByPath["nightly/annotation-summary.sql"] == nil {
		t.Fatalf("embedded catalog missing nightly/annotation-summary path")
	}
	if catalog.ByPath["overview/aliases/codex-framework-summary.alias.yaml"] == nil {
		t.Fatalf("embedded catalog missing codex-framework-summary alias")
	}
	if catalog.ByName["file-history"] == nil {
		t.Fatalf("embedded catalog missing history/file-history command")
	}
	if catalog.ByName["ticket-timeline"] == nil {
		t.Fatalf("embedded catalog missing history/ticket-timeline command")
	}
	if catalog.ByName["context-window"] == nil {
		t.Fatalf("embedded catalog missing history/context-window command")
	}
	if catalog.ByPath["history/file-history"] == nil {
		t.Fatalf("embedded catalog missing history/file-history path")
	}
	if catalog.ByPath["history/ticket-timeline"] == nil {
		t.Fatalf("embedded catalog missing history/ticket-timeline path")
	}
	if catalog.ByPath["history/context-window"] == nil {
		t.Fatalf("embedded catalog missing history/context-window path")
	}
	if catalog.ByName["session-activity"] == nil {
		t.Fatalf("embedded catalog missing overview/session-activity command")
	}
	if catalog.ByName["file-activity"] == nil {
		t.Fatalf("embedded catalog missing files/file-activity command")
	}
	// Both verbs used to live in one skill file, which would have registered
	// them at the doubled paths overview/session-activity/session-activity and
	// overview/session-activity/file-activity. They are split one verb per file
	// so the catalog collapses each to a flat path like every other command.
	if catalog.ByPath["overview/session-activity"] == nil {
		t.Fatalf("embedded catalog missing overview/session-activity path")
	}
	if catalog.ByPath["files/file-activity"] == nil {
		t.Fatalf("embedded catalog missing files/file-activity path")
	}
}
