package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/adapters"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
)

const (
	AdapterVersion      = "go-minitrace-codex-adapter-dev"
	SourceFormatExec    = "codex-exec-jsonl-v1"
	SourceFormatSession = "codex-session-jsonl-v1"
)

var (
	readPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^cat\s`),
		regexp.MustCompile(`^head\s`),
		regexp.MustCompile(`^tail\s`),
		regexp.MustCompile(`^less\s`),
		regexp.MustCompile(`^more\s`),
		regexp.MustCompile(`^find\s`),
		regexp.MustCompile(`^ls\s`),
		regexp.MustCompile(`^tree\s`),
		regexp.MustCompile(`^wc\s`),
		regexp.MustCompile(`^grep\s`),
		regexp.MustCompile(`^rg\s`),
		regexp.MustCompile(`^ag\s`),
		regexp.MustCompile(`^ack\s`),
		regexp.MustCompile(`^file\s`),
		regexp.MustCompile(`^stat\s`),
		regexp.MustCompile(`^du\s`),
		regexp.MustCompile(`^df\s`),
		regexp.MustCompile(`^git\s+(log|show|diff|status|branch|blame)\b`),
		regexp.MustCompile(`^python3?\s+-c\s+.*open.*read`),
	}
	modifyPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^sed\s+-i`),
		regexp.MustCompile(`^perl\s+-i`),
		regexp.MustCompile(`^patch\s`),
		regexp.MustCompile(`^git\s+apply\b`),
		regexp.MustCompile(`^chmod\s`),
		regexp.MustCompile(`^chown\s`),
	}
)

func ConvertLocator(locator adapters.SessionLocator) (*minitrace.Session, error) {
	records, err := parseJSONLFile(locator.SourcePath)
	if err != nil {
		return nil, err
	}
	return ConvertRecords(records, locator.ID, locator.SourcePath, locator.FormatHint)
}

func ConvertRecords(records []map[string]any, sessionID, sourcePath, formatHint string) (*minitrace.Session, error) {
	actualFormat := formatHint
	if actualFormat == "" || actualFormat == "unknown-jsonl" {
		actualFormat = detectFormatRecords(records)
	}

	var (
		turns       []minitrace.Turn
		toolCalls   []minitrace.ToolCall
		annotations []minitrace.Annotation
		timestamps  []time.Time
		tokenTotals *minitrace.TokenTotals
		metadata    codexMetadata
	)

	switch actualFormat {
	case "session-jsonl-v1":
		turns, toolCalls, annotations, timestamps, tokenTotals, metadata = parseSessionJSONL(records)
	case "exec-jsonl-v1":
		turns, toolCalls, annotations, timestamps, tokenTotals, metadata = parseExecJSONL(records)
	default:
		return nil, errors.Errorf("unsupported Codex format hint: %s", actualFormat)
	}

	if metadata.SessionID != "" {
		sessionID = metadata.SessionID
	}

	session := minitrace.BuildSessionSkeleton(sessionID, "codex", sourceFormatName(actualFormat), AdapterVersion)
	session.Environment.PlatformType = ptr("agent")
	session.Environment.SystemPrompt = optionalString(metadata.SystemPrompt)
	session.Environment.Model = optionalString(metadata.Model)
	session.Environment.AgentVersion = optionalString(metadata.CLIVersion)
	session.Environment.ProviderHint = providerHint(metadata.ModelProvider)
	session.Environment.ToolsEnabled = uniqueToolNames(toolCalls)
	session.OperationalContext.WorkingDirectory = optionalNormalizedPath(metadata.CWD)
	session.OperationalContext.AutonomyLevel = optionalString(mapApprovalPolicy(metadata.ApprovalPolicy))
	session.OperationalContext.Sandbox = sandboxValue(metadata.SandboxPolicy)
	session.OperationalContext.FrameworkConfig = frameworkConfig(metadata)
	if sourcePath != "" {
		session.Provenance.SourcePath = ptr(sourcePath)
	}

	deduped, duplicateCount := minitrace.DeduplicateToolCalls(toolCalls, nil)
	toolCalls = deduped
	if duplicateCount > 0 {
		annotations = append(annotations, minitrace.BuildAnnotation(
			"ann-dedup-"+truncateID(sessionID),
			"adapter",
			"session",
			sessionID,
			"observation",
			fmt.Sprintf("Deduplicated %d duplicate tool calls", duplicateCount),
			fmt.Sprintf("Removed %d tool calls with duplicate IDs.", duplicateCount),
			[]string{"deduplication", "data-quality"},
			nil,
		))
	}
	minitrace.ComputeToolCallContext(toolCalls)

	timing := minitrace.ComputeTiming(timestamps)
	quality := minitrace.AssignQualityTier(turns, toolCalls)
	containsPII := minitrace.DetectPIIInPaths(toolCalls)

	session.Quality = &quality
	session.Title = minitrace.ExtractTitle(turns, 80)
	session.Timing = timing
	session.Turns = turns
	session.ToolCalls = toolCalls
	session.Annotations = annotations
	session.Metrics = minitrace.ComputeMetrics(turns, toolCalls, timing, 0, tokenTotals)
	session.Flags.ContainsPII = containsPII
	session.Flags.ForResearch = quality == "A" && !containsPII
	session.Flags.NeedsCleaning = quality != "A" || containsPII
	if containsPII {
		session.Classification = "confidential"
	}

	return &session, nil
}

