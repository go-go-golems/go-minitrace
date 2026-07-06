package copilot

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/adapters"
	"gopkg.in/yaml.v3"
)

const (
	AdapterVersion     = "go-minitrace-copilot-adapter-dev"
	SourceFormatEvents = "copilot-cli-events-jsonl-v1"
)

type WorkspaceMetadata struct {
	ID              string         `yaml:"id" json:"id,omitempty"`
	Name            string         `yaml:"name" json:"name,omitempty"`
	CWD             string         `yaml:"cwd" json:"cwd,omitempty"`
	GitRoot         string         `yaml:"git_root" json:"git_root,omitempty"`
	Branch          string         `yaml:"branch" json:"branch,omitempty"`
	Repository      string         `yaml:"repository" json:"repository,omitempty"`
	HostType        string         `yaml:"host_type" json:"host_type,omitempty"`
	ClientName      string         `yaml:"client_name" json:"client_name,omitempty"`
	RemoteSteerable *bool          `yaml:"remote_steerable" json:"remote_steerable,omitempty"`
	CreatedAt       any            `yaml:"created_at" json:"created_at,omitempty"`
	UpdatedAt       any            `yaml:"updated_at" json:"updated_at,omitempty"`
	Raw             map[string]any `yaml:"-" json:"raw,omitempty"`
}

func Discover(sourceDir string) ([]adapters.SessionLocator, error) {
	root, err := expandHome(sourceDir)
	if err != nil {
		return nil, err
	}

	sessionDirs, err := discoverSessionDirs(root)
	if err != nil {
		return nil, err
	}

	locators := make([]adapters.SessionLocator, 0, len(sessionDirs))
	for _, sessionDir := range sessionDirs {
		eventsPath := filepath.Join(sessionDir, "events.jsonl")
		if !isFile(eventsPath) {
			continue
		}
		workspace, _ := ReadWorkspace(filepath.Join(sessionDir, "workspace.yaml"))
		sessionID := filepath.Base(sessionDir)
		cwd := ""
		startedAt := ""
		if workspace != nil {
			if strings.TrimSpace(workspace.ID) != "" {
				sessionID = strings.TrimSpace(workspace.ID)
			}
			cwd = strings.TrimSpace(workspace.CWD)
			startedAt = timestampString(workspace.CreatedAt)
		}
		locators = append(locators, adapters.SessionLocator{
			ID:         sessionID,
			FormatHint: SourceFormatEvents,
			SourcePath: eventsPath,
			Cwd:        cwd,
			StartedAt:  startedAt,
		})
	}

	sort.Slice(locators, func(i, j int) bool { return locators[i].SourcePath < locators[j].SourcePath })
	return locators, nil
}

func discoverSessionDirs(root string) ([]string, error) {
	root = filepath.Clean(root)
	if isFile(filepath.Join(root, "events.jsonl")) {
		return []string{root}, nil
	}

	searchRoot := root
	if isDir(filepath.Join(root, "session-state")) {
		searchRoot = filepath.Join(root, "session-state")
	}

	entries, err := os.ReadDir(searchRoot)
	if err != nil {
		return nil, err
	}
	ret := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(searchRoot, entry.Name())
		if isFile(filepath.Join(path, "events.jsonl")) {
			ret = append(ret, path)
		}
	}
	return ret, nil
}

func ReadWorkspace(path string) (*WorkspaceMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var workspace WorkspaceMetadata
	if err := yaml.Unmarshal(data, &workspace); err != nil {
		return nil, err
	}
	workspace.Raw = raw
	return &workspace, nil
}

func expandHome(path string) (string, error) {
	if path == "" {
		return filepath.Clean(path), nil
	}
	if path[0] != '~' {
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

// timestampString renders the free-form created_at/updated_at workspace
// metadata values as a string when cheap to do so.
func timestampString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case time.Time:
		return v.UTC().Format(time.RFC3339)
	}
	return ""
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func isFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
