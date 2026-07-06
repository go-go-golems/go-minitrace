package pi

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/adapters"
	"github.com/pkg/errors"
)

func Discover(sourceDir string) ([]adapters.SessionLocator, error) {
	root, err := expandHome(sourceDir)
	if err != nil {
		return nil, err
	}

	ret := []adapters.SessionLocator{}
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}

		ret = append(ret, locatorForFile(path))
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(ret, func(i, j int) bool { return ret[i].SourcePath < ret[j].SourcePath })
	return ret, nil
}

// LocateSession builds a SessionLocator for an explicitly listed Pi session
// JSONL file, validating that the file exists.
func LocateSession(path string) (adapters.SessionLocator, error) {
	root, err := expandHome(path)
	if err != nil {
		return adapters.SessionLocator{}, err
	}
	if _, err := os.Stat(root); err != nil {
		return adapters.SessionLocator{}, errors.Wrapf(err, "pi source session %s", path)
	}
	return locatorForFile(root), nil
}

func locatorForFile(path string) adapters.SessionLocator {
	sessionID := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	if index := strings.LastIndex(sessionID, "_"); index >= 0 && index < len(sessionID)-1 {
		sessionID = sessionID[index+1:]
	}

	cwd, startedAt := readSessionHeader(path)
	return adapters.SessionLocator{
		ID:         sessionID,
		FormatHint: "jsonl-v3",
		SourcePath: path,
		Cwd:        cwd,
		StartedAt:  startedAt,
	}
}

// readSessionHeader cheaply extracts the working directory and start
// timestamp from the leading "session" record of a Pi session JSONL file,
// reading at most a bounded prefix of the file.
func readSessionHeader(path string) (string, string) {
	var cwd, startedAt string
	_ = adapters.ScanJSONLHead(path, adapters.HeadMaxLines, adapters.HeadMaxBytes, func(record map[string]any) bool {
		type_, _ := record["type"].(string)
		if type_ != "session" {
			return false
		}
		cwd, _ = record["cwd"].(string)
		startedAt, _ = record["timestamp"].(string)
		return true
	})
	return cwd, startedAt
}

func expandHome(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return filepath.Clean(path), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
}