type codexMetadata struct {
	SessionID         string
	Model             string
	ModelProvider     string
	CWD               string
	CLIVersion        string
	Originator        string
	SystemPrompt      string
	ApprovalPolicy    string
	SandboxPolicy     string
	Personality       string
	CollaborationMode string
	ReasoningEffort   string
	Timezone          string
	ContextWindow     int
}

func parseSessionJSONL(records []map[string]any) ([]minitrace.Turn, []minitrace.ToolCall, []minitrace.Annotation, []time.Time, *minitrace.TokenTotals, codexMetadata) {
	turns := []minitrace.Turn{}
	toolCalls := []minitrace.ToolCall{}
	annotations := []minitrace.Annotation{}
	timestamps := []time.Time{}
	tokenTotals := &minitrace.TokenTotals{}
	metadata := codexMetadata{}

	pendingFunctionCalls := map[string]int{}
	pendingTurnToolIDs := map[string]struct{}{}
	currentThinking := []string{}
	turnIndex := 0
	toolCounter := 0

	for _, record := range records {
		recordType := stringValue(record["type"])
		timestamp := stringValue(record["timestamp"])
		if parsed, ok := minitrace.ParseTimestamp(timestamp); ok {
			timestamps = append(timestamps, parsed)
		}
		timestampPtr := optionalString(timestamp)
		payload := mapValue(record["payload"])
		if payload == nil {
			continue
		}

		switch recordType {
		case "session_meta":
			metadata.SessionID = firstNonEmpty(stringValue(payload["id"]), metadata.SessionID)
			metadata.CWD = firstNonEmpty(stringValue(payload["cwd"]), metadata.CWD)
			metadata.CLIVersion = firstNonEmpty(stringValue(payload["cli_version"]), metadata.CLIVersion)
			metadata.Originator = firstNonEmpty(stringValue(payload["originator"]), metadata.Originator)
			metadata.ModelProvider = firstNonEmpty(stringValue(payload["model_provider"]), metadata.ModelProvider)
			baseInstructions := mapValue(payload["base_instructions"])
			if baseInstructions != nil {
				metadata.SystemPrompt = firstNonEmpty(stringValue(baseInstructions["text"]), metadata.SystemPrompt)
			}
		case "turn_context":
			metadata.CWD = firstNonEmpty(stringValue(payload["cwd"]), metadata.CWD)
			metadata.Model = firstNonEmpty(stringValue(payload["model"]), metadata.Model)
			metadata.ApprovalPolicy = firstNonEmpty(stringValue(payload["approval_policy"]), metadata.ApprovalPolicy)
			metadata.SandboxPolicy = firstNonEmpty(extractSandboxPolicy(payload["sandbox_policy"]), metadata.SandboxPolicy)
			metadata.Personality = firstNonEmpty(stringValue(payload["personality"]), metadata.Personality)
			metadata.Timezone = firstNonEmpty(stringValue(payload["timezone"]), metadata.Timezone)
			metadata.ReasoningEffort = firstNonEmpty(stringValue(payload["effort"]), metadata.ReasoningEffort)
			if collaboration := mapValue(payload["collaboration_mode"]); collaboration != nil {
				metadata.CollaborationMode = firstNonEmpty(stringValue(collaboration["mode"]), metadata.CollaborationMode)
				settings := mapValue(collaboration["settings"])
				if settings != nil {
					metadata.ReasoningEffort = firstNonEmpty(stringValue(settings["reasoning_effort"]), metadata.ReasoningEffort)
				}
			}
		case "event_msg":
			eventType := stringValue(payload["type"])
			switch eventType {
			case "task_started":
				metadata.ContextWindow = firstNonZero(minitrace.SafeInt(payload["model_context_window"], 0), metadata.ContextWindow)
			case "user_message":
				source := ptr("human")
				turn := minitrace.BuildTurn(turnIndex, timestampPtr, "user", source, stringValue(payload["message"]))
				turn.InputChannel = ptr("user_input")
				turns = append(turns, turn)
				turnIndex++
			case "agent_reasoning":
				if text := stringValue(payload["text"]); text != "" {
					currentThinking = append(currentThinking, text)
				}
			case "agent_message":
				source := ptr("model")
				turn := minitrace.BuildTurn(turnIndex, timestampPtr, "assistant", source, stringValue(payload["message"]))
				if len(currentThinking) > 0 {
					thinking := strings.Join(currentThinking, "\n")
					turn.Thinking = &thinking
				}
				turn.Model = optionalString(metadata.Model)
				toolIDs := pendingTurnToolIDsSlice(pendingTurnToolIDs)
				for _, toolID := range toolIDs {
					if index, ok := pendingFunctionCalls[toolID]; ok {
						turnIndexCopy := turnIndex
						toolCalls[index].EmittingTurnIndex = &turnIndexCopy
					}
				}
				turn.ToolCallsInTurn = toolIDs
				turns = append(turns, turn)
				turnIndex++
				currentThinking = nil
				pendingTurnToolIDs = map[string]struct{}{}
			case "token_count":
				info := mapValue(payload["info"])
				lastUsage := mapValue(info["last_token_usage"])
				if lastUsage == nil {
					continue
				}
				inputTokens := minitrace.SafeInt(lastUsage["input_tokens"], 0)
				outputTokens := minitrace.SafeInt(lastUsage["output_tokens"], 0)
				cacheReadTokens := minitrace.SafeInt(lastUsage["cached_input_tokens"], 0)
				reasoningTokens := minitrace.SafeInt(lastUsage["reasoning_output_tokens"], 0)
				tokenTotals.Input += inputTokens
				tokenTotals.Output += outputTokens
				tokenTotals.CacheRead += cacheReadTokens
				tokenTotals.Reasoning += reasoningTokens
				metadata.ContextWindow = firstNonZero(minitrace.SafeInt(info["model_context_window"], 0), metadata.ContextWindow)
				if len(turns) > 0 && turns[len(turns)-1].Role == "assistant" {
					usage := &minitrace.Usage{}
					if inputTokens != 0 {
						usage.InputTokens = &inputTokens
					}
					if outputTokens != 0 {
						usage.OutputTokens = &outputTokens
					}
					if cacheReadTokens != 0 {
						usage.CacheReadTokens = &cacheReadTokens
					}
					if reasoningTokens != 0 {
						usage.ReasoningTokens = &reasoningTokens
					}
					turns[len(turns)-1].Usage = usage
				}
			}
		case "response_item":
			payloadType := stringValue(payload["type"])
			switch payloadType {
			case "reasoning":
				for _, item := range listValue(payload["summary"]) {
					summary := mapValue(item)
					if summary == nil {
						continue
					}
					if text := stringValue(summary["text"]); text != "" {
						currentThinking = append(currentThinking, text)
					}
				}
			case "function_call":
				callID := stringValue(payload["call_id"])
				if callID == "" {
					callID = fmt.Sprintf("tc-codex-%04d", toolCounter)
				}
				toolCounter++
				funcName := firstNonEmpty(stringValue(payload["name"]), "unknown")
				args := parseArguments(payload["arguments"])
				command := ""
				if funcName == "exec_command" {
					command = stringValue(args["cmd"])
				}
				toolCall := minitrace.BuildToolCall(
					callID,
					nil,
					timestampPtr,
					funcName,
					classifyFunction(funcName, args),
					optionalString(extractFilePathFromCommand(command)),
					optionalString(command),
					args,
					true,
					nil,
					nil,
					nil,
					codexFrameworkMetadata(funcName, args),
					nil,
					classifyContentOrigin(funcName),
					nil,
				)
				if justification := stringValue(args["justification"]); justification != "" {
					toolCall.Input.Justification = &justification
				}
				toolCalls = append(toolCalls, toolCall)
				pendingFunctionCalls[callID] = len(toolCalls) - 1
				pendingTurnToolIDs[callID] = struct{}{}
			case "function_call_output":
				callID := stringValue(payload["call_id"])
				index, ok := pendingFunctionCalls[callID]
				if !ok {
					continue
				}
				result, exitCode, durationMS := parseFunctionOutput(stringValue(payload["output"]))
				truncated, fullBytes, fullHash := minitrace.TruncateContent(result, minitrace.TruncateLimit)
				toolCalls[index].Output.Result = truncated
				toolCalls[index].Output.Truncated = fullBytes != nil
				toolCalls[index].Output.FullBytes = fullBytes
				toolCalls[index].Output.FullHash = fullHash
				toolCalls[index].Output.DurationMS = durationMS
				toolCalls[index].Output.ExitCode = exitCode
				if exitCode != nil {
					toolCalls[index].Output.Success = *exitCode == 0
					if *exitCode != 0 {
						errorText := result
						if len(errorText) > 1024 {
							errorText = errorText[:1024]
						}
						toolCalls[index].Output.Error = &errorText
					}
				}
			}
		}
	}

	if len(pendingTurnToolIDs) > 0 {
		lastTurnIndex := 0
		if len(turns) > 0 {
			lastTurnIndex = len(turns) - 1
		}
		for toolID := range pendingTurnToolIDs {
			if index, ok := pendingFunctionCalls[toolID]; ok {
				toolCalls[index].EmittingTurnIndex = &lastTurnIndex
			}
		}
	}

	return turns, toolCalls, annotations, timestamps, tokenTotals, metadata
}

