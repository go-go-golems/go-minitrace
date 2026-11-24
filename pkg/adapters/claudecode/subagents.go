package claudecode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/adapters"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
)

const subagentsDirName = "subagents"

// SubagentLocator identifies one subagent transcript below a parent session's
// directory: <project>/<session>/subagents/**/agent-<id>.jsonl. Fields after
// SourcePath come from the transcript head and the optional sibling
// agent-<id>.meta.json; they are empty when the source does not record them.
type SubagentLocator struct {
	AgentID         string
	ParentSessionID string
	SourcePath      string
	// ToolUseID is the parent's tool_use id that spawned this subagent.
	ToolUseID   string
	AgentType   string
	Description string
	// HasMeta reports whether agent-<id>.meta.json was found. Modern Claude
	// Code always writes one; its absence marks an older transcript whose
	// parent link can only be inferred from ordering.
	HasMeta        bool
	Cwd            string
	StartedAt      string
	LastActivityAt string
}

type subagentMeta struct {
	AgentType   string `json:"agentType"`
	Description string `json:"description"`
	ToolUseID   string `json:"toolUseId"`
}

// IsSubagentTranscriptPath reports whether path names a subagent transcript
// (agent-*.jsonl below a subagents/ directory). Other JSONL files under
// subagents/, such as workflow journal.jsonl files, are not transcripts.
func IsSubagentTranscriptPath(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, "agent-") && filepath.Ext(base) == ".jsonl" && isUnderSubagentsDir(path)
}

func isUnderSubagentsDir(path string) bool {
	return subagentsIndex(strings.Split(filepath.Clean(path), string(filepath.Separator))) >= 0
}

// subagentsIndex returns the index of the subagents directory nearest the
// file named by parts (its last element), or -1. It searches from the file
// outward: a directory above the projects tree may itself be called
// subagents, and the parent session id is the component just above the
// innermost one.
func subagentsIndex(parts []string) int {
	for i := len(parts) - 2; i >= 0; i-- {
		if parts[i] == subagentsDirName {
			return i
		}
	}
	return -1
}

// LocateSubagent builds a SubagentLocator for an explicitly listed subagent
// transcript. The parent session id is the directory that contains
// subagents/.
func LocateSubagent(path string) (SubagentLocator, error) {
	root, err := expandHome(path)
	if err != nil {
		return SubagentLocator{}, err
	}
	if _, err := os.Stat(root); err != nil {
		return SubagentLocator{}, errors.Wrapf(err, "claude-code subagent transcript %s", path)
	}
	parts := strings.Split(root, string(filepath.Separator))
	index := subagentsIndex(parts)
	if !IsSubagentTranscriptPath(root) || index <= 0 {
		return SubagentLocator{}, errors.Errorf("%s is not a Claude Code subagent transcript (expected .../<session>/subagents/**/agent-<id>.jsonl)", path)
	}
	return subagentLocatorForFile(root, parts[index-1]), nil
}

// SubagentsForSession lists the subagent transcripts that belong to a primary
// session: <stem>/subagents/** for a JSONL transcript <stem>.jsonl, and
// <dir>/subagents/** for a dir-v1 session directory.
func SubagentsForSession(locator adapters.SessionLocator) ([]SubagentLocator, error) {
	sessionDir := locator.SourcePath
	if locator.FormatHint != "dir-v1" {
		sessionDir = strings.TrimSuffix(locator.SourcePath, filepath.Ext(locator.SourcePath))
	}
	root := filepath.Join(sessionDir, subagentsDirName)
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return nil, nil
	}
	ret := []SubagentLocator{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !IsSubagentTranscriptPath(path) {
			return nil
		}
		ret = append(ret, subagentLocatorForFile(path, locator.ID))
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "listing subagents of %s", locator.SourcePath)
	}
	sortSubagentLocators(ret)
	return ret, nil
}

// DiscoverSubagents walks sourceDir for every subagent transcript.
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
		if d.IsDir() || !IsSubagentTranscriptPath(path) {
			return nil
		}
		parts := strings.Split(path, string(filepath.Separator))
		index := subagentsIndex(parts)
		if index <= 0 {
			return nil
		}
		ret = append(ret, subagentLocatorForFile(path, parts[index-1]))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sortSubagentLocators(ret)
	return ret, nil
}

