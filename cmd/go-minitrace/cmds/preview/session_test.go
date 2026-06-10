package preview

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/minitracejs"
)

func TestDiscoverPreviewPathsLatestPi(t *testing.T) {
	dir := t.TempDir()
	older := filepath.Join(dir, "2026-06-10T10-00-00-000Z_old.jsonl")
	newer := filepath.Join(dir, "2026-06-10T11-00-00-000Z_new.jsonl")
	writePiFixture(t, older, "old-session")
	writePiFixture(t, newer, "new-session")
	now := time.Now()
	if err := os.Chtimes(older, now.Add(-time.Hour), now.Add(-time.Hour)); err != nil {
		t.Fatalf("set older time: %v", err)
	}
	if err := os.Chtimes(newer, now, now); err != nil {
		t.Fatalf("set newer time: %v", err)
	}

	paths, err := discoverPreviewPaths("pi", dir, 1)
	if err != nil {
		t.Fatalf("discoverPreviewPaths: %v", err)
	}
	if len(paths) != 1 || paths[0] != newer {
		t.Fatalf("expected newest path %s, got %+v", newer, paths)
	}
}

func TestDiscoverPreviewLocatorsPreservesFormatHint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exec-session.jsonl")
	writeCodexExecFixture(t, path)

	locators, err := discoverPreviewLocators("codex", dir, 1)
	if err != nil {
		t.Fatalf("discoverPreviewLocators: %v", err)
	}
	if len(locators) != 1 {
		t.Fatalf("expected one locator, got %+v", locators)
	}
	if locators[0].SourcePath != path || locators[0].FormatHint != "exec-jsonl-v1" {
		t.Fatalf("expected exec locator for %s, got %+v", path, locators[0])
	}
}

func TestPreviewSessionLocatorPreservesCodexFormatHint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exec-session.jsonl")
	writeCodexExecFixture(t, path)

	locators, err := discoverPreviewLocators("codex", dir, 1)
	if err != nil {
		t.Fatalf("discoverPreviewLocators: %v", err)
	}
	preview, err := previewSessionLocator("codex", locators[0], minitracejs.PreviewOptions{SampleLimit: 2, Privacy: "structural"})
	if err != nil {
		t.Fatalf("previewSessionLocator: %v", err)
	}
	if preview.Adapter != "codex" || preview.Format != "exec-jsonl-v1" {
		t.Fatalf("expected codex exec preview, got adapter=%q format=%q", preview.Adapter, preview.Format)
	}
	if preview.SessionID != "thread-1" {
		t.Fatalf("expected thread session id, got %q", preview.SessionID)
	}
	if preview.ToolCounts["exec_command"] != 1 {
		t.Fatalf("expected exec_command count, got %+v", preview.ToolCounts)
	}
}

func TestPreviewSessionPathUsesOptions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2026-06-10T10-00-00-000Z_session.jsonl")
	writePiFixture(t, path, "preview-session")

	preview, err := previewSessionPath(path, minitracejs.PreviewOptions{SampleLimit: 1, Privacy: "structural"})
	if err != nil {
		t.Fatalf("previewSessionPath: %v", err)
	}
	if preview.Adapter != "pi" || preview.Format != "pi-jsonl" {
		t.Fatalf("expected pi preview, got adapter=%q format=%q", preview.Adapter, preview.Format)
	}
	if len(preview.SampleTurns) != 1 {
		t.Fatalf("expected one sampled turn, got %d", len(preview.SampleTurns))
	}
	if preview.SampleTurns[0].Preview != "" {
		t.Fatalf("expected structural privacy to suppress turn preview, got %q", preview.SampleTurns[0].Preview)
	}
}

func writeCodexExecFixture(t *testing.T, path string) {
	t.Helper()
	content := `{"type":"thread.started","thread_id":"thread-1"}
{"type":"item.completed","item":{"id":"cmd-1","type":"command_execution","command":"go test ./...","aggregated_output":"ok","exit_code":0,"status":"completed","turn_id":"turn-1","source":"exec-runner","parsed_cmd":[{"type":"test","cmd":"go test ./..."}],"stdout":"ok","stderr":""}}
{"type":"item.completed","item":{"type":"agent_message","text":"Tests passed.","turn_id":"turn-1","phase":"commentary"}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func writePiFixture(t *testing.T, path, sessionID string) {
	t.Helper()
	content := `{"type":"session","id":"` + sessionID + `","version":3,"timestamp":"2026-06-10T12:00:00Z","cwd":"/tmp/project"}
{"type":"message","message":{"role":"user","timestamp":"2026-06-10T12:00:01Z","content":[{"type":"text","text":"Please inspect the repo."}]}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