func parseExecJSONL(records []map[string]any) ([]minitrace.Turn, []minitrace.ToolCall, []minitrace.Annotation, []time.Time, *minitrace.TokenTotals, codexMetadata) {
	turns := []minitrace.Turn{}
	toolCalls := []minitrace.ToolCall{}
	annotations := []minitrace.Annotation{}
	timestamps := []time.Time{}
	tokenTotals := &minitrace.TokenTotals{}
	metadata := codexMetadata{}

	currentThinking := []string{}
	turnIndex := 0

	for _, record := range records {
		recordType := stringValue(record["type"])
		timestamp := stringValue(record["timestamp"])
		if parsed, ok := minitrace.ParseTimestamp(timestamp); ok {
			timestamps = append(timestamps, parsed)
		}
		timestampPtr := optionalString(timestamp)

		switch recordType {
		case "thread.started":
			metadata.SessionID = stringValue(record["thread_id"])
		case "item.completed":
			item := mapValue(record["item"])
			if item == nil {
				continue
			}
			itemType := stringValue(item["type"])
			itemID := firstNonEmpty(stringValue(item["id"]), fmt.Sprintf("item-%d", len(toolCalls)))
			switch itemType {
			case "error":
				message := stringValue(item["message"])
				annotations = append(annotations, minitrace.BuildAnnotation(
					"ann-error-"+truncateID(itemID),
					"adapter",
					"session",
					firstNonEmpty(metadata.SessionID, "unknown"),
					"observation",
					"Codex error: "+truncateTitle(message, 60),
					message,
					[]string{"codex-error"},
					nil,
				))
			case "reasoning":
				if text := stringValue(item["text"]); text != "" {
					currentThinking = append(currentThinking, text)
				}
			case "command_execution":
				command := stringValue(item["command"])
				output := stringValue(item["aggregated_output"])
				var success bool
				var exitCode *int
				if exitCodeValue, ok := item["exit_code"]; ok {
					parsedExitCode := minitrace.SafeInt(exitCodeValue, 1)
					exitCode = &parsedExitCode
					success = parsedExitCode == 0
				} else {
					success = stringValue(item["status"]) == "completed"
				}
				toolCall := minitrace.BuildToolCall(
					itemID,
					&turnIndex,
					timestampPtr,
					"exec_command",
					classifyOperationFromCommand(command),
					optionalString(extractFilePathFromCommand(command)),
					optionalString(command),
					map[string]any{"cmd": command},
					success,
					output,
					nil,
					nil,
					map[string]any{
						"codex_function": "exec_command",
						"exit_code":      item["exit_code"],
						"status":         item["status"],
					},
					nil,
					classifyContentOrigin("exec_command"),
					nil,
				)
				toolCall.Output.ExitCode = exitCode
				toolCalls = append(toolCalls, toolCall)
			case "agent_message":
				source := ptr("model")
				turn := minitrace.BuildTurn(turnIndex, timestampPtr, "assistant", source, stringValue(item["text"]))
				if len(currentThinking) > 0 {
					thinking := strings.Join(currentThinking, "\n")
					turn.Thinking = &thinking
				}
				toolIDs := []string{}
				for _, toolCall := range toolCalls {
					if toolCall.EmittingTurnIndex != nil && *toolCall.EmittingTurnIndex == turnIndex {
						toolIDs = append(toolIDs, toolCall.ID)
					}
				}
				turn.ToolCallsInTurn = toolIDs
				turns = append(turns, turn)
				turnIndex++
				currentThinking = nil
			}
		}
	}

	return turns, toolCalls, annotations, timestamps, tokenTotals, metadata
}

