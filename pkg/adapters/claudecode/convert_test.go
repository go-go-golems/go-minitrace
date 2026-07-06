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
	if len(session.Events) != 4 {
		t.Fatalf("expected mode/permission/title/attachment events, got %d", len(session.Events))
	}
	if session.Events[0].Kind != "mode_change" || session.Events[1].Kind != "permission_mode_change" || session.Events[2].Kind != "title_change" || session.Events[3].Kind != "attachment" {
		t.Fatalf("unexpected Claude lifecycle events: %+v", session.Events)
	}
	if len(session.Attachments) != 1 {
		t.Fatalf("expected first-class attachment, got %d", len(session.Attachments))
	}
	attachment := session.Attachments[0]
	if attachment.ID != "attachment-1" || attachment.Kind != "image" || attachment.MediaType != "image/png" || attachment.Name != "screenshot.png" {
		t.Fatalf("unexpected attachment: %+v", attachment)
	}
	if attachment.EventID == nil || *attachment.EventID != session.Events[3].ID {
		t.Fatalf("expected attachment to link to attachment event, got %+v", attachment.EventID)
	}
	if session.Events[3].AttachmentID == nil || *session.Events[3].AttachmentID != attachment.ID {
		t.Fatalf("expected attachment event to link to attachment, got %+v", session.Events[3].AttachmentID)
	}
	if len(session.Annotations) != 0 {
		t.Fatalf("expected source attachments to avoid annotation fallback, got %d", len(session.Annotations))
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
	if child.Coordination.PredecessorSession == nil || *child.Coordination.PredecessorSession != "parent-1" {
		t.Fatalf("expected predecessor parent-1, got %+v", child.Coordination.PredecessorSession)
	}

	LinkParentSubagents(&parent, []string{"agent-123"})
	if parent.ToolCalls[0].SpawnedAgent == nil || parent.ToolCalls[0].SpawnedAgent.SubSessionID == nil || *parent.ToolCalls[0].SpawnedAgent.SubSessionID != "agent-123" {
		t.Fatalf("expected parent tool call to backlink subagent, got %+v", parent.ToolCalls[0].SpawnedAgent)
	}
}

func TestConvertRecordsDerivesToolDurationAndPreservesEmitTimestamp(t *testing.T) {
	records := []map[string]any{
		{
			"type":      "assistant",
			"timestamp": "2026-03-29T10:00:10Z",
			"message": map[string]any{
				"content": []any{
					map[string]any{
						"type":  "tool_use",
						"id":    "tool-1",
						"name":  "Bash",
						"input": map[string]any{"command": "ls"},
					},
				},
			},
		},
		{
			"type":      "user",
			"timestamp": "2026-03-29T10:00:12.500Z",
			"message": map[string]any{
				"content": []any{
					map[string]any{
						"type":        "tool_result",
						"tool_use_id": "tool-1",
						"content":     "file.txt",
						"is_error":    false,
					},
				},
			},
		},
	}

	session, err := ConvertRecords(records, "session-duration", "/tmp/session-duration.jsonl")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}
	if len(session.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(session.ToolCalls))
	}
	toolCall := session.ToolCalls[0]
	if toolCall.Timestamp == nil || *toolCall.Timestamp != "2026-03-29T10:00:10Z" {
		t.Fatalf("expected emit timestamp to be preserved, got %+v", toolCall.Timestamp)
	}
	if toolCall.Output.DurationMS == nil || *toolCall.Output.DurationMS != 2500 {
		t.Fatalf("expected 2500ms duration, got %+v", toolCall.Output.DurationMS)
	}
}

