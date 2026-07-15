package convert

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

// collectSourceSessions merges explicit --source-session paths with the
// contents of an optional --source-list file. The list file contains one
// session path per line; blank lines and lines starting with # are ignored.
// It normalizes, deduplicates, and sorts paths before conversion so equivalent
// invocation inputs produce the same source order.
func collectSourceSessions(sourceSessions []string, sourceListPath string) ([]string, error) {
	paths := make([]string, 0, len(sourceSessions))
	appendPath := func(path string) error {
		path = strings.TrimSpace(path)
		if path == "" {
			return nil
		}
		absolutePath, err := filepath.Abs(filepath.Clean(path))
		if err != nil {
			return errors.Wrapf(err, "normalizing source session %s", path)
		}
		paths = append(paths, absolutePath)
		return nil
	}
	for _, path := range sourceSessions {
		if err := appendPath(path); err != nil {
			return nil, err
		}
	}

	sourceListPath = strings.TrimSpace(sourceListPath)
	if sourceListPath != "" {
		data, err := os.ReadFile(sourceListPath)
		if err != nil {
			return nil, errors.Wrapf(err, "reading source list %s", sourceListPath)
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if err := appendPath(line); err != nil {
				return nil, err
			}
		}
	}

	unique := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		unique[path] = struct{}{}
	}
	paths = paths[:0]
	for path := range unique {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}
