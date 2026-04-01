package serve

import (
	"math"
	"testing"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestBuildRawSessionBlocksSplitsOnUserTurnsAndComputesGap(t *testing.T) {
	sessionID := "blocks-unit"
	userSource := "human"
	assistantSource := "agent"

	ts1 := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	ts2 := ts1.Add(15 * time.Minute)
	ts1Formatted := minitrace.FormatTimestamp(ts1)
	ts2Formatted := minitrace.FormatTimestamp(ts2)
	durationMS := 7

	toolCall := minitrace.BuildToolCall(
		sessionID+"-tool-1",
		intPtr(1),
		&ts1Formatted,
		"exec_command",
		"execute",
		nil,
		nil,
		map[string]any{"cmd": "pwd"},
		true,
		"/tmp/project\n",
		nil,
		&durationMS,
		nil,
		nil,
		nil,
		nil,
	)

	session := minitrace.BuildSessionSkeleton(sessionID, "codex", "fixture", "test")
	session.Turns = []minitrace.Turn{
		minitrace.BuildTurn(0, &ts1Formatted, "user", &userSource, "first request"),
		{
			Index:             1,
			Timestamp:         &ts1Formatted,
			Role:              "assistant",
			Source:            &assistantSource,
			Content:           "first response",
			ToolCallsInTurn:   []string{toolCall.ID},
			Streaming:         minitrace.Streaming{},
			FrameworkMetadata: nil,
		},
		minitrace.BuildTurn(2, &ts2Formatted, "user", &userSource, "second request"),
		{
			Index:             3,
			Timestamp:         &ts2Formatted,
			Role:              "assistant",
			Source:            &assistantSource,
			Content:           "second response",
			ToolCallsInTurn:   nil,
			Streaming:         minitrace.Streaming{},
			FrameworkMetadata: nil,
		},
	}
	session.ToolCalls = []minitrace.ToolCall{toolCall}

	tcByID := map[string]minitrace.ToolCall{toolCall.ID: toolCall}
	blocks := buildRawSessionBlocks(session, tcByID)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	first := blocks[0]
	if first.UserTurnIdx != 0 {
		t.Fatalf("expected first block user turn idx 0, got %d", first.UserTurnIdx)
	}
	if first.AgentTurns != 1 {
		t.Fatalf("expected first block to contain 1 agent turn, got %d", first.AgentTurns)
	}
	if first.ToolCalls != 1 {
		t.Fatalf("expected first block to contain 1 tool call, got %d", first.ToolCalls)
	}

	second := blocks[1]
	if second.UserContent != "second request" {
		t.Fatalf("expected second block user content %q, got %q", "second request", second.UserContent)
	}
	if second.GapMinutes == nil {
		t.Fatal("expected second block gap to be computed")
	}
	if math.Abs(*second.GapMinutes-15.0) > 0.001 {
		t.Fatalf("expected 15 minute gap, got %.3f", *second.GapMinutes)
	}
}
