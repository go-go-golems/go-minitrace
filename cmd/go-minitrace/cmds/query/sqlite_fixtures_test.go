package query

// Shared fixtures and helpers for the SQLite query-path tests. These are
// intentionally self-contained: command_runtime_js_test.go defines similar
// helpers, but its _js filename suffix makes the Go toolchain treat it as
// GOOS=js-constrained, so nothing in it is compiled on normal platforms.

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

type captureProcessor struct {
	rows []types.Row
}

func (c *captureProcessor) AddRow(_ context.Context, row types.Row) error {
	c.rows = append(c.rows, row)
	return nil
}

func (c *captureProcessor) Close(context.Context) error { return nil }

func rowToMap(row types.Row) map[string]interface{} {
	ret := map[string]interface{}{}
	for pair := row.Oldest(); pair != nil; pair = pair.Next() {
		ret[pair.Key] = pair.Value
	}
	return ret
}

func writeAdvancedFixtureArchive(t *testing.T) string {
	t.Helper()
	archiveRoot := t.TempDir()
	for _, session := range buildAdvancedFixtureSessions(t) {
		if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
			t.Fatalf("WriteSession returned error: %v", err)
		}
	}
	return filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")
}

func buildAdvancedFixtureSessions(t *testing.T) []*minitrace.Session {
	t.Helper()
	mkSession := func(sessionID, title, workingDir, model string, start time.Time, userTurns, assistantTurns int, toolNames []string) *minitrace.Session {
		session := minitrace.BuildSessionSkeleton(sessionID, "pi", "fixture", "test")
		session.Title = fixtureStringPtr(title)
		session.Environment.Model = fixtureStringPtr(model)
		session.Environment.SystemPrompt = fixtureStringPtr("Run tests before reporting, commit focused work, and avoid legacy APIs.")
		session.OperationalContext.WorkingDirectory = fixtureStringPtr(workingDir)

		turns := []minitrace.Turn{}
		times := []time.Time{}
		index := 0
		for i := 0; i < userTurns; i++ {
			ts := start.Add(time.Duration(index) * time.Minute)
			formatted := minitrace.FormatTimestamp(ts)
			turns = append(turns, minitrace.BuildTurn(index, &formatted, "user", fixtureStringPtr("human"), fmt.Sprintf("user turn %d", i+1)))
			times = append(times, ts)
			index++
		}
		for i := 0; i < assistantTurns; i++ {
			ts := start.Add(time.Duration(index) * time.Minute)
			formatted := minitrace.FormatTimestamp(ts)
			turns = append(turns, minitrace.BuildTurn(index, &formatted, "assistant", fixtureStringPtr("model"), fmt.Sprintf("assistant turn %d", i+1)))
			times = append(times, ts)
			index++
		}
		session.Turns = turns

		toolCalls := []minitrace.ToolCall{}
		for i, toolName := range toolNames {
			ts := start.Add(time.Duration(index+i) * time.Minute)
			formatted := minitrace.FormatTimestamp(ts)
			turnIndex := assistantTurns + userTurns - 1
			var filePath *string
			var command *string
			operationType := "EXECUTE"
			switch toolName {
			case "read":
				operationType = "read"
				filePath = fixtureStringPtr(filepath.Join(workingDir, "README.md"))
			case "edit", "write":
				operationType = "modify"
				filePath = fixtureStringPtr(filepath.Join(workingDir, "src", "main.go"))
			case "bash":
				command = fixtureStringPtr("go test ./...")
			case "understand_image":
				operationType = "read"
				filePath = fixtureStringPtr(filepath.Join(workingDir, "screenshots", "ui.png"))
			}
			durationMS := 100 + i*10
			toolCall := minitrace.BuildToolCall(
				fmt.Sprintf("%s-tool-%02d", sessionID, i+1),
				&turnIndex,
				&formatted,
				toolName,
				operationType,
				filePath,
				command,
				map[string]any{"example": true},
				true,
				"ok",
				nil,
				&durationMS,
				nil,
				nil,
				fixtureStringPtr("fixture"),
				nil,
			)
			toolCalls = append(toolCalls, toolCall)
			times = append(times, ts)
		}
		session.ToolCalls = toolCalls
		session.Annotations = []minitrace.Annotation{}
		session.Timing = minitrace.ComputeTiming(times)
		quality := minitrace.AssignQualityTier(session.Turns, session.ToolCalls)
		session.Quality = &quality
		session.Metrics = minitrace.ComputeMetrics(session.Turns, session.ToolCalls, session.Timing, 0, nil)
		return &session
	}

	sessions := []*minitrace.Session{
		mkSession("fixture-alpha-1", "Alpha planning", "~/projects/alpha/app", "gpt-5", time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC), 2, 6, []string{"bash", "bash", "bash", "read", "read", "edit"}),
		mkSession("fixture-alpha-2", "Alpha follow-up", "~/projects/alpha/app", "gpt-5", time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC), 1, 4, []string{"bash", "bash", "write", "write"}),
		mkSession("fixture-beta-1", "Beta vision", "~/projects/beta/lab", "claude-sonnet-4-6", time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC), 1, 3, []string{"understand_image", "understand_image", "read", "bash"}),
	}

	gitBranch := "main"
	autonomyLevel := "supervised"
	sandbox := true
	sessions[0].OperationalContext.GitBranch = &gitBranch
	sessions[0].OperationalContext.AutonomyLevel = &autonomyLevel
	sessions[0].OperationalContext.Sandbox = &sandbox
	sessions[0].Coordination.HumanAttention = "medium"
	fromSession := "fixture-prior"
	toSession := "fixture-next"
	warmState := "warm"
	readyState := "ready"
	sessions[0].Handover.Received = &minitrace.HandoverDocument{FromSession: &fromSession, Document: "prior handover", StateDescription: &warmState}
	sessions[0].Handover.Produced = &minitrace.HandoverDocument{ToSession: &toSession, Document: "next handover", StateDescription: &readyState}
	classification := "high"
	sessions[0].Annotations = []minitrace.Annotation{{
		ID:        "fixture-ann-1",
		Timestamp: minitrace.FormatTimestamp(time.Date(2026, 4, 1, 9, 45, 0, 0, time.UTC)),
		Annotator: "fixture",
		Scope:     minitrace.AnnotationScope{Type: "tool_call", TargetID: sessions[0].ToolCalls[0].ID},
		Content:   minitrace.AnnotationContent{Category: "risk", Tags: []string{"spawned-agent"}, Title: "Subagent review", Detail: "Spawned agent should be reviewed."},
		TaxonomyMappings: minitrace.TaxonomyMappings{
			Minitrace: []string{"risk/subagent"},
		},
		Classification: &classification,
	}}
	subSessionID := "fixture-subagent-1"
	position := 0.35
	sessions[0].ToolCalls[0].SpawnedAgent = &minitrace.SpawnedAgent{AgentType: "codex", TaskScope: "audit", SubSessionID: &subSessionID, OutcomeSummary: "completed"}
	sessions[0].ToolCalls[0].Context.PositionInSession = &position
	inputTokens := 11
	outputTokens := 22
	cacheReadTokens := 33
	toolTokens := 44
	sessions[0].Metrics.TotalInputTokens = &inputTokens
	sessions[0].Metrics.TotalOutputTokens = &outputTokens
	sessions[0].Metrics.TotalCacheReadTokens = &cacheReadTokens
	sessions[0].Metrics.TotalToolTokens = &toolTokens
	sessions[0].Metrics.SubagentCount = 1
	sessions[0].Metrics.SubagentToolCalls = 1
	return sessions
}

func fixtureStringPtr(value string) *string { return &value }
