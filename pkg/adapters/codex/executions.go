package codex

import (
	"fmt"
	"net/url"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

type codexExecution struct {
	id           string
	turnID       string
	threadID     string
	firstLine    int
	outputLine   int
	timestamp    *string
	argv         []string
	cwd          string
	status       string
	exitCode     *int
	durationMS   *int
	startedAtMS  *int
	completed    bool
	conflict     bool
	text         string
	stdout       string
	stderr       string
	parentCallID string
	callID       string
	sources      []map[string]any
	diagnostics  []string
}

// Only outer native lifecycle records are execution evidence. This pass never
// inspects JavaScript arguments or quoted JSON in message/output text.
func collectCodexExecutions(records []map[string]any) []*codexExecution {
	var executions []*codexExecution
	byID := map[string]*codexExecution{}
	currentTurn := ""
	for index, record := range records {
		payload := mapValue(record["payload"])
		kind := stringValue(record["type"])
		if kind == "turn_context" || (kind == "event_msg" && payload["type"] == "task_started") {
			currentTurn = codexNativeTurnID(payload, currentTurn)
		}
		if kind != "event_msg" || (payload["type"] != "item_started" && payload["type"] != "item_completed") {
			continue
		}
		item := mapValue(payload["item"])
		if item["type"] != "CommandExecution" {
			continue
		}
		id := stringValue(item["id"])
		missingID := id == ""
		if missingID {
			id = fmt.Sprintf("anonymous-line-%d", index+1)
		}
		execution := byID[id]
		if execution == nil {
			execution = &codexExecution{id: id, turnID: codexNativeTurnID(payload, currentTurn), threadID: stringValue(payload["thread_id"]), firstLine: index + 1,
				timestamp: optionalString(stringValue(record["timestamp"]))}
			byID[id] = execution
			executions = append(executions, execution)
			if missingID {
				execution.diagnose("missing_execution_id")
			}
		}
		execution.merge(record, item, index+1)
	}
	return executions
}

func (execution *codexExecution) diagnose(code string) {
	for _, existing := range execution.diagnostics {
		if existing == code {
			return
		}
	}
	execution.diagnostics = append(execution.diagnostics, code)
}

func (execution *codexExecution) merge(record, item map[string]any, line int) {
	payload := mapValue(record["payload"])
	source := map[string]any{"source_line": line, "event_type": payload["type"]}
	for _, key := range []string{"timestamp", "ordinal"} {
		if value, ok := record[key]; ok {
			source[key] = value
		}
	}
	for _, key := range []string{"started_at_ms", "completed_at_ms", "thread_id", "turn_id"} {
		if value, ok := payload[key]; ok {
			source[key] = value
		}
	}
	if threadID := stringValue(payload["thread_id"]); threadID != "" && execution.threadID != "" && threadID != execution.threadID {
		execution.conflict = true
		execution.diagnose("conflicting_execution_thread_id")
	}
	for _, key := range []string{"status", "exit_code", "process_id"} {
		if value, ok := item[key]; ok {
			source[key] = value
		}
	}
	execution.sources = append(execution.sources, source)
	if start, ok := codexInteger(payload["started_at_ms"]); ok && execution.startedAtMS == nil {
		execution.startedAtMS = &start
	}
	if turn := stringValue(payload["turn_id"]); turn != "" && execution.turnID != "" && turn != execution.turnID {
		execution.conflict = true
		execution.diagnose("conflicting_execution_turn_id")
	}
	if value, present := item["command"]; present {
		argv := []string{}
		valid := true
		for _, value := range listValue(value) {
			word, ok := value.(string)
			if !ok {
				valid = false
				break
			}
			argv = append(argv, word)
		}
		if !valid || len(argv) == 0 {
			execution.diagnose("unsupported_execution_command")
		} else if len(execution.argv) > 0 && !reflect.DeepEqual(execution.argv, argv) {
			execution.conflict = true
			execution.diagnose("conflicting_execution_command")
			source["argv"] = argv
		} else {
			execution.argv = argv
		}
	}
	if cwd := stringValue(item["cwd"]); cwd != "" {
		if execution.cwd != "" && execution.cwd != cwd {
			execution.conflict = true
			execution.diagnose("conflicting_execution_cwd")
			source["cwd"] = cwd
		} else {
			execution.cwd = cwd
		}
	}
	for _, field := range []struct {
		key         string
		destination *string
	}{
		{"parent_call_id", &execution.parentCallID}, {"call_id", &execution.callID},
	} {
		if value := firstNonEmpty(stringValue(item[field.key]), stringValue(payload[field.key])); value != "" {
			if *field.destination != "" && *field.destination != value {
				execution.conflict = true
				execution.diagnose("conflicting_" + field.key)
			} else {
				*field.destination = value
			}
		}
	}
	if payload["type"] != "item_completed" {
		return
	}
	status := stringValue(item["status"])
	if execution.completed && ((execution.status == "cancelled") != (status == "cancelled")) {
		execution.conflict = true
		execution.diagnose("conflicting_execution_status")
	}
	execution.completed = true
	execution.status = status
	execution.outputLine = line
	if raw, present := item["exit_code"]; present && raw != nil {
		code, ok := codexInteger(raw)
		if !ok || code < -2147483648 || code > 2147483647 {
			execution.conflict = true
			execution.diagnose("invalid_execution_exit_code")
		} else if execution.exitCode != nil && *execution.exitCode != code {
			execution.conflict = true
			execution.diagnose("conflicting_execution_exit_code")
		} else {
			execution.exitCode = &code
		}
	}
	if execution.exitCode != nil && ((status == "failed" && *execution.exitCode == 0) || status == "cancelled") {
		execution.conflict = true
		execution.diagnose("conflicting_execution_outcome")
	}
	if stdout, ok := item["stdout"].(string); ok {
		execution.stdout = stdout
	}
	if stderr, ok := item["stderr"].(string); ok {
		execution.stderr = stderr
	}
	if text, ok := item["aggregated_output"].(string); ok {
		execution.text = text
	} else {
		execution.text = execution.stdout + execution.stderr
	}
	end, endOK := codexInteger(payload["completed_at_ms"])
	if execution.startedAtMS != nil && endOK && end >= *execution.startedAtMS {
		execution.durationMS = ptr(end - *execution.startedAtMS)
	}
}

func (execution *codexExecution) toolCall() minitrace.ToolCall {
	metadata := map[string]any{
		"record_kind": "execution", "native_execution_id": execution.id, "turn_id": execution.turnID, "thread_id": execution.threadID,
		"source_line": execution.firstLine, "execution_sources": execution.sources,
		"argv": execution.argv, "native_cwd": execution.cwd, "turn_association": "unknown",
		"parent_association": "unknown",
	}
	cwd := execution.cwd
	if strings.HasPrefix(cwd, "file:") {
		parsed, err := url.Parse(cwd)
		if err == nil && (parsed.Host == "" || parsed.Host == "localhost") && parsed.Path != "" {
			cwd = parsed.Path
		} else {
			execution.diagnose("unsupported_execution_cwd_uri")
			cwd = ""
		}
	}
	metadata["cwd"] = minitrace.NormalizePath(cwd)
	if execution.parentCallID != "" {
		metadata["parent_call_id"] = execution.parentCallID
		metadata["parent_association"] = "explicit_parent_call_id"
	}
	if execution.callID != "" {
		metadata["response_call_id"] = execution.callID
	}
	command, rendering := codexExecutionCommand(execution.argv)
	metadata["command_rendering"] = rendering
	call := minitrace.BuildToolCall("codex-execution:"+execution.id, nil, execution.timestamp,
		"exec_command", "EXECUTE", nil, optionalString(command), map[string]any{"argv": execution.argv, "cwd": cwd},
		false, nil, nil, execution.durationMS, metadata, nil, ptr("local_exec"), nil)
	call.Output.Success = nil
	call.Output.Status = minitrace.ToolOutcomePending
	if execution.completed {
		call.Output.Status = minitrace.ToolOutcomeUnknown
		call.Output.Result, call.Output.FullBytes, call.Output.FullHash = minitrace.TruncateContent(execution.text, minitrace.TruncateLimit)
		call.Output.Truncated = call.Output.FullBytes != nil
		call.Output.FullReference = ptr(fmt.Sprintf("line:%d", execution.outputLine))
		stdout, _, _ := minitrace.TruncateContent(execution.stdout, minitrace.TruncateLimit)
		stderr, _, _ := minitrace.TruncateContent(execution.stderr, minitrace.TruncateLimit)
		metadata["stdout"], metadata["stderr"] = stdout, stderr
		metadata["status"] = execution.status
		if !execution.conflict {
			switch {
			case execution.status == "cancelled":
				call.Output.Status = minitrace.ToolOutcomeCancelled
			case execution.exitCode != nil:
				call.Output.ExitCode = execution.exitCode
				call.Output.SetSuccess(*execution.exitCode == 0)
			case execution.status == "failed":
				call.Output.SetSuccess(false)
			default:
				execution.diagnose("missing_execution_outcome")
			}
		}
		if call.Output.Failed() {
			errorText := firstNonEmpty(execution.stderr, execution.text, "command execution failed")
			if len(errorText) > 1024 {
				errorText = errorText[:1024]
			}
			call.Output.Error = &errorText
		}
	}
	if len(execution.diagnostics) > 0 {
		metadata["fidelity_diagnostics"] = execution.diagnostics
	}
	return call
}

func appendCodexExecutions(records []map[string]any, calls []minitrace.ToolCall) []minitrace.ToolCall {
	byID := map[string]int{}
	for index, call := range calls {
		byID[call.ID] = index
	}
	for _, execution := range collectCodexExecutions(records) {
		call := execution.toolCall()
		if index, exists := byID[execution.callID]; execution.callID != "" && exists && calls[index].ToolName == "exec_command" {
			// This native execution explicitly identifies an already-recorded
			// direct invocation. Enrich that record instead of double-counting it.
			existing := &calls[index]
			metadata := mapValue(existing.FrameworkMetadata)
			sourceLine := metadata["source_line"]
			if existing.Output.ExitCode != nil && call.Output.ExitCode != nil && *existing.Output.ExitCode != *call.Output.ExitCode {
				mapValue(call.FrameworkMetadata)["response_exit_code"] = *existing.Output.ExitCode
				execution.diagnose("conflicting_response_execution_exit_code")
				mapValue(call.FrameworkMetadata)["fidelity_diagnostics"] = execution.diagnostics
				call.Output.Success = nil
				call.Output.Status = minitrace.ToolOutcomeUnknown
				call.Output.ExitCode = nil
				call.Output.Error = nil
			}
			if call.Input.Command != nil {
				existing.Input.Command = call.Input.Command
			}
			existing.Output = call.Output
			existing.OperationType = call.OperationType
			existing.FrameworkMetadata = mergeMetadataMap(existing.FrameworkMetadata, mapValue(call.FrameworkMetadata))
			mapValue(existing.FrameworkMetadata)["source_line"] = sourceLine
			continue
		}
		baseID := call.ID
		for suffix := 1; ; suffix++ {
			if _, collision := byID[call.ID]; !collision {
				break
			}
			call.ID = fmt.Sprintf("%s:line-%d:%d", baseID, execution.firstLine, suffix)
			mapValue(call.FrameworkMetadata)["identity_diagnostic"] = "execution_id_namespace_collision"
		}
		byID[call.ID] = len(calls)
		calls = append(calls, call)
	}
	sort.SliceStable(calls, func(i, j int) bool {
		left, _ := codexInteger(mapValue(calls[i].FrameworkMetadata)["source_line"])
		right, _ := codexInteger(mapValue(calls[j].FrameworkMetadata)["source_line"])
		return left < right
	})
	return calls
}

func codexExecutionCommand(argv []string) (string, string) {
	if len(argv) >= 3 && (argv[1] == "-c" || argv[1] == "-lc") {
		switch filepath.Base(argv[0]) {
		case "sh", "bash", "dash", "zsh", "ksh":
			return argv[2], "shell_script"
		}
	}
	words := make([]string, len(argv))
	for i, word := range argv {
		words[i] = "'" + strings.ReplaceAll(word, "'", "'\"'\"'") + "'"
	}
	return strings.Join(words, " "), "quoted_argv_display"
}
