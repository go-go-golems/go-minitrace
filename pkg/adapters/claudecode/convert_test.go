package claudecode

import (
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestConvertRecordsMatchesToolResultsAndBuildsSession(t *testing.T) {
	records := []map[string]any{
		{
			"type":      "system",
			"timestamp": "2026-03-29T10:00:00Z",
			"cwd":       "/home/manuel/project",
			"version":   "2.1.76",
			"gitBranch": "main",
			"message": map[string]any{
				"content": "You are Claude Code.",
			},
		},
		{
			"type":      "user",
			"timestamp": "2026-03-29T10:00:05Z",
			"message": map[string]any{
				"content": "Read app.go",
			},
		},
		{
			"type":      "assistant",
			"timestamp": "2026-03-29T10:00:10Z",
			"message": map[string]any{
				"model": "claude-opus-4-1",
				"usage": map[string]any{
					"input_tokens":                10,
					"output_tokens":               20,
					"cache_read_input_tokens":     3,
					"cache_creation_input_tokens": 1,
				},
				"content": []any{
					map[string]any{"type": "text", "text": "I'll inspect the file."},
					map[string]any{
						"type": "tool_use",
						"id":   "tool-1",
						"name": "Read",
						"input": map[string]any{
							"file_path": "/home/manuel/project/app.go",
						},
					},
				},
			},
		},
		{
			"type":      "user",
			"timestamp": "2026-03-29T10:00:11Z",
			"message": map[string]any{
				"content": []any{
					map[string]any{
						"type":        "tool_result",
						"tool_use_id": "tool-1",
						"content":     "package main",
						"is_error":    false,
					},
				},
			},
		},
	}

	session, err := ConvertRecords(records, "session-1", "/tmp/session-1.jsonl")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}

	if session.Environment.Model == nil || *session.Environment.Model != "claude-opus-4-1" {
		t.Fatalf("expected model to be captured, got %+v", session.Environment.Model)
	}
	expectedWorkingDirectory := minitrace.NormalizePath("/home/manuel/project")
	if session.OperationalContext.WorkingDirectory == nil || *session.OperationalContext.WorkingDirectory != expectedWorkingDirectory {
		t.Fatalf("expected normalized working directory, got %+v", session.OperationalContext.WorkingDirectory)
	}
	if len(session.Turns) != 3 {
		t.Fatalf("expected 3 turns, got %d", len(session.Turns))
	}
	if len(session.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(session.ToolCalls))
	}
	toolCall := session.ToolCalls[0]
	if toolCall.Output.Success != true {
		t.Fatalf("expected successful tool result")
	}
	if toolCall.Output.Result == nil || *toolCall.Output.Result != "package main" {
		t.Fatalf("unexpected tool result: %+v", toolCall.Output.Result)
	}
	expectedFilePath := minitrace.NormalizePath("/home/manuel/project/app.go")
	if toolCall.Input.FilePath == nil || *toolCall.Input.FilePath != expectedFilePath {
		t.Fatalf("expected normalized file path, got %+v", toolCall.Input.FilePath)
	}
	if session.Metrics.TotalInputTokens == nil || *session.Metrics.TotalInputTokens != 10 {
		t.Fatalf("expected input tokens, got %+v", session.Metrics.TotalInputTokens)
	}
	if session.Metrics.TotalOutputTokens == nil || *session.Metrics.TotalOutputTokens != 20 {
		t.Fatalf("expected output tokens, got %+v", session.Metrics.TotalOutputTokens)
	}
	if session.Quality == nil || *session.Quality != "B" {
		t.Fatalf("expected quality B, got %+v", session.Quality)
	}
}

func TestConvertRecordsAnnotatesOrphanToolCalls(t *testing.T) {
	records := []map[string]any{
		{
			"type":      "assistant",
			"timestamp": "2026-03-29T10:00:10Z",
			"message": map[string]any{
				"content": []any{
					map[string]any{
						"type": "tool_use",
						"id":   "tool-1",
						"name": "Read",
						"input": map[string]any{
							"file_path": "/tmp/app.go",
						},
					},
				},
			},
		},
	}

	session, err := ConvertRecords(records, "session-2", "/tmp/session-2.jsonl")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}

	if len(session.ToolCalls) != 1 {
		t.Fatalf("expected orphan tool call to be retained")
	}
	if session.ToolCalls[0].Output.Error == nil || *session.ToolCalls[0].Output.Error != "no tool_result received" {
		t.Fatalf("expected orphan tool call error annotation, got %+v", session.ToolCalls[0].Output.Error)
	}
	if len(session.Annotations) != 1 {
		t.Fatalf("expected orphan annotation, got %d", len(session.Annotations))
	}
}

