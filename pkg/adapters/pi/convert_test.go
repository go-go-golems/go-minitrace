package pi

import "testing"

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
				"role":     "assistant",
				"provider": "claude-agent-sdk",
				"model":    "claude-opus-4-6",
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
	if session.OperationalContext.WorkingDirectory == nil || *session.OperationalContext.WorkingDirectory != "~/project" {
		t.Fatalf("expected normalized cwd, got %+v", session.OperationalContext.WorkingDirectory)
	}
	if len(session.Turns) != 3 {
		t.Fatalf("expected 3 turns, got %d", len(session.Turns))
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
