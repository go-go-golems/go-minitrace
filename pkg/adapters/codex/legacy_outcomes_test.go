package codex

import (
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"testing"
)

func TestLegacyEndOutcomeArrivalOrder(t *testing.T) {
	call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "call_id": "call", "name": "exec_command", "arguments": `{"cmd":"exit 7"}`}}
	output := map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call_output", "call_id": "call", "output": "plain output without transport metadata"}}
	end := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "exec_command_end", "call_id": "call", "exit_code": 7}}
	for _, records := range [][]map[string]any{{call, output, end}, {end, output, call}, {call, end, output}} {
		session := convertMessageTestRecords(t, records)
		if len(session.ToolCalls) != 1 || !session.ToolCalls[0].Output.Failed() || *session.ToolCalls[0].Output.ExitCode != 7 {
			t.Fatalf("lost authoritative outcome: %#v", session.ToolCalls)
		}
	}
	conflicting := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "exec_command_end", "call_id": "call", "exit_code": 0}}
	session := convertMessageTestRecords(t, []map[string]any{call, end, conflicting, end})
	if session.ToolCalls[0].Output.Success != nil || session.ToolCalls[0].Output.ExitCode != nil {
		t.Fatal("terminal conflict was overwritten by replay")
	}
}

func TestExecStreamCompletionWithoutExitCodeIsUnknown(t *testing.T) {
	for _, code := range []any{nil, "invalid", 0, 7} {
		item := map[string]any{"type": "command_execution", "id": "cmd", "command": "some command", "status": "completed", "exit_code": code}
		session, err := ConvertRecords([]map[string]any{{"type": "item.completed", "item": item}}, "id", "/tmp/test", "exec-jsonl-v1")
		if err != nil {
			t.Fatal(err)
		}
		if len(session.ToolCalls) != 1 {
			t.Fatalf("missing execution: %#v", session)
		}
		want := minitrace.ToolOutcomeUnknown
		if code == 0 {
			want = minitrace.ToolOutcomeSucceeded
		}
		if code == 7 {
			want = minitrace.ToolOutcomeFailed
		}
		if session.ToolCalls[0].Output.OutcomeStatus() != want {
			t.Fatalf("code=%v got=%s", code, session.ToolCalls[0].Output.OutcomeStatus())
		}
	}
}
