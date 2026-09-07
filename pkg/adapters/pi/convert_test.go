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
	if len(session.Events) != 2 {
		t.Fatalf("expected model/thinking source events, got %d", len(session.Events))
	}
	if session.Events[0].Kind != "model_change" || session.Events[1].Kind != "thinking_level_change" {
		t.Fatalf("unexpected Pi lifecycle events: %+v", session.Events)
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

func TestConvertRecordsPreservesPiNonMessageRecordsAsEvents(t *testing.T) {
	records := []map[string]any{
		{
			"type":      "session",
			"id":        "session-events",
			"timestamp": "2026-06-10T12:00:00Z",
		},
		{
			"type":      "session_info",
			"name":      "Configurable Context Diagram StyleSet Cutover",
			"timestamp": "2026-06-10T12:00:01Z",
		},
		{
			"type":       "custom",
			"customType": "pinned-skills-state",
			"timestamp":  "2026-06-10T12:00:02Z",
			"data": map[string]any{
				"activeConfig": map[string]any{"skills": []any{"diary", "docmgr"}},
			},
		},
		{
			"type":      "compaction",
			"timestamp": "2026-06-10T12:00:03Z",
			"details": map[string]any{
				"modifiedFiles":                 []any{"pkg/foo.go", "pkg/bar.go"},
				"customInstructionsAppended":    false,
				"edited":                        map[string]any{"prompt": true},
				"largeUnmodeledSourceFieldName": "kept in raw json",
			},
		},
	}

	session, err := ConvertRecords(records, "fallback", "/tmp/session.jsonl")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}

	if session.Title == nil || *session.Title != "Configurable Context Diagram StyleSet Cutover" {
		t.Fatalf("expected session_info title, got %+v", session.Title)
	}
	if len(session.Events) != 3 {
		t.Fatalf("expected 3 Pi source events, got %d", len(session.Events))
	}
	kinds := []string{session.Events[0].Kind, session.Events[1].Kind, session.Events[2].Kind}
	wantKinds := []string{"session_info", "custom.pinned-skills-state", "compaction"}
	for i, want := range wantKinds {
		if kinds[i] != want {
			t.Fatalf("event kind[%d] = %q, want %q; events=%+v", i, kinds[i], want, session.Events)
		}
	}
	customMetadata, ok := session.Events[1].FrameworkMetadata.(map[string]any)
	if !ok || customMetadata["custom_type"] != "pinned-skills-state" {
		t.Fatalf("expected custom event metadata, got %+v", session.Events[1].FrameworkMetadata)
	}
	compactionMetadata, ok := session.Events[2].FrameworkMetadata.(map[string]any)
	if !ok {
		t.Fatalf("expected compaction metadata, got %+v", session.Events[2].FrameworkMetadata)
	}
	if compactionMetadata["modified_file_count"] != 2 {
		t.Fatalf("expected modified_file_count=2, got %+v", compactionMetadata)
	}
	frameworkConfig, ok := session.OperationalContext.FrameworkConfig.(map[string]any)
	if !ok {
		t.Fatalf("expected framework config map, got %+v", session.OperationalContext.FrameworkConfig)
	}
	if frameworkConfig["session_info"] == nil || frameworkConfig["pi_custom"] == nil {
		t.Fatalf("expected session_info and pi_custom framework config, got %+v", frameworkConfig)
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
	if !successCall.Output.Succeeded() {
		t.Fatalf("expected tc-success to be successful")
	}
	if successCall.Output.Result == nil || *successCall.Output.Result != "ok" {
		t.Fatalf("expected tc-success result to be ok, got %+v", successCall.Output.Result)
	}

	failedCall, ok := toolCallsByID["tc-fail"]
	if !ok {
		t.Fatalf("expected tc-fail tool call in session")
	}
	if !failedCall.Output.Failed() {
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

func TestConvertRecordsDerivesToolDurationAndPreservesEmitTimestamp(t *testing.T) {
	records := []map[string]any{
		{
			"type":      "message",
			"timestamp": "2026-03-29T10:00:10Z",
			"message": map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{
						"type":      "toolCall",
						"id":        "tc-1",
						"name":      "bash",
						"arguments": map[string]any{"command": "sleep 1"},
					},
				},
			},
		},
		{
			"type":      "message",
			"timestamp": "2026-03-29T10:00:13.250Z",
			"message": map[string]any{
				"role":       "toolResult",
				"toolCallId": "tc-1",
				"content": []any{
					map[string]any{"type": "text", "text": "done"},
				},
			},
		},
	}

	session, err := ConvertRecords(records, "session-duration", "/tmp/pi-duration.jsonl")
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
	if toolCall.Output.DurationMS == nil || *toolCall.Output.DurationMS != 3250 {
		t.Fatalf("expected 3250ms duration, got %+v", toolCall.Output.DurationMS)
	}
}

func TestConvertRecordsPreservesParentSessionLineage(t *testing.T) {
	parentPath := "/home/me/.pi/agent/sessions/project/2026-07-01T21-32-52-892Z_parent-session-id.jsonl"
	records := []map[string]any{
		{
			"type":          "session",
			"id":            "child-session-id",
			"version":       3,
			"timestamp":     "2026-07-01T22:00:00Z",
			"cwd":           "/tmp/project",
			"parentSession": parentPath,
		},
		{
			"type":      "message",
			"id":        "m1",
			"timestamp": "2026-07-01T22:00:01Z",
			"message": map[string]any{
				"role":    "user",
				"content": []any{map[string]any{"type": "text", "text": "continue"}},
			},
		},
	}

	session, err := ConvertRecords(records, "fallback", "/tmp/child.jsonl")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}
	if session.Coordination.PredecessorSession == nil || *session.Coordination.PredecessorSession != "parent-session-id" {
		t.Fatalf("expected predecessor parent-session-id, got %+v", session.Coordination.PredecessorSession)
	}
	config, ok := session.OperationalContext.FrameworkConfig.(map[string]any)
	if !ok {
		t.Fatalf("expected framework config map, got %+v", session.OperationalContext.FrameworkConfig)
	}
	if config["parent_session_id"] != "parent-session-id" {
		t.Fatalf("expected parent_session_id metadata, got %+v", config)
	}
	if config["parent_session"] == "" {
		t.Fatalf("expected parent_session metadata, got %+v", config)
	}
}

