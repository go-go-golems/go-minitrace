package chatgpt

import "testing"

func TestConvertTranscriptConversationBuildsAssistantToolCalls(t *testing.T) {
	conv := map[string]any{
		"conversation_id": "alt-conv-1",
		"title":           "Alt Export",
		"current_node":    "n7",
		"mapping": map[string]any{
			"root": map[string]any{
				"id":       "root",
				"parent":   nil,
				"children": []any{"n1"},
			},
			"n1": map[string]any{
				"id":       "n1",
				"parent":   "root",
				"children": []any{"n2"},
				"message": map[string]any{
					"author": map[string]any{"role": "user"},
					"content": map[string]any{
						"content_type": "text",
						"parts":        []any{"why duckdb?"},
					},
					"create_time": 1743240000.0,
					"metadata": map[string]any{
						"turn_exchange_id": "ex-1",
					},
				},
			},
			"n2": map[string]any{
				"id":       "n2",
				"parent":   "n1",
				"children": []any{"n3"},
				"message": map[string]any{
					"author": map[string]any{"role": "assistant"},
					"content": map[string]any{
						"content_type": "thoughts",
						"summary":      "Thought for 4 seconds",
					},
					"create_time": 1743240001.0,
					"metadata": map[string]any{
						"turn_exchange_id": "ex-1",
						"model_slug":       "gpt-5-4-thinking",
					},
				},
			},
			"n3": map[string]any{
				"id":       "n3",
				"parent":   "n2",
				"children": []any{"n4"},
				"message": map[string]any{
					"author":    map[string]any{"role": "assistant"},
					"recipient": "web.run",
					"content": map[string]any{
						"content_type": "code",
						"language":     "json",
						"text":         `{"search_query":[{"q":"duckdb official docs"}],"response_length":"short"}`,
					},
					"create_time": 1743240002.0,
					"metadata": map[string]any{
						"turn_exchange_id": "ex-1",
						"model_slug":       "gpt-5-4-thinking",
					},
				},
			},
			"n4": map[string]any{
				"id":       "n4",
				"parent":   "n3",
				"children": []any{"n5"},
				"message": map[string]any{
					"author": map[string]any{"role": "tool", "name": "web.run"},
					"content": map[string]any{
						"content_type": "text",
						"parts":        []any{""},
					},
					"create_time": 1743240003.0,
					"metadata": map[string]any{
						"turn_exchange_id": "ex-1",
						"search_model_queries": map[string]any{
							"queries": []any{"duckdb official docs"},
						},
					},
				},
			},
			"n5": map[string]any{
				"id":       "n5",
				"parent":   "n4",
				"children": []any{"n6"},
				"message": map[string]any{
					"author": map[string]any{"role": "assistant"},
					"content": map[string]any{
						"content_type": "reasoning_recap",
						"content":      "Thought for 4 seconds",
					},
					"create_time": 1743240004.0,
					"metadata": map[string]any{
						"turn_exchange_id": "ex-1",
						"model_slug":       "gpt-5-4-thinking",
					},
				},
			},
			"n6": map[string]any{
				"id":       "n6",
				"parent":   "n5",
				"children": []any{"n7"},
				"message": map[string]any{
					"author": map[string]any{"role": "assistant"},
					"content": map[string]any{
						"content_type": "text",
						"parts":        []any{"DuckDB is an in-process analytical database."},
					},
					"create_time": 1743240005.0,
					"metadata": map[string]any{
						"turn_exchange_id": "ex-1",
						"model_slug":       "gpt-5-4-thinking",
					},
				},
			},
			"n7": map[string]any{
				"id":       "n7",
				"parent":   "n6",
				"children": []any{},
				"message": map[string]any{
					"author": map[string]any{"role": "system"},
					"content": map[string]any{
						"content_type": "text",
						"parts":        []any{""},
					},
					"metadata": map[string]any{
						"is_visually_hidden_from_conversation": true,
						"turn_exchange_id":                     "ex-1",
					},
					"weight": 0.0,
				},
			},
		},
	}

	session, err := ConvertTranscriptConversation(conv, "/tmp/chatgpt-exports/example.json")
	if err != nil {
		t.Fatalf("ConvertTranscriptConversation returned error: %v", err)
	}

	if session.Provenance.SourceFormat != TranscriptSourceFormat {
		t.Fatalf("expected %s, got %s", TranscriptSourceFormat, session.Provenance.SourceFormat)
	}
	if len(session.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(session.Turns))
	}
	if session.Turns[1].Thinking == nil || *session.Turns[1].Thinking == "" {
		t.Fatalf("expected assistant thinking to be preserved")
	}
	if len(session.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(session.ToolCalls))
	}
	toolCall := session.ToolCalls[0]
	if toolCall.ToolName != "web.run" {
		t.Fatalf("expected web.run tool, got %s", toolCall.ToolName)
	}
	if toolCall.OperationType != "READ" {
		t.Fatalf("expected READ operation, got %s", toolCall.OperationType)
	}
	if toolCall.Output.ContentOrigin == nil || *toolCall.Output.ContentOrigin != "web" {
		t.Fatalf("expected web content origin, got %+v", toolCall.Output.ContentOrigin)
	}
	if toolCall.Input.Arguments == nil {
		t.Fatalf("expected parsed tool arguments")
	}
	if session.Environment.Model == nil || *session.Environment.Model != "gpt-5-4-thinking" {
		t.Fatalf("expected most recent model gpt-5-4-thinking, got %+v", session.Environment.Model)
	}
}