func TestConvertRecordsMapsToolUseResult(t *testing.T) {
	// Record shapes modeled on real ~/.claude/projects transcripts: Bash
	// results carry {stdout, stderr, interrupted, isImage, noOutputExpected}
	// and failing Bash results are plain strings "Error: Exit code N\n...".
	records := []map[string]any{
		{
			"type":      "assistant",
			"timestamp": "2026-03-29T10:00:10Z",
			"message": map[string]any{
				"content": []any{
					map[string]any{"type": "tool_use", "id": "tool-ok", "name": "Bash", "input": map[string]any{"command": "echo hi"}},
					map[string]any{"type": "tool_use", "id": "tool-exit", "name": "Bash", "input": map[string]any{"command": "false"}},
					map[string]any{"type": "tool_use", "id": "tool-stderr", "name": "Bash", "input": map[string]any{"command": "boom"}},
					map[string]any{"type": "tool_use", "id": "tool-int", "name": "Bash", "input": map[string]any{"command": "sleep 100"}},
				},
			},
		},
		{
			"type":      "user",
			"timestamp": "2026-03-29T10:00:11Z",
			"toolUseResult": map[string]any{
				"stdout":           "hi",
				"stderr":           "",
				"interrupted":      false,
				"isImage":          false,
				"noOutputExpected": false,
			},
			"message": map[string]any{
				"content": []any{
					map[string]any{"type": "tool_result", "tool_use_id": "tool-ok", "content": "hi", "is_error": false},
				},
			},
		},
		{
			"type":          "user",
			"timestamp":     "2026-03-29T10:00:12Z",
			"toolUseResult": "Error: Exit code 2\nsome output",
			"message": map[string]any{
				"content": []any{
					map[string]any{"type": "tool_result", "tool_use_id": "tool-exit", "content": "Error: Exit code 2\nsome output", "is_error": true},
				},
			},
		},
		{
			"type":      "user",
			"timestamp": "2026-03-29T10:00:13Z",
			"toolUseResult": map[string]any{
				"stdout":           "",
				"stderr":           "boom: command not found",
				"interrupted":      false,
				"isImage":          false,
				"noOutputExpected": false,
			},
			"message": map[string]any{
				"content": []any{
					map[string]any{"type": "tool_result", "tool_use_id": "tool-stderr", "content": "failure text", "is_error": true},
				},
			},
		},
		{
			"type":      "user",
			"timestamp": "2026-03-29T10:00:14Z",
			"toolUseResult": map[string]any{
				"stdout":           "",
				"stderr":           "",
				"interrupted":      true,
				"isImage":          false,
				"noOutputExpected": false,
			},
			"message": map[string]any{
				"content": []any{
					map[string]any{"type": "tool_result", "tool_use_id": "tool-int", "content": "[Request interrupted by user]", "is_error": false},
				},
			},
		},
	}

	session, err := ConvertRecords(records, "session-tur", "/tmp/session-tur.jsonl")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}
	if len(session.ToolCalls) != 4 {
		t.Fatalf("expected 4 tool calls, got %d", len(session.ToolCalls))
	}
	byID := map[string]minitrace.ToolCall{}
	for _, toolCall := range session.ToolCalls {
		byID[toolCall.ID] = toolCall
	}

	okCall := byID["tool-ok"]
	okMetadata, ok := okCall.FrameworkMetadata.(map[string]any)
	if !ok {
		t.Fatalf("expected framework metadata on successful call, got %+v", okCall.FrameworkMetadata)
	}
	preserved, ok := okMetadata["tool_use_result"].(map[string]any)
	if !ok || preserved["stdout"] != "hi" {
		t.Fatalf("expected toolUseResult preserved in metadata, got %+v", okMetadata["tool_use_result"])
	}

	exitCall := byID["tool-exit"]
	if exitCall.Output.ExitCode == nil || *exitCall.Output.ExitCode != 2 {
		t.Fatalf("expected exit code 2 parsed from string toolUseResult, got %+v", exitCall.Output.ExitCode)
	}
	exitMetadata := exitCall.FrameworkMetadata.(map[string]any)
	if exitMetadata["tool_use_result"] != "Error: Exit code 2\nsome output" {
		t.Fatalf("expected string toolUseResult preserved, got %+v", exitMetadata["tool_use_result"])
	}

	stderrCall := byID["tool-stderr"]
	if stderrCall.Output.Success {
		t.Fatalf("expected stderr call to be failed")
	}
	if stderrCall.Output.Error == nil || *stderrCall.Output.Error != "boom: command not found" {
		t.Fatalf("expected stderr in error, got %+v", stderrCall.Output.Error)
	}

	interruptedCall := byID["tool-int"]
	if interruptedCall.Output.Success {
		t.Fatalf("expected interrupted call to be failed")
	}
	if interruptedCall.Output.Error == nil || *interruptedCall.Output.Error != "interrupted by user" {
		t.Fatalf("expected interruption error, got %+v", interruptedCall.Output.Error)
	}
	interruptedMetadata := interruptedCall.FrameworkMetadata.(map[string]any)
	if interruptedMetadata["interrupted"] != true {
		t.Fatalf("expected interrupted metadata flag, got %+v", interruptedMetadata)
	}
}

func TestConvertRecordsReadsSessionContextFromAnyRecord(t *testing.T) {
	records := []map[string]any{
		{
			"type":      "file-history-snapshot",
			"messageId": "snapshot-1",
		},
		{
			"type":      "user",
			"timestamp": "2026-03-29T10:00:00Z",
			"cwd":       "/home/manuel/project",
			"version":   "2.1.76",
			"gitBranch": "feature/foo",
			"message": map[string]any{
				"content": "hello",
			},
		},
	}

	session, err := ConvertRecords(records, "session-ctx", "/tmp/session-ctx.jsonl")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}
	expectedCwd := minitrace.NormalizePath("/home/manuel/project")
	if session.OperationalContext.WorkingDirectory == nil || *session.OperationalContext.WorkingDirectory != expectedCwd {
		t.Fatalf("expected cwd from later record, got %+v", session.OperationalContext.WorkingDirectory)
	}
	if session.Environment.AgentVersion == nil || *session.Environment.AgentVersion != "2.1.76" {
		t.Fatalf("expected version from later record, got %+v", session.Environment.AgentVersion)
	}
	if session.OperationalContext.GitBranch == nil || *session.OperationalContext.GitBranch != "feature/foo" {
		t.Fatalf("expected git branch from later record, got %+v", session.OperationalContext.GitBranch)
	}
}

func TestConvertRecordsPreservesSignedThinkingPresence(t *testing.T) {
	records := []map[string]any{
		{
			"type":      "assistant",
			"timestamp": "2026-07-06T10:00:00Z",
			"message": map[string]any{
				"role":  "assistant",
				"model": "claude-opus-4-1",
				"content": []any{
					map[string]any{"type": "thinking", "thinking": "", "signature": "sig-1"},
					map[string]any{"type": "text", "text": "answer"},
				},
			},
		},
	}

	session, err := ConvertRecords(records, "session-signed-thinking", "/tmp/signed-thinking.jsonl")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}
	if len(session.Turns) != 1 {
		t.Fatalf("expected one turn, got %d", len(session.Turns))
	}
	if session.Turns[0].Thinking != nil {
		t.Fatalf("empty signed thinking must not synthesize cleartext thinking, got %+v", session.Turns[0].Thinking)
	}
	metadata, ok := session.Turns[0].FrameworkMetadata.(map[string]any)
	if !ok {
		t.Fatalf("expected framework metadata map, got %+v", session.Turns[0].FrameworkMetadata)
	}
	if metadata["signed_thinking_blocks"] != 1 || metadata["thinking_signature_present"] != true {
		t.Fatalf("expected signed thinking metadata, got %+v", metadata)
	}
}
