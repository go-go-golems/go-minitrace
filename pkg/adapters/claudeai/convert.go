package claudeai

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
)

const (
	AdapterVersion  = "go-minitrace-claude-ai-adapter-dev"
	SourceFormat    = "claude-ai-export-v1"
	RedactedMarker  = "This block is not supported on your current device yet."
	MaxDecompressed = 2 * 1024 * 1024 * 1024
)

var operationType = map[string]string{
	"view":                                "READ",
	"web_search":                          "READ",
	"web_fetch":                           "READ",
	"project_knowledge_search":            "READ",
	"conversation_search":                 "READ",
	"recent_chats":                        "READ",
	"present_files":                       "READ",
	"Filesystem:read_file":                "READ",
	"Filesystem:read_text_file":           "READ",
	"Filesystem:read_multiple_files":      "READ",
	"Filesystem:get_file_info":            "READ",
	"Filesystem:list_directory":           "READ",
	"Filesystem:directory_tree":           "READ",
	"Filesystem:search_files":             "READ",
	"Filesystem:list_allowed_directories": "READ",
	"str_replace":                         "MODIFY",
	"memory_user_edits":                   "MODIFY",
	"Filesystem:edit_file":                "MODIFY",
	"Filesystem:move_file":                "MODIFY",
	"create_file":                         "NEW",
	"artifacts":                           "NEW",
	"Filesystem:write_file":               "NEW",
	"Filesystem:create_directory":         "NEW",
	"Filesystem:copy_file_user_to_claude": "NEW",
	"bash_tool":                           "EXECUTE",
	"repl":                                "EXECUTE",
	"launch_extended_search_task":         "DELEGATE",
}

var contentOrigin = map[string]string{
	"bash_tool":                           "local_exec",
	"repl":                                "local_exec",
	"view":                                "local_file",
	"Filesystem:read_file":                "local_file",
	"Filesystem:read_text_file":           "local_file",
	"Filesystem:read_multiple_files":      "local_file",
	"web_search":                          "web",
	"web_fetch":                           "web",
	"project_knowledge_search":            "database",
	"conversation_search":                 "database",
	"recent_chats":                        "database",
	"artifacts":                           "model_echo",
	"create_file":                         "model_echo",
	"str_replace":                         "model_echo",
	"Filesystem:write_file":               "model_echo",
	"Filesystem:edit_file":                "model_echo",
	"Filesystem:copy_file_user_to_claude": "user_provided",
	"launch_extended_search_task":         "sub_agent",
	"memory_user_edits":                   "model_echo",
}

func StreamConversations(zipPath string, uuidFilter []string, fn func(map[string]any) error) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return errors.Wrap(err, "opening claude.ai export ZIP")
	}
	defer func() { _ = reader.Close() }()

	var conversationsFile *zip.File
	for _, file := range reader.File {
		if file.Name == "conversations.json" {
			conversationsFile = file
			break
		}
	}
	if conversationsFile == nil {
		return errors.New("ZIP does not contain conversations.json")
	}
	if conversationsFile.UncompressedSize64 > MaxDecompressed {
		return errors.Errorf(
			"conversations.json too large: %d bytes (limit %d)",
			conversationsFile.UncompressedSize64,
			MaxDecompressed,
		)
	}

	rc, err := conversationsFile.Open()
	if err != nil {
		return errors.Wrap(err, "opening conversations.json from ZIP")
	}
	defer func() { _ = rc.Close() }()

	limited := io.LimitReader(rc, MaxDecompressed+1)
	decoder := json.NewDecoder(limited)

	token, err := decoder.Token()
	if err != nil {
		return errors.Wrap(err, "reading opening JSON token")
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '[' {
		return errors.New("conversations.json is not a JSON array")
	}

	for decoder.More() {
		var conversation map[string]any
		if err := decoder.Decode(&conversation); err != nil {
			return errors.Wrap(err, "decoding conversation entry")
		}
		if matchesUUIDFilter(stringValue(conversation["uuid"]), uuidFilter) {
			if err := fn(conversation); err != nil {
				return err
			}
		}
	}

	if _, err := decoder.Token(); err != nil {
		return errors.Wrap(err, "reading closing JSON token")
	}
	return nil
}

