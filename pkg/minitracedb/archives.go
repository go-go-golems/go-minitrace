package minitracedb

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ExpandArchiveGlobs resolves a list of archive glob patterns to a
// deduplicated list of absolute file paths, preserving glob order.
func ExpandArchiveGlobs(archiveGlobs []string) ([]string, error) {
	normalizedGlobs := normalizeArchiveGlobList(archiveGlobs)
	if len(normalizedGlobs) == 0 {
		return nil, fmt.Errorf("archive glob is required")
	}

	resolvedFiles := make([]string, 0)
	seenFiles := make(map[string]struct{})

	for _, archiveGlob := range normalizedGlobs {
		files, err := filepath.Glob(archiveGlob)
		if err != nil {
			return nil, fmt.Errorf("expanding archive glob %q: %w", archiveGlob, err)
		}
		for _, filePath := range files {
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return nil, fmt.Errorf("resolving absolute path for %s: %w", filePath, err)
			}
			if _, ok := seenFiles[absPath]; ok {
				continue
			}
			seenFiles[absPath] = struct{}{}
			resolvedFiles = append(resolvedFiles, absPath)
		}
	}

	if len(resolvedFiles) == 0 {
		return nil, fmt.Errorf("archive globs matched no files: %s", strings.Join(normalizedGlobs, ", "))
	}

	return resolvedFiles, nil
}

func normalizeArchiveGlobList(values []string) []string {
	ret := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ret = append(ret, value)
	}
	return ret
}
