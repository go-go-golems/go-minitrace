package pi

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/adapters"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
)

const (
	AdapterVersion = "go-minitrace-pi-adapter-dev"
	SourceFormat   = "pi-agent-jsonl-v3"
)

func ConvertLocator(locator adapters.SessionLocator) (*minitrace.Session, error) {
	records, err := parseJSONLFile(locator.SourcePath)
	if err != nil {
		return nil, err
	}
	return ConvertRecords(records, locator.ID, locator.SourcePath)
}

func ConvertFile(path string) (*minitrace.Session, error) {
	root, err := expandHome(path)
	if err != nil {
		return nil, err
	}
	records, err := parseJSONLFile(root)
	if err != nil {
		return nil, err
	}
	sessionID := strings.TrimSuffix(filepathBase(root), ".jsonl")
	if index := strings.LastIndex(sessionID, "_"); index >= 0 && index < len(sessionID)-1 {
		sessionID = sessionID[index+1:]
	}
	return ConvertRecords(records, sessionID, root)
}

func ConvertRecords(records []map[string]any, fallbackID, sourcePath string) (*minitrace.Session, error) {
	sessionMeta := map[string]any{}
	modelInfo := map[string]any{}
	turns := make([]minitrace.Turn, 0, len(records))
	toolCalls := make([]minitrace.ToolCall, 0)
	annotations := make([]minitrace.Annotation, 0)
	allTimestamps := make([]time.Time, 0, len(records))
	tokenTotals := &minitrace.TokenTotals{}
	pendingToolCalls := map[string]int{}
	turnIndex := 0

	for _, record := range records {
		recordType := stringValue(record["type"])
		switch recordType {
		case "session":
			sessionMeta = record
			if ts, ok := minitrace.ParseTimestamp(stringValue(record["timestamp"])); ok {
				allTimestamps = append(allTimestamps, ts)
			}
		case "model_change":
			modelInfo["provider"] = record["provider"]
			modelInfo["model"] = record["modelId"]
			if api := stringValue(record["api"]); api != "" {
				modelInfo["api"] = api
			}
		case "thinking_level_change":
			modelInfo["thinking_level"] = record["thinkingLevel"]
		case "message":
			msg := mapValue(record["message"])
			if msg == nil {
				continue
			}
			timestamp := firstNonEmpty(stringValue(record["timestamp"]), stringValue(msg["timestamp"]))
			if ts, ok := minitrace.ParseTimestamp(timestamp); ok {
				allTimestamps = append(allTimestamps, ts)
			}
			timestampPtr := optionalString(timestamp)
			role := stringValue(msg["role"])
			contentBlocks := listValue(msg["content"])
			if len(contentBlocks) == 0 && msg["content"] != nil {
				contentBlocks = []any{msg["content"]}
			}

			usage := buildUsage(mapValue(msg["usage"]), tokenTotals)
			if provider := stringValue(msg["provider"]); provider != "" {
				modelInfo["provider"] = provider
			}
			if model := stringValue(msg["model"]); model != "" {
				modelInfo["model"] = model
			}
			if api := stringValue(msg["api"]); api != "" {
				modelInfo["api"] = api
			}

			textParts := make([]string, 0, len(contentBlocks))
			thinkingParts := make([]string, 0)
			toolIDs := make([]string, 0)

			for _, rawBlock := range contentBlocks {
				block := mapValue(rawBlock)
				if block == nil {
					text := strings.TrimSpace(fmt.Sprint(rawBlock))
					if text != "" {
						textParts = append(textParts, text)
					}
					continue
				}

				switch stringValue(block["type"]) {
				case "text":
					if text := stringValue(block["text"]); text != "" {
						textParts = append(textParts, text)
					}
				case "thinking":
					if thinking := stringValue(block["thinking"]); thinking != "" {
						thinkingParts = append(thinkingParts, thinking)
						tokenTotals.Reasoning += minitrace.SafeInt(block["token_count"], 0)
					}
				case "toolCall", "tool_use":
					toolCallID := firstNonEmpty(stringValue(block["id"]), fmt.Sprintf("tc-%04d", len(toolCalls)))
					toolName := firstNonEmpty(stringValue(block["name"]), "unknown")
					arguments := mapValue(firstNonNil(block["arguments"], block["input"]))
					filePath := extractFilePath(arguments)
					command := extractCommand(toolName, arguments)

					var filePathPtr *string
					if filePath != "" {
						filePathPtr = &filePath
					}
					var commandPtr *string
					if command != "" {
						commandPtr = &command
					}
					emittingTurnIndex := turnIndex
					toolCall := minitrace.BuildToolCall(
						toolCallID,
						&emittingTurnIndex,
						timestampPtr,
						toolName,
						classifyOperation(toolName, arguments),
						filePathPtr,
						commandPtr,
						arguments,
						true,
						nil,
						nil,
						nil,
						nil,
						nil,
						classifyContentOrigin(toolName),
						nil,
					)
					pendingToolCalls[toolCallID] = len(toolCalls)
					toolCalls = append(toolCalls, toolCall)
					toolIDs = append(toolIDs, toolCallID)
				case "toolResult", "tool_result":
					toolCallID := firstNonEmpty(stringValue(block["toolUseId"]), stringValue(block["tool_use_id"]))
					if toolCallID != "" {
						if pendingIndex, ok := pendingToolCalls[toolCallID]; ok {
							applyToolResult(&toolCalls[pendingIndex], block["content"], boolValue(firstNonNil(block["isError"], block["is_error"])), timestampPtr)
							delete(pendingToolCalls, toolCallID)
						}
					}
				}
			}

			if role == "toolResult" {
				toolCallID := stringValue(msg["toolCallId"])
				if toolCallID != "" {
					if pendingIndex, ok := pendingToolCalls[toolCallID]; ok {
						applyToolResult(&toolCalls[pendingIndex], contentBlocks, false, timestampPtr)
						delete(pendingToolCalls, toolCallID)
					}
				}
			}

			source, normalizedRole := classifyTurnRole(role)
			content := strings.TrimSpace(strings.Join(textParts, "\n"))
			turn := minitrace.BuildTurn(turnIndex, timestampPtr, normalizedRole, source, content)
			if len(thinkingParts) > 0 {
				thinking := strings.Join(thinkingParts, "\n")
				turn.Thinking = &thinking
			}
			if usage != nil {
				turn.Usage = usage
			}
			if model, ok := modelInfo["model"].(string); ok && model != "" {
				turn.Model = &model
			}
			if len(toolIDs) > 0 {
				turn.ToolCallsInTurn = toolIDs
			}
			if normalizedRole == "assistant" {
				turn.Streaming.WasStreamed = true
			}
			turns = append(turns, turn)
			turnIndex++
		}
	}

	for toolCallID, pendingIndex := range pendingToolCalls {
		errorText := "no tool result received"
		toolCalls[pendingIndex].Output.Success = false
		toolCalls[pendingIndex].Output.Error = &errorText

		annotation := minitrace.BuildAnnotation(
			"ann-orphan-"+truncateID(toolCallID),
			"adapter",
			"tool_call",
			toolCallID,
			"observation",
			"Tool call "+toolCalls[pendingIndex].ToolName+" never received result",
			"Pi tool call id="+toolCallID+" has no matching tool result record.",
			[]string{"data-quality", "orphan-tool-call"},
			nil,
		)
		annotations = append(annotations, annotation)
	}

	sort.Slice(toolCalls, func(i, j int) bool {
		left := ""
		if toolCalls[i].Timestamp != nil {
			left = *toolCalls[i].Timestamp
		}
		right := ""
		if toolCalls[j].Timestamp != nil {
			right = *toolCalls[j].Timestamp
		}
		return left < right
	})

	deduped, duplicateCount := minitrace.DeduplicateToolCalls(toolCalls, nil)
	toolCalls = deduped
	if duplicateCount > 0 {
		annotations = append(annotations, minitrace.BuildAnnotation(
			"ann-dedup-"+truncateID(fallbackID),
			"adapter",
			"session",
			fallbackID,
			"observation",
			fmt.Sprintf("Deduplicated %d duplicate tool calls", duplicateCount),
			fmt.Sprintf("Removed %d Pi tool calls with duplicate IDs.", duplicateCount),
			[]string{"deduplication", "data-quality"},
			nil,
		))
	}

	sessionID := firstNonEmpty(stringValue(sessionMeta["id"]), fallbackID)
	session := minitrace.BuildSessionSkeleton(sessionID, "pi", SourceFormat, AdapterVersion)
	session.Provenance.SourcePath = optionalString(sourcePath)
	session.Environment.PlatformType = ptr("agent")
	if provider := stringValue(modelInfo["provider"]); provider != "" {
		session.Environment.ProviderHint = &provider
	}
	if model := stringValue(modelInfo["model"]); model != "" {
		session.Environment.Model = &model
	}
	if cwd := stringValue(sessionMeta["cwd"]); cwd != "" {
		normalized := minitrace.NormalizePath(cwd)
		session.OperationalContext.WorkingDirectory = &normalized
	}
	if frameworkConfig := buildFrameworkConfig(modelInfo); frameworkConfig != nil {
		session.OperationalContext.FrameworkConfig = frameworkConfig
	}

	minitrace.ComputeToolCallContext(toolCalls)
	timing := minitrace.ComputeTiming(allTimestamps)
	quality := minitrace.AssignQualityTier(turns, toolCalls)
	containsPII := minitrace.DetectPIIInPaths(toolCalls)

	session.Environment.ToolsEnabled = uniqueToolNames(toolCalls)
	session.Title = minitrace.ExtractTitle(turns, 80)
	session.Quality = &quality
	session.Timing = timing
	session.Turns = turns
	session.ToolCalls = toolCalls
	session.Annotations = annotations
	session.Metrics = minitrace.ComputeMetrics(turns, toolCalls, timing, 0, tokenTotals)
	session.Flags.NeedsCleaning = false
	session.Flags.ContainsPII = containsPII
	if containsPII {
		session.Flags.NeedsCleaning = true
		session.Classification = "confidential"
	}

	return &session, nil
}

