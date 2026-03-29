package claudecode

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/adapters"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
)

const (
	AdapterVersion = "go-minitrace-claude-adapter-dev"
	SourceFormatV2 = "claude-code-jsonl-v2"
	SourceFormatV1 = "claude-code-dir-v1"
)

var discardTypes = map[string]struct{}{
	"file-history-snapshot": {},
	"last-prompt":           {},
}

func ConvertLocator(locator adapters.SessionLocator) (*minitrace.Session, error) {
	switch locator.FormatHint {
	case "jsonl-v2":
		records, err := parseJSONLFile(locator.SourcePath)
		if err != nil {
			return nil, err
		}
		return ConvertRecords(records, locator.ID, locator.SourcePath)
	case "dir-v1":
		return convertDirSession(locator.SourcePath, locator.ID)
	default:
		return nil, errors.Errorf("unsupported Claude Code format hint: %s", locator.FormatHint)
	}
}

func ConvertRecords(records []map[string]any, sessionID, sourcePath string) (*minitrace.Session, error) {
	session := minitrace.BuildSessionSkeleton(sessionID, "claude-code", SourceFormatV2, AdapterVersion)
	session.Environment.PlatformType = ptr("agent")
	session.Environment.ProviderHint = ptr("anthropic")
	if sourcePath != "" {
		session.Provenance.SourcePath = ptr(sourcePath)
	}

	turns := make([]minitrace.Turn, 0, len(records))
	toolCalls := make([]minitrace.ToolCall, 0)
	annotations := make([]minitrace.Annotation, 0)
	allTimestamps := make([]time.Time, 0, len(records))
	pendingToolCalls := map[string]minitrace.ToolCall{}
	tokenTotals := &minitrace.TokenTotals{}

	first := map[string]any{}
	if len(records) > 0 {
		first = records[0]
	}
	if cwd := stringValue(first["cwd"]); cwd != "" {
		normalized := minitrace.NormalizePath(cwd)
		session.OperationalContext.WorkingDirectory = &normalized
	}
	if version := stringValue(first["version"]); version != "" {
		session.Environment.AgentVersion = &version
	}
	if gitBranch := stringValue(first["gitBranch"]); gitBranch != "" {
		session.OperationalContext.GitBranch = &gitBranch
	}

	turnIndex := 0
	toolCounter := 0
	var sessionModel *string

	for _, record := range records {
		recordType := stringValue(record["type"])
		if _, ok := discardTypes[recordType]; ok || recordType == "progress" {
			continue
		}

		timestamp := stringValue(record["timestamp"])
		if parsed, ok := minitrace.ParseTimestamp(timestamp); ok {
			allTimestamps = append(allTimestamps, parsed)
		}
		timestampPtr := optionalString(timestamp)

		switch recordType {
		case "system":
			message := mapValue(record["message"])
			content := flattenClaudeText(message["content"])
			source := ptr("framework")
			turn := minitrace.BuildTurn(turnIndex, timestampPtr, "system", source, content)
			turn.InputChannel = classifyInputChannel(record)
			turns = append(turns, turn)
			turnIndex++
		case "user":
			message := mapValue(record["message"])
			contentBlocks, isList := message["content"].([]any)
			if !isList {
				if rawBlocks, ok := message["content"].([]map[string]any); ok {
					contentBlocks = make([]any, 0, len(rawBlocks))
					for _, block := range rawBlocks {
						contentBlocks = append(contentBlocks, block)
					}
					isList = true
				}
			}

			if isList {
				hasToolResults := false
				textParts := make([]string, 0, len(contentBlocks))
				for _, item := range contentBlocks {
					block := mapValue(item)
					if block == nil {
						continue
					}
					if stringValue(block["type"]) == "tool_result" {
						hasToolResults = true
						toolUseID := stringValue(block["tool_use_id"])
						if pending, ok := pendingToolCalls[toolUseID]; ok {
							resultText := stringifyContent(block["content"])
							truncated, fullBytes, fullHash := minitrace.TruncateContent(resultText, minitrace.TruncateLimit)
							pending.Output.Success = !boolValue(block["is_error"])
							pending.Output.Result = truncated
							if boolValue(block["is_error"]) {
								errText := resultText
								if len(errText) > 500 {
									errText = errText[:500]
								}
								pending.Output.Error = &errText
							} else {
								pending.Output.Error = nil
							}
							pending.Output.Truncated = fullBytes != nil
							pending.Output.FullBytes = fullBytes
							pending.Output.FullHash = fullHash
							pending.Timestamp = timestampPtr
							toolCalls = append(toolCalls, pending)
							delete(pendingToolCalls, toolUseID)
						}
						continue
					}
					if text := blockText(block); text != "" {
						textParts = append(textParts, text)
					}
				}
				if hasToolResults {
					continue
				}
				content := strings.Join(textParts, "\n")
				source := classifySource(record)
				turn := minitrace.BuildTurn(turnIndex, timestampPtr, "user", source, content)
				turn.InputChannel = classifyInputChannel(record)
				turns = append(turns, turn)
				turnIndex++
				continue
			}

			content := stringifyContent(message["content"])
			source := classifySource(record)
			turn := minitrace.BuildTurn(turnIndex, timestampPtr, "user", source, content)
			turn.InputChannel = classifyInputChannel(record)
			turns = append(turns, turn)
			turnIndex++
		case "assistant":
			message := mapValue(record["message"])
			model := stringValue(message["model"])
			if model != "" {
				sessionModel = &model
			}
			usage := mapValue(message["usage"])
			turnUsage := buildUsage(usage, tokenTotals)
			contentBlocks := listValue(message["content"])

			textParts := make([]string, 0, len(contentBlocks))
			thinkingParts := make([]string, 0)
			toolIDs := make([]string, 0)

			for _, item := range contentBlocks {
				block := mapValue(item)
				if block == nil {
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
					}
				case "tool_use":
					toolCallID := stringValue(block["id"])
					if toolCallID == "" {
						toolCallID = fmt.Sprintf("tc-%04d", toolCounter)
					}
					toolCounter++

					toolName := stringValue(block["name"])
					toolInput := mapValue(block["input"])
					filePath := firstNonEmpty(stringValue(toolInput["file_path"]), stringValue(toolInput["path"]))
					command := stringValue(toolInput["command"])

					var filePathPtr *string
					if filePath != "" {
						filePathPtr = &filePath
					}
					var commandPtr *string
					if command != "" {
						commandPtr = &command
					}
					turnIndexCopy := turnIndex
					toolCall := minitrace.BuildToolCall(
						toolCallID,
						&turnIndexCopy,
						timestampPtr,
						toolName,
						classifyOperation(toolName),
						filePathPtr,
						commandPtr,
						toolInput,
						true,
						nil,
						nil,
						nil,
						nil,
						nil,
						classifyContentOrigin(toolName),
						nil,
					)
					if spawned := buildSpawnedAgent(toolName, toolInput); spawned != nil {
						toolCall.SpawnedAgent = spawned
					}
					pendingToolCalls[toolCallID] = toolCall
					toolIDs = append(toolIDs, toolCallID)
				}
			}

			source := ptr("model")
			turn := minitrace.BuildTurn(turnIndex, timestampPtr, "assistant", source, strings.Join(textParts, "\n"))
			if len(thinkingParts) > 0 {
				thinking := strings.Join(thinkingParts, "\n")
				turn.Thinking = &thinking
			}
			turn.ToolCallsInTurn = toolIDs
			turn.Usage = turnUsage
			turn.Streaming.WasStreamed = true
			turn.Model = sessionModel
			turns = append(turns, turn)
			turnIndex++
		}
	}

	for toolCallID, toolCall := range pendingToolCalls {
		errorText := "no tool_result received"
		toolCall.Output.Success = false
		toolCall.Output.Error = &errorText
		toolCalls = append(toolCalls, toolCall)

		annotation := minitrace.BuildAnnotation(
			"ann-orphan-"+truncateID(toolCallID),
			"adapter",
			"tool_call",
			toolCallID,
			"observation",
			"Tool call "+toolCall.ToolName+" never received result",
			"tool_use id="+toolCallID+" has no matching tool_result. Model may have crashed or timed out.",
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
		annotation := minitrace.BuildAnnotation(
			"ann-dedup-"+truncateID(sessionID),
			"adapter",
			"session",
			sessionID,
			"observation",
			fmt.Sprintf("Deduplicated %d duplicate tool calls", duplicateCount),
			fmt.Sprintf("Removed %d tool calls with duplicate IDs.", duplicateCount),
			[]string{"deduplication", "data-quality"},
			nil,
		)
		annotations = append(annotations, annotation)
	}

	minitrace.ComputeToolCallContext(toolCalls)

	timing := minitrace.ComputeTiming(allTimestamps)
	quality := minitrace.AssignQualityTier(turns, toolCalls)
	containsPII := minitrace.DetectPIIInPaths(toolCalls)

	session.Environment.Model = sessionModel
	session.Environment.ToolsEnabled = uniqueToolNames(toolCalls)
	session.Quality = &quality
	session.Title = minitrace.ExtractTitle(turns, 80)
	session.Timing = timing
	session.Turns = turns
	session.ToolCalls = toolCalls
	session.Annotations = annotations
	session.Metrics = minitrace.ComputeMetrics(turns, toolCalls, timing, countSubagents(toolCalls), tokenTotals)
	session.Flags.ContainsPII = containsPII
	session.Flags.ForResearch = quality == "A" && !containsPII
	session.Flags.NeedsCleaning = quality != "A" || containsPII
	if containsPII {
		session.Classification = "confidential"
	}

	return &session, nil
}

