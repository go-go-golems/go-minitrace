package convert

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectSourceSessionsMergesFlagsAndListFile(t *testing.T) {
	dir := t.TempDir()
	listPath := filepath.Join(dir, "sessions.txt")
	content := `# comment line
/tmp/session-a.jsonl

  /tmp/session-b.jsonl
   # indented comment
`
	if err := os.WriteFile(listPath, []byte(content), 0o644); err != nil {
		t.Fatalf("writing list file: %v", err)
	}

	paths, err := collectSourceSessions([]string{"/tmp/explicit.jsonl", " "}, listPath)
	if err != nil {
		t.Fatalf("collectSourceSessions returned error: %v", err)
	}
	expected := []string{"/tmp/explicit.jsonl", "/tmp/session-a.jsonl", "/tmp/session-b.jsonl"}
	if len(paths) != len(expected) {
		t.Fatalf("expected %d paths, got %+v", len(expected), paths)
	}
	for i, path := range expected {
		if paths[i] != path {
			t.Fatalf("expected paths[%d]=%s, got %+v", i, path, paths)
		}
	}
}

func TestCollectSourceSessionsMissingListFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.txt")
	if _, err := collectSourceSessions(nil, missing); err == nil {
		t.Fatalf("expected error for missing source list file")
	}
}

func TestCollectSourceSessionsEmptyInputs(t *testing.T) {
	paths, err := collectSourceSessions(nil, "")
	if err != nil {
		t.Fatalf("collectSourceSessions returned error: %v", err)
	}
	if len(paths) != 0 {
		t.Fatalf("expected no paths, got %+v", paths)
	}
}