func TestConvertRecordsPreservesClaudeFrameworkMetadata(t *testing.T) {
	records := []map[string]any{
		{
			"type":        "assistant",
			"timestamp":   "2026-03-29T10:00:10Z",
			"entrypoint":  "sdk-ts",
			"slug":        "curious-otter",
			"parentUuid":  "parent-1",
			"isSidechain": false,
			"message": map[string]any{
				"model":         "claude-opus-4-1",
				"stop_reason":   "tool_use",
				"stop_sequence": nil,
				"usage": map[string]any{
					"input_tokens":                10,
					"output_tokens":               20,
					"cache_read_input_tokens":     3,
					"cache_creation_input_tokens": 1,
					"cache_creation": map[string]any{
						"ephemeral_5m_input_tokens": 1,
						"ephemeral_1h_input_tokens": 2,
					},
				},
				"content": []any{
					map[string]any{"type": "text", "text": "I'll inspect the file."},
					map[string]any{
						"type": "tool_use",
						"id":   "tool-1",
						"name": "Read",
						"input": map[string]any{
							"file_path": "/home/manuel/project/app.go",
						},
						"caller": map[string]any{"type": "direct"},
					},
				},
			},
		},
		{
			"type":        "user",
			"timestamp":   "2026-03-29T10:00:11Z",
			"entrypoint":  "sdk-ts",
			"slug":        "curious-otter",
			"parentUuid":  "assistant-1",
			"isSidechain": false,
			"message": map[string]any{
				"content": []any{
					map[string]any{
						"type":        "tool_result",
						"tool_use_id": "tool-1",
						"content":     "package main",
						"is_error":    false,
					},
				},
			},
		},
	}

	session, err := ConvertRecords(records, "session-meta", "/tmp/session-meta.jsonl")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}

	config, ok := session.OperationalContext.FrameworkConfig.(map[string]any)
	if !ok || config["entrypoint"] != "sdk-ts" {
		t.Fatalf("expected entrypoint in framework config, got %+v", session.OperationalContext.FrameworkConfig)
	}
	turnMetadata, ok := session.Turns[0].FrameworkMetadata.(map[string]any)
	if !ok {
		t.Fatalf("expected assistant turn metadata map, got %+v", session.Turns[0].FrameworkMetadata)
	}
	if turnMetadata["entrypoint"] != "sdk-ts" || turnMetadata["slug"] != "curious-otter" || turnMetadata["parent_uuid"] != "parent-1" {
		t.Fatalf("unexpected assistant turn metadata: %+v", turnMetadata)
	}
	if turnMetadata["stop_reason"] != "tool_use" {
		t.Fatalf("expected stop_reason metadata, got %+v", turnMetadata)
	}
	cacheCreation, ok := turnMetadata["cache_creation"].(map[string]any)
	if !ok || cacheCreation["ephemeral_1h_input_tokens"] != 2 {
		t.Fatalf("expected cache_creation bucket detail, got %+v", turnMetadata)
	}
	toolMetadata, ok := session.ToolCalls[0].FrameworkMetadata.(map[string]any)
	if !ok {
		t.Fatalf("expected tool metadata map, got %+v", session.ToolCalls[0].FrameworkMetadata)
	}
	caller, ok := toolMetadata["caller"].(map[string]any)
	if !ok || caller["type"] != "direct" {
		t.Fatalf("expected caller metadata, got %+v", toolMetadata)
	}
	if toolMetadata["slug"] != "curious-otter" || toolMetadata["entrypoint"] != "sdk-ts" || toolMetadata["parent_uuid"] != "assistant-1" {
		t.Fatalf("expected tool-result metadata to be preserved, got %+v", toolMetadata)
	}
}

