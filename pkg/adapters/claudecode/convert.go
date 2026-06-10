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

func ConvertSubagentLocator(locator SubagentLocator) (*minitrace.Session, string, error) {
	records, err := parseJSONLFile(locator.SourcePath)
	if err != nil {
		return nil, "", err
	}
	if len(records) == 0 {
		return nil, "", errors.New("empty subagent transcript")
	}

	first := records[0]
	agentID := firstNonEmpty(stringValue(first["agentId"]), locator.AgentID)
	slug := stringValue(first["slug"])

	session, err := ConvertRecords(records, agentID, locator.SourcePath)
	if err != nil {
		return nil, "", err
	}
	AdjustSubagentSession(session, agentID, locator.ParentSessionID, slug)
	return session, agentID, nil
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
	events := make([]minitrace.Event, 0)
	attachments := make([]minitrace.Attachment, 0)
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
	attachmentCounter := 0
	var sessionModel *string
	aiTitle := ""
	frameworkConfig := map[string]any{}

	for _, record := range records {
		captureClaudeSessionMetadata(frameworkConfig, record)

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
		case "mode":
			if mode := stringValue(record["mode"]); mode != "" {
				frameworkConfig["mode"] = mode
				events = append(events, buildClaudeLifecycleEvent(record, len(events), "mode_change", "Claude Code mode change", mode))
			}
			continue
		case "permission-mode":
			if permissionMode := stringValue(record["permissionMode"]); permissionMode != "" {
				frameworkConfig["permission_mode"] = permissionMode
				if autonomy := mapClaudePermissionMode(permissionMode); autonomy != "" {
					session.OperationalContext.AutonomyLevel = &autonomy
				}
				events = append(events, buildClaudeLifecycleEvent(record, len(events), "permission_mode_change", "Claude Code permission mode change", permissionMode))
			}
			continue
		case "ai-title":
			if title := stringValue(record["aiTitle"]); title != "" {
				aiTitle = title
				frameworkConfig["ai_title"] = title
				events = append(events, buildClaudeLifecycleEvent(record, len(events), "title_change", "Claude Code title change", title))
			}
			continue
		case "attachment":
			attachmentCounter++
			attachment := buildClaudeAttachment(record, attachmentCounter)
			event := buildClaudeAttachmentEvent(record, attachment, len(events))
			attachment.EventID = &event.ID
			attachments = append(attachments, attachment)
			events = append(events, event)
			continue
		case "system":
			message := mapValue(record["message"])
			content := flattenClaudeText(message["content"])
			source := ptr("framework")
			turn := minitrace.BuildTurn(turnIndex, timestampPtr, "system", source, content)
			turn.InputChannel = classifyInputChannel(record)
			turn.FrameworkMetadata = claudeTurnMetadata(record, message)
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
							pending.FrameworkMetadata = mergeMetadataMap(pending.FrameworkMetadata, claudeToolResultMetadata(record))
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
				turn.FrameworkMetadata = claudeTurnMetadata(record, message)
				turns = append(turns, turn)
				turnIndex++
				continue
			}

			content := stringifyContent(message["content"])
			source := classifySource(record)
			turn := minitrace.BuildTurn(turnIndex, timestampPtr, "user", source, content)
			turn.InputChannel = classifyInputChannel(record)
			turn.FrameworkMetadata = claudeTurnMetadata(record, message)
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
						claudeToolMetadata(record, block),
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
			turn.FrameworkMetadata = claudeTurnMetadata(record, message)
			turns = append(turns, turn)
			turnIndex++
		}
	}

	if len(frameworkConfig) > 0 {
		session.OperationalContext.FrameworkConfig = frameworkConfig
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
	if aiTitle != "" {
		session.Title = &aiTitle
	} else {
		session.Title = minitrace.ExtractTitle(turns, 80)
	}
	session.Timing = timing
	session.Turns = turns
	session.ToolCalls = toolCalls
	session.Events = events
	session.Attachments = attachments
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

func AdjustSubagentSession(session *minitrace.Session, agentID, parentSessionID, slug string) {
	if session == nil {
		return
	}

	session.ID = agentID
	session.Provenance.OriginalSessionID = &agentID
	session.Provenance.SourceFormat = SourceFormatV2 + "+subagent"
	session.Flags.Category = appendUnique(session.Flags.Category, "subagent")

	switch {
	case slug != "":
		title := "[subagent] " + slug
		session.Title = &title
	case session.Title != nil && *session.Title != "":
		title := "[subagent] " + *session.Title
		session.Title = &title
	default:
		title := "[subagent] " + agentID
		session.Title = &title
	}

	config, _ := session.OperationalContext.FrameworkConfig.(map[string]any)
	if config == nil {
		config = map[string]any{}
	}
	config["parent_session"] = parentSessionID
	session.OperationalContext.FrameworkConfig = config
}

func LinkParentSubagents(session *minitrace.Session, subagentIDs []string) {
	if session == nil || len(subagentIDs) == 0 {
		return
	}

	index := 0
	for i := range session.ToolCalls {
		if session.ToolCalls[i].SpawnedAgent == nil {
			continue
		}
		if index >= len(subagentIDs) {
			return
		}
		subSessionID := subagentIDs[index]
		session.ToolCalls[i].SpawnedAgent.SubSessionID = &subSessionID
		index++
	}
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

func captureClaudeSessionMetadata(config map[string]any, record map[string]any) {
	if config == nil || record == nil {
		return
	}
	if entrypoint := stringValue(record["entrypoint"]); entrypoint != "" {
		config["entrypoint"] = entrypoint
	}
	if agentID := stringValue(record["agentId"]); agentID != "" {
		config["agent_id"] = agentID
	}
	if sessionID := stringValue(record["sessionId"]); sessionID != "" {
		config["session_id"] = sessionID
	}
	if parentUUID, ok := record["parentUuid"]; ok {
		config["parent_uuid"] = parentUUID
	}
	if isSidechain, ok := record["isSidechain"]; ok {
		config["is_sidechain"] = isSidechain
	}
	if userType := stringValue(record["userType"]); userType != "" {
		config["user_type"] = userType
	}
	if attributionAgent := stringValue(record["attributionAgent"]); attributionAgent != "" {
		config["attribution_agent"] = attributionAgent
	}
}

func mapClaudePermissionMode(permissionMode string) string {
	switch permissionMode {
	case "bypassPermissions":
		return "full-auto"
	case "plan":
		return "suggest"
	default:
		return ""
	}
}

func buildClaudeLifecycleEvent(record map[string]any, ordinal int, kind, title, summary string) minitrace.Event {
	timestamp := optionalString(stringValue(record["timestamp"]))
	event := minitrace.BuildEvent(fmt.Sprintf("claude-%s-%06d", strings.ReplaceAll(kind, "_", "-"), ordinal), timestamp, kind, title, summary, record)
	event.Role = "system"
	event.FrameworkMetadata = claudeRecordMetadata(record)
	return event
}

func buildClaudeAttachment(record map[string]any, ordinal int) minitrace.Attachment {
	attachment := mapValue(record["attachment"])
	attachmentType := firstNonEmpty(stringValue(attachment["type"]), "attachment")
	attachmentID := firstNonEmpty(stringValue(record["uuid"]), fmt.Sprintf("attachment-%d", ordinal))
	kind := attachmentType
	if hasClaudeImageSignal(attachment) {
		kind = "image"
	}
	mediaType := firstNonEmpty(stringValue(attachment["mediaType"]), stringValue(attachment["mimeType"]))
	name := firstNonEmpty(stringValue(attachment["fileName"]), stringValue(attachment["name"]), attachmentID)
	ret := minitrace.BuildAttachment(attachmentID, optionalString(stringValue(record["timestamp"])), kind, name, mediaType, record)
	if path := firstNonEmpty(stringValue(attachment["path"]), stringValue(attachment["filePath"])); path != "" {
		normalized := minitrace.NormalizePath(path)
		ret.Path = normalized
	}
	ret.URL = stringValue(attachment["url"])
	rawSize := attachment["sizeBytes"]
	if rawSize == nil {
		rawSize = attachment["fileSize"]
	}
	if sizeBytes := minitrace.SafeInt(rawSize, 0); sizeBytes != 0 {
		ret.SizeBytes = &sizeBytes
	}
	ret.Hash = firstNonEmpty(stringValue(attachment["hash"]), stringValue(attachment["sha256"]), stringValue(attachment["contentHash"]))
	ret.ContentRef = firstNonEmpty(stringValue(attachment["contentRef"]), stringValue(attachment["reference"]), stringValue(attachment["ref"]))
	if content := stringValue(attachment["content"]); content != "" {
		ret.TextPreview = truncateText(content, 240)
	}
	ret.FrameworkMetadata = mergeMetadataMap(claudeRecordMetadata(record), map[string]any{
		"attachment_type":  attachmentType,
		"has_image_signal": hasClaudeImageSignal(attachment),
	})
	return ret
}

func buildClaudeAttachmentEvent(record map[string]any, attachment minitrace.Attachment, ordinal int) minitrace.Event {
	event := minitrace.BuildEvent(fmt.Sprintf("claude-attachment-%06d", ordinal), optionalString(stringValue(record["timestamp"])), "attachment", "Claude Code attachment", summarizeClaudeAttachment(record, mapValue(record["attachment"])), record)
	event.Role = "system"
	event.AttachmentID = &attachment.ID
	event.FrameworkMetadata = attachment.FrameworkMetadata
	return event
}

func claudeRecordMetadata(record map[string]any) map[string]any {
	metadata := map[string]any{}
	if entrypoint := stringValue(record["entrypoint"]); entrypoint != "" {
		metadata["entrypoint"] = entrypoint
	}
	if agentID := stringValue(record["agentId"]); agentID != "" {
		metadata["agent_id"] = agentID
	}
	if sessionID := stringValue(record["sessionId"]); sessionID != "" {
		metadata["session_id"] = sessionID
	}
	if parentUUID := stringValue(record["parentUuid"]); parentUUID != "" {
		metadata["parent_uuid"] = parentUUID
	}
	if _, ok := record["isSidechain"]; ok {
		metadata["is_sidechain"] = boolValue(record["isSidechain"])
	}
	if attributionAgent := stringValue(record["attributionAgent"]); attributionAgent != "" {
		metadata["attribution_agent"] = attributionAgent
	}
	return metadata
}

func summarizeClaudeAttachment(record map[string]any, attachment map[string]any) string {
	if attachment == nil {
		return "Claude Code attachment with no structured payload."
	}
	parts := []string{}
	if uuid := stringValue(record["uuid"]); uuid != "" {
		parts = append(parts, "uuid="+uuid)
	}
	if parentUUID := stringValue(record["parentUuid"]); parentUUID != "" {
		parts = append(parts, "parent_uuid="+parentUUID)
	}
	if agentID := stringValue(record["agentId"]); agentID != "" {
		parts = append(parts, "agent_id="+agentID)
	}
	if attachmentType := stringValue(attachment["type"]); attachmentType != "" {
		parts = append(parts, "type="+attachmentType)
	}
	if fileName := stringValue(attachment["fileName"]); fileName != "" {
		parts = append(parts, "file="+fileName)
	}
	if mediaType := firstNonEmpty(stringValue(attachment["mediaType"]), stringValue(attachment["mimeType"])); mediaType != "" {
		parts = append(parts, "media="+mediaType)
	}
	if itemCount := minitrace.SafeInt(attachment["itemCount"], 0); itemCount != 0 {
		parts = append(parts, fmt.Sprintf("items=%d", itemCount))
	}
	if content := stringValue(attachment["content"]); content != "" {
		parts = append(parts, "content="+truncateText(content, 240))
	}
	if len(parts) == 0 {
		return "Claude Code attachment metadata preserved."
	}
	return strings.Join(parts, "; ")
}

func truncateText(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}

func hasClaudeImageSignal(attachment map[string]any) bool {
	if attachment == nil {
		return false
	}
	candidates := []string{
		stringValue(attachment["type"]),
		stringValue(attachment["fileName"]),
		stringValue(attachment["mediaType"]),
		stringValue(attachment["mimeType"]),
	}
	for _, value := range candidates {
		value = strings.ToLower(value)
		if strings.Contains(value, "image") || strings.HasSuffix(value, ".png") || strings.HasSuffix(value, ".jpg") || strings.HasSuffix(value, ".jpeg") || strings.HasSuffix(value, ".webp") {
			return true
		}
	}
	return false
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

func claudeTurnMetadata(record map[string]any, message map[string]any) any {
	metadata := map[string]any{}
	if entrypoint := stringValue(record["entrypoint"]); entrypoint != "" {
		metadata["entrypoint"] = entrypoint
	}
	if slug := stringValue(record["slug"]); slug != "" {
		metadata["slug"] = slug
	}
	if agentID := stringValue(record["agentId"]); agentID != "" {
		metadata["agent_id"] = agentID
	}
	if sessionID := stringValue(record["sessionId"]); sessionID != "" {
		metadata["session_id"] = sessionID
	}
	if parentUUID, ok := record["parentUuid"]; ok {
		metadata["parent_uuid"] = parentUUID
	}
	if isSidechain, ok := record["isSidechain"]; ok {
		metadata["is_sidechain"] = isSidechain
	}
	if message != nil {
		if stopReason, ok := message["stop_reason"]; ok {
			metadata["stop_reason"] = stopReason
		}
		if stopSequence, ok := message["stop_sequence"]; ok {
			metadata["stop_sequence"] = stopSequence
		}
		if usage := mapValue(message["usage"]); usage != nil {
			if cacheCreation, ok := usage["cache_creation"]; ok {
				metadata["cache_creation"] = cacheCreation
			}
		}
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func claudeToolMetadata(record map[string]any, block map[string]any) any {
	metadata := map[string]any{}
	if caller := mapValue(block["caller"]); caller != nil {
		metadata["caller"] = caller
	}
	if entrypoint := stringValue(record["entrypoint"]); entrypoint != "" {
		metadata["entrypoint"] = entrypoint
	}
	if slug := stringValue(record["slug"]); slug != "" {
		metadata["slug"] = slug
	}
	if agentID := stringValue(record["agentId"]); agentID != "" {
		metadata["agent_id"] = agentID
	}
	if sessionID := stringValue(record["sessionId"]); sessionID != "" {
		metadata["session_id"] = sessionID
	}
	if parentUUID, ok := record["parentUuid"]; ok {
		metadata["parent_uuid"] = parentUUID
	}
	if isSidechain, ok := record["isSidechain"]; ok {
		metadata["is_sidechain"] = isSidechain
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func claudeToolResultMetadata(record map[string]any) map[string]any {
	metadata := map[string]any{}
	if entrypoint := stringValue(record["entrypoint"]); entrypoint != "" {
		metadata["entrypoint"] = entrypoint
	}
	if slug := stringValue(record["slug"]); slug != "" {
		metadata["slug"] = slug
	}
	if agentID := stringValue(record["agentId"]); agentID != "" {
		metadata["agent_id"] = agentID
	}
	if sessionID := stringValue(record["sessionId"]); sessionID != "" {
		metadata["session_id"] = sessionID
	}
	if parentUUID, ok := record["parentUuid"]; ok {
		metadata["parent_uuid"] = parentUUID
	}
	if isSidechain, ok := record["isSidechain"]; ok {
		metadata["is_sidechain"] = isSidechain
	}
	return metadata
}

func mergeMetadataMap(existing any, fields map[string]any) any {
	if len(fields) == 0 {
		return existing
	}
	metadata := map[string]any{}
	if current, ok := existing.(map[string]any); ok {
		for key, value := range current {
			metadata[key] = value
		}
	}
	for key, value := range fields {
		metadata[key] = value
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
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

func appendUnique(values []string, additions ...string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range additions {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func ptr[T any](value T) *T {
	return &value
}
