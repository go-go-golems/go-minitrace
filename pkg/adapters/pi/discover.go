package pi

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/adapters"
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

		sessionID := strings.TrimSuffix(filepath.Base(path), ".jsonl")
		if index := strings.LastIndex(sessionID, "_"); index >= 0 && index < len(sessionID)-1 {
			sessionID = sessionID[index+1:]
		}

		ret = append(ret, adapters.SessionLocator{
			ID:         sessionID,
			FormatHint: "jsonl-v3",
			SourcePath: path,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(ret, func(i, j int) bool { return ret[i].SourcePath < ret[j].SourcePath })
	return ret, nil
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
