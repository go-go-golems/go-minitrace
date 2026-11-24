package query

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = old
	payload, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return string(payload), runErr
}

func TestContextWindowUsesCompactionMarker(t *testing.T) {
	setIsolatedConfigHome(t)
	transcript, err := filepath.Abs("../../../../testdata/claude-code/projects/-tmp-demo/11111111-1111-4111-8111-111111111111.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	for method, want := range map[string]struct {
		boundary float64
		source   string
	}{
		"auto":        {3, "compact-boundary"},
		"marker-only": {3, "compact-boundary"},
	} {
		root, err := NewCommandsCommand(nil)
		if err != nil {
			t.Fatal(err)
		}
		root.SetArgs([]string{"history", "context-window", "--archive-glob", transcript, "--session", "11111111-1111-4111-8111-111111111111", "--turn", "5", "--boundary-method", method, "--output", "json"})
		output, err := captureStdout(t, root.Execute)
		if err != nil {
			t.Fatalf("%s: %v", method, err)
		}
		var rows []map[string]any
		if err := json.Unmarshal([]byte(output), &rows); err != nil || len(rows) != 1 {
			t.Fatalf("%s: decode %q: %v", method, output, err)
		}
		if rows[0]["boundary_turn"] != want.boundary || rows[0]["boundary_source"] != want.source {
			t.Fatalf("%s: boundary = %v/%v, want %v/%v", method, rows[0]["boundary_turn"], rows[0]["boundary_source"], want.boundary, want.source)
		}
	}
}
