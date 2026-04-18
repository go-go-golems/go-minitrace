package pi

import (
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestConvertRecordsMatchesToolResultsAndBuildsSession(t *testing.T) {
	records := []map[string]any{
		{
			"type":      "session",
			"id":        "session-1",
			"version":   3,
			"timestamp": "2026-03-29T12:00:00Z",
			"cwd":       "/home/manuel/project",
		},
		{
			"type":      "model_change",
			"provider":  "claude-agent-sdk",
			"modelId":   "claude-opus-4-6",
			"timestamp": "2026-03-29T12:00:01Z",
		},
		{
			"type":          "thinking_level_change",
			"thinkingLevel": "high",
			"timestamp":     "2026-03-29T12:00:01Z",
		},
		{
			"type":      "message",
			"timestamp": "2026-03-29T12:00:02Z",
			"message": map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": "Read app.go"},
				},
			},
		},
		{
			"type":      "message",
			"timestamp": "2026-03-29T12:00:03Z",
			"message": map[string]any{
				"role":         "assistant",
				"provider":     "claude-agent-sdk",
				"model":        "claude-opus-4-6",
				"stopReason":   "toolUse",
				"errorMessage": "",
				"usage": map[string]any{
					"input":      10,
					"output":     20,
					"cacheRead":  3,
					"cacheWrite": 1,
					"cost": map[string]any{
						"total": 0.25,
					},
				},
				"content": []any{
					map[string]any{"type": "text", "text": "I'll inspect the file."},
					map[string]any{
						"type": "toolCall",
						"id":   "tool-1",
						"name": "read",
						"arguments": map[string]any{
							"path": "/home/manuel/project/app.go",
						},
					},
				},
			},
		},
		{
			"type":      "message",
			"timestamp": "2026-03-29T12:00:04Z",
			"message": map[string]any{
				"role":       "toolResult",
				"toolCallId": "tool-1",
				"toolName":   "read",
				"content": []any{
					map[string]any{"type": "text", "text": "package main"},
				},
			},
		},
	}

	session, err := ConvertRecords(records, "fallback", "/tmp/session.jsonl")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}

	if session.ID != "session-1" {
		t.Fatalf("expected session id session-1, got %s", session.ID)
	}
	if session.Environment.Model == nil || *session.Environment.Model != "claude-opus-4-6" {
		t.Fatalf("expected model claude-opus-4-6, got %+v", session.Environment.Model)
	}
	if session.Environment.ProviderHint == nil || *session.Environment.ProviderHint != "claude-agent-sdk" {
		t.Fatalf("expected provider claude-agent-sdk, got %+v", session.Environment.ProviderHint)
	}
	expectedWorkingDirectory := minitrace.NormalizePath("/home/manuel/project")
	if session.OperationalContext.WorkingDirectory == nil || *session.OperationalContext.WorkingDirectory != expectedWorkingDirectory {
		t.Fatalf("expected normalized cwd, got %+v", session.OperationalContext.WorkingDirectory)
	}
	if len(session.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(session.Turns))
	}
	if len(session.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(session.ToolCalls))
	}
	if session.ToolCalls[0].OperationType != "READ" {
		t.Fatalf("expected READ operation, got %s", session.ToolCalls[0].OperationType)
	}
	if session.ToolCalls[0].Output.Result == nil || *session.ToolCalls[0].Output.Result != "package main" {
		t.Fatalf("unexpected tool output: %+v", session.ToolCalls[0].Output.Result)
	}
	if session.Metrics.TotalInputTokens == nil || *session.Metrics.TotalInputTokens != 10 {
		t.Fatalf("expected input tokens 10, got %+v", session.Metrics.TotalInputTokens)
	}
	assistantMetadata, ok := session.Turns[1].FrameworkMetadata.(map[string]any)
	if !ok {
		t.Fatalf("expected assistant framework metadata, got %+v", session.Turns[1].FrameworkMetadata)
	}
	if assistantMetadata["stop_reason"] != "toolUse" || assistantMetadata["error_message"] != "" {
		t.Fatalf("unexpected assistant metadata: %+v", assistantMetadata)
	}
	if session.Metrics.SessionCost == nil || *session.Metrics.SessionCost != 0.25 {
		t.Fatalf("expected session cost 0.25, got %+v", session.Metrics.SessionCost)
	}
}

