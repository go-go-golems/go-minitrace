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

func writePiFixture(t *testing.T, path, sessionID string) {
	t.Helper()
	content := `{"type":"session","id":"` + sessionID + `","version":3,"timestamp":"2026-06-10T12:00:00Z","cwd":"/tmp/project"}
{"type":"message","message":{"role":"user","timestamp":"2026-06-10T12:00:01Z","content":[{"type":"text","text":"Please inspect the repo."}]}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
