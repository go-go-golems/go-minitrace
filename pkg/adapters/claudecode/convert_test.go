package claudecode

import "testing"

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
	if session.OperationalContext.WorkingDirectory == nil || *session.OperationalContext.WorkingDirectory != "~/project" {
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
	if toolCall.Input.FilePath == nil || *toolCall.Input.FilePath != "~/project/app.go" {
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
