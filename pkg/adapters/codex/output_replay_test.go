package codex

import "testing"

func TestRepeatedEarlyOutputsRetainConflictAndProvenance(t *testing.T) {
	for _, early := range []bool{false, true} {
		call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "call_id": "call", "name": "exec_command", "arguments": `{"cmd":"pwd"}`}}
		var records []map[string]any
		if !early {
			records = append(records, call)
		}
		for _, text := range []string{`{"output":"ok","metadata":{"exit_code":0}}`, "plain text", `{"output":"failure","metadata":{"exit_code":7}}`, `{"output":"ok","metadata":{"exit_code":0}}`} {
			records = append(records, map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call_output", "call_id": "call", "output": text}})
		}
		if early {
			records = append(records, call)
		}
		for _, native := range []bool{false, true} {
			input := append([]map[string]any{}, records...)
			if native {
				end := executionRecord("item_completed", "native", "completed", ptr(0))
				mapValue(mapValue(end["payload"])["item"])["call_id"] = "call"
				input = append(input, end)
			}
			session := convertMessageTestRecords(t, input)
			if len(session.ToolCalls) != 1 {
				t.Fatal("replayed output created additional calls")
			}
			result := session.ToolCalls[0]
			if result.Output.Success != nil || result.Output.ExitCode != nil {
				t.Fatalf("conflict lost (early=%v native=%v)", early, native)
			}
			metadata := mapValue(result.FrameworkMetadata)
			sources, _ := metadata["output_sources"].([]map[string]any)
			if len(sources) != 4 || metadata["output_outcome_conflict"] != true {
				t.Fatalf("source evidence lost: %#v", metadata)
			}
		}
	}
}