func parseJSONLFile(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "opening Codex JSONL")
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	buffer := make([]byte, 0, 1024*1024)
	scanner.Buffer(buffer, 10*1024*1024)

	records := []map[string]any{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "scanning Codex JSONL")
	}
	return records, nil
}

func detectFormatRecords(records []map[string]any) string {
	for i, record := range records {
		if i >= 5 {
			break
		}
		switch stringValue(record["type"]) {
		case "session_meta", "response_item", "event_msg", "turn_context":
			return "session-jsonl-v1"
		case "thread.started", "turn.started", "turn.completed", "item.started", "item.completed":
			return "exec-jsonl-v1"
		}
	}
	return "unknown-jsonl"
}

func sourceFormatName(formatHint string) string {
	switch formatHint {
	case "session-jsonl-v1":
		return SourceFormatSession
	case "exec-jsonl-v1":
		return SourceFormatExec
	default:
		return formatHint
	}
}

func parseArguments(raw any) map[string]any {
	switch value := raw.(type) {
	case map[string]any:
		return value
	case string:
		if value == "" {
			return map[string]any{}
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(value), &parsed); err == nil && parsed != nil {
			return parsed
		}
		return map[string]any{"raw": value}
	default:
		return map[string]any{}
	}
}

func codexFrameworkMetadata(functionName string, args map[string]any) map[string]any {
	metadata := map[string]any{
		"codex_function": functionName,
	}
	if justification := stringValue(args["justification"]); justification != "" {
		metadata["justification"] = justification
	}
	return metadata
}

