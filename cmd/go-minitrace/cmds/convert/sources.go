package convert

import (
	"os"
	"strings"

	"github.com/pkg/errors"
)

// collectSourceSessions merges explicit --source-session paths with the
// contents of an optional --source-list file. The list file contains one
// session path per line; blank lines and lines starting with # are ignored.
func collectSourceSessions(sourceSessions []string, sourceListPath string) ([]string, error) {
	paths := make([]string, 0, len(sourceSessions))
	for _, path := range sourceSessions {
		path = strings.TrimSpace(path)
		if path != "" {
			paths = append(paths, path)
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
			paths = append(paths, line)
		}
	}

	return paths, nil
}
