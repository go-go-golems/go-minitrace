package claudecode

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/adapters"
)

type SubagentLocator struct {
	AgentID         string
	ParentSessionID string
	SourcePath      string
}

func Discover(sourceDir string) ([]adapters.SessionLocator, error) {
	root, err := expandHome(sourceDir)
	if err != nil {
		return nil, err
	}

	locators := map[string]adapters.SessionLocator{}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".jsonl" {
			return nil
		}
		if strings.Contains(path, string(filepath.Separator)+"subagents"+string(filepath.Separator)) {
			return nil
		}

		sid := strings.TrimSuffix(filepath.Base(path), ".jsonl")
		if len(sid) < 32 {
			return nil
		}
		locators[sid] = adapters.SessionLocator{
			ID:         sid,
			FormatHint: "jsonl-v2",
			SourcePath: path,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() || d.Name() != "tool-results" {
			return nil
		}
		sessionDir := filepath.Dir(path)
		sid := filepath.Base(sessionDir)
		if len(sid) < 32 {
			return nil
		}
		if _, ok := locators[sid]; ok {
			return nil
		}
		locators[sid] = adapters.SessionLocator{
			ID:         sid,
			FormatHint: "dir-v1",
			SourcePath: sessionDir,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	ret := make([]adapters.SessionLocator, 0, len(locators))
	for _, locator := range locators {
		ret = append(ret, locator)
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i].ID < ret[j].ID })
	return ret, nil
}

func DiscoverSubagents(sourceDir string) ([]SubagentLocator, error) {
	root, err := expandHome(sourceDir)
	if err != nil {
		return nil, err
	}

	ret := []SubagentLocator{}
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		subagentNeedle := string(filepath.Separator) + "subagents" + string(filepath.Separator)
		if !strings.Contains(path, subagentNeedle) {
			return nil
		}

		parts := strings.Split(path, string(filepath.Separator))
		subagentIndex := -1
		for i, part := range parts {
			if part == "subagents" {
				subagentIndex = i
				break
			}
		}
		if subagentIndex <= 0 {
			return nil
		}

		ret = append(ret, SubagentLocator{
			AgentID:         strings.TrimSuffix(filepath.Base(path), ".jsonl"),
			ParentSessionID: parts[subagentIndex-1],
			SourcePath:      path,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(ret, func(i, j int) bool {
		if ret[i].ParentSessionID == ret[j].ParentSessionID {
			return ret[i].SourcePath < ret[j].SourcePath
		}
		return ret[i].ParentSessionID < ret[j].ParentSessionID
	})
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