func TestConvertRecordsCreatesImageAttachmentWithoutEmbeddingData(t *testing.T) {
	imageData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB"
	records := []map[string]any{
		{
			"type":      "session",
			"id":        "session-image",
			"version":   3,
			"timestamp": "2026-07-06T10:00:00Z",
			"cwd":       "/tmp/project",
		},
		{
			"type":      "message",
			"id":        "m1",
			"timestamp": "2026-07-06T10:00:01Z",
			"message": map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "text", "text": "Here is the screenshot."},
					map[string]any{"type": "image", "mimeType": "image/png", "data": imageData},
				},
			},
		},
	}

	session, err := ConvertRecords(records, "fallback", "/tmp/session-image.jsonl")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}
	if len(session.Attachments) != 1 {
		t.Fatalf("expected one image attachment, got %d", len(session.Attachments))
	}
	attachment := session.Attachments[0]
	if attachment.Kind != "image" || attachment.MediaType != "image/png" || attachment.Name != "pi-image-0001.png" {
		t.Fatalf("unexpected attachment identity: %+v", attachment)
	}
	if attachment.TurnIndex == nil || *attachment.TurnIndex != 0 {
		t.Fatalf("expected attachment linked to turn 0, got %+v", attachment.TurnIndex)
	}
	if attachment.Hash == "" || attachment.ContentRef != "inline:image" {
		t.Fatalf("expected hash/content ref for inline image, got %+v", attachment)
	}
	raw, ok := attachment.RawJSON.(map[string]any)
	if !ok {
		t.Fatalf("expected sanitized raw map, got %+v", attachment.RawJSON)
	}
	if _, ok := raw["data"]; ok {
		t.Fatalf("attachment raw json must not embed image data")
	}
}

func TestConvertRecordsLinksToolResultImageAttachmentToToolCall(t *testing.T) {
	records := []map[string]any{
		{"type": "session", "id": "session-tool-image", "version": 3, "timestamp": "2026-07-06T10:00:00Z", "cwd": "/tmp/project"},
		{
			"type":      "message",
			"id":        "m1",
			"timestamp": "2026-07-06T10:00:01Z",
			"message": map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "toolCall", "id": "tc-image", "name": "read", "arguments": map[string]any{"path": "/tmp/image.png"}},
				},
			},
		},
		{
			"type":      "message",
			"id":        "m2",
			"timestamp": "2026-07-06T10:00:02Z",
			"message": map[string]any{
				"role":       "toolResult",
				"toolCallId": "tc-image",
				"isError":    false,
				"content": []any{
					map[string]any{"type": "image", "mimeType": "image/png", "data": "abc123"},
				},
			},
		},
	}

	session, err := ConvertRecords(records, "fallback", "/tmp/session-tool-image.jsonl")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}
	if len(session.Attachments) != 1 {
		t.Fatalf("expected one image attachment, got %d", len(session.Attachments))
	}
	attachment := session.Attachments[0]
	if attachment.ToolCallID == nil || *attachment.ToolCallID != "tc-image" {
		t.Fatalf("expected attachment linked to tool call, got %+v", attachment.ToolCallID)
	}
	if attachment.TurnIndex != nil {
		t.Fatalf("tool-result attachment should not be linked to synthetic turn, got %+v", attachment.TurnIndex)
	}
	if len(session.ToolCalls) != 1 || session.ToolCalls[0].Output.Result == nil || *session.ToolCalls[0].Output.Result != "[image image/png]" {
		t.Fatalf("expected image placeholder tool result, got %+v", session.ToolCalls)
	}
}
