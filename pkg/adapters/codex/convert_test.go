package codex

import (
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestConvertRecordsSessionJSONL(t *testing.T) {
	records := []map[string]any{
		{
			"type": "session_meta",
			"payload": map[string]any{
				"id":             "session-a",
				"cwd":            "/home/manuel/project",
				"cli_version":    "0.114.0",
				"model_provider": "openai",
				"base_instructions": map[string]any{
					"text": "System prompt",
				},
			},
		},
		{
			"timestamp": "2026-03-29T11:00:00Z",
			"type":      "event_msg",
			"payload": map[string]any{
				"type":    "user_message",
				"message": "fix main.go",
			},
		},
		{
			"timestamp": "2026-03-29T11:00:00Z",
			"type":      "turn_context",
			"payload": map[string]any{
				"cwd":             "/home/manuel/project",
				"approval_policy": "never",
				"sandbox_policy": map[string]any{
					"mode": "danger-full-access",
				},
				"model":  "gpt-5-codex",
				"effort": "high",
			},
		},
		{
			"timestamp": "2026-03-29T11:00:01Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type":      "function_call",
				"call_id":   "call-1",
				"name":      "exec_command",
				"arguments": "{\"cmd\":\"cat main.go\",\"justification\":\"inspect file\"}",
			},
		},
		{
			"timestamp": "2026-03-29T11:00:02Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type":    "function_call_output",
				"call_id": "call-1",
				"output":  "{\"output\":\"package main\",\"metadata\":{\"exit_code\":0,\"duration_seconds\":0.2}}",
			},
		},
		{
			"timestamp": "2026-03-29T11:00:03Z",
			"type":      "event_msg",
			"payload": map[string]any{
				"type": "agent_reasoning",
				"text": "**Inspecting file**",
			},
		},
		{
			"timestamp": "2026-03-29T11:00:04Z",
			"type":      "event_msg",
			"payload": map[string]any{
				"type":    "agent_message",
				"message": "I inspected the file.",
			},
		},
		{
			"timestamp": "2026-03-29T11:00:04Z",
			"type":      "event_msg",
			"payload": map[string]any{
				"type": "token_count",
				"info": map[string]any{
					"last_token_usage": map[string]any{
						"input_tokens":            30,
						"output_tokens":           12,
						"cached_input_tokens":     4,
						"reasoning_output_tokens": 3,
					},
					"model_context_window": 272000,
				},
			},
		},
	}

	session, err := ConvertRecords(records, "fallback-id", "/tmp/session.jsonl", "session-jsonl-v1")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}

	if session.ID != "session-a" {
		t.Fatalf("expected session_meta ID to win, got %s", session.ID)
	}
	if session.Environment.Model == nil || *session.Environment.Model != "gpt-5-codex" {
		t.Fatalf("expected model gpt-5-codex, got %+v", session.Environment.Model)
	}
	if session.Environment.ProviderHint == nil || *session.Environment.ProviderHint != "openai-compatible" {
		t.Fatalf("expected openai-compatible provider, got %+v", session.Environment.ProviderHint)
	}
	expectedWorkingDirectory := minitrace.NormalizePath("/home/manuel/project")
	if session.OperationalContext.WorkingDirectory == nil || *session.OperationalContext.WorkingDirectory != expectedWorkingDirectory {
		t.Fatalf("expected normalized cwd, got %+v", session.OperationalContext.WorkingDirectory)
	}
	if session.OperationalContext.AutonomyLevel == nil || *session.OperationalContext.AutonomyLevel != "full-auto" {
		t.Fatalf("expected full-auto autonomy, got %+v", session.OperationalContext.AutonomyLevel)
	}
	if session.OperationalContext.Sandbox == nil || *session.OperationalContext.Sandbox != false {
		t.Fatalf("expected sandbox=false, got %+v", session.OperationalContext.Sandbox)
	}
	if len(session.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(session.ToolCalls))
	}
	if session.ToolCalls[0].OperationType != "READ" {
		t.Fatalf("expected READ operation, got %s", session.ToolCalls[0].OperationType)
	}
	if session.ToolCalls[0].Input.Justification == nil || *session.ToolCalls[0].Input.Justification != "inspect file" {
		t.Fatalf("expected justification to be promoted, got %+v", session.ToolCalls[0].Input.Justification)
	}
	if session.ToolCalls[0].Output.Result == nil || *session.ToolCalls[0].Output.Result != "package main" {
		t.Fatalf("unexpected tool output: %+v", session.ToolCalls[0].Output.Result)
	}
	if session.ToolCalls[0].Output.ExitCode == nil || *session.ToolCalls[0].Output.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %+v", session.ToolCalls[0].Output.ExitCode)
	}
	if session.Metrics.TotalInputTokens == nil || *session.Metrics.TotalInputTokens != 30 {
		t.Fatalf("expected input tokens 30, got %+v", session.Metrics.TotalInputTokens)
	}
}