func classifyOperationFromCommand(command string) string {
	if command == "" {
		return "EXECUTE"
	}
	commandLower := strings.ToLower(strings.TrimSpace(command))
	if len(commandLower) > 1024 {
		commandLower = commandLower[:1024]
	}
	switch {
	case strings.Contains(command, ">>"):
		return "MODIFY"
	case strings.Contains(command, ">"):
		return "NEW"
	case regexp.MustCompile(`^tee\s`).MatchString(commandLower):
		return "NEW"
	case regexp.MustCompile(`^(touch|mkdir|cp)\s`).MatchString(commandLower):
		return "NEW"
	}
	for _, pattern := range modifyPatterns {
		if pattern.MatchString(commandLower) {
			return "MODIFY"
		}
	}
	for _, pattern := range readPatterns {
		if pattern.MatchString(commandLower) {
			return "READ"
		}
	}
	return "EXECUTE"
}

func classifyFunction(functionName string, args map[string]any) string {
	if functionName == "exec_command" {
		return classifyOperationFromCommand(stringValue(args["cmd"]))
	}
	switch functionName {
	case "read_file":
		return "READ"
	case "write_file":
		return "NEW"
	case "edit_file", "apply_patch", "apply_diff":
		return "MODIFY"
	default:
		return "OTHER"
	}
}

