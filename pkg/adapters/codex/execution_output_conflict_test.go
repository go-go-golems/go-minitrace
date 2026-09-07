package codex

import "testing"

func TestNativeExecutionOutputReplayConflicts(t *testing.T) {
	for _, field := range []string{"stdout", "stderr", "aggregated_output"} {
		t.Run(field, func(t *testing.T) {
			makeRecord := func(text string) map[string]any {
				r := executionRecord("item_completed", "same", "completed", ptr(0))
				mapValue(mapValue(r["payload"])["item"])[field] = text
				return r
			}
			duplicate := convertMessageTestRecords(t, []map[string]any{makeRecord("first"), makeRecord("first")})
			if len(duplicate.ToolCalls) != 1 || !duplicate.ToolCalls[0].Output.Succeeded() {
				t.Fatal("identical replay became conflict")
			}
			session := convertMessageTestRecords(t, []map[string]any{makeRecord("first"), makeRecord("different"), makeRecord("first")})
			if len(session.ToolCalls) != 1 {
				t.Fatal("replayed execution duplicated")
			}
			call := session.ToolCalls[0]
			if call.Output.Success != nil || call.Output.ExitCode != nil {
				t.Fatal("conflicting output remained authoritative")
			}
			metadata := mapValue(call.FrameworkMetadata)
			sources := metadata["execution_sources"].([]map[string]any)
			if len(sources) != 3 || sources[0][field+"_hash"] == sources[1][field+"_hash"] || sources[0][field+"_hash"] != sources[2][field+"_hash"] {
				t.Fatal("per-source output hashes lost")
			}
			if metadata["fidelity_diagnostics"] == nil {
				t.Fatal("conflict not diagnosed")
			}
		})
	}
}

func TestCompletionWithoutOutputDoesNotEraseEarlierAggregate(t *testing.T) {
	first := executionRecord("item_completed", "same", "completed", ptr(0))
	mapValue(mapValue(first["payload"])["item"])["aggregated_output"] = "keep"
	session := convertMessageTestRecords(t, []map[string]any{first, executionRecord("item_completed", "same", "completed", ptr(0))})
	call := session.ToolCalls[0]
	if call.Output.Result == nil || *call.Output.Result != "keep" || !call.Output.Succeeded() {
		t.Fatal("missing replay output erased original")
	}
}