func convertDirSession(sessionDir, sessionID string) (*minitrace.Session, error) {
	session := minitrace.BuildSessionSkeleton(sessionID, "claude-code", SourceFormatV1, AdapterVersion)
	session.Environment.PlatformType = ptr("agent")
	session.Environment.ProviderHint = ptr("anthropic")
	session.Provenance.SourcePath = ptr(sessionDir)
	session.Flags.NeedsCleaning = true
	session.Flags.Category = append(session.Flags.Category, "dir-v1", "no-transcript")

	toolCalls := []minitrace.ToolCall{}
	toolResultsDir := filepath.Join(sessionDir, "tool-results")
	entries, err := os.ReadDir(toolResultsDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "reading Claude Code dir-v1 tool-results")
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".txt" {
			continue
		}
		path := filepath.Join(toolResultsDir, entry.Name())
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		toolCallID := strings.TrimSuffix(entry.Name(), ".txt")
		toolCall := minitrace.BuildToolCall(
			toolCallID,
			nil,
			nil,
			"unknown",
			"OTHER",
			nil,
			nil,
			nil,
			true,
			string(content),
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		)
		toolCalls = append(toolCalls, toolCall)
	}

	deduped, _ := minitrace.DeduplicateToolCalls(toolCalls, nil)
	toolCalls = deduped
	minitrace.ComputeToolCallContext(toolCalls)

	quality := "D"
	hasToolIO := false
	for _, toolCall := range toolCalls {
		if toolCall.Output.Result != nil {
			hasToolIO = true
			break
		}
	}
	switch {
	case hasToolIO && len(toolCalls) > 10:
		quality = "B"
	case len(toolCalls) > 0:
		quality = "C"
	}

	session.Quality = &quality
	session.ToolCalls = toolCalls
	session.Metrics = minitrace.ComputeMetrics(nil, toolCalls, session.Timing, 0, nil)
	return &session, nil
}

