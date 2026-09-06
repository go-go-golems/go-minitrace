package codex

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

type codexNativeOutput struct {
	value any
	line  int
}

func (output codexNativeOutput) apply(call *minitrace.ToolCall) {
	applyCodexFunctionOutput(call, output.value)
	call.Output.FullReference = ptr("line:" + strconv.Itoa(output.line))
	metadata := mapValue(call.FrameworkMetadata)
	metadata["output_source_line"] = output.line
	sources, _ := metadata["output_sources"].([]map[string]any)
	sources = append(sources, map[string]any{"source_line": output.line, "blocks": metadata["output_blocks"]})
	metadata["output_sources"] = sources
	conflict := metadata["output_outcome_conflict"] == true
	if call.Output.ExitCode != nil {
		if previous, exists := metadata["first_output_exit_code"].(int); exists {
			conflict = conflict || previous != *call.Output.ExitCode
		} else {
			metadata["first_output_exit_code"] = *call.Output.ExitCode
		}
	}
	if conflict {
		call.Output.Success = nil
		call.Output.ExitCode = nil
		call.Output.Status = minitrace.ToolOutcomeUnknown
		call.Output.Error = nil
		if metadata["output_outcome_conflict"] != true {
			diagnostics, _ := metadata["fidelity_diagnostics"].([]string)
			metadata["fidelity_diagnostics"] = append(diagnostics, "conflicting_response_output_exit_codes")
		}
		metadata["output_outcome_conflict"] = true
	}
}

// Output evidence is retained per block, not zipped to JS calls: parallel
// wrappers can return results in any order, including multiple child outcomes.
type codexOutputEvidence struct {
	Block      int    `json:"block"`
	Kind       string `json:"kind"`
	ExitCode   *int   `json:"exit_code,omitempty"`
	DurationMS *int   `json:"duration_ms,omitempty"`
	Diagnostic string `json:"diagnostic,omitempty"`
}

func decodeCodexOutput(value any) (string, []codexOutputEvidence) {
	if text, ok := value.(string); ok {
		result, code, duration := parseFunctionOutput(text)
		return result, []codexOutputEvidence{{Kind: "text", ExitCode: code, DurationMS: duration}}
	}
	blocks, ok := value.([]any)
	if !ok {
		return "[unsupported output shape]", []codexOutputEvidence{{Kind: "unknown", Diagnostic: "unsupported_output_shape"}}
	}
	texts := make([]string, 0, len(blocks))
	evidence := make([]codexOutputEvidence, 0, len(blocks))
	for index, value := range blocks {
		block := mapValue(value)
		kind := stringValue(block["type"])
		item := codexOutputEvidence{Block: index, Kind: kind}
		switch strings.ToLower(kind) {
		case "text", "input_text", "output_text":
			if text, ok := block["text"].(string); ok {
				result, code, duration := parseFunctionOutput(text)
				texts = append(texts, result)
				item.ExitCode, item.DurationMS = code, duration
			} else {
				texts = append(texts, "[invalid text output block]")
				item.Diagnostic = "invalid_output_text"
			}
		case "image", "input_image", "output_image":
			texts = append(texts, "[image output]")
		default:
			texts = append(texts, "[unsupported output block]")
			item.Diagnostic = "unsupported_output_block"
		}
		evidence = append(evidence, item)
	}
	return strings.Join(texts, "\n"), evidence
}

