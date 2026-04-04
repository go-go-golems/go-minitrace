package annotate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestSyncSessionDryRun(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "sess-001.minitrace.json")

	// Write a minimal session file.
	session := map[string]any{
		"id":       "sess-001",
		"metadata": "existing",
	}
	writeSession(t, filePath, session)

	anns := []minitrace.Annotation{
		{
			ID:        "ann-001",
			Timestamp: minitrace.FormatTimestamp(time.Now().UTC()),
			Annotator: "tester",
			Scope:     minitrace.AnnotationScope{Type: "session", TargetID: "sess-001"},
			Content:   minitrace.AnnotationContent{Category: "observation", Title: "Test"},
		},
	}

	err := SyncSession(ctx, filePath, anns, SyncOptions{DryRun: true})
	if err != nil {
		t.Fatalf("SyncSession dry-run: %v", err)
	}

	// File should be unchanged.
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var after map[string]any
	if err := json.Unmarshal(data, &after); err != nil {
		t.Fatalf("unmarshal after: %v", err)
	}
	if _, ok := after["annotations"]; ok {
		t.Errorf("dry-run should not modify file, but annotations field is present")
	}
}

func TestSyncSessionWrite(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "sess-002.minitrace.json")

	// Write a session with no annotations field.
	session := map[string]any{
		"id":         "sess-002",
		"schema_ver": "0.2.0",
	}
	writeSession(t, filePath, session)

	anns := []minitrace.Annotation{
		{
			ID:               "ann-002",
			Timestamp:        minitrace.FormatTimestamp(time.Now().UTC()),
			Annotator:        "tester",
			Scope:            minitrace.AnnotationScope{Type: "session", TargetID: "sess-002"},
			Content:          minitrace.AnnotationContent{Category: "ai-failure", Title: "Failure", Tags: []string{"auth"}},
			TaxonomyMappings: minitrace.TaxonomyMappings{Minitrace: []string{"F-AUT"}},
		},
	}

	err := SyncSession(ctx, filePath, anns, SyncOptions{DryRun: false})
	if err != nil {
		t.Fatalf("SyncSession: %v", err)
	}

	// Verify the file was updated.
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile after: %v", err)
	}
	var after map[string]any
	if err := json.Unmarshal(data, &after); err != nil {
		t.Fatalf("unmarshal after: %v", err)
	}
	annsField, ok := after["annotations"]
	if !ok {
		t.Fatalf("annotations field missing from file")
	}
	arr, ok := annsField.([]any)
	if !ok {
		t.Fatalf("annotations is not an array")
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(arr))
	}
	// Verify the annotation content.
	ann := arr[0].(map[string]any)
	if ann["id"] != "ann-002" {
		t.Errorf("annotation id = %v, want ann-002", ann["id"])
	}
}

func TestSyncSessionNilAnnotationsProducesEmptyArray(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "sess-003.minitrace.json")

	// Write a session that already has annotations: null.
	session := map[string]any{
		"id":          "sess-003",
		"annotations": nil,
	}
	writeSession(t, filePath, session)

	// Sync with nil annotations (session with no annotations).
	err := SyncSession(ctx, filePath, nil, SyncOptions{DryRun: false})
	if err != nil {
		t.Fatalf("SyncSession with nil: %v", err)
	}

	data, errRead := os.ReadFile(filePath)
	if errRead != nil {
		t.Fatalf("ReadFile: %v", errRead)
	}
	var after map[string]any
	if err := json.Unmarshal(data, &after); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	anns := after["annotations"]
	// Must be [] not null.
	b, _ := json.Marshal(anns)
	if string(b) != "[]" {
		t.Errorf("annotations = %s, want []", b)
	}
}

func TestSyncSessionOverwritesExistingAnnotations(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "sess-004.minitrace.json")

	// Write a session that already has annotations.
	session := map[string]any{
		"id": "sess-004",
		"annotations": []any{
			map[string]any{
				"id":    "old-001",
				"title": "Old annotation",
			},
		},
	}
	writeSession(t, filePath, session)

	newAnns := []minitrace.Annotation{
		{
			ID:        "new-001",
			Timestamp: minitrace.FormatTimestamp(time.Now().UTC()),
			Annotator: "tester",
			Scope:     minitrace.AnnotationScope{Type: "session", TargetID: "sess-004"},
			Content:   minitrace.AnnotationContent{Category: "observation", Title: "New"},
		},
	}

	err := SyncSession(ctx, filePath, newAnns, SyncOptions{DryRun: false})
	if err != nil {
		t.Fatalf("SyncSession overwrite: %v", err)
	}

	data, errRead := os.ReadFile(filePath)
	if errRead != nil {
		t.Fatalf("ReadFile: %v", errRead)
	}
	var after map[string]any
	if err := json.Unmarshal(data, &after); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	anns := after["annotations"].([]any)
	if len(anns) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(anns))
	}
	if anns[0].(map[string]any)["id"] != "new-001" {
		t.Errorf("old annotation still present")
	}
}

func TestSyncSessionFileNotFound(t *testing.T) {
	ctx := context.Background()
	err := SyncSession(ctx, "/nonexistent/sess.json", nil, SyncOptions{})
	if err == nil {
		t.Errorf("expected error for nonexistent file, got nil")
	}
}

// writeSession writes a session map to a JSON file.
func writeSession(t *testing.T, path string, session map[string]any) {
	t.Helper()
	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("marshaling session: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing session file: %v", err)
	}
}
