package validate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ValidAnnotationCategories is the set of known annotation categories.
var ValidAnnotationCategories = map[string]bool{
	"observation":       true,
	"ai-failure":        true,
	"user-error":        true,
	"environment-issue": true,
	"success":           true,
	"question":          true,
	"to-discuss":        true,
	"to-improve":        true,
}

// ValidClassificationLevels is the ordered set of classification levels.
// De-escalation (going from more to less restrictive) is not allowed.
var ValidClassificationLevels = []string{
	"public",
	"internal",
	"confidential",
	"customer-confidential",
}

// ValidAnnotationScopeTypes is the set of known scope types.
var ValidAnnotationScopeTypes = map[string]bool{
	"session":   true,
	"turn":      true,
	"tool_call": true,
}

type Result struct {
	Path  string
	Valid bool
	Error string
}

func ValidatePath(path string, recursive bool) ([]Result, error) {
	targets, err := collectTargets(path, recursive)
	if err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(targets))
	for _, target := range targets {
		result := validateFile(target)
		results = append(results, result)
	}
	return results, nil
}

// validateFile reads a JSON file, checks JSON syntax, and validates the
// minitrace session schema including annotations if present.
func validateFile(path string) Result {
	result := Result{Path: path, Valid: true}

	data, err := os.ReadFile(path)
	if err != nil {
		result.Valid = false
		result.Error = err.Error()
		return result
	}

	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		result.Valid = false
		result.Error = fmt.Sprintf("JSON parse error: %s", err.Error())
		return result
	}

	// Validate minitrace session structure.
	session, ok := payload.(map[string]any)
	if !ok {
		// Not a minitrace session (might be a manifest file) — skip semantic validation.
		return result
	}

	if anns, ok := session["annotations"]; ok {
		if errs := validateAnnotations(anns); len(errs) > 0 {
			result.Valid = false
			result.Error = strings.Join(errs, "; ")
			return result
		}
	}

	return result
}

// validateAnnotations checks the annotations field and returns a list of error
// strings. Returns nil if valid.
func validateAnnotations(field any) []string {
	if field == nil {
		return nil // annotations: null is valid (no annotations)
	}

	anns, ok := field.([]any)
	if !ok {
		return []string{fmt.Sprintf("annotations must be an array, got %T", field)}
	}

	var errs []string
	for i, item := range anns {
		ann, ok := item.(map[string]any)
		if !ok {
			errs = append(errs, fmt.Sprintf("annotations[%d]: must be an object", i))
			continue
		}

		if innerErr := validateAnnotation(ann); innerErr != "" {
			errs = append(errs, fmt.Sprintf("annotations[%d]: %s", i, innerErr))
		}
	}
	return errs
}

func validateAnnotation(ann map[string]any) string {
	// id: required string
	id, _ := ann["id"].(string)
	if id == "" {
		return `annotation missing required field "id"`
	}

	// timestamp: required string (ISO 8601)
	ts, _ := ann["timestamp"].(string)
	if ts == "" {
		return `annotation missing required field "timestamp"`
	}

	// annotator: required string
	annotator, _ := ann["annotator"].(string)
	if annotator == "" {
		return `annotation missing required field "annotator"`
	}

	// scope: required object
	scope, ok := ann["scope"].(map[string]any)
	if !ok {
		return `annotation missing required field "scope"`
	}
	scopeType, _ := scope["type"].(string)
	if !ValidAnnotationScopeTypes[scopeType] {
		return fmt.Sprintf(`annotation.scope.type %q is not valid (must be one of: session, turn, tool_call)`, scopeType)
	}
	_, hasTarget := scope["target_id"]
	if !hasTarget {
		return `annotation.scope missing required field "target_id"`
	}

	// content: required object
	content, ok := ann["content"].(map[string]any)
	if !ok {
		return `annotation missing required field "content"`
	}
	category, _ := content["category"].(string)
	if !ValidAnnotationCategories[category] {
		return fmt.Sprintf(`annotation.content.category %q is not a known category`, category)
	}
	title, _ := content["title"].(string)
	if title == "" {
		return `annotation.content missing required field "title"`
	}

	// tags: optional array of strings (inside content)
	if tags, ok := content["tags"].([]any); ok {
		for i, t := range tags {
			if _, ok := t.(string); !ok {
				return fmt.Sprintf("annotation.content.tags[%d]: must be a string", i)
			}
		}
	}

	// taxonomy_mappings: optional object
	if taxMaps, ok := ann["taxonomy_mappings"].(map[string]any); ok {
		for _, key := range []string{"minitrace", "mast", "toolemu"} {
			if arr, ok := taxMaps[key].([]any); ok {
				for i, v := range arr {
					if _, ok := v.(string); !ok {
						return fmt.Sprintf("annotation.taxonomy_mappings.%s[%d]: must be a string", key, i)
					}
				}
			}
		}
	}

	// classification: optional string, must be a known level
	if classif, ok := ann["classification"].(string); ok && classif != "" {
		if !isValidClassification(classif) {
			return fmt.Sprintf(`annotation.classification %q is not a known level`, classif)
		}
	}

	return ""
}

// isValidClassification returns true if the classification is a known level.
func isValidClassification(classif string) bool {
	for _, valid := range ValidClassificationLevels {
		if classif == valid {
			return true
		}
	}
	return false
}

func collectTargets(path string, recursive bool) ([]string, error) {
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return []string{path}, nil
	}

	targets := []string{}
	if recursive {
		err = filepath.WalkDir(path, func(current string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if isJSONFile(current) {
				targets = append(targets, current)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			current := filepath.Join(path, entry.Name())
			if isJSONFile(current) {
				targets = append(targets, current)
			}
		}
	}

	sort.Strings(targets)
	return targets, nil
}

func isJSONFile(path string) bool {
	return strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".minitrace.json")
}