func extractFilePathFromCommand(command string) string {
	if command == "" {
		return ""
	}
	command = strings.TrimSpace(command)
	if match := regexp.MustCompile(`>{1,2}\s*(\S+)\s*$`).FindStringSubmatch(command); len(match) == 2 {
		return match[1]
	}
	if match := regexp.MustCompile(`\|\s*tee\s+(?:-a\s+)?(\S+)`).FindStringSubmatch(command); len(match) == 2 {
		return match[1]
	}
	if match := regexp.MustCompile(`^(?:cat|head|tail|less|more|wc|file|stat)\s+(?:-\S+\s+(?:\d+\s+)*)*(\S+)`).FindStringSubmatch(command); len(match) == 2 && !strings.HasPrefix(match[1], "-") {
		return match[1]
	}
	if match := regexp.MustCompile(`^(?:touch|mkdir)\s+(?:-\S+\s+)*(\S+)`).FindStringSubmatch(command); len(match) == 2 {
		return match[1]
	}
	if match := regexp.MustCompile(`^(?:cp|mv)\s+(?:-\S+\s+)*(\S+)\s+(\S+)`).FindStringSubmatch(command); len(match) == 3 {
		return match[2]
	}
	if match := regexp.MustCompile(`^sed\s+-i[^\s]*\s+.*\s+(\S+)\s*$`).FindStringSubmatch(command); len(match) == 2 {
		return match[1]
	}
	if match := regexp.MustCompile(`^(?:chmod|chown)\s+(?:-\S+\s+)*\S+\s+(\S+)`).FindStringSubmatch(command); len(match) == 2 {
		return match[1]
	}
	return ""
}

func classifyContentOrigin(functionName string) *string {
	switch functionName {
	case "exec_command":
		return ptr("local_exec")
	case "read_file":
		return ptr("local_file")
	case "write_file", "edit_file", "apply_patch", "apply_diff":
		return ptr("model_echo")
	default:
		return nil
	}
}

func parseFunctionOutput(raw string) (string, *int, *int) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil, nil
	}

	var structured struct {
		Output   string `json:"output"`
		Metadata struct {
			ExitCode        *int    `json:"exit_code"`
			DurationSeconds float64 `json:"duration_seconds"`
		} `json:"metadata"`
	}
	if strings.HasPrefix(raw, "{") && json.Unmarshal([]byte(raw), &structured) == nil {
		var durationMS *int
		if structured.Metadata.DurationSeconds > 0 {
			value := int(structured.Metadata.DurationSeconds * 1000)
			durationMS = &value
		}
		return structured.Output, structured.Metadata.ExitCode, durationMS
	}

	var exitCode *int
	var durationMS *int
	lines := strings.Split(raw, "\n")
	outputStarted := false
	outputLines := []string{}
	for _, line := range lines {
		switch {
		case outputStarted:
			outputLines = append(outputLines, line)
		case strings.HasPrefix(line, "Output:"):
			outputStarted = true
			rest := strings.TrimSpace(strings.TrimPrefix(line, "Output:"))
			if rest != "" {
				outputLines = append(outputLines, rest)
			}
		case strings.HasPrefix(line, "Process exited with code "):
			value := strings.TrimSpace(strings.TrimPrefix(line, "Process exited with code "))
			if parsed, err := strconv.Atoi(value); err == nil {
				exitCode = &parsed
			}
		case strings.HasPrefix(line, "Wall time: "):
			value := strings.TrimSpace(strings.TrimPrefix(line, "Wall time: "))
			if seconds, err := strconv.ParseFloat(strings.Fields(value)[0], 64); err == nil {
				parsed := int(seconds * 1000)
				durationMS = &parsed
			}
		}
	}
	if len(outputLines) > 0 {
		return strings.Join(outputLines, "\n"), exitCode, durationMS
	}
	return raw, exitCode, durationMS
}

