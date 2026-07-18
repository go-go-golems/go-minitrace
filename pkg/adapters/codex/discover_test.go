package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverExtractsCwdAndStartedAtFromSessionMeta(t *testing.T) {
	root := t.TempDir()
	sessionsDir := filepath.Join(root, "sessions", "2026", "03", "29")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("creating sessions dir: %v", err)
	}
	content := `{"timestamp":"2026-03-29T11:00:00Z","type":"session_meta","payload":{"id":"session-a","cwd":"/home/manuel/project"}}
{"timestamp":"2026-03-29T11:00:01Z","type":"event_msg","payload":{"type":"user_message","message":"hi"}}
`
	if err := os.WriteFile(filepath.Join(sessionsDir, "rollout-session-a.jsonl"), []byte(content), 0o644); err != nil {
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
	if locator.FormatHint != "session-jsonl-v1" {
		t.Fatalf("expected session-jsonl-v1 format hint, got %s", locator.FormatHint)
	}
	if locator.Cwd != "/home/manuel/project" {
		t.Fatalf("expected cwd from session_meta payload, got %q", locator.Cwd)
	}
	if locator.StartedAt != "2026-03-29T11:00:00Z" {
		t.Fatalf("expected started_at from session_meta timestamp, got %q", locator.StartedAt)
	}
}

func TestLastActivityAtUsesPayloadTimestampFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	content := `{"timestamp":"2026-04-01T08:00:00Z","type":"session_meta"}
{"type":"event_msg","payload":{"timestamp":"2026-04-02T08:00:00Z"}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	latest, err := LastActivityAt(path)
	if err != nil {
		t.Fatalf("LastActivityAt returned error: %v", err)
	}
	if latest != "2026-04-02T08:00:00Z" {
		t.Fatalf("expected payload timestamp fallback, got %q", latest)
	}
}

func TestDiscoverLeavesHeaderEmptyForExecJSONL(t *testing.T) {
	root := t.TempDir()
	content := `{"type":"thread.started","thread_id":"thread-1"}
{"type":"turn.started"}
`
	if err := os.WriteFile(filepath.Join(root, "exec-run.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing exec fixture: %v", err)
	}

	locators, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if len(locators) != 1 {
		t.Fatalf("expected 1 locator, got %d", len(locators))
	}
	if locators[0].Cwd != "" || locators[0].StartedAt != "" {
		t.Fatalf("expected empty header for exec JSONL, got cwd=%q started_at=%q", locators[0].Cwd, locators[0].StartedAt)
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
	path := filepath.Join(dir, "rollout-explicit.jsonl")
	content := `{"timestamp":"2026-06-10T15:04:24Z","type":"session_meta","payload":{"id":"explicit","cwd":"/tmp/project"}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing session fixture: %v", err)
	}

	locator, err := LocateSession(path)
	if err != nil {
		t.Fatalf("LocateSession returned error: %v", err)
	}
	if locator.ID != "rollout-explicit" || locator.FormatHint != "session-jsonl-v1" {
		t.Fatalf("unexpected locator: %+v", locator)
	}
	if locator.Cwd != "/tmp/project" || locator.StartedAt != "2026-06-10T15:04:24Z" {
		t.Fatalf("expected header fields on explicit locator, got %+v", locator)
	}
}