func sortSubagentLocators(locators []SubagentLocator) {
	sort.Slice(locators, func(i, j int) bool {
		if locators[i].ParentSessionID == locators[j].ParentSessionID {
			return locators[i].SourcePath < locators[j].SourcePath
		}
		return locators[i].ParentSessionID < locators[j].ParentSessionID
	})
}

func subagentLocatorForFile(path, parentSessionID string) SubagentLocator {
	locator := SubagentLocator{
		AgentID:         strings.TrimPrefix(strings.TrimSuffix(filepath.Base(path), ".jsonl"), "agent-"),
		ParentSessionID: parentSessionID,
		SourcePath:      path,
	}
	_ = adapters.ScanJSONLHead(path, adapters.HeadMaxLines, adapters.HeadMaxBytes, func(record map[string]any) bool {
		if agentID := stringValue(record["agentId"]); agentID != "" {
			locator.AgentID = agentID
		}
		if cwd := stringValue(record["cwd"]); cwd != "" && locator.Cwd == "" {
			locator.Cwd = cwd
			locator.StartedAt = stringValue(record["timestamp"])
		}
		return locator.Cwd != ""
	})
	metaPath := strings.TrimSuffix(path, ".jsonl") + ".meta.json"
	if payload, err := os.ReadFile(metaPath); err == nil {
		var meta subagentMeta
		if json.Unmarshal(payload, &meta) == nil {
			locator.HasMeta = true
			locator.ToolUseID = meta.ToolUseID
			locator.AgentType = meta.AgentType
			locator.Description = meta.Description
		}
	}
	return locator
}

// SubagentLink is what a parent needs to backlink a converted subagent.
type SubagentLink struct {
	SessionID string
	ToolUseID string
	StartedAt string
	// Inferable marks a link without native identity (no meta file), which
	// may be paired with an unlinked spawn by start order.
	Inferable bool
}

// LinkParentSubagents sets spawned_agent.sub_session_id on the parent's
// delegating tool calls. Links already present (from the tool result's
// toolUseResult.agentId) are kept; the subagent meta toolUseId fills the
// rest; only subagents without native identity fall back to pairing by start
// time with spawns that are still unlinked.
func LinkParentSubagents(session *minitrace.Session, links []SubagentLink) {
	if session == nil || len(links) == 0 {
		return
	}
	linked := map[string]struct{}{}
	for _, toolCall := range session.ToolCalls {
		if toolCall.SpawnedAgent != nil && toolCall.SpawnedAgent.SubSessionID != nil {
			linked[*toolCall.SpawnedAgent.SubSessionID] = struct{}{}
		}
	}
	byToolUseID := map[string]int{}
	for i, toolCall := range session.ToolCalls {
		if toolCall.SpawnedAgent != nil {
			byToolUseID[toolCall.ID] = i
		}
	}
	inferable := []SubagentLink{}
	for _, link := range links {
		if _, ok := linked[link.SessionID]; ok {
			continue
		}
		if i, ok := byToolUseID[link.ToolUseID]; ok && link.ToolUseID != "" && session.ToolCalls[i].SpawnedAgent.SubSessionID == nil {
			sessionID := link.SessionID
			session.ToolCalls[i].SpawnedAgent.SubSessionID = &sessionID
			linked[sessionID] = struct{}{}
			continue
		}
		if link.Inferable {
			inferable = append(inferable, link)
		}
	}
	sort.SliceStable(inferable, func(i, j int) bool { return inferable[i].StartedAt < inferable[j].StartedAt })
	next := 0
	for i := range session.ToolCalls {
		if next >= len(inferable) {
			return
		}
		spawned := session.ToolCalls[i].SpawnedAgent
		if spawned == nil || spawned.SubSessionID != nil {
			continue
		}
		sessionID := inferable[next].SessionID
		spawned.SubSessionID = &sessionID
		next++
	}
}