func parseJSONLFile(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "opening Claude Code JSONL")
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
		return nil, errors.Wrap(err, "scanning Claude Code JSONL")
	}
	return records, nil
}

func classifyOperation(toolName string) string {
	switch toolName {
	case "Read", "Glob", "Grep", "TaskGet", "TaskList", "TaskOutput", "WebFetch", "WebSearch", "ToolSearch":
		return "READ"
	case "Edit", "NotebookEdit":
		return "MODIFY"
	case "Write":
		return "NEW"
	case "Bash", "TaskStop":
		return "EXECUTE"
	case "Agent", "Task", "TaskCreate", "TaskUpdate":
		return "DELEGATE"
	default:
		return "OTHER"
	}
}

func classifySource(record map[string]any) *string {
	recordType := stringValue(record["type"])
	switch recordType {
	case "system":
		return ptr("system")
	case "assistant":
		return ptr("model")
	case "user":
		message := mapValue(record["message"])
		content := message["content"]
		for _, item := range listValue(content) {
			block := mapValue(item)
			if block != nil && stringValue(block["type"]) == "tool_result" {
				return ptr("framework")
			}
		}
		contentText := stringifyContent(content)
		if strings.Contains(contentText, "<system-reminder>") || strings.Contains(contentText, "<command-name>") {
			return ptr("framework")
		}
		return ptr("human")
	default:
		return nil
	}
}

func classifyInputChannel(record map[string]any) *string {
	recordType := stringValue(record["type"])
	switch recordType {
	case "system":
		return ptr("system_prompt")
	case "user":
		message := mapValue(record["message"])
		content := message["content"]
		for _, item := range listValue(content) {
			block := mapValue(item)
			if block != nil && stringValue(block["type"]) == "tool_result" {
				return ptr("tool_output")
			}
		}
		contentText := stringifyContent(content)
		switch {
		case strings.Contains(contentText, "<system-reminder>"):
			return ptr("framework_control")
		case strings.Contains(contentText, "<command-name>"):
			return ptr("framework_control")
		case strings.Contains(contentText, "<available-deferred-tools>"):
			return ptr("framework_control")
		default:
			return ptr("user_input")
		}
	default:
		return nil
	}
}

