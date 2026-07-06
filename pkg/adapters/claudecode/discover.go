package claudecode

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/adapters"
	"github.com/pkg/errors"
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
		locators[sid] = locatorForJSONLFile(sid, path)
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

// LocateSession builds a SessionLocator for an explicitly listed Claude Code
// session, validating that the path exists. A directory is treated as a
// dir-v1 session; a file is treated as a JSONL v2 transcript.
func LocateSession(path string) (adapters.SessionLocator, error) {
	root, err := expandHome(path)
	if err != nil {
		return adapters.SessionLocator{}, err
	}
	st, err := os.Stat(root)
	if err != nil {
		return adapters.SessionLocator{}, errors.Wrapf(err, "claude-code source session %s", path)
	}
	if st.IsDir() {
		return adapters.SessionLocator{
			ID:         filepath.Base(root),
			FormatHint: "dir-v1",
			SourcePath: root,
		}, nil
	}
	sid := strings.TrimSuffix(filepath.Base(root), ".jsonl")
	return locatorForJSONLFile(sid, root), nil
}

func locatorForJSONLFile(sid, path string) adapters.SessionLocator {
	cwd, startedAt := readSessionHeader(path)
	return adapters.SessionLocator{
		ID:         sid,
		FormatHint: "jsonl-v2",
		SourcePath: path,
		Cwd:        cwd,
		StartedAt:  startedAt,
	}
}

// readSessionHeader cheaply extracts the working directory and its timestamp
// from the leading records of a Claude Code JSONL transcript. The first
// record is often a file-history-snapshot without cwd, so the scan keeps
// going until a record carrying cwd shows up, bounded by a line and byte cap.
func readSessionHeader(path string) (string, string) {
	var cwd, startedAt string
	_ = adapters.ScanJSONLHead(path, adapters.HeadMaxLines, adapters.HeadMaxBytes, func(record map[string]any) bool {
		recordCwd, _ := record["cwd"].(string)
		if recordCwd == "" {
			return false
		}
		cwd = recordCwd
		startedAt, _ = record["timestamp"].(string)
		return true
	})
	return cwd, startedAt
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