func TestConvertRecordsExecJSONL(t *testing.T) {
	records := []map[string]any{
		{
			"type":      "thread.started",
			"thread_id": "thread-1",
		},
		{
			"type": "item.completed",
			"item": map[string]any{
				"id":                "cmd-1",
				"type":              "command_execution",
				"command":           "go test ./...",
				"aggregated_output": "ok",
				"exit_code":         0,
				"status":            "completed",
				"turn_id":           "turn-1",
				"source":            "exec-runner",
				"parsed_cmd": []any{
					map[string]any{"type": "test", "cmd": "go test ./..."},
				},
				"stdout": "ok",
				"stderr": "",
			},
		},
		{
			"type": "item.completed",
			"item": map[string]any{
				"type": "reasoning",
				"text": "Running tests",
			},
		},
		{
			"type": "item.completed",
			"item": map[string]any{
				"type":    "agent_message",
				"text":    "Tests passed.",
				"turn_id": "turn-1",
				"phase":   "commentary",
			},
		},
	}

	session, err := ConvertRecords(records, "fallback-id", "/tmp/exec.jsonl", "exec-jsonl-v1")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}

	if session.ID != "thread-1" {
		t.Fatalf("expected thread.started ID, got %s", session.ID)
	}
	if len(session.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(session.ToolCalls))
	}
	if session.ToolCalls[0].ToolName != "exec_command" {
		t.Fatalf("expected exec_command, got %s", session.ToolCalls[0].ToolName)
	}
	if session.ToolCalls[0].Output.ExitCode == nil || *session.ToolCalls[0].Output.ExitCode != 0 {
		t.Fatalf("expected exec exit code 0, got %+v", session.ToolCalls[0].Output.ExitCode)
	}
	metadata, ok := session.ToolCalls[0].FrameworkMetadata.(map[string]any)
	if !ok {
		t.Fatalf("expected tool metadata map, got %+v", session.ToolCalls[0].FrameworkMetadata)
	}
	if metadata["source"] != "exec-runner" {
		t.Fatalf("expected exec source metadata, got %+v", metadata)
	}
	if metadata["turn_id"] != "turn-1" {
		t.Fatalf("expected tool turn_id metadata, got %+v", metadata)
	}
	if metadata["stdout"] != "ok" {
		t.Fatalf("expected stdout metadata, got %+v", metadata)
	}
	if len(session.Turns) != 1 || session.Turns[0].Thinking == nil || *session.Turns[0].Thinking != "Running tests" {
		t.Fatalf("expected reasoning to attach to assistant turn, got %+v", session.Turns)
	}
	turnMetadata, ok := session.Turns[0].FrameworkMetadata.(map[string]any)
	if !ok {
		t.Fatalf("expected turn metadata map, got %+v", session.Turns[0].FrameworkMetadata)
	}
	if turnMetadata["turn_id"] != "turn-1" || turnMetadata["phase"] != "commentary" {
		t.Fatalf("expected exec turn metadata, got %+v", turnMetadata)
	}
}

