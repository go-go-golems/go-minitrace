package codex

import (
	"bufio"
	"encoding/json"
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

		ret = append(ret, locatorForFile(path))
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(ret, func(i, j int) bool { return ret[i].SourcePath < ret[j].SourcePath })
	return ret, nil
}

// LocateSession builds a SessionLocator for an explicitly listed Codex
// session JSONL file, validating that the file exists.
func LocateSession(path string) (adapters.SessionLocator, error) {
	root, err := expandHome(path)
	if err != nil {
		return adapters.SessionLocator{}, err
	}
	if _, err := os.Stat(root); err != nil {
		return adapters.SessionLocator{}, errors.Wrapf(err, "codex source session %s", path)
	}
	return locatorForFile(root), nil
}

func locatorForFile(path string) adapters.SessionLocator {
	cwd, startedAt := readSessionHeader(path)
	return adapters.SessionLocator{
		ID:         strings.TrimSuffix(filepath.Base(path), ".jsonl"),
		FormatHint: detectFormat(path),
		SourcePath: path,
		Cwd:        cwd,
		StartedAt:  startedAt,
	}
}

// readSessionHeader cheaply extracts the working directory and start
// timestamp from the leading session_meta record of a Codex session JSONL
// file. It reads at most a bounded prefix of the file; exec JSONL streams
// carry no session_meta record and yield empty values.
// LastActivityAt returns the latest valid timestamp emitted by a persisted
// Codex session JSONL transcript. Codex exec JSONL has no authoritative native
// timestamp field, so it is explicitly rejected instead of silently omitted
// by --active-since.
func LastActivityAt(path string) (string, error) {
	if detectFormat(path) == "exec-jsonl-v1" {
		return "", errors.Errorf("Codex exec JSONL source %s has no authoritative activity timestamp; --active-since is unsupported for this format", path)
	}
	return adapters.ScanJSONLLastTimestamp(path, func(record map[string]any) []string {
		timestamp, _ := record["timestamp"].(string)
		payload, _ := record["payload"].(map[string]any)
		payloadTimestamp, _ := payload["timestamp"].(string)
		return []string{timestamp, payloadTimestamp}
	})
}

func readSessionHeader(path string) (string, string) {
	var cwd, startedAt string
	_ = adapters.ScanJSONLHead(path, adapters.HeadMaxLines, adapters.HeadMaxBytes, func(record map[string]any) bool {
		type_, _ := record["type"].(string)
		if type_ != "session_meta" {
			return false
		}
		payload, _ := record["payload"].(map[string]any)
		if payload != nil {
			cwd, _ = payload["cwd"].(string)
		}
		startedAt, _ = record["timestamp"].(string)
		if startedAt == "" && payload != nil {
			startedAt, _ = payload["timestamp"].(string)
		}
		return true
	})
	return cwd, startedAt
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