func applyCodexFunctionOutput(toolCall *minitrace.ToolCall, rawOutput any) {
	result, evidence := decodeCodexOutput(rawOutput)
	truncated, fullBytes, fullHash := minitrace.TruncateContent(result, minitrace.TruncateLimit)
	toolCall.Output.Result = truncated
	toolCall.Output.Truncated = fullBytes != nil
	toolCall.Output.FullBytes = fullBytes
	toolCall.Output.FullHash = fullHash
	toolCall.Output.Success = nil
	toolCall.Output.Status = minitrace.ToolOutcomeUnknown
	toolCall.Output.ExitCode = nil
	toolCall.Output.DurationMS = nil
	toolCall.Output.Error = nil
	metadata := mapValue(toolCall.FrameworkMetadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["output_blocks"] = evidence
	outcomes, durations := 0, 0
	var code, duration *int
	for _, block := range evidence {
		if block.ExitCode != nil {
			outcomes++
			code = block.ExitCode
		}
		if block.DurationMS != nil {
			durations++
			duration = block.DurationMS
		}
		if strings.Contains(strings.ToLower(block.Kind), "image") {
			metadata["has_image_signal"] = true
		}
	}
	// exec is JavaScript orchestration, not a subprocess. Its printed child
	// result envelopes never establish the wrapper's own binary success.
	if toolCall.ToolName != "exec" && durations == 1 {
		toolCall.Output.DurationMS = duration
	}
	if toolCall.ToolName != "exec" && outcomes == 1 {
		toolCall.Output.ExitCode = code
		toolCall.Output.SetSuccess(*code == 0)
		if *code != 0 {
			errorText := result
			if len(errorText) > 1024 {
				errorText = errorText[:1024]
			}
			toolCall.Output.Error = &errorText
		}
	} else if outcomes > 1 {
		metadata["outcome_association"] = "multiple_output_envelopes"
	}
	toolCall.FrameworkMetadata = metadata
	promoteCodexOutputMetadata(toolCall, result)
}

// Recognize only known tool result envelopes. A random JSON-looking stdout
// string must survive unchanged; it is not automatically transport metadata.
func parseFunctionOutput(raw string) (string, *int, *int) {
	var object map[string]any
	if json.Unmarshal([]byte(raw), &object) == nil && object != nil {
		if text, code, duration, ok := codexOutputEnvelope(object); ok {
			return text, code, duration
		}
		// Promise.allSettled is an observed wrapper output shape. Unwrap only a
		// fulfilled value that is itself a recognized tool envelope.
		if object["status"] == "fulfilled" {
			if value := mapValue(object["value"]); value != nil {
				if text, code, duration, ok := codexOutputEnvelope(value); ok {
					return text, code, duration
				}
			}
		}
		return raw, nil, nil
	}
	var code, duration *int
	lines := strings.Split(raw, "\n")
	outputStart := -1
	for index, line := range lines {
		if strings.HasPrefix(line, "Output:") || strings.HasPrefix(line, "Final output:") {
			outputStart = index
			break
		}
	}
	// Without the renderer's explicit output boundary, these phrases could be
	// arbitrary stdout. Do not turn them into a successful process outcome.
	if outputStart < 0 {
		return raw, nil, nil
	}
	for _, line := range lines[:outputStart] {
		for _, prefix := range []string{"Process exited with code ", "Exit code: "} {
			if value, ok := strings.CutPrefix(line, prefix); ok {
				if parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
					code = ptr(int(parsed))
				}
			}
		}
		if value, ok := strings.CutPrefix(line, "Wall time: "); ok {
			fields := strings.Fields(value)
			if len(fields) > 0 {
				if seconds, err := strconv.ParseFloat(fields[0], 64); err == nil {
					duration = codexDurationMS(seconds)
				}
			}
		}
	}
	if code == nil && duration == nil {
		return raw, nil, nil
	}
	first := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(lines[outputStart], "Final output:"), "Output:"))
	output := lines[outputStart+1:]
	if first != "" {
		output = append([]string{first}, output...)
	}
	return strings.Join(output, "\n"), code, duration
}

func codexOutputEnvelope(object map[string]any) (string, *int, *int, bool) {
	text, ok := object["output"].(string)
	if !ok {
		return "", nil, nil, false
	}
	metadata := mapValue(object["metadata"])
	var codeValue, durationValue any
	if metadata != nil {
		_, hasExit := metadata["exit_code"]
		_, hasDuration := metadata["duration_seconds"]
		if !hasExit && !hasDuration {
			return "", nil, nil, false
		}
		codeValue, durationValue = metadata["exit_code"], metadata["duration_seconds"]
	} else if chunk, ok := object["chunk_id"].(string); ok && chunk != "" {
		codeValue, durationValue = object["exit_code"], object["wall_time_seconds"]
	} else {
		return "", nil, nil, false
	}
	var code, duration *int
	if value, ok := codexInteger(codeValue); ok && value >= math.MinInt32 && value <= math.MaxInt32 {
		code = &value
	} else if codeValue != nil {
		return "", nil, nil, false
	}
	if value, ok := durationValue.(float64); ok {
		duration = codexDurationMS(value)
	} else if durationValue != nil {
		return "", nil, nil, false
	}
	return text, code, duration, true
}

func codexDurationMS(seconds float64) *int {
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 || seconds*1000 >= float64(math.MaxInt) {
		return nil
	}
	return ptr(int(seconds * 1000))
}

func codexInteger(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Trunc(v) != v || v >= float64(math.MaxInt) || v < float64(math.MinInt) {
			return 0, false
		}
		return int(v), true
	default:
		return 0, false
	}
}
