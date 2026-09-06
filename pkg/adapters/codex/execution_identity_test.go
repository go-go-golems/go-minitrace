package codex

import "testing"

func TestMultipleExecutionsCannotOverwriteOneInvocation(t *testing.T) {
	invocation := map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "call_id": "direct", "name": "exec_command", "arguments": `{"cmd":"pwd"}`}}
	first := executionRecord("item_completed", "first", "completed", ptr(0))
	second := executionRecord("item_completed", "second", "failed", ptr(7))
	for _, record := range []map[string]any{first, second} {
		mapValue(mapValue(record["payload"])["item"])["call_id"] = "direct"
	}
	session := convertMessageTestRecords(t, []map[string]any{invocation, first, second})
	if len(session.ToolCalls) != 3 {
		t.Fatalf("distinct executions lost: got %d records", len(session.ToolCalls))
	}
	seen := map[string]bool{}
	for _, call := range session.ToolCalls {
		metadata := mapValue(call.FrameworkMetadata)
		if id := stringValue(metadata["native_execution_id"]); id != "" {
			seen[id] = true
			if id == "second" && !call.Output.Failed() {
				t.Fatal("second execution failure lost")
			}
		}
		if call.ID == "direct" && call.Output.Success != nil {
			t.Fatal("child outcome promoted to ambiguous invocation")
		}
	}
	if !seen["first"] || !seen["second"] {
		t.Fatalf("missing native identities: %v", seen)
	}
}

func TestExecutionCannotMasqueradeAsOriginalInvocation(t *testing.T) {
	first := executionRecord("item_completed", "first", "completed", ptr(0))
	second := executionRecord("item_completed", "second", "failed", ptr(7))
	mapValue(mapValue(second["payload"])["item"])["call_id"] = "codex-execution:first"
	session := convertMessageTestRecords(t, []map[string]any{first, second})
	if len(session.ToolCalls) != 2 {
		t.Fatal("native execution consumed another execution as an invocation")
	}
	if !session.ToolCalls[0].Output.Succeeded() || !session.ToolCalls[1].Output.Failed() {
		t.Fatal("native identities/outcomes overwritten")
	}
}

func TestDirectInvocationAndExecutionExitConflict(t *testing.T) {
	invocation := map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "call_id": "direct", "name": "exec_command", "arguments": `{"cmd":"pwd"}`}}
	output := map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call_output", "call_id": "direct", "output": `{"output":"ok","metadata":{"exit_code":0}}`}}
	execution := executionRecord("item_completed", "native", "failed", ptr(7))
	mapValue(mapValue(execution["payload"])["item"])["call_id"] = "direct"
	session := convertMessageTestRecords(t, []map[string]any{invocation, output, execution})
	if len(session.ToolCalls) != 1 {
		t.Fatal("one-to-one native association duplicated")
	}
	call := session.ToolCalls[0]
	if call.Output.Success != nil || call.Output.ExitCode != nil {
		t.Fatal("contradictory terminal evidence resolved confidently")
	}
	metadata := mapValue(call.FrameworkMetadata)
	if metadata["response_exit_code"] != 0 || metadata["fidelity_diagnostics"] == nil {
		t.Fatal("conflicting evidence not retained/diagnosed")
	}
}
