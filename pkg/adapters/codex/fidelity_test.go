package codex

import (
	"fmt"
	"testing"
)

func TestCodexUnsupportedEvidenceIsDiagnosedAndBounded(t *testing.T) {
	var records []map[string]any
	for i := 0; i < 250; i++ {
		records = append(records, map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "item": map[string]any{"type": fmt.Sprintf("FutureType%d", i)}}})
	}
	records = append(records, map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call_output", "call_id": "absent", "output": "some output"}})
	session := convertMessageTestRecords(t, records)
	config := mapValue(session.OperationalContext.FrameworkConfig)
	report, ok := config["fidelity"].(codexFidelityReport)
	if !ok || report.Total != 251 || len(report.Examples) != 100 || len(report.Counts) > 129 {
		t.Fatalf("diagnostics not bounded/complete: %#v", config)
	}
	if !session.Flags.NeedsCleaning || session.Flags.ForResearch {
		t.Fatal("unsupported evidence advertised as clean/research ready")
	}
	if len(session.Events) != 1 || session.Events[0].Kind != "fidelity_warning" || session.Events[0].Severity != "warning" {
		t.Fatalf("diagnostic not visible as source event: %#v", session.Events)
	}
}

func TestCodexEarlyOutputDoesNotProduceOrphanDiagnostic(t *testing.T) {
	session := convertMessageTestRecords(t, []map[string]any{
		{"type": "response_item", "payload": map[string]any{"type": "function_call_output", "call_id": "later", "output": "result"}},
		{"type": "response_item", "payload": map[string]any{"type": "function_call", "call_id": "later", "name": "exec_command", "arguments": `{"cmd":"pwd"}`}},
	})
	if config := mapValue(session.OperationalContext.FrameworkConfig); config["fidelity"] != nil {
		t.Fatalf("early output falsely diagnosed: %#v", config)
	}
}