func TestConvertRecordsPreservesLatestClaudeSessionMetadata(t *testing.T) {
	records := []map[string]any{
		{
			"type":             "mode",
			"mode":             "plan",
			"sessionId":        "session-latest",
			"timestamp":        "2026-06-10T12:00:00Z",
			"entrypoint":       "cli",
			"agentId":          "agent-abc",
			"parentUuid":       "parent-uuid",
			"isSidechain":      true,
			"attributionAgent": "reviewer",
		},
		{
			"type":           "permission-mode",
			"permissionMode": "bypassPermissions",
			"sessionId":      "session-latest",
			"timestamp":      "2026-06-10T12:00:01Z",
		},
		{
			"type":      "ai-title",
			"aiTitle":   "Investigate latest Claude session metadata",
			"sessionId": "session-latest",
		},
		{
			"type":        "attachment",
			"uuid":        "attachment-1",
			"timestamp":   "2026-06-10T12:00:02Z",
			"entrypoint":  "cli",
			"parentUuid":  "parent-uuid",
			"isSidechain": true,
			"agentId":     "agent-abc",
			"sessionId":   "session-latest",
			"attachment": map[string]any{
				"type":      "image",
				"fileName":  "screenshot.png",
				"mediaType": "image/png",
				"content":   "<blob omitted>",
			},
		},
		{
			"type":      "user",
			"timestamp": "2026-06-10T12:00:03Z",
			"agentId":   "agent-abc",
			"sessionId": "session-latest",
			"message": map[string]any{
				"content": "Please inspect this screenshot.",
			},
		},
	}

	session, err := ConvertRecords(records, "fallback", "/tmp/claude-latest.jsonl")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}
	if session.Title == nil || *session.Title != "Investigate latest Claude session metadata" {
		t.Fatalf("expected ai-title to become session title, got %+v", session.Title)
	}
	config, ok := session.OperationalContext.FrameworkConfig.(map[string]any)
	if !ok {
		t.Fatalf("expected framework config map, got %+v", session.OperationalContext.FrameworkConfig)
	}
	if config["mode"] != "plan" || config["permission_mode"] != "bypassPermissions" || config["ai_title"] != "Investigate latest Claude session metadata" {
		t.Fatalf("expected mode/permission/title config, got %+v", config)
	}
	if config["agent_id"] != "agent-abc" || config["session_id"] != "session-latest" || config["parent_uuid"] != "parent-uuid" || config["is_sidechain"] != true {
		t.Fatalf("expected subagent/session metadata in config, got %+v", config)
	}
	if len(session.Annotations) != 1 {
		t.Fatalf("expected attachment annotation, got %d", len(session.Annotations))
	}
	annotation := session.Annotations[0]
	if annotation.Scope.Type != "session" || annotation.Scope.TargetID != "fallback" {
		t.Fatalf("unexpected attachment scope: %+v", annotation.Scope)
	}
	if annotation.Content.Category != "attachment" || !containsString(annotation.Content.Tags, "image") {
		t.Fatalf("expected image attachment annotation, got %+v", annotation.Content)
	}
	turnMetadata, ok := session.Turns[0].FrameworkMetadata.(map[string]any)
	if !ok || turnMetadata["agent_id"] != "agent-abc" || turnMetadata["session_id"] != "session-latest" {
		t.Fatalf("expected turn agent/session metadata, got %+v", session.Turns[0].FrameworkMetadata)
	}
}

func TestAdjustSubagentSessionAndParentLinking(t *testing.T) {
	parent := minitrace.BuildSessionSkeleton("parent-1", "claude-code", SourceFormatV2, AdapterVersion)
	turnIndex := 0
	parent.ToolCalls = []minitrace.ToolCall{
		minitrace.BuildToolCall(
			"tool-1",
			&turnIndex,
			nil,
			"Agent",
			"DELEGATE",
			nil,
			nil,
			map[string]any{"prompt": "inspect"},
			true,
			nil,
			nil,
			nil,
			nil,
			&minitrace.SpawnedAgent{
				AgentType:      "general",
				TaskScope:      "inspect",
				SubSessionID:   nil,
				OutcomeSummary: "",
			},
			classifyContentOrigin("Agent"),
			nil,
		),
	}

	child := minitrace.BuildSessionSkeleton("child-raw", "claude-code", SourceFormatV2, AdapterVersion)
	title := "Investigate file"
	child.Title = &title

	AdjustSubagentSession(&child, "agent-123", "parent-1", "bug-hunt")
	if child.ID != "agent-123" {
		t.Fatalf("expected adjusted child ID, got %s", child.ID)
	}
	if child.Title == nil || *child.Title != "[subagent] bug-hunt" {
		t.Fatalf("expected subagent title prefix, got %+v", child.Title)
	}
	if child.OperationalContext.FrameworkConfig == nil {
		t.Fatalf("expected framework_config with parent_session")
	}
	config := child.OperationalContext.FrameworkConfig.(map[string]any)
	if config["parent_session"] != "parent-1" {
		t.Fatalf("expected parent_session backlink metadata, got %+v", config)
	}

	LinkParentSubagents(&parent, []string{"agent-123"})
	if parent.ToolCalls[0].SpawnedAgent == nil || parent.ToolCalls[0].SpawnedAgent.SubSessionID == nil || *parent.ToolCalls[0].SpawnedAgent.SubSessionID != "agent-123" {
		t.Fatalf("expected parent tool call to backlink subagent, got %+v", parent.ToolCalls[0].SpawnedAgent)
	}
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