func ConvertConversation(conv map[string]any, sourcePath string) (*minitrace.Session, error) {
	convUUID := firstNonEmpty(stringValue(conv["uuid"]), "unknown")
	messages := listValue(conv["chat_messages"])
	turns := make([]minitrace.Turn, 0, len(messages))
	toolCalls := make([]minitrace.ToolCall, 0)
	annotations := make([]minitrace.Annotation, 0)
	allTimestamps := make([]time.Time, 0, len(messages))
	turnIndex := 0
	toolCounter := 0

	for _, rawMessage := range messages {
		message := mapValue(rawMessage)
		if message == nil {
			continue
		}

		timestamp := stringValue(message["created_at"])
		if ts, ok := minitrace.ParseTimestamp(timestamp); ok {
			allTimestamps = append(allTimestamps, ts)
		}
		timestampPtr := optionalString(timestamp)

		switch stringValue(message["sender"]) {
		case "human":
			contentParts := []string{stringValue(message["text"])}
			var frameworkMetadata map[string]any
			attachments := listValue(message["attachments"])
			if len(attachments) > 0 {
				summaries := make([]map[string]any, 0, len(attachments))
				for _, rawAttachment := range attachments {
					attachment := mapValue(rawAttachment)
					if attachment == nil {
						continue
					}
					if extracted := stringValue(attachment["extracted_content"]); extracted != "" {
						contentParts = append(contentParts, extracted)
					}
					summaries = append(summaries, map[string]any{
						"file_name": stringValue(attachment["file_name"]),
						"file_size": attachment["file_size"],
						"file_type": stringValue(attachment["file_type"]),
					})
				}
				if len(summaries) > 0 {
					frameworkMetadata = map[string]any{"attachments": summaries}
				}
			}

			turn := minitrace.BuildTurn(turnIndex, timestampPtr, "user", ptr("human"), joinNonEmpty(contentParts, "\n\n"))
			turn.InputChannel = ptr("user_input")
			if frameworkMetadata != nil {
				turn.FrameworkMetadata = frameworkMetadata
			}
			turns = append(turns, turn)
			turnIndex++
		case "assistant":
			contentBlocks := listValue(message["content"])
			textParts := make([]string, 0, len(contentBlocks))
			toolIDs := make([]string, 0)
			thinkingParts := make([]string, 0)

			for i := 0; i < len(contentBlocks); i++ {
				block := mapValue(contentBlocks[i])
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
					toolName := firstNonEmpty(stringValue(block["name"]), "unknown")
					toolInput := mapValue(block["input"])
					integrationName := stringValue(block["integration_name"])
					messageText := stringValue(block["message"])

					durationMS := durationMillis(block["start_timestamp"], block["stop_timestamp"])
					origin := classifyContentOrigin(toolName, integrationName)
					filePath, command := extractToolFields(toolInput)
					var filePathPtr *string
					if filePath != "" {
						filePathPtr = &filePath
					}
					var commandPtr *string
					if command != "" {
						commandPtr = &command
					}

					toolID := firstNonEmpty(
						stringValue(block["id"]),
						fmt.Sprintf("tc-%s-%04d", truncateID(convUUID, 8), toolCounter),
					)

					var resultText any
					var errorText *string
					var redacted *bool
					success := true
					nextConsumed := false

					if i+1 < len(contentBlocks) {
						nextBlock := mapValue(contentBlocks[i+1])
						if nextBlock != nil && stringValue(nextBlock["type"]) == "tool_result" {
							nextConsumed = true
							if toolUseID := stringValue(nextBlock["tool_use_id"]); toolUseID != "" && toolUseID != toolID {
								annotations = append(annotations, minitrace.BuildAnnotation(
									"ann-tool-id-mismatch-"+truncateID(toolID, 24),
									"adapter",
									"tool_call",
									toolID,
									"observation",
									"Tool result ID mismatch",
									fmt.Sprintf("tool_use id=%s paired with tool_result.tool_use_id=%s", toolID, toolUseID),
									[]string{"data-quality", "pairing-mismatch"},
									nil,
								))
							}
							if resultName := stringValue(nextBlock["name"]); resultName != "" && resultName != toolName {
								annotations = append(annotations, minitrace.BuildAnnotation(
									"ann-tool-name-mismatch-"+truncateID(toolID, 24),
									"adapter",
									"tool_call",
									toolID,
									"observation",
									"Tool result name mismatch",
									fmt.Sprintf("tool_use name=%s paired with tool_result name=%s", toolName, resultName),
									[]string{"data-quality", "pairing-mismatch"},
									nil,
								))
							}
							resultString := stringifyToolResult(nextBlock["content"])
							if resultString != "" {
								resultText = resultString
								if strings.Contains(resultString, RedactedMarker) {
									value := true
									redacted = &value
								}
							}
							if boolValue(nextBlock["is_error"]) {
								success = false
								if resultString != "" {
									errorText = ptr(truncateString(resultString, 500))
								}
							}
						}
					}

					if !nextConsumed {
						success = false
						errorText = ptr("no tool result received")
						annotations = append(annotations, minitrace.BuildAnnotation(
							"ann-orphan-"+truncateID(toolID, 24),
							"adapter",
							"tool_call",
							toolID,
							"observation",
							"Tool call "+toolName+" has no tool_result",
							fmt.Sprintf("tool_use at assistant content index %d was not followed by tool_result", i),
							[]string{"data-quality", "orphan-tool-call"},
							nil,
						))
					} else {
						i++
					}

					emittingTurnIndex := turnIndex
					toolCall := minitrace.BuildToolCall(
						toolID,
						&emittingTurnIndex,
						firstNonNilString(stringValue(block["start_timestamp"]), timestamp),
						toolName,
						classifyOperation(toolName),
						filePathPtr,
						commandPtr,
						toolInput,
						success,
						resultText,
						errorText,
						durationMS,
						buildToolFrameworkMetadata(block, messageText, integrationName),
						nil,
						origin,
						redacted,
					)
					if toolName == "launch_extended_search_task" {
						toolCall.SpawnedAgent = &minitrace.SpawnedAgent{
							AgentType:      "extended_search",
							TaskScope:      truncateString(stringifyMap(toolInput), 200),
							SubSessionID:   nil,
							OutcomeSummary: "",
						}
					}

					toolCalls = append(toolCalls, toolCall)
					toolIDs = append(toolIDs, toolID)
					toolCounter++
				}
			}

			content := firstNonEmpty(joinNonEmpty(textParts, "\n"), stringValue(message["text"]))
			turn := minitrace.BuildTurn(turnIndex, timestampPtr, "assistant", ptr("model"), content)
			if len(thinkingParts) > 0 {
				thinking := joinNonEmpty(thinkingParts, "\n")
				turn.Thinking = &thinking
			}
			turn.ToolCallsInTurn = toolIDs
			turn.Streaming.WasStreamed = true
			turns = append(turns, turn)
			turnIndex++
		}
	}

	minitrace.ComputeToolCallContext(toolCalls)
	timing := minitrace.ComputeTiming(allTimestamps)
	quality := minitrace.AssignQualityTier(turns, toolCalls)

	session := minitrace.BuildSessionSkeleton(convUUID, "claude.ai", SourceFormat, AdapterVersion)
	session.Environment.PlatformType = ptr("web")
	session.Environment.ProviderHint = ptr("anthropic")
	session.Environment.ToolsEnabled = uniqueToolNames(toolCalls)
	session.Provenance.OriginalSessionID = ptr(convUUID)
	if sourcePath != "" {
		session.Provenance.SourcePath = ptr(sourcePath)
	}
	if title := stringValue(conv["name"]); title != "" {
		session.Title = &title
	}
	if summary := stringValue(conv["summary"]); summary != "" {
		session.Summary = &summary
	}
	session.Timing = timing
	session.Turns = turns
	session.ToolCalls = toolCalls
	session.Annotations = annotations
	session.Quality = &quality
	session.Metrics = minitrace.ComputeMetrics(turns, toolCalls, timing, countDelegates(toolCalls), nil)
	session.Flags.ContainsPII = true
	session.Flags.ForResearch = false
	session.Flags.NeedsCleaning = true
	session.Classification = "confidential"

	return &session, nil
}