func TestConvertRecordsSessionJSONLPromotesLatestToolSemantics(t *testing.T) {
	records := []map[string]any{
		{
			"timestamp": "2026-06-10T11:00:00Z",
			"type":      "session_meta",
			"payload": map[string]any{
				"id":             "codex-latest",
				"cwd":            "/tmp/project",
				"model_provider": "openai",
			},
		},
		{
			"timestamp": "2026-06-10T11:00:01Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type":      "function_call",
				"call_id":   "call-spawn",
				"name":      "spawn_agent",
				"namespace": "multi_agent_v1",
				"arguments": `{"agent_type":"explorer","message":"Review cmd/demo/main.go","reasoning_effort":"medium"}`,
			},
		},
		{
			"timestamp": "2026-06-10T11:00:02Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type":    "function_call_output",
				"call_id": "call-spawn",
				"output":  "019eb210-1fd0-7d93-8c61-7f99c0dfe463",
			},
		},
		{
			"timestamp": "2026-06-10T11:00:03Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type":      "function_call",
				"call_id":   "call-wait",
				"name":      "wait_agent",
				"namespace": "multi_agent_v1",
				"arguments": `{"targets":["019eb210-1fd0-7d93-8c61-7f99c0dfe463"],"timeout_ms":30000}`,
			},
		},
		{
			"timestamp": "2026-06-10T11:00:04Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type":    "function_call_output",
				"call_id": "call-wait",
				"output":  `{"status":{"019eb210-1fd0-7d93-8c61-7f99c0dfe463":{"completed":"No findings."}},"timed_out":false}`,
			},
		},
		{
			"timestamp": "2026-06-10T11:00:05Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type":      "function_call",
				"call_id":   "call-image",
				"name":      "view_image",
				"arguments": `{"path":"/tmp/screenshot.png","detail":"high"}`,
			},
		},
		{
			"timestamp": "2026-06-10T11:00:06Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type":    "function_call_output",
				"call_id": "call-image",
				"output":  "image loaded",
			},
		},
		{
			"timestamp": "2026-06-10T11:00:07Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type":    "custom_tool_call",
				"status":  "completed",
				"call_id": "call-patch",
				"name":    "apply_patch",
				"input":   "*** Begin Patch\n*** Update File: cmd/demo/main.go\n*** End Patch\n",
			},
		},
		{
			"timestamp": "2026-06-10T11:00:08Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type":    "custom_tool_call_output",
				"call_id": "call-patch",
				"output":  "Exit code: 0\nWall time: 0.1 seconds\nOutput:\nSuccess. Updated the following files:\nM cmd/demo/main.go\n",
			},
		},
		{
			"timestamp": "2026-06-10T11:00:09Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type":      "function_call",
				"call_id":   "call-stdin",
				"name":      "write_stdin",
				"arguments": `{"session_id":83612,"yield_time_ms":1000}`,
			},
		},
	}

	session, err := ConvertRecords(records, "fallback-id", "/tmp/codex-latest.jsonl", "session-jsonl-v1")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}
	if len(session.ToolCalls) != 5 {
		t.Fatalf("expected 5 tool calls, got %d", len(session.ToolCalls))
	}
	byID := map[string]minitrace.ToolCall{}
	for _, toolCall := range session.ToolCalls {
		byID[toolCall.ID] = toolCall
	}

	spawn := byID["call-spawn"]
	if spawn.OperationType != "DELEGATE" || spawn.SpawnedAgent == nil {
		t.Fatalf("expected spawn_agent delegate with spawned agent, got %+v", spawn)
	}
	if spawn.SpawnedAgent.AgentType != "explorer" || spawn.SpawnedAgent.TaskScope != "Review cmd/demo/main.go" {
		t.Fatalf("unexpected spawned agent: %+v", spawn.SpawnedAgent)
	}
	if spawn.SpawnedAgent.SubSessionID == nil || *spawn.SpawnedAgent.SubSessionID != "019eb210-1fd0-7d93-8c61-7f99c0dfe463" {
		t.Fatalf("expected spawned sub-session from output, got %+v", spawn.SpawnedAgent)
	}

	wait := byID["call-wait"]
	if wait.OperationType != "DELEGATE" || wait.SpawnedAgent == nil || wait.SpawnedAgent.OutcomeSummary != "No findings." {
		t.Fatalf("expected wait_agent delegate outcome, got %+v", wait)
	}
	waitMetadata := wait.FrameworkMetadata.(map[string]any)
	if waitMetadata["targets"] == nil || waitMetadata["timed_out"] != false {
		t.Fatalf("expected wait metadata targets/timed_out, got %+v", waitMetadata)
	}

	image := byID["call-image"]
	if image.OperationType != "READ" || image.Input.FilePath == nil || *image.Input.FilePath != minitrace.NormalizePath("/tmp/screenshot.png") {
		t.Fatalf("expected image read with normalized path, got %+v", image)
	}
	if image.Output.ContentOrigin == nil || *image.Output.ContentOrigin != "image" {
		t.Fatalf("expected image content origin, got %+v", image.Output.ContentOrigin)
	}
	imageMetadata := image.FrameworkMetadata.(map[string]any)
	if imageMetadata["has_image_signal"] != true {
		t.Fatalf("expected image metadata signal, got %+v", imageMetadata)
	}
	if len(session.Attachments) != 1 {
		t.Fatalf("expected one image attachment, got %d", len(session.Attachments))
	}
	attachment := session.Attachments[0]
	if attachment.Kind != "image" || attachment.MediaType != "image/png" || attachment.Path != minitrace.NormalizePath("/tmp/screenshot.png") || attachment.ToolCallID == nil || *attachment.ToolCallID != "call-image" {
		t.Fatalf("unexpected image attachment: %+v", attachment)
	}
	if len(session.Events) != 3 {
		t.Fatalf("expected spawn/wait/image events, got %d: %+v", len(session.Events), session.Events)
	}
	if session.Events[0].Kind != "subagent_spawn" || session.Events[1].Kind != "subagent_wait" || session.Events[2].Kind != "image_view" {
		t.Fatalf("unexpected Codex events: %+v", session.Events)
	}
	if session.Events[2].AttachmentID == nil || *session.Events[2].AttachmentID != attachment.ID {
		t.Fatalf("expected image event to link attachment, got %+v", session.Events[2].AttachmentID)
	}

	patch := byID["call-patch"]
	if patch.OperationType != "MODIFY" || patch.Output.ExitCode == nil || *patch.Output.ExitCode != 0 {
		t.Fatalf("expected successful apply_patch modify call, got %+v", patch)
	}
	patchMetadata := patch.FrameworkMetadata.(map[string]any)
	if patchMetadata["status"] != "completed" || patchMetadata["custom_tool"] != true {
		t.Fatalf("expected custom tool metadata, got %+v", patchMetadata)
	}

	stdin := byID["call-stdin"]
	if stdin.OperationType != "EXECUTE" {
		t.Fatalf("expected write_stdin execute operation, got %+v", stdin.OperationType)
	}
}

