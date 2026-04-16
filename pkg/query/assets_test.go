package query

import "testing"

func TestListPresetEntries_IncludesFoldersAndPaths(t *testing.T) {
	entries := ListPresetEntries()
	if len(entries) == 0 {
		t.Fatal("expected embedded preset entries")
	}

	seenSessionList := false
	seenToolFailures := false
	for _, entry := range entries {
		switch entry.Path {
		case "overview/session-list.sql":
			seenSessionList = true
			if entry.Name != "session-list" {
				t.Fatalf("session-list Name = %q, want session-list", entry.Name)
			}
			if entry.Folder != "overview" {
				t.Fatalf("session-list Folder = %q, want overview", entry.Folder)
			}
		case "tools/tool-failures.sql":
			seenToolFailures = true
			if entry.Name != "tool-failures" {
				t.Fatalf("tool-failures Name = %q, want tool-failures", entry.Name)
			}
			if entry.Folder != "tools" {
				t.Fatalf("tool-failures Folder = %q, want tools", entry.Folder)
			}
		}
	}
	if !seenSessionList || !seenToolFailures {
		t.Fatalf("expected overview/session-list.sql and tools/tool-failures.sql in %#v", entries)
	}
}

func TestResolvePresetSQLAcceptsPathIdentifier(t *testing.T) {
	sqlText, err := ResolvePresetSQL("overview/session-list", "custom_sessions")
	if err != nil {
		t.Fatalf("ResolvePresetSQL returned error: %v", err)
	}
	if sqlText == "" {
		t.Fatal("expected sql text")
	}
	if want := "FROM custom_sessions"; !contains(sqlText, want) {
		t.Fatalf("expected %q in sql text, got:\n%s", want, sqlText)
	}
}