func classifyOperation(toolName string) string {
	if operation, ok := operationType[toolName]; ok {
		return operation
	}
	return "OTHER"
}

func classifyContentOrigin(toolName, integrationName string) *string {
	if origin, ok := contentOrigin[toolName]; ok {
		return &origin
	}
	if integrationName != "" {
		return ptr("mcp_server")
	}
	return nil
}

func buildToolFrameworkMetadata(block map[string]any, messageText, integrationName string) map[string]any {
	metadata := map[string]any{}
	if integrationName != "" {
		metadata["integration_name"] = integrationName
	}
	if messageText != "" {
		metadata["message"] = messageText
	}
	if isMCPApp, ok := block["is_mcp_app"].(bool); ok {
		metadata["is_mcp_app"] = isMCPApp
	}
	if serverURL := stringValue(block["mcp_server_url"]); serverURL != "" {
		metadata["mcp_server_url"] = serverURL
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func extractToolFields(input map[string]any) (string, string) {
	if input == nil {
		return "", ""
	}
	filePath := firstNonEmpty(
		stringValue(input["file_path"]),
		stringValue(input["path"]),
		stringValue(input["file"]),
	)
	command := stringValue(input["command"])
	return filePath, command
}

func stringifyToolResult(content any) string {
	switch typed := content.(type) {
	case string:
		return typed
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			switch block := item.(type) {
			case string:
				if strings.TrimSpace(block) != "" {
					parts = append(parts, block)
				}
			case map[string]any:
				if text := stringValue(block["text"]); text != "" {
					parts = append(parts, text)
				} else if nested := stringifyToolResult(block["content"]); nested != "" {
					parts = append(parts, nested)
				} else {
					parts = append(parts, stringifyMap(block))
				}
			default:
				parts = append(parts, fmt.Sprint(item))
			}
		}
		return joinNonEmpty(parts, "\n")
	case map[string]any:
		if text := stringValue(typed["text"]); text != "" {
			return text
		}
		return stringifyMap(typed)
	default:
		return fmt.Sprint(content)
	}
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

func countDelegates(toolCalls []minitrace.ToolCall) int {
	count := 0
	for _, toolCall := range toolCalls {
		if toolCall.SpawnedAgent != nil {
			count++
		}
	}
	return count
}

func durationMillis(startRaw, stopRaw any) *int {
	start, ok := minitrace.ParseTimestamp(stringValue(startRaw))
	if !ok {
		return nil
	}
	stop, ok := minitrace.ParseTimestamp(stringValue(stopRaw))
	if !ok {
		return nil
	}
	value := int(stop.Sub(start).Milliseconds())
	return &value
}

func matchesUUIDFilter(uuid string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, filter := range filters {
		if strings.HasPrefix(uuid, filter) {
			return true
		}
	}
	return false
}

func joinNonEmpty(items []string, sep string) string {
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			filtered = append(filtered, item)
		}
	}
	return strings.Join(filtered, sep)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstNonNilString(values ...string) *string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return &value
		}
	}
	return nil
}

func truncateID(value string, n int) string {
	if n <= 0 || len(value) <= n {
		return value
	}
	return value[:n]
}

func truncateString(value string, n int) string {
	if n <= 0 || len(value) <= n {
		return value
	}
	return value[:n]
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		return fmt.Sprint(value)
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

func stringifyMap(value map[string]any) string {
	if value == nil {
		return ""
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(payload)
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func ptr[T any](value T) *T {
	return &value
}