func parseJSONLFile(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "opening Pi JSONL")
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
		return nil, errors.Wrap(err, "scanning Pi JSONL")
	}
	return records, nil
}

func buildUsage(raw map[string]any, totals *minitrace.TokenTotals) *minitrace.Usage {
	if raw == nil {
		return nil
	}

	input := minitrace.SafeInt(raw["input"], 0)
	output := minitrace.SafeInt(raw["output"], 0)
	cacheRead := minitrace.SafeInt(raw["cacheRead"], 0)
	cacheWrite := minitrace.SafeInt(raw["cacheWrite"], 0)
	reasoning := minitrace.SafeInt(raw["reasoning"], 0)

	totals.Input += input
	totals.Output += output
	totals.CacheRead += cacheRead
	totals.CacheCreation += cacheWrite
	totals.Reasoning += reasoning

	if cost := floatValuePath(raw, "cost", "total"); cost != nil {
		if totals.Cost == nil {
			totals.Cost = ptr(0.0)
		}
		*totals.Cost += *cost
	}

	usage := &minitrace.Usage{}
	if input != 0 {
		usage.InputTokens = &input
	}
	if output != 0 {
		usage.OutputTokens = &output
	}
	if cacheRead != 0 {
		usage.CacheReadTokens = &cacheRead
	}
	if cacheWrite != 0 {
		usage.CacheCreationTokens = &cacheWrite
	}
	if reasoning != 0 {
		usage.ReasoningTokens = &reasoning
	}
	return usage
}

