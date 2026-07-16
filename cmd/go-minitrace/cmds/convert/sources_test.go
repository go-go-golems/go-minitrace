package convert

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestApplySourceFingerprintMakesStagedRerunsIdempotent(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.jsonl")
	if err := os.WriteFile(source, []byte("source bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	session := minitrace.BuildSessionSkeleton("session", "pi", "pi-v3", "test")
	if err := applySourceFingerprint(&session, source); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "output")
	first, err := minitrace.PublishSessionBatch([]*minitrace.Session{&session}, output, minitrace.CollisionError)
	if err != nil {
		t.Fatal(err)
	}
	second, err := minitrace.PublishSessionBatch([]*minitrace.Session{&session}, output, minitrace.CollisionError)
	if err != nil {
		t.Fatalf("unchanged rerun failed: %v", err)
	}
	if first[0].Status != minitrace.PublicationCreated || second[0].Status != minitrace.PublicationUnchanged {
		t.Fatalf("unexpected statuses: %s then %s", first[0].Status, second[0].Status)
	}
}

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

	paths, err := collectSourceSessions([]string{"/tmp/explicit.jsonl", "/tmp/session-a.jsonl", " "}, listPath)
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

func TestCollectSourceSessionsSortsRelativePaths(t *testing.T) {
	dir := t.TempDir()
	paths, err := collectSourceSessions([]string{filepath.Join(dir, "b.jsonl"), filepath.Join(dir, "a.jsonl")}, "")
	if err != nil {
		t.Fatalf("collectSourceSessions returned error: %v", err)
	}
	if got, want := paths, []string{filepath.Join(dir, "a.jsonl"), filepath.Join(dir, "b.jsonl")}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("paths = %+v, want %+v", got, want)
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