func classifyContentOrigin(toolName string) *string {
	if strings.HasPrefix(toolName, "mcp__") {
		return ptr("mcp_server")
	}
	switch toolName {
	case "Read", "Glob", "Grep", "Edit", "Write", "NotebookEdit":
		return ptr("local_file")
	case "Bash":
		return ptr("local_exec")
	case "WebFetch", "WebSearch":
		return ptr("web")
	case "Agent", "Task", "TaskCreate", "TaskUpdate", "TaskGet", "TaskList", "TaskOutput", "TaskStop":
		return ptr("sub_agent")
	case "AskUserQuestion":
		return ptr("user_provided")
	default:
		return nil
	}
}

func buildSpawnedAgent(toolName string, toolInput map[string]any) *minitrace.SpawnedAgent {
	switch toolName {
	case "Agent", "Task", "TaskCreate":
		agentType := firstNonEmpty(stringValue(toolInput["subagent_type"]), stringValue(toolInput["type"]), "general")
		taskScope := firstNonEmpty(stringValue(toolInput["prompt"]), stringValue(toolInput["description"]))
		if len(taskScope) > 200 {
			taskScope = taskScope[:200]
		}
		return &minitrace.SpawnedAgent{
			AgentType:      agentType,
			TaskScope:      taskScope,
			SubSessionID:   nil,
			OutcomeSummary: "",
		}
	default:
		return nil
	}
}

func buildUsage(usage map[string]any, tokenTotals *minitrace.TokenTotals) *minitrace.Usage {
	if len(usage) == 0 {
		return nil
	}

	inputTokens := minitrace.SafeInt(usage["input_tokens"], 0)
	outputTokens := minitrace.SafeInt(usage["output_tokens"], 0)
	cacheReadTokens := minitrace.SafeInt(usage["cache_read_input_tokens"], 0)
	cacheCreationTokens := minitrace.SafeInt(usage["cache_creation_input_tokens"], 0)
	tokenTotals.Input += inputTokens
	tokenTotals.Output += outputTokens
	tokenTotals.CacheRead += cacheReadTokens
	tokenTotals.CacheCreation += cacheCreationTokens

	u := &minitrace.Usage{}
	if inputTokens != 0 {
		u.InputTokens = &inputTokens
	}
	if outputTokens != 0 {
		u.OutputTokens = &outputTokens
	}
	if cacheReadTokens != 0 {
		u.CacheReadTokens = &cacheReadTokens
	}
	if cacheCreationTokens != 0 {
		u.CacheCreationTokens = &cacheCreationTokens
	}
	return u
}

func countSubagents(toolCalls []minitrace.ToolCall) int {
	count := 0
	for _, toolCall := range toolCalls {
		if toolCall.SpawnedAgent != nil {
			count++
		}
	}
	return count
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

func mapValue(value any) map[string]any {
	switch v := value.(type) {
	case map[string]any:
		return v
	default:
		return nil
	}
}

func listValue(value any) []any {
	switch v := value.(type) {
	case []any:
		return v
	default:
		return nil
	}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(value)
	}
}

func boolValue(value any) bool {
	if v, ok := value.(bool); ok {
		return v
	}
	return false
}

func stringifyContent(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			block := mapValue(item)
			if block != nil {
				if text := blockText(block); text != "" {
					parts = append(parts, text)
				}
				continue
			}
			parts = append(parts, fmt.Sprint(item))
		}
		return strings.Join(parts, "\n")
	default:
		return fmt.Sprint(value)
	}
}

func flattenClaudeText(value any) string {
	parts := make([]string, 0)
	for _, item := range listValue(value) {
		block := mapValue(item)
		if block == nil {
			continue
		}
		if stringValue(block["type"]) == "text" {
			if text := stringValue(block["text"]); text != "" {
				parts = append(parts, text)
			}
		}
	}
	if len(parts) == 0 {
		return stringifyContent(value)
	}
	return strings.Join(parts, "\n")
}

func blockText(block map[string]any) string {
	switch stringValue(block["type"]) {
	case "text":
		return stringValue(block["text"])
	case "thinking":
		return stringValue(block["thinking"])
	case "tool_result":
		return stringifyContent(block["content"])
	default:
		if text := stringValue(block["text"]); text != "" {
			return text
		}
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
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

func ptr[T any](value T) *T {
	return &value
}
