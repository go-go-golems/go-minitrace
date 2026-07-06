package claudecode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSessionID = "0123456789abcdef0123456789abcdef"

func TestDiscoverExtractsCwdAndStartedAtSkippingSnapshotRecord(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "-home-manuel-project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("creating project dir: %v", err)
	}
	// The first record is a file-history-snapshot without cwd; the second
	// record carries cwd and timestamp.
	content := `{"type":"file-history-snapshot","messageId":"snap-1","snapshot":{}}
{"type":"user","cwd":"/home/manuel/project","sessionId":"` + testSessionID + `","timestamp":"2026-04-01T09:30:00Z","message":{"role":"user","content":"hello"}}
`
	if err := os.WriteFile(filepath.Join(projectDir, testSessionID+".jsonl"), []byte(content), 0o644); err != nil {
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
	if locator.ID != testSessionID || locator.FormatHint != "jsonl-v2" {
		t.Fatalf("unexpected locator: %+v", locator)
	}
	if locator.Cwd != "/home/manuel/project" {
		t.Fatalf("expected cwd from first record carrying cwd, got %q", locator.Cwd)
	}
	if locator.StartedAt != "2026-04-01T09:30:00Z" {
		t.Fatalf("expected started_at from cwd record, got %q", locator.StartedAt)
	}
}

func TestDiscoverLeavesHeaderEmptyWhenNoCwdInHead(t *testing.T) {
	root := t.TempDir()
	content := `{"type":"file-history-snapshot","messageId":"snap-1","snapshot":{}}
`
	if err := os.WriteFile(filepath.Join(root, testSessionID+".jsonl"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing session fixture: %v", err)
	}

	locators, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if len(locators) != 1 {
		t.Fatalf("expected 1 locator, got %d", len(locators))
	}
	if locators[0].Cwd != "" || locators[0].StartedAt != "" {
		t.Fatalf("expected empty header, got cwd=%q started_at=%q", locators[0].Cwd, locators[0].StartedAt)
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

func TestLocateSessionPopulatesLocator(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, testSessionID+".jsonl")
	content := `{"type":"user","cwd":"/tmp/project","timestamp":"2026-05-02T08:00:00Z","message":{"role":"user","content":"hi"}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing session fixture: %v", err)
	}

	locator, err := LocateSession(path)
	if err != nil {
		t.Fatalf("LocateSession returned error: %v", err)
	}
	if locator.ID != testSessionID || locator.FormatHint != "jsonl-v2" {
		t.Fatalf("unexpected locator: %+v", locator)
	}
	if locator.Cwd != "/tmp/project" || locator.StartedAt != "2026-05-02T08:00:00Z" {
		t.Fatalf("expected header fields on explicit locator, got %+v", locator)
	}
}