func TestConvertRecordsSessionJSONLPreservesFrameworkMetadata(t *testing.T) {
	records := []map[string]any{
		{
			"timestamp": "2026-03-29T11:00:00Z",
			"type":      "session_meta",
			"payload": map[string]any{
				"id":             "session-meta",
				"cwd":            "/home/manuel/project",
				"cli_version":    "0.114.0",
				"model_provider": "openai",
				"source":         "cli",
			},
		},
		{
			"timestamp": "2026-03-29T11:00:00Z",
			"type":      "turn_context",
			"payload": map[string]any{
				"turn_id":         "turn-ctx-1",
				"cwd":             "/home/manuel/project",
				"approval_policy": "on-request",
				"sandbox_policy": map[string]any{
					"type":           "workspace-write",
					"network_access": false,
				},
				"collaboration_mode": map[string]any{
					"mode": "default",
					"settings": map[string]any{
						"reasoning_effort": "high",
					},
				},
				"truncation_policy": map[string]any{
					"kind": "auto",
				},
				"model":       "gpt-5-codex",
				"personality": "pragmatic",
				"timezone":    "America/New_York",
				"effort":      "high",
			},
		},
		{
			"timestamp": "2026-03-29T11:00:01Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type":      "function_call",
				"call_id":   "call-1",
				"name":      "exec_command",
				"arguments": "{\"cmd\":\"rg --files\"}",
			},
		},
		{
			"timestamp": "2026-03-29T11:00:02Z",
			"type":      "event_msg",
			"payload": map[string]any{
				"type":    "exec_command_end",
				"call_id": "call-1",
				"turn_id": "turn-ctx-1",
				"source":  "unified_exec_startup",
				"parsed_cmd": []any{
					map[string]any{"type": "list_files", "cmd": "rg --files"},
				},
				"stdout":            "main.go",
				"stderr":            "",
				"exit_code":         0,
				"status":            "completed",
				"aggregated_output": "main.go",
			},
		},
		{
			"timestamp": "2026-03-29T11:00:03Z",
			"type":      "event_msg",
			"payload": map[string]any{
				"type":        "token_count",
				"rate_limits": map[string]any{"plan_type": "pro"},
				"info":        map[string]any{"model_context_window": 272000},
			},
		},
		{
			"timestamp": "2026-03-29T11:00:04Z",
			"type":      "event_msg",
			"payload": map[string]any{
				"type":            "agent_message",
				"message":         "Done.",
				"phase":           "commentary",
				"memory_citation": "memory://abc",
			},
		},
	}

	session, err := ConvertRecords(records, "fallback-id", "/tmp/session-meta.jsonl", "session-jsonl-v1")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}

	config, ok := session.OperationalContext.FrameworkConfig.(map[string]any)
	if !ok {
		t.Fatalf("expected framework config map, got %+v", session.OperationalContext.FrameworkConfig)
	}
	if config["approval_policy"] != "on-request" {
		t.Fatalf("expected approval_policy in framework config, got %+v", config)
	}
	if config["session_source"] != "cli" {
		t.Fatalf("expected session_source in framework config, got %+v", config)
	}
	if config["rate_limits"] == nil || config["truncation_policy"] == nil || config["sandbox_policy"] == nil {
		t.Fatalf("expected detailed Codex config metadata, got %+v", config)
	}
	turnMetadata, ok := session.Turns[0].FrameworkMetadata.(map[string]any)
	if !ok {
		t.Fatalf("expected assistant turn metadata, got %+v", session.Turns[0].FrameworkMetadata)
	}
	if turnMetadata["turn_id"] != "turn-ctx-1" || turnMetadata["phase"] != "commentary" || turnMetadata["memory_citation"] != "memory://abc" {
		t.Fatalf("unexpected turn metadata: %+v", turnMetadata)
	}
	toolMetadata, ok := session.ToolCalls[0].FrameworkMetadata.(map[string]any)
	if !ok {
		t.Fatalf("expected tool metadata map, got %+v", session.ToolCalls[0].FrameworkMetadata)
	}
	if toolMetadata["source"] != "unified_exec_startup" || toolMetadata["turn_id"] != "turn-ctx-1" {
		t.Fatalf("unexpected tool metadata: %+v", toolMetadata)
	}
	if toolMetadata["parsed_cmd"] == nil || toolMetadata["stdout"] != "main.go" {
		t.Fatalf("expected parsed_cmd/stdout metadata, got %+v", toolMetadata)
	}
	if len(session.Events) != 1 || session.Events[0].Kind != "rate_limits" {
		t.Fatalf("expected rate_limits source event, got %+v", session.Events)
	}
	if session.Events[0].FrameworkMetadata == nil {
		t.Fatalf("expected rate_limits event metadata")
	}
}

