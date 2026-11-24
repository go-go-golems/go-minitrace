package claudecode

import (
	"path/filepath"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/adapters"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestSubagentsForSessionReadsMetaAndSkipsWorkflowJournal(t *testing.T) {
	dir := fixtureProjectDir(t)
	locator := adapters.SessionLocator{ID: fixtureParentID, FormatHint: "jsonl-v2", SourcePath: filepath.Join(dir, fixtureParentID+".jsonl")}
	subagents, err := SubagentsForSession(locator)
	if err != nil {
		t.Fatalf("SubagentsForSession: %v", err)
	}
	if len(subagents) != 2 {
		t.Fatalf("subagents = %+v, want the two agent-*.jsonl transcripts", subagents)
	}
	first := subagents[0]
	if first.AgentID != "a1111111111111111" || first.ParentSessionID != fixtureParentID || first.ToolUseID != "toolu_agent1" || first.AgentType != "Explore" || !first.HasMeta || first.Cwd != "/tmp/demo" || first.StartedAt == "" {
		t.Fatalf("unexpected subagent locator: %+v", first)
	}

	discovered, err := DiscoverSubagents(filepath.Dir(dir))
	if err != nil {
		t.Fatalf("DiscoverSubagents: %v", err)
	}
	if len(discovered) != 2 {
		t.Fatalf("DiscoverSubagents should skip journal.jsonl, got %+v", discovered)
	}

	none, err := SubagentsForSession(adapters.SessionLocator{ID: fixtureStubID, FormatHint: "jsonl-v2", SourcePath: filepath.Join(dir, fixtureStubID+".jsonl")})
	if err != nil || len(none) != 0 {
		t.Fatalf("session without subagents directory: %+v, %v", none, err)
	}
}

func TestLocateSubagentAndLocateSessionDistinguishSubagentPaths(t *testing.T) {
	path := filepath.Join(fixtureProjectDir(t), fixtureParentID, "subagents", "agent-a2222222222222222.jsonl")
	locator, err := LocateSubagent(path)
	if err != nil {
		t.Fatalf("LocateSubagent: %v", err)
	}
	if locator.ParentSessionID != fixtureParentID || locator.AgentID != "a2222222222222222" || locator.ToolUseID != "toolu_agent2" {
		t.Fatalf("unexpected locator: %+v", locator)
	}
	if _, err := LocateSession(path); err == nil {
		t.Fatalf("LocateSession should reject a subagent transcript")
	}
	if _, err := LocateSubagent(filepath.Join(fixtureProjectDir(t), fixtureParentID+".jsonl")); err == nil {
		t.Fatalf("LocateSubagent should reject a primary transcript")
	}
}

func TestLinkParentSubagentsPrefersNativeIdentity(t *testing.T) {
	session := convertFixtureParent(t)
	dir := fixtureProjectDir(t)
	subagents, err := SubagentsForSession(adapters.SessionLocator{ID: fixtureParentID, FormatHint: "jsonl-v2", SourcePath: filepath.Join(dir, fixtureParentID+".jsonl")})
	if err != nil {
		t.Fatalf("SubagentsForSession: %v", err)
	}
	links := []SubagentLink{}
	for _, locator := range subagents {
		child, link, err := ConvertSubagentLocator(locator)
		if err != nil {
			t.Fatalf("ConvertSubagentLocator: %v", err)
		}
		if child.Coordination.PredecessorSession == nil || *child.Coordination.PredecessorSession != fixtureParentID {
			t.Fatalf("subagent %s predecessor = %v", child.ID, child.Coordination.PredecessorSession)
		}
		if child.OperationalContext.FrameworkConfig.(map[string]any)["parent_tool_use_id"] != locator.ToolUseID {
			t.Fatalf("subagent %s should record its parent tool use id", child.ID)
		}
		links = append(links, link)
	}
	// Reverse order: linking must not depend on link order.
	links[0], links[1] = links[1], links[0]
	LinkParentSubagents(session, links)
	for toolUseID, want := range map[string]string{"toolu_agent1": "a1111111111111111", "toolu_agent2": "a2222222222222222"} {
		spawned := toolCallByID(t, session, toolUseID).SpawnedAgent
		if spawned == nil || spawned.SubSessionID == nil || *spawned.SubSessionID != want {
			t.Fatalf("%s linked to %+v, want %s", toolUseID, spawned, want)
		}
	}
}

func TestLinkParentSubagentsFallsBackToStartOrderOnlyWithoutMeta(t *testing.T) {
	newSession := func() *minitrace.Session {
		session := minitrace.BuildSessionSkeleton("parent", "claude-code", SourceFormatV2, AdapterVersion)
		for _, id := range []string{"t1", "t2"} {
			session.ToolCalls = append(session.ToolCalls, minitrace.ToolCall{ID: id, ToolName: "Agent", SpawnedAgent: &minitrace.SpawnedAgent{AgentType: "general"}})
		}
		return &session
	}
	session := newSession()
	LinkParentSubagents(session, []SubagentLink{
		{SessionID: "late", StartedAt: "2026-11-20T10:05:00Z", Inferable: true},
		{SessionID: "early", StartedAt: "2026-11-20T10:01:00Z", Inferable: true},
	})
	if *session.ToolCalls[0].SpawnedAgent.SubSessionID != "early" || *session.ToolCalls[1].SpawnedAgent.SubSessionID != "late" {
		t.Fatalf("start-order fallback mismatched: %v %v", *session.ToolCalls[0].SpawnedAgent.SubSessionID, *session.ToolCalls[1].SpawnedAgent.SubSessionID)
	}

	session = newSession()
	LinkParentSubagents(session, []SubagentLink{{SessionID: "workflow-agent", StartedAt: "2026-11-20T10:01:00Z"}})
	if session.ToolCalls[0].SpawnedAgent.SubSessionID != nil {
		t.Fatalf("a subagent with meta but no toolUseId (workflow agent) must not be guessed onto a spawn")
	}
}