func applyToolResult(toolCall *minitrace.ToolCall, content any, isError bool, timestamp *string) {
	truncated, fullBytes, fullHash := minitrace.TruncateContent(stringifyContent(content), minitrace.TruncateLimit)
	toolCall.Output.Success = !isError
	toolCall.Output.Result = truncated
	toolCall.Output.Truncated = fullBytes != nil
	toolCall.Output.FullBytes = fullBytes
	toolCall.Output.FullHash = fullHash
	toolCall.Timestamp = timestamp
	if isError && truncated != nil {
		errorText := *truncated
		if len(errorText) > 500 {
			errorText = errorText[:500]
		}
		toolCall.Output.Error = &errorText
	}
}

func classifyOperation(toolName string, arguments map[string]any) string {
	switch strings.ToLower(toolName) {
	case "bash", "shell":
		command := strings.ToLower(strings.TrimSpace(stringValue(arguments["command"])))
		switch {
		case startsWithAny(command, "cat ", "head ", "tail ", "less ", "ls ", "find ", "grep ", "rg ", "fd "):
			return "READ"
		case strings.Contains(command, " > ") || startsWithAny(command, "touch ", "mkdir "):
			return "NEW"
		case strings.Contains(command, " >> ") || strings.Contains(command, "sed -i"):
			return "MODIFY"
		default:
			return "EXECUTE"
		}
	case "read", "read_file", "search", "list_files", "glob", "grep":
		return "READ"
	case "write", "write_file":
		return "NEW"
	case "edit", "edit_file":
		return "MODIFY"
	default:
		return "OTHER"
	}
}

