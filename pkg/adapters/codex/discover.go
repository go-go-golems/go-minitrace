package codex

import (
	"bufio"
	"encoding/json"
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

	searchRoot := filepath.Join(root, "sessions")
	if st, err := os.Stat(searchRoot); err != nil || !st.IsDir() {
		searchRoot = root
	}

	ret := []adapters.SessionLocator{}
	err = filepath.WalkDir(searchRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}

		sid := strings.TrimSuffix(filepath.Base(path), ".jsonl")
		ret = append(ret, adapters.SessionLocator{
			ID:         sid,
			FormatHint: detectFormat(path),
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

func detectFormat(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return "unknown-jsonl"
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for i := 0; i < 5 && scanner.Scan(); i++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			continue
		}
		type_, _ := payload["type"].(string)
		switch type_ {
		case "session_meta", "response_item", "event_msg", "turn_context":
			return "session-jsonl-v1"
		case "thread.started", "turn.started", "turn.completed", "item.started", "item.completed":
			return "exec-jsonl-v1"
		}
	}
	return "unknown-jsonl"
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
