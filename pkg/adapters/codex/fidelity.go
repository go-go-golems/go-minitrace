package codex

import (
	"fmt"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

// Keep complete counts but bounded examples. Diagnostics must not copy private
// transcript bodies or allow unknown event names to grow archive metadata.
type codexFidelityReport struct {
	Counts   map[string]int         `json:"counts"`
	Examples []codexFidelityExample `json:"examples"`
	Total    int                    `json:"total"`
}

type codexFidelityExample struct {
	Code       string `json:"code"`
	SourceLine int    `json:"source_line,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

func (report *codexFidelityReport) add(code string, line int, id string) {
	if len(code) > 120 {
		code = code[:120]
	}
	if len(id) > 160 {
		id = id[:160]
	}
	report.Total++
	if _, exists := report.Counts[code]; !exists && len(report.Counts) >= 128 {
		code = "other_fidelity_diagnostic"
	}
	report.Counts[code]++
	if len(report.Examples) < 100 {
		report.Examples = append(report.Examples, codexFidelityExample{Code: code, SourceLine: line, ToolCallID: id})
	}
}

func codexFidelity(records []map[string]any, turns []minitrace.Turn, calls []minitrace.ToolCall) codexFidelityReport {
	report := codexFidelityReport{Counts: map[string]int{}}
	callIDs := map[string]bool{}
	for _, call := range calls {
		callIDs[call.ID] = true
		metadata := mapValue(call.FrameworkMetadata)
		line, _ := codexInteger(metadata["source_line"])
		if codes, ok := metadata["fidelity_diagnostics"].([]string); ok {
			for _, code := range codes {
				report.add(code, line, call.ID)
			}
		}
		if code, ok := metadata["identity_diagnostic"].(string); ok {
			report.add(code, line, call.ID)
		}
		if blocks, ok := metadata["output_blocks"].([]codexOutputEvidence); ok {
			outputLine, _ := codexInteger(metadata["output_source_line"])
			for _, block := range blocks {
				if block.Diagnostic != "" {
					report.add(block.Diagnostic, outputLine, call.ID)
				}
			}
		}
	}
	for _, turn := range turns {
		metadata := mapValue(turn.FrameworkMetadata)
		if codes, ok := metadata["fidelity_diagnostics"].([]string); ok {
			for _, code := range codes {
				report.add(code, 0, "")
			}
		}
	}
	for index, record := range records {
		if record["type"] != "event_msg" && record["type"] != "response_item" {
			continue
		}
		payload := mapValue(record["payload"])
		if record["type"] == "response_item" && (payload["type"] == "function_call_output" || payload["type"] == "custom_tool_call_output") {
			id := stringValue(payload["call_id"])
			if !callIDs[id] {
				report.add("output_without_invocation", index+1, id)
			}
		}
		if record["type"] != "event_msg" || (payload["type"] != "item_completed" && payload["type"] != "item_started") {
			continue
		}
		item := mapValue(payload["item"])
		switch stringValue(item["type"]) {
		case "UserMessage", "AgentMessage", "Reasoning", "ContextCompaction", "CommandExecution", "FileChange":
			// Supported or deliberately non-execution contextual records.
		default:
			report.add("unsupported_item_type:"+stringValue(item["type"]), index+1, "")
		}
	}
	return report
}

func applyCodexFidelityReport(session *minitrace.Session, report codexFidelityReport) {
	if report.Total == 0 {
		return
	}
	session.OperationalContext.FrameworkConfig = mergeMetadataMap(session.OperationalContext.FrameworkConfig, map[string]any{"fidelity": report})
	session.Flags.NeedsCleaning = true
	session.Flags.ForResearch = false
	session.Flags.Category = append(session.Flags.Category, "codex-fidelity-warning")
	event := minitrace.BuildEvent("codex-fidelity", session.Timing.EndedAt, "fidelity_warning", "Codex fidelity diagnostics", fmt.Sprintf("%d fidelity diagnostics; inspect framework metadata for counts and bounded source references", report.Total), nil)
	event.Severity = "warning"
	event.FrameworkMetadata = report
	session.Events = append(session.Events, event)
}
