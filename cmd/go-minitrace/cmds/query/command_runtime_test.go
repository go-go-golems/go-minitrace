package query

import (
	"testing"

	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
)

func TestNewMinitraceCatalogGlazeCommand_SeparatesRuntimeSection(t *testing.T) {
	catalog, err := minitracecmd.LoadEmbeddedCatalog()
	if err != nil {
		t.Fatalf("LoadEmbeddedCatalog returned error: %v", err)
	}

	command := catalog.ByName["session-list"]
	if command == nil {
		t.Fatalf("embedded catalog missing session-list command")
	}

	glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
	if err != nil {
		t.Fatalf("NewMinitraceCatalogGlazeCommand returned error: %v", err)
	}

	if _, ok := glazeCommand.GetSection(QueryRuntimeSectionSlug); !ok {
		t.Fatalf("expected %q section to be present", QueryRuntimeSectionSlug)
	}

	defaults := glazeCommand.GetDefaultFlags()
	if _, ok := defaults.Get("framework"); !ok {
		t.Fatalf("default section missing command flag framework")
	}
	if _, ok := defaults.Get("archive-glob"); ok {
		t.Fatalf("default section should not contain runtime flag archive-glob")
	}
}
