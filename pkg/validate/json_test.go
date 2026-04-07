package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAnnotations_Valid(t *testing.T) {
	session := `{
		"id": "sess-001",
		"annotations": [
			{
				"id": "ann-001",
				"timestamp": "2026-04-04T00:00:00Z",
				"annotator": "user",
				"scope": { "type": "session", "target_id": "sess-001" },
				"content": {
					"category": "ai-failure",
					"title": "Test failure",
					"tags": ["auth"],
					"detail": ""
				},
				"taxonomy_mappings": {
					"minitrace": ["F-AUT"],
					"mast": [],
					"toolemu": []
				},
				"classification": "internal"
			}
		]
	}`

	result := validateFileFromJSON(t, session)
	if !result.Valid {
		t.Errorf("expected valid, got error: %s", result.Error)
	}
}

func TestValidateAnnotations_MissingID(t *testing.T) {
	session := `{
		"id": "sess-001",
		"annotations": [
			{
				"timestamp": "2026-04-04T00:00:00Z",
				"annotator": "user",
				"scope": { "type": "session", "target_id": "sess-001" },
				"content": { "category": "ai-failure", "title": "T" }
			}
		]
	}`
	result := validateFileFromJSON(t, session)
	if result.Valid {
		t.Error("expected invalid, got valid")
	}
	if !strings.Contains(result.Error, "id") {
		t.Errorf("expected 'id' error, got: %s", result.Error)
	}
}

func TestValidateAnnotations_UnknownCategory(t *testing.T) {
	session := `{
		"id": "sess-001",
		"annotations": [
			{
				"id": "ann-001",
				"timestamp": "2026-04-04T00:00:00Z",
				"annotator": "user",
				"scope": { "type": "session", "target_id": "sess-001" },
				"content": { "category": "not-a-category", "title": "T" }
			}
		]
	}`
	result := validateFileFromJSON(t, session)
	if result.Valid {
		t.Error("expected invalid, got valid")
	}
	if !strings.Contains(result.Error, "not-a-category") {
		t.Errorf("expected 'not-a-category' error, got: %s", result.Error)
	}
}

func TestValidateAnnotations_UnknownClassification(t *testing.T) {
	session := `{
		"id": "sess-001",
		"annotations": [
			{
				"id": "ann-001",
				"timestamp": "2026-04-04T00:00:00Z",
				"annotator": "user",
				"scope": { "type": "session", "target_id": "sess-001" },
				"content": { "category": "ai-failure", "title": "T" },
				"classification": "top-secret"
			}
		]
	}`
	result := validateFileFromJSON(t, session)
	if result.Valid {
		t.Error("expected invalid, got valid")
	}
	if !strings.Contains(result.Error, "top-secret") {
		t.Errorf("expected 'top-secret' error, got: %s", result.Error)
	}
}

func TestValidateAnnotations_MissingTitle(t *testing.T) {
	session := `{
		"id": "sess-001",
		"annotations": [
			{
				"id": "ann-001",
				"timestamp": "2026-04-04T00:00:00Z",
				"annotator": "user",
				"scope": { "type": "session", "target_id": "sess-001" },
				"content": { "category": "ai-failure" }
			}
		]
	}`
	result := validateFileFromJSON(t, session)
	if result.Valid {
		t.Error("expected invalid, got valid")
	}
	if !strings.Contains(result.Error, "title") {
		t.Errorf("expected 'title' error, got: %s", result.Error)
	}
}

func TestValidateAnnotations_InvalidScopeType(t *testing.T) {
	session := `{
		"id": "sess-001",
		"annotations": [
			{
				"id": "ann-001",
				"timestamp": "2026-04-04T00:00:00Z",
				"annotator": "user",
				"scope": { "type": "bears", "target_id": "sess-001" },
				"content": { "category": "ai-failure", "title": "T" }
			}
		]
	}`
	result := validateFileFromJSON(t, session)
	if result.Valid {
		t.Error("expected invalid, got valid")
	}
	if !strings.Contains(result.Error, "bears") {
		t.Errorf("expected 'bears' error, got: %s", result.Error)
	}
}

func TestValidateAnnotations_NullAnnotations(t *testing.T) {
	session := `{"id": "sess-001", "annotations": null}`
	result := validateFileFromJSON(t, session)
	if !result.Valid {
		t.Errorf("expected valid for null annotations, got error: %s", result.Error)
	}
}

func TestValidateAnnotations_EmptyArray(t *testing.T) {
	session := `{"id": "sess-001", "annotations": []}`
	result := validateFileFromJSON(t, session)
	if !result.Valid {
		t.Errorf("expected valid for empty annotations, got error: %s", result.Error)
	}
}

func TestValidateAnnotations_NotAnArray(t *testing.T) {
	session := `{"id": "sess-001", "annotations": "not-an-array"}`
	result := validateFileFromJSON(t, session)
	if result.Valid {
		t.Error("expected invalid, got valid")
	}
	if !strings.Contains(result.Error, "must be an array") {
		t.Errorf("expected 'must be an array' error, got: %s", result.Error)
	}
}

func TestValidateAnnotations_TagsNotStrings(t *testing.T) {
	session := `{
		"id": "sess-001",
		"annotations": [
			{
				"id": "ann-001",
				"timestamp": "2026-04-04T00:00:00Z",
				"annotator": "user",
				"scope": { "type": "session", "target_id": "sess-001" },
				"content": { "category": "ai-failure", "title": "T", "tags": ["ok", 123, "also-ok"] }
			}
		]
	}`
	result := validateFileFromJSON(t, session)
	if result.Valid {
		t.Error("expected invalid, got valid")
	}
	if !strings.Contains(result.Error, "must be a string") {
		t.Errorf("expected 'must be a string' error, got: %s", result.Error)
	}
}

func TestValidatePath_ValidAnnotations(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "session.minitrace.json")
	validSession := `{
		"id": "sess-001",
		"annotations": [
			{
				"id": "ann-001",
				"timestamp": "2026-04-04T00:00:00Z",
				"annotator": "user",
				"scope": { "type": "session", "target_id": "sess-001" },
				"content": { "category": "ai-failure", "title": "Test" }
			}
		]
	}`
	if err := os.WriteFile(file, []byte(validSession), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	results, err := ValidatePath(file, false)
	if err != nil {
		t.Fatalf("ValidatePath: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Valid {
		t.Errorf("expected valid, got: %s", results[0].Error)
	}
}

func TestValidatePath_InvalidAnnotations(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "bad.minitrace.json")
	badSession := `{
		"id": "sess-001",
		"annotations": [
			{
				"id": "ann-001",
				"timestamp": "2026-04-04T00:00:00Z",
				"annotator": "user",
				"scope": { "type": "session", "target_id": "sess-001" },
				"content": { "category": "unknown-cat", "title": "Test" }
			}
		]
	}`
	if err := os.WriteFile(file, []byte(badSession), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	results, err := ValidatePath(file, false)
	if err != nil {
		t.Fatalf("ValidatePath: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Valid {
		t.Error("expected invalid, got valid")
	}
	if !strings.Contains(results[0].Error, "unknown-cat") {
		t.Errorf("expected 'unknown-cat' error, got: %s", results[0].Error)
	}
}

// validateFileFromJSON calls validateFile on a temp file with the given JSON content.
func validateFileFromJSON(t *testing.T, content string) Result {
	dir := t.TempDir()
	file := filepath.Join(dir, "session.minitrace.json")
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	results, err := ValidatePath(file, false)
	if err != nil {
		t.Fatalf("ValidatePath: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("no results")
	}
	return results[0]
}
