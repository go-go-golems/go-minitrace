package validate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
		result := Result{Path: target, Valid: true}

		data, err := os.ReadFile(target)
		if err != nil {
			result.Valid = false
			result.Error = err.Error()
			results = append(results, result)
			continue
		}

		var payload any
		if err := json.Unmarshal(data, &payload); err != nil {
			result.Valid = false
			result.Error = err.Error()
		}
		results = append(results, result)
	}
	return results, nil
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
