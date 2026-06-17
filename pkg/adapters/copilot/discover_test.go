package copilot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverCopilotHomeSessionState(t *testing.T) {
	root := t.TempDir()
	writeSession(t, filepath.Join(root, "session-state", "b-session"), "workspace-id-b")
	writeSession(t, filepath.Join(root, "session-state", "a-session"), "")
	if err := os.MkdirAll(filepath.Join(root, "session-state", "scaffold-only"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "session-state", "scaffold-only", "workspace.yaml"), []byte("id: scaffold-only\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	locators, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if len(locators) != 2 {
		t.Fatalf("expected 2 locators, got %d: %#v", len(locators), locators)
	}
	if locators[0].ID != "a-session" {
		t.Fatalf("expected fallback directory ID for first sorted locator, got %q", locators[0].ID)
	}
	if locators[1].ID != "workspace-id-b" {
		t.Fatalf("expected workspace ID to win, got %q", locators[1].ID)
	}
	for _, locator := range locators {
		if locator.FormatHint != SourceFormatEvents {
			t.Fatalf("unexpected format hint: %q", locator.FormatHint)
		}
		if filepath.Base(locator.SourcePath) != "events.jsonl" {
			t.Fatalf("expected events.jsonl source path, got %q", locator.SourcePath)
		}
	}
}

func TestDiscoverDirectSessionStateAndSessionDir(t *testing.T) {
	root := t.TempDir()
	sessionState := filepath.Join(root, "session-state")
	sessionDir := filepath.Join(sessionState, "session-1")
	writeSession(t, sessionDir, "session-1")

	fromState, err := Discover(sessionState)
	if err != nil {
		t.Fatalf("Discover(session-state) returned error: %v", err)
	}
	if len(fromState) != 1 || fromState[0].ID != "session-1" {
		t.Fatalf("unexpected locators from session-state: %#v", fromState)
	}

	fromSession, err := Discover(sessionDir)
	if err != nil {
		t.Fatalf("Discover(session-dir) returned error: %v", err)
	}
	if len(fromSession) != 1 || fromSession[0].SourcePath != filepath.Join(sessionDir, "events.jsonl") {
		t.Fatalf("unexpected locators from direct session dir: %#v", fromSession)
	}
}

func writeSession(t *testing.T, dir, workspaceID string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	workspace := "cwd: /tmp/project\n"
	if workspaceID != "" {
		workspace = "id: " + workspaceID + "\n" + workspace
	}
	if err := os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte(workspace), 0o644); err != nil {
		t.Fatal(err)
	}
}
