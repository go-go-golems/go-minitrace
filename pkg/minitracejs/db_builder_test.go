package minitracejs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
)

func TestNewArchiveQueryTargetDeduplicatesOverlappingRelativeGlobs(t *testing.T) {
	archiveRoot := t.TempDir()
	session := minitrace.BuildSessionSkeleton("overlap-session", "codex", "test", "test")
	turnIndex := 0
	filePath := "main.go"
	contentOrigin := "file"
	session.ToolCalls = []minitrace.ToolCall{
		minitrace.BuildToolCall(
			"tool-1",
			&turnIndex,
			nil,
			"read_file",
			"READ",
			&filePath,
			nil,
			map[string]any{"path": filePath},
			true,
			"package main",
			nil,
			nil,
			nil,
			nil,
			&contentOrigin,
			nil,
		),
	}
	if _, err := minitrace.WriteSession(&session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(archiveRoot); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restoring working directory: %v", err)
		}
	})

	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	handle, err := NewArchiveQueryTarget(context.Background(), []string{
		filepath.Join(".", "active", "*", "*.minitrace.json"),
		filepath.Join("active", "*", "*.minitrace.json"),
	}, minitracedb.DefaultQueryOptions())
	if err != nil {
		t.Fatalf("NewArchiveQueryTarget returned error: %v", err)
	}
	defer func() { _ = handle.Close() }()

	if len(handle.sources) != 1 {
		t.Fatalf("expected one deduplicated source, got %#v", handle.sources)
	}
	row, err := handle.QueryOne(context.Background(), `SELECT COUNT(*) AS n FROM files`)
	if err != nil {
		t.Fatalf("querying files count: %v", err)
	}
	if row["n"] != int64(1) {
		t.Fatalf("expected one materialized file operation, got %#v", row["n"])
	}
}
