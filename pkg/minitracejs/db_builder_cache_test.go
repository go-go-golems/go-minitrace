package minitracejs

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func piSessionJSONL(sessionID, userText string) string {
	return fmt.Sprintf(`{"type":"session","id":%q,"version":3,"timestamp":"2026-06-10T12:00:00Z","cwd":"/tmp/project"}
{"type":"message","message":{"role":"user","timestamp":"2026-06-10T12:00:01Z","content":[{"type":"text","text":%q}]}}`, sessionID, userText)
}

func buildDiskCachedDB(t *testing.T, cacheDir, sourceName, content string, cacheMaxBytes int64) string {
	t.Helper()
	builder := NewDBBuilder(context.Background())
	builder.cacheMode = "disk"
	builder.cacheDir = cacheDir
	builder.cacheMaxBytes = cacheMaxBytes
	builder.autoConvert = true
	builder.sources = append(builder.sources, dbSource{Kind: "content", Name: sourceName, Content: content})
	handle, err := builder.Build()
	if err != nil {
		t.Fatalf("Build(%s): %v", sourceName, err)
	}
	cachePath := handle.cachePath
	if err := handle.Close(); err != nil {
		t.Fatalf("Close(%s): %v", sourceName, err)
	}
	if cachePath == "" {
		t.Fatalf("expected a disk cache path for %s", sourceName)
	}
	return cachePath
}

func TestDiskCacheEvictsOldestFilesWhenOverSizeLimit(t *testing.T) {
	cacheDir := t.TempDir()

	firstPath := buildDiskCachedDB(t, cacheDir, "first.jsonl", piSessionJSONL("session-one", "Inspect main.go"), 1)
	if _, err := os.Stat(firstPath); err != nil {
		t.Fatalf("expected first cache file to survive its own install (never delete just-installed file): %v", err)
	}

	// Age the first cache file so mtime ordering is deterministic.
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(firstPath, old, old); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	secondPath := buildDiskCachedDB(t, cacheDir, "second.jsonl", piSessionJSONL("session-two", "Inspect util.go"), 1)
	if firstPath == secondPath {
		t.Fatalf("expected distinct cache keys for different content, both were %s", firstPath)
	}

	if _, err := os.Stat(firstPath); !os.IsNotExist(err) {
		t.Fatalf("expected older cache file %s to be evicted, stat err = %v", firstPath, err)
	}
	if _, err := os.Stat(secondPath); err != nil {
		t.Fatalf("expected newly installed cache file %s to exist: %v", secondPath, err)
	}
}

func TestDiskCacheMaxBytesExplicitOverrideAndFallback(t *testing.T) {
	builder := NewDBBuilder(context.Background())
	builder.cacheMaxBytes = 12345
	if got := builder.diskCacheMaxBytes(); got != 12345 {
		t.Fatalf("diskCacheMaxBytes() = %d, want 12345", got)
	}

	for _, value := range []int64{0, -5} {
		builder.cacheMaxBytes = value
		if got := builder.diskCacheMaxBytes(); got != defaultDiskCacheMaxBytes {
			t.Fatalf("diskCacheMaxBytes() with %d = %d, want default %d", value, got, defaultDiskCacheMaxBytes)
		}
	}
}