func TestConvertRecordsExecJSONLAssignsPerTurnEmittingIndexes(t *testing.T) {
	records := []map[string]any{
		{"type": "thread.started", "thread_id": "thread-turns"},
		{
			"type": "item.completed",
			"item": map[string]any{
				"id":                "cmd-1",
				"type":              "command_execution",
				"command":           "ls",
				"aggregated_output": "file.txt",
				"exit_code":         0,
				"status":            "completed",
			},
		},
		{
			"type": "item.completed",
			"item": map[string]any{"type": "agent_message", "text": "First turn."},
		},
		{
			"type": "item.completed",
			"item": map[string]any{
				"id":                "cmd-2",
				"type":              "command_execution",
				"command":           "go build ./...",
				"aggregated_output": "",
				"exit_code":         0,
				"status":            "completed",
			},
		},
		{
			"type": "item.completed",
			"item": map[string]any{"type": "agent_message", "text": "Second turn."},
		},
	}

	session, err := ConvertRecords(records, "fallback-id", "/tmp/exec-turns.jsonl", "exec-jsonl-v1")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}
	if len(session.ToolCalls) != 2 || len(session.Turns) != 2 {
		t.Fatalf("expected 2 tool calls and 2 turns, got %d/%d", len(session.ToolCalls), len(session.Turns))
	}
	byID := map[string]minitrace.ToolCall{}
	for _, toolCall := range session.ToolCalls {
		byID[toolCall.ID] = toolCall
	}
	first := byID["cmd-1"]
	second := byID["cmd-2"]
	if first.EmittingTurnIndex == nil || *first.EmittingTurnIndex != 0 {
		t.Fatalf("expected cmd-1 emitted at turn 0, got %+v", first.EmittingTurnIndex)
	}
	if second.EmittingTurnIndex == nil || *second.EmittingTurnIndex != 1 {
		t.Fatalf("expected cmd-2 emitted at turn 1, got %+v", second.EmittingTurnIndex)
	}
	if len(session.Turns[0].ToolCallsInTurn) != 1 || session.Turns[0].ToolCallsInTurn[0] != "cmd-1" {
		t.Fatalf("expected turn 0 to own cmd-1 only, got %+v", session.Turns[0].ToolCallsInTurn)
	}
	if len(session.Turns[1].ToolCallsInTurn) != 1 || session.Turns[1].ToolCallsInTurn[0] != "cmd-2" {
		t.Fatalf("expected turn 1 to own cmd-2 only, got %+v", session.Turns[1].ToolCallsInTurn)
	}
}

