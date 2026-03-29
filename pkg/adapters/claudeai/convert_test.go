package claudeai

import "testing"

func TestConvertConversationBuildsSessionAndToolCalls(t *testing.T) {
	conv := map[string]any{
		"uuid":    "conv-1",
		"name":    "Example conversation",
		"summary": "Summary",
		"chat_messages": []any{
			map[string]any{
				"sender":     "human",
				"created_at": "2026-03-29T10:00:00Z",
				"text":       "Read file and create output",
				"content": []any{
					map[string]any{"type": "text", "text": "Read file and create output"},
				},
			},
			map[string]any{
				"sender":     "assistant",
				"created_at": "2026-03-29T10:00:05Z",
				"text":       "Done.",
				"content": []any{
					map[string]any{"type": "thinking", "thinking": "Need to inspect file"},
					map[string]any{"type": "text", "text": "I'll inspect the file."},
					map[string]any{
						"type":            "tool_use",
						"id":              "tool-1",
						"name":            "view",
						"start_timestamp": "2026-03-29T10:00:06Z",
						"stop_timestamp":  "2026-03-29T10:00:07Z",
						"input": map[string]any{
							"path": "/mnt/user-data/outputs/app.go",
						},
						"message": "Reading app.go",
					},
					map[string]any{
						"type":        "tool_result",
						"tool_use_id": "tool-1",
						"name":        "view",
						"content": []any{
							map[string]any{"type": "text", "text": "package main"},
						},
						"is_error": false,
					},
				},
			},
		},
	}

	session, err := ConvertConversation(conv, "/tmp/data.zip")
	if err != nil {
		t.Fatalf("ConvertConversation returned error: %v", err)
	}

	if session.ID != "conv-1" {
		t.Fatalf("expected session id conv-1, got %s", session.ID)
	}
	if session.Environment.PlatformType == nil || *session.Environment.PlatformType != "web" {
		t.Fatalf("expected platform web, got %+v", session.Environment.PlatformType)
	}
	if session.Environment.ProviderHint == nil || *session.Environment.ProviderHint != "anthropic" {
		t.Fatalf("expected anthropic provider, got %+v", session.Environment.ProviderHint)
	}
	if session.Provenance.SourcePath == nil || *session.Provenance.SourcePath != "/tmp/data.zip" {
		t.Fatalf("expected source path, got %+v", session.Provenance.SourcePath)
	}
	if len(session.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(session.Turns))
	}
	if session.Turns[1].Thinking == nil || *session.Turns[1].Thinking != "Need to inspect file" {
		t.Fatalf("expected assistant thinking, got %+v", session.Turns[1].Thinking)
	}
	if len(session.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(session.ToolCalls))
	}
	toolCall := session.ToolCalls[0]
	if toolCall.OperationType != "READ" {
		t.Fatalf("expected READ operation, got %s", toolCall.OperationType)
	}
	if toolCall.Output.Result == nil || *toolCall.Output.Result != "package main" {
		t.Fatalf("unexpected tool result: %+v", toolCall.Output.Result)
	}
	if toolCall.Output.DurationMS == nil || *toolCall.Output.DurationMS != 1000 {
		t.Fatalf("expected duration 1000ms, got %+v", toolCall.Output.DurationMS)
	}
	if session.Flags.ContainsPII != true || session.Classification != "confidential" {
		t.Fatalf("expected confidential PII handling, got flags=%+v classification=%s", session.Flags, session.Classification)
	}
}

func TestConvertConversationAnnotatesOrphanAndRedactedToolCalls(t *testing.T) {
	conv := map[string]any{
		"uuid": "conv-2",
		"chat_messages": []any{
			map[string]any{
				"sender":     "assistant",
				"created_at": "2026-03-29T10:00:05Z",
				"content": []any{
					map[string]any{
						"type": "tool_use",
						"id":   "tool-1",
						"name": "bash_tool",
						"input": map[string]any{
							"command": "echo hi",
						},
					},
				},
			},
			map[string]any{
				"sender":     "assistant",
				"created_at": "2026-03-29T10:00:06Z",
				"content": []any{
					map[string]any{
						"type": "tool_use",
						"id":   "tool-2",
						"name": "view",
						"input": map[string]any{
							"path": "/mnt/user-data/outputs/app.go",
						},
					},
					map[string]any{
						"type":        "tool_result",
						"tool_use_id": "tool-2",
						"name":        "view",
						"content": []any{
							map[string]any{"type": "text", "text": RedactedMarker},
						},
						"is_error": false,
					},
				},
			},
		},
	}

	session, err := ConvertConversation(conv, "")
	if err != nil {
		t.Fatalf("ConvertConversation returned error: %v", err)
	}

	if len(session.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(session.ToolCalls))
	}
	if session.ToolCalls[0].Output.Error == nil || *session.ToolCalls[0].Output.Error != "no tool result received" {
		t.Fatalf("expected orphan tool error, got %+v", session.ToolCalls[0].Output.Error)
	}
	if session.ToolCalls[1].Output.Redacted == nil || *session.ToolCalls[1].Output.Redacted != true {
		t.Fatalf("expected redacted tool result, got %+v", session.ToolCalls[1].Output.Redacted)
	}
	if len(session.Annotations) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(session.Annotations))
	}
}