func providerHint(modelProvider string) *string {
	switch modelProvider {
	case "ollama", "openai", "":
		return ptr("openai-compatible")
	default:
		return ptr(modelProvider)
	}
}

func mapApprovalPolicy(policy string) string {
	switch policy {
	case "never":
		return "full-auto"
	case "always":
		return "suggest"
	default:
		return policy
	}
}

func sandboxValue(policy string) *bool {
	switch {
	case policy == "danger-full-access":
		return ptr(false)
	case strings.Contains(strings.ToLower(policy), "sandbox"), strings.Contains(strings.ToLower(policy), "read-only"), strings.Contains(strings.ToLower(policy), "workspace-write"):
		return ptr(true)
	default:
		return nil
	}
}

func frameworkConfig(metadata codexMetadata) any {
	config := map[string]any{}
	if metadata.Personality != "" {
		config["personality"] = metadata.Personality
	}
	if metadata.CollaborationMode != "" {
		config["collaboration_mode"] = metadata.CollaborationMode
	}
	if metadata.ReasoningEffort != "" {
		config["reasoning_effort"] = metadata.ReasoningEffort
	}
	if metadata.Originator != "" {
		config["originator"] = metadata.Originator
	}
	if metadata.ContextWindow != 0 {
		config["model_context_window"] = metadata.ContextWindow
	}
	if metadata.Timezone != "" {
		config["timezone"] = metadata.Timezone
	}
	if len(config) == 0 {
		return nil
	}
	return config
}

func extractSandboxPolicy(value any) string {
	if value == nil {
		return ""
	}
	if raw, ok := value.(string); ok {
		return raw
	}
	sandbox := mapValue(value)
	if sandbox == nil {
		return ""
	}
	return firstNonEmpty(stringValue(sandbox["type"]), stringValue(sandbox["mode"]))
}

func pendingTurnToolIDsSlice(ids map[string]struct{}) []string {
	ret := make([]string, 0, len(ids))
	for id := range ids {
		ret = append(ret, id)
	}
	sort.Strings(ret)
	return ret
}

func uniqueToolNames(toolCalls []minitrace.ToolCall) []string {
	seen := map[string]struct{}{}
	names := make([]string, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		if toolCall.ToolName == "" {
			continue
		}
		if _, ok := seen[toolCall.ToolName]; ok {
			continue
		}
		seen[toolCall.ToolName] = struct{}{}
		names = append(names, toolCall.ToolName)
	}
	sort.Strings(names)
	return names
}

func optionalNormalizedPath(path string) *string {
	if path == "" {
		return nil
	}
	normalized := minitrace.NormalizePath(path)
	return &normalized
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func truncateID(value string) string {
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}

func truncateTitle(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func mapValue(value any) map[string]any {
	if value == nil {
		return nil
	}
	if v, ok := value.(map[string]any); ok {
		return v
	}
	return nil
}

func listValue(value any) []any {
	if value == nil {
		return nil
	}
	if v, ok := value.([]any); ok {
		return v
	}
	return nil
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if v, ok := value.(string); ok {
		return v
	}
	return fmt.Sprint(value)
}

func ptr[T any](value T) *T {
	return &value
}