func TestConvertRecordsSessionJSONLCapturesSubagentMetadataAndCount(t *testing.T) {
	records := []map[string]any{
		{
			"type":      "session_meta",
			"timestamp": "2026-06-10T15:04:24.524Z",
			"payload": map[string]any{
				"id":               "child-thread",
				"parent_thread_id": "parent-thread",
				"agent_nickname":   "Dewey",
				"agent_role":       "explorer",
				"cwd":              "/tmp/project",
			},
		},
		{
			"type":      "response_item",
			"timestamp": "2026-06-10T15:04:30Z",
			"payload": map[string]any{
				"type":      "function_call",
				"call_id":   "spawn-1",
				"name":      "spawn_agent",
				"arguments": `{"message": "explore the repo"}`,
			},
		},
		{
			"type":      "response_item",
			"timestamp": "2026-06-10T15:04:35Z",
			"payload": map[string]any{
				"type":    "function_call_output",
				"call_id": "spawn-1",
				"output":  "0199aaaa-bbbb-cccc-dddd-eeeeffff0000",
			},
		},
		{
			"type":      "event_msg",
			"timestamp": "2026-06-10T15:04:40Z",
			"payload": map[string]any{
				"type":    "agent_message",
				"message": "Spawned a subagent.",
			},
		},
	}

	session, err := ConvertRecords(records, "fallback-id", "/tmp/subagent.jsonl", "session-jsonl-v1")
	if err != nil {
		t.Fatalf("ConvertRecords returned error: %v", err)
	}
	config, ok := session.OperationalContext.FrameworkConfig.(map[string]any)
	if !ok {
		t.Fatalf("expected framework config map, got %+v", session.OperationalContext.FrameworkConfig)
	}
	if config["parent_thread_id"] != "parent-thread" {
		t.Fatalf("expected parent_thread_id in framework config, got %+v", config)
	}
	if config["agent_nickname"] != "Dewey" || config["agent_role"] != "explorer" {
		t.Fatalf("expected agent nickname/role in framework config, got %+v", config)
	}
	if session.Metrics.SubagentCount != 1 {
		t.Fatalf("expected subagent count 1 from spawn_agent, got %d", session.Metrics.SubagentCount)
	}
}
