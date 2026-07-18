package pi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverExtractsCwdAndStartedAtFromSessionRecord(t *testing.T) {
	root := t.TempDir()
	content := `{"type":"session","id":"pi-session-1","cwd":"/home/manuel/pi-project","timestamp":"2026-05-01T08:00:00Z"}
{"type":"message","message":{"role":"user","content":"hello"},"timestamp":"2026-05-01T08:00:01Z"}
`
	if err := os.WriteFile(filepath.Join(root, "project_pi-session-1.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing session fixture: %v", err)
	}

	locators, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if len(locators) != 1 {
		t.Fatalf("expected 1 locator, got %d", len(locators))
	}
	locator := locators[0]
	if locator.ID != "pi-session-1" {
		t.Fatalf("unexpected locator ID: %s", locator.ID)
	}
	if locator.Cwd != "/home/manuel/pi-project" {
		t.Fatalf("expected cwd from leading session record, got %q", locator.Cwd)
	}
	if locator.StartedAt != "2026-05-01T08:00:00Z" {
		t.Fatalf("expected started_at from leading session record, got %q", locator.StartedAt)
	}
}

func TestLastActivityAtUsesNestedMessageTimestamp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	content := `{"type":"session","timestamp":"2026-04-01T08:00:00Z"}
{"type":"message","message":{"timestamp":"2026-04-02T08:00:00Z"}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	latest, err := LastActivityAt(path)
	if err != nil {
		t.Fatalf("LastActivityAt returned error: %v", err)
	}
	if latest != "2026-04-02T08:00:00Z" {
		t.Fatalf("expected nested message timestamp, got %q", latest)
	}
}

func TestLocateSessionMissingPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.jsonl")
	_, err := LocateSession(missing)
	if err == nil {
		t.Fatalf("expected error for missing session path")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Fatalf("expected error to name missing path %s, got %v", missing, err)
	}
}
