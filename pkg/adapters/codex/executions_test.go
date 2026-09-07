package codex

import (
	"reflect"
	"strings"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestCodexNativeExecutionsFromFixture(t *testing.T) {
	records, err := parseJSONLFile("testdata/paginated-fidelity.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	session := convertMessageTestRecords(t, records)
	executions := map[string]minitrace.ToolCall{}
	for _, call := range session.ToolCalls {
		metadata := mapValue(call.FrameworkMetadata)
		if metadata["record_kind"] == "execution" {
			executions[stringValue(metadata["native_execution_id"])] = call
		}
		if call.Input.Command != nil && strings.Contains(*call.Input.Command, "never-created") {
			t.Fatal("false JS branch became execution")
		}
	}
	if len(executions) != 6 {
		t.Fatalf("want six distinct executions, got %d", len(executions))
	}
	failed := executions["exec-failed"]
	if !failed.Output.Failed() || failed.Output.ExitCode == nil || *failed.Output.ExitCode != 7 {
		t.Fatalf("lost child failure: %#v", failed.Output)
	}
	if failed.Input.Command == nil || *failed.Input.Command != "printf first > first.txt; printf second >> second.txt; exit 7" {
		t.Fatalf("lost executed command: %#v", failed.Input)
	}
	metadata := mapValue(failed.FrameworkMetadata)
	if metadata["cwd"] != "/workspace/actual" || metadata["parent_association"] != "unknown" {
		t.Fatalf("cwd or parent provenance changed: %#v", metadata)
	}
	if failed.Output.DurationMS == nil || *failed.Output.DurationMS != 1000 {
		t.Fatalf("duration lost: %#v", failed.Output)
	}
	if len(metadata["execution_sources"].([]map[string]any)) != 3 {
		t.Fatal("start and both completion references must survive reconciliation")
	}
	for id, status := range map[string]minitrace.ToolOutcomeStatus{
		"exec-ok": minitrace.ToolOutcomeSucceeded, "exec-ok-again": minitrace.ToolOutcomeSucceeded,
		"exec-unknown": minitrace.ToolOutcomeUnknown, "exec-pending": minitrace.ToolOutcomePending,
		"exec-cancelled": minitrace.ToolOutcomeCancelled,
	} {
		if executions[id].Output.OutcomeStatus() != status {
			t.Errorf("%s outcome=%s, want %s", id, executions[id].Output.OutcomeStatus(), status)
		}
	}
	if executions["exec-ok"].ID == executions["exec-ok-again"].ID {
		t.Fatal("distinct real executions collapsed")
	}
	if !reflect.DeepEqual(session.ToolCalls, convertMessageTestRecords(t, records).ToolCalls) {
		t.Fatal("execution identities/provenance not stable")
	}
}

func TestCodexExecutionLifecycleConflictAndDirectLink(t *testing.T) {
	start := executionRecord("item_started", "exec", "in_progress", nil)
	mapValue(start["payload"])["started_at_ms"] = 1000
	end := executionRecord("item_completed", "exec", "completed", ptr(0))
	mapValue(end["payload"])["completed_at_ms"] = 1500
	records := []map[string]any{start, end, start, end}
	session := convertMessageTestRecords(t, records)
	if len(session.ToolCalls) != 1 || !session.ToolCalls[0].Output.Succeeded() || *session.ToolCalls[0].Output.DurationMS != 500 {
		t.Fatalf("lifecycle reconciliation failed: %#v", session.ToolCalls)
	}
	conflict := executionRecord("item_completed", "exec", "failed", ptr(2))
	session = convertMessageTestRecords(t, append(records, conflict))
	if len(session.ToolCalls) != 1 || session.ToolCalls[0].Output.Success != nil || session.ToolCalls[0].Output.ExitCode != nil {
		t.Fatalf("conflicting outcomes must stay unknown: %#v", session.ToolCalls)
	}
	if mapValue(session.ToolCalls[0].FrameworkMetadata)["fidelity_diagnostics"] == nil {
		t.Fatal("conflict not diagnosed")
	}

	// Explicit native call identity enriches the existing direct invocation.
	mapValue(mapValue(end["payload"])["item"])["call_id"] = "direct"
	session = convertMessageTestRecords(t, []map[string]any{
		{"type": "response_item", "payload": map[string]any{"type": "function_call", "call_id": "direct", "name": "exec_command", "arguments": `{"cmd":"printf ok"}`}}, end,
	})
	if len(session.ToolCalls) != 1 || session.ToolCalls[0].ID != "direct" || !session.ToolCalls[0].Output.Succeeded() {
		t.Fatalf("explicitly linked execution duplicated: %#v", session.ToolCalls)
	}
}

func TestCodexExecutionStdoutIsNotAnEnvelope(t *testing.T) {
	record := executionRecord("item_completed", "exec", "completed", ptr(3))
	text := `{"output":"invented","metadata":{"exit_code":0}}`
	mapValue(mapValue(record["payload"])["item"])["stdout"] = text
	session := convertMessageTestRecords(t, []map[string]any{record})
	output := session.ToolCalls[0].Output
	if output.Result == nil || *output.Result != text || !output.Failed() || *output.ExitCode != 3 {
		t.Fatalf("stdout reinterpreted as transport: %#v", output)
	}
}

func TestCodexExecutionArgvDisplay(t *testing.T) {
	command, kind := codexExecutionCommand([]string{"program", "a b", "it's"})
	if kind != "quoted_argv_display" || command != `'program' 'a b' 'it'"'"'s'` {
		t.Fatalf("argv quoting lost: %q %q", command, kind)
	}
	command, kind = codexExecutionCommand([]string{"/bin/sh", "-lc", "printf ok"})
	if command != "printf ok" || kind != "shell_script" {
		t.Fatalf("shell script not surfaced: %q %q", command, kind)
	}
}

func executionRecord(kind, id, status string, code *int) map[string]any {
	item := map[string]any{"type": "CommandExecution", "id": id, "status": status, "command": []any{"/bin/sh", "-lc", "printf ok"}, "cwd": "file:///workspace/test"}
	if code != nil {
		item["exit_code"] = *code
	}
	return map[string]any{"type": "event_msg", "payload": map[string]any{"type": kind, "turn_id": "turn", "item": item}}
}