func classifyContentOrigin(toolName string) *string {
	switch strings.ToLower(toolName) {
	case "read", "read_file", "write", "write_file", "edit", "edit_file", "search", "list_files", "glob", "grep":
		return ptr("local_file")
	case "bash", "shell":
		return ptr("runtime")
	default:
		return nil
	}
}

func extractFilePath(arguments map[string]any) string {
	for _, key := range []string{"path", "file_path", "file", "filename"} {
		if value := stringValue(arguments[key]); value != "" {
			return value
		}
	}
	return ""
}

func extractCommand(toolName string, arguments map[string]any) string {
	switch strings.ToLower(toolName) {
	case "bash", "shell":
		return stringValue(arguments["command"])
	default:
		return ""
	}
}

func classifyTurnRole(role string) (*string, string) {
	switch role {
	case "user":
		return ptr("human"), "user"
	case "assistant":
		return ptr("model"), "assistant"
	case "toolResult":
		return ptr("framework"), "user"
	default:
		return ptr("framework"), "user"
	}
}

func buildFrameworkConfig(modelInfo map[string]any) map[string]any {
	ret := map[string]any{}
	if thinkingLevel := stringValue(modelInfo["thinking_level"]); thinkingLevel != "" {
		ret["thinking_level"] = thinkingLevel
	}
	if api := stringValue(modelInfo["api"]); api != "" {
		ret["api"] = api
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}

func uniqueToolNames(toolCalls []minitrace.ToolCall) []string {
	seen := map[string]struct{}{}
	ret := []string{}
	for _, toolCall := range toolCalls {
		if toolCall.ToolName == "" {
			continue
		}
		if _, ok := seen[toolCall.ToolName]; ok {
			continue
		}
		seen[toolCall.ToolName] = struct{}{}
		ret = append(ret, toolCall.ToolName)
	}
	sort.Strings(ret)
	return ret
}

func stringifyContent(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, stringifyContent(item))
		}
		return strings.Join(filterEmpty(parts), "\n")
	case map[string]any:
		if text := stringValue(typed["text"]); text != "" {
			return text
		}
		if content := typed["content"]; content != nil {
			return stringifyContent(content)
		}
		blob, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(blob)
	default:
		return fmt.Sprint(value)
	}
}

func filterEmpty(items []string) []string {
	ret := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			ret = append(ret, item)
		}
	}
	return ret
}

func startsWithAny(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func truncateID(id string) string {
	if len(id) > 24 {
		return id[:24]
	}
	return id
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(value)
	}
}

func mapValue(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	default:
		return nil
	}
}

func listValue(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []map[string]any:
		ret := make([]any, 0, len(typed))
		for _, item := range typed {
			ret = append(ret, item)
		}
		return ret
	default:
		return nil
	}
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(typed, "true")
	default:
		return false
	}
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func floatValuePath(root map[string]any, keys ...string) *float64 {
	current := any(root)
	for _, key := range keys {
		next, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = next[key]
	}
	switch typed := current.(type) {
	case float64:
		return &typed
	case float32:
		value := float64(typed)
		return &value
	case int:
		value := float64(typed)
		return &value
	case int64:
		value := float64(typed)
		return &value
	default:
		return nil
	}
}

func filepathBase(path string) string {
	parts := strings.Split(strings.ReplaceAll(path, "\\", "/"), "/")
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}

func ptr[T any](value T) *T {
	return &value
}
