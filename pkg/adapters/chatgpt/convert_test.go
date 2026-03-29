package chatgpt

import "testing"

func TestConvertConversationLinearizesTreeAndCapturesModels(t *testing.T) {
	conv := map[string]any{
		"conversation_id": "conv-1",
		"title":           "Example ChatGPT Conversation",
		"current_node":    "n3",
		"mapping": map[string]any{
			"root": map[string]any{
				"id":       "root",
				"parent":   nil,
				"children": []any{"n1"},
				"message":  nil,
			},
			"n1": map[string]any{
				"id":       "n1",
				"parent":   "root",
				"children": []any{"n2"},
				"message": map[string]any{
					"author": map[string]any{"role": "user"},
					"content": map[string]any{
						"content_type": "text",
						"parts":        []any{"Hello"},
					},
					"create_time": 1743240000.0,
					"metadata":    map[string]any{},
				},
			},
			"n2": map[string]any{
				"id":       "n2",
				"parent":   "n1",
				"children": []any{"n3"},
				"message": map[string]any{
					"author": map[string]any{"role": "assistant"},
					"content": map[string]any{
						"content_type": "multimodal_text",
						"parts": []any{
							"Hi there",
							map[string]any{
								"content_type":  "image_asset_pointer",
								"asset_pointer": "sediment://image-1",
							},
						},
					},
					"create_time": 1743240005.0,
					"metadata": map[string]any{
						"model_slug": "gpt-4o",
						"finish_details": map[string]any{
							"type": "stop",
						},
					},
				},
			},
			"n3": map[string]any{
				"id":       "n3",
				"parent":   "n2",
				"children": []any{},
				"message": map[string]any{
					"author": map[string]any{"role": "assistant"},
					"content": map[string]any{
						"content_type": "text",
						"parts":        []any{"Second reply"},
					},
					"create_time": 1743240010.0,
					"metadata": map[string]any{
						"model_slug": "gpt-5-2",
					},
				},
			},
		},
	}

	session, err := ConvertConversation(conv, "/tmp/chatgpt.zip")
	if err != nil {
		t.Fatalf("ConvertConversation returned error: %v", err)
	}

	if session.ID != "conv-1" {
		t.Fatalf("expected conv-1, got %s", session.ID)
	}
	if session.Environment.Model == nil || *session.Environment.Model != "gpt-5-2" {
		t.Fatalf("expected most recent assistant model gpt-5-2, got %+v", session.Environment.Model)
	}
	if session.Environment.PlatformType == nil || *session.Environment.PlatformType != "web" {
		t.Fatalf("expected web platform, got %+v", session.Environment.PlatformType)
	}
	if len(session.ToolCalls) != 0 {
		t.Fatalf("expected no tool calls, got %d", len(session.ToolCalls))
	}
	if len(session.Turns) != 3 {
		t.Fatalf("expected 3 turns, got %d", len(session.Turns))
	}
	if session.Turns[0].InputChannel == nil || *session.Turns[0].InputChannel != "user_input" {
		t.Fatalf("expected user_input channel, got %+v", session.Turns[0].InputChannel)
	}
	if session.Turns[1].ContentType == nil || *session.Turns[1].ContentType != "multimodal_text" {
		t.Fatalf("expected multimodal_text content type, got %+v", session.Turns[1].ContentType)
	}
	if session.Metrics.ModelSwitches == nil || *session.Metrics.ModelSwitches != 1 {
		t.Fatalf("expected model switch count 1, got %+v", session.Metrics.ModelSwitches)
	}
	if session.Metrics.UniqueModels == nil || *session.Metrics.UniqueModels != 2 {
		t.Fatalf("expected unique models 2, got %+v", session.Metrics.UniqueModels)
	}
}

func TestConvertConversationSkipsHiddenAndArtifactNodes(t *testing.T) {
	conv := map[string]any{
		"id":           "conv-2",
		"current_node": "n4",
		"mapping": map[string]any{
			"n1": map[string]any{
				"id":       "n1",
				"parent":   nil,
				"children": []any{"n2"},
				"message": map[string]any{
					"author":  map[string]any{"role": "system"},
					"content": map[string]any{"content_type": "text", "parts": []any{}},
					"weight":  0.0,
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
						"parts":        []any{"Prompt"},
					},
				},
			},
			"n3": map[string]any{
				"id":       "n3",
				"parent":   "n2",
				"children": []any{"n4"},
				"message": map[string]any{
					"author": map[string]any{"role": "assistant"},
					"content": map[string]any{
						"content_type": "code",
						"parts":        []any{},
					},
					"metadata": map[string]any{
						"is_visually_hidden_from_conversation": true,
					},
				},
			},
			"n4": map[string]any{
				"id":       "n4",
				"parent":   "n3",
				"children": []any{},
				"message": map[string]any{
					"author": map[string]any{"role": "assistant"},
					"content": map[string]any{
						"content_type": "text",
						"parts":        []any{"Visible reply"},
					},
					"metadata": map[string]any{
						"model_slug": "gpt-4o-mini",
					},
				},
			},
		},
	}

	session, err := ConvertConversation(conv, "")
	if err != nil {
		t.Fatalf("ConvertConversation returned error: %v", err)
	}

	if len(session.Turns) != 2 {
		t.Fatalf("expected 2 turns after skipping hidden/system artifact nodes, got %d", len(session.Turns))
	}
	if session.Turns[1].Content != "Visible reply" {
		t.Fatalf("unexpected final reply: %q", session.Turns[1].Content)
	}
}
