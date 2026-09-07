package codex

import "github.com/go-go-golems/go-minitrace/pkg/minitrace"

// Terminal native evidence is reconciled after response results, so arrival
// order cannot leave a pending status paired with an authoritative exit code.
func reconcileCodexLegacyEnds(records []map[string]any, calls []minitrace.ToolCall) {
	byID := map[string]int{}
	for i := range calls {
		byID[calls[i].ID] = i
	}
	conflicts := map[string]bool{}
	for line, record := range records {
		payload := mapValue(record["payload"])
		if record["type"] != "event_msg" || payload["type"] != "exec_command_end" {
			continue
		}
		id := stringValue(payload["call_id"])
		index, exists := byID[id]
		if !exists {
			continue
		}
		call := &calls[index]
		metadata := mapValue(mergeMetadataMap(call.FrameworkMetadata, codexToolExecutionMetadata(payload)))
		sources, _ := metadata["execution_end_sources"].([]map[string]any)
		metadata["execution_end_sources"] = append(sources, map[string]any{"source_line": line + 1, "exit_code": payload["exit_code"], "timestamp": record["timestamp"]})
		call.FrameworkMetadata = metadata
		conflicts[id] = conflicts[id] || metadata["output_outcome_conflict"] == true
		code, valid := codexInteger(payload["exit_code"])
		if !valid {
			continue
		}
		if call.Output.ExitCode != nil && *call.Output.ExitCode != code {
			conflicts[id] = true
			metadata["response_or_previous_exit_code"] = *call.Output.ExitCode
			codes, _ := metadata["fidelity_diagnostics"].([]string)
			metadata["fidelity_diagnostics"] = append(codes, "conflicting_legacy_execution_exit_code")
		}
		if conflicts[id] {
			call.Output.Success = nil
			call.Output.ExitCode = nil
			call.Output.Status = minitrace.ToolOutcomeUnknown
			call.Output.Error = nil
		} else {
			call.Output.ExitCode = &code
			call.Output.SetSuccess(code == 0)
		}
	}
}