func TestConvertTranscriptConversationPreservesFileSearchAndSkipsHiddenSystem(t *testing.T) {
	conv := map[string]any{
		"id":           "alt-conv-2",
		"current_node": "n6",
		"mapping": map[string]any{
			"n1": map[string]any{
				"id":       "n1",
				"parent":   nil,
				"children": []any{"n2"},
				"message": map[string]any{
					"author": map[string]any{"role": "system"},
					"content": map[string]any{
						"content_type": "text",
						"parts":        []any{""},
					},
					"metadata": map[string]any{
						"is_visually_hidden_from_conversation": true,
					},
					"weight": 0.0,
				},
			},
			"n2": map[string]any{
				"id":       "n2",
				"parent":   "n1",
				"children": []any{"n3"},
				"message": map[string]any{
					"author": map[string]any{"role": "user"},
					"content": map[string]any{
						"content_type": "text",
						"parts":        []any{"search local notes"},
					},
					"metadata": map[string]any{
						"turn_exchange_id": "ex-2",
					},
				},
			},
			"n3": map[string]any{
				"id":       "n3",
				"parent":   "n2",
				"children": []any{"n4"},
				"message": map[string]any{
					"author":    map[string]any{"role": "assistant"},
					"recipient": "file_search",
					"content": map[string]any{
						"content_type": "code",
						"language":     "json",
						"text":         `{"query":"duckdb vs sqlite"}`,
					},
					"metadata": map[string]any{
						"turn_exchange_id": "ex-2",
						"model_slug":       "gpt-5-4-thinking",
					},
				},
			},
			"n4": map[string]any{
				"id":       "n4",
				"parent":   "n3",
				"children": []any{"n5"},
				"message": map[string]any{
					"author": map[string]any{"role": "tool", "name": "file_search"},
					"content": map[string]any{
						"content_type": "text",
						"parts":        []any{""},
					},
					"metadata": map[string]any{
						"turn_exchange_id": "ex-2",
						"matches":          []any{"note-1.md"},
					},
				},
			},
			"n5": map[string]any{
				"id":       "n5",
				"parent":   "n4",
				"children": []any{"n6"},
				"message": map[string]any{
					"author": map[string]any{"role": "assistant"},
					"content": map[string]any{
						"content_type": "text",
						"parts":        []any{"I found one relevant note."},
					},
					"metadata": map[string]any{
						"turn_exchange_id": "ex-2",
						"model_slug":       "gpt-5-4-thinking",
					},
				},
			},
			"n6": map[string]any{
				"id":       "n6",
				"parent":   "n5",
				"children": []any{},
			},
		},
	}

	session, err := ConvertTranscriptConversation(conv, "")
	if err != nil {
		t.Fatalf("ConvertTranscriptConversation returned error: %v", err)
	}

	if len(session.Turns) != 2 {
		t.Fatalf("expected 2 visible turns, got %d", len(session.Turns))
	}
	if len(session.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(session.ToolCalls))
	}
	if session.ToolCalls[0].ToolName != "file_search" {
		t.Fatalf("expected file_search tool, got %s", session.ToolCalls[0].ToolName)
	}
	if session.ToolCalls[0].Output.ContentOrigin == nil || *session.ToolCalls[0].Output.ContentOrigin != "database" {
		t.Fatalf("expected database content origin, got %+v", session.ToolCalls[0].Output.ContentOrigin)
	}
}