func TestConvertRecordsAnnotatesOrphanToolCall(t *testing.T) {
	records := []map[string]any{
		{
			"type":      "message",
			"timestamp": "2026-03-29T12:00:03Z",
			"message": map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{
						"type": "toolCall",
						"id":   "tool-1",
						"name": "read",
						"arguments": map[string]any{
							"path": "/tmp/app.go",
						},
					},
				},
			},
		},
	}

	session, err := ConvertRecords(records, "fallback", "/tmp/session.jsonl")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}

	if len(session.ToolCalls) != 1 {
		t.Fatalf("expected orphan tool call to be kept")
	}
	if session.ToolCalls[0].Output.Error == nil || *session.ToolCalls[0].Output.Error != "no tool result received" {
		t.Fatalf("expected orphan tool error, got %+v", session.ToolCalls[0].Output.Error)
	}
	if len(session.Annotations) != 1 {
		t.Fatalf("expected one orphan annotation, got %d", len(session.Annotations))
	}
}

func TestConvertRecordsMessageLevelToolResultPreservesIsError(t *testing.T) {
	records := []map[string]any{
		{
			"type":      "session",
			"id":        "session-tool-results",
			"version":   3,
			"timestamp": "2026-04-16T00:00:00Z",
			"cwd":       "/tmp",
		},
		{
			"type":      "message",
			"id":        "m1",
			"parentId":  nil,
			"timestamp": "2026-04-16T00:00:01Z",
			"message": map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": "test"},
				},
			},
		},
		{
			"type":      "message",
			"id":        "m2",
			"parentId":  "m1",
			"timestamp": "2026-04-16T00:00:02Z",
			"message": map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{
						"type": "toolCall",
						"id":   "tc-success",
						"name": "read",
						"arguments": map[string]any{
							"path": "/tmp/ok.md",
						},
					},
					map[string]any{
						"type": "toolCall",
						"id":   "tc-fail",
						"name": "edit",
						"arguments": map[string]any{
							"path": "/tmp/missing.md",
							"edits": []any{
								map[string]any{"oldText": "a", "newText": "b"},
							},
						},
					},
				},
			},
		},
		{
			"type":      "message",
			"id":        "m3",
			"parentId":  "m2",
			"timestamp": "2026-04-16T00:00:03Z",
			"message": map[string]any{
				"role":       "toolResult",
				"toolCallId": "tc-success",
				"toolName":   "read",
				"isError":    false,
				"content": []any{
					map[string]any{"type": "text", "text": "ok"},
				},
			},
		},
		{
			"type":      "message",
			"id":        "m4",
			"parentId":  "m2",
			"timestamp": "2026-04-16T00:00:04Z",
			"message": map[string]any{
				"role":       "toolResult",
				"toolCallId": "tc-fail",
				"toolName":   "edit",
				"isError":    true,
				"details": map[string]any{
					"diff":             "- old\n+ new",
					"firstChangedLine": 13,
				},
				"content": []any{
					map[string]any{"type": "text", "text": "File not found: /tmp/missing.md"},
				},
			},
		},
	}

	session, err := ConvertRecords(records, "fallback", "/tmp/session.jsonl")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}

	if len(session.Turns) != 2 {
		t.Fatalf("expected toolResult messages to be absorbed into tool calls, got %d turns", len(session.Turns))
	}
	if len(session.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(session.ToolCalls))
	}

	toolCallsByID := map[string]minitrace.ToolCall{}
	for _, toolCall := range session.ToolCalls {
		toolCallsByID[toolCall.ID] = toolCall
	}

	successCall, ok := toolCallsByID["tc-success"]
	if !ok {
		t.Fatalf("expected tc-success tool call in session")
	}
	if !successCall.Output.Success {
		t.Fatalf("expected tc-success to be successful")
	}
	if successCall.Output.Result == nil || *successCall.Output.Result != "ok" {
		t.Fatalf("expected tc-success result to be ok, got %+v", successCall.Output.Result)
	}

	failedCall, ok := toolCallsByID["tc-fail"]
	if !ok {
		t.Fatalf("expected tc-fail tool call in session")
	}
	if failedCall.Output.Success {
		t.Fatalf("expected tc-fail to be marked as failed")
	}
	if failedCall.Output.Error == nil || *failedCall.Output.Error != "File not found: /tmp/missing.md" {
		t.Fatalf("expected tc-fail error to be propagated, got %+v", failedCall.Output.Error)
	}
	failedMetadata, ok := failedCall.FrameworkMetadata.(map[string]any)
	if !ok {
		t.Fatalf("expected failed tool metadata map, got %+v", failedCall.FrameworkMetadata)
	}
	if failedMetadata["diff"] != "- old\n+ new" || failedMetadata["first_changed_line"] != 13 {
		t.Fatalf("expected diff metadata to be preserved, got %+v", failedMetadata)
	}
}
