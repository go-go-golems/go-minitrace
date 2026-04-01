package chatgpt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
)

const TranscriptSourceFormat = "chatgpt-json-transcript-v1"

type exchangeStep struct {
	kind       string
	exchangeID string
	nodes      []map[string]any
}

func StreamTranscriptFiles(sourceDir string, idFilter []string, fn func(map[string]any, string) error) error {
	root, err := expandHomePath(sourceDir)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return errors.Wrap(err, "reading ChatGPT transcript export directory")
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		paths = append(paths, filepath.Join(root, entry.Name()))
	}
	sort.Strings(paths)

	for _, path := range paths {
		payload, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "reading ChatGPT transcript %s", path)
		}

		var conversation map[string]any
		if err := json.Unmarshal(payload, &conversation); err != nil {
			return errors.Wrapf(err, "decoding ChatGPT transcript %s", path)
		}

		convID := firstNonEmpty(stringValue(conversation["conversation_id"]), stringValue(conversation["id"]))
		if !matchesIDFilter(convID, idFilter) {
			continue
		}
		if err := fn(conversation, path); err != nil {
			return err
		}
	}

	return nil
}

func ConvertTranscriptConversation(conv map[string]any, sourcePath string) (*minitrace.Session, error) {
	convID := firstNonEmpty(stringValue(conv["conversation_id"]), stringValue(conv["id"]), "unknown")
	title := stringValue(conv["title"])
	mapping := mapValue(conv["mapping"])
	currentNode := stringValue(conv["current_node"])
	nodes := linearizeTree(mapping, currentNode)
	if len(nodes) == 0 {
		return nil, errors.Errorf("cannot linearize transcript conversation %s", truncateID(convID, 12))
	}

	steps := groupNodesIntoSteps(nodes)
	turns := make([]minitrace.Turn, 0, len(steps))
	toolCalls := make([]minitrace.ToolCall, 0)
	annotations := make([]minitrace.Annotation, 0)
	allTimestamps := []time.Time{}

	for _, step := range steps {
		switch step.kind {
		case "user":
			turn := buildTranscriptUserTurn(step, &allTimestamps)
			if turn != nil {
				turn.Index = len(turns)
				turns = append(turns, *turn)
			}
		case "assistant":
			assistantTurn, assistantToolCalls, assistantAnnotations := buildTranscriptAssistantStep(step, len(turns), &allTimestamps)
			toolCalls = append(toolCalls, assistantToolCalls...)
			annotations = append(annotations, assistantAnnotations...)
			if assistantTurn != nil {
				assistantTurn.Index = len(turns)
				if len(assistantToolCalls) > 0 {
					assistantTurn.ToolCallsInTurn = make([]string, 0, len(assistantToolCalls))
					for _, toolCall := range assistantToolCalls {
						assistantTurn.ToolCallsInTurn = append(assistantTurn.ToolCallsInTurn, toolCall.ID)
					}
				}
				turns = append(turns, *assistantTurn)
				lastIndex := len(turns) - 1
				for i := range assistantToolCalls {
					toolCalls[len(toolCalls)-len(assistantToolCalls)+i].EmittingTurnIndex = &lastIndex
				}
			}
		case "system":
			turn := buildTranscriptSystemTurn(step, &allTimestamps)
			if turn != nil {
				turn.Index = len(turns)
				turns = append(turns, *turn)
			}
		}
	}

	timing := minitrace.ComputeTiming(allTimestamps)
	minitrace.ComputeToolCallContext(toolCalls)
	quality := minitrace.AssignQualityTier(turns, toolCalls)
	primaryModel := transcriptPrimaryModel(nodes)

	session := minitrace.BuildSessionSkeleton(convID, "chatgpt-web", TranscriptSourceFormat, AdapterVersion)
	session.Environment.Model = optionalString(primaryModel)
	session.Environment.PlatformType = ptr("web")
	session.Environment.ProviderHint = ptr("openai")
	session.Provenance.OriginalSessionID = ptr(convID)
	if sourcePath != "" {
		session.Provenance.SourcePath = ptr(sourcePath)
	}
	if title != "" {
		session.Title = &title
	}
	session.Environment.ToolsEnabled = uniqueTranscriptToolNames(toolCalls)
	session.Quality = &quality
	session.Timing = timing
	session.Turns = turns
	session.ToolCalls = toolCalls
	session.Annotations = annotations
	session.Metrics = minitrace.ComputeMetrics(turns, toolCalls, timing, 0, nil)

	modelSwitches, uniqueModels := computeModelSwitches(nodes)
	session.Metrics.ModelSwitches = modelSwitches
	session.Metrics.UniqueModels = uniqueModels
	session.Flags.ContainsPII = true
	session.Flags.ForResearch = false
	session.Flags.NeedsCleaning = true
	session.Classification = "confidential"

	return &session, nil
}

func groupNodesIntoSteps(nodes []map[string]any) []exchangeStep {
	steps := []exchangeStep{}
	var current *exchangeStep

	flush := func() {
		if current == nil || len(current.nodes) == 0 {
			return
		}
		steps = append(steps, *current)
		current = nil
	}

	for _, node := range nodes {
		message := mapValue(node["message"])
		if message == nil {
			continue
		}

		role := stringValuePath(message, "author", "role")
		if role == "" {
			continue
		}
		if shouldSkipTranscriptNode(message) {
			continue
		}

		exchangeID := stringValuePath(message, "metadata", "turn_exchange_id")
		switch role {
		case "user":
			flush()
			steps = append(steps, exchangeStep{kind: "user", exchangeID: exchangeID, nodes: []map[string]any{node}})
		case "assistant", "tool":
			if current != nil && current.kind == "assistant" && sameTranscriptStep(current.exchangeID, exchangeID) {
				current.nodes = append(current.nodes, node)
			} else {
				flush()
				current = &exchangeStep{kind: "assistant", exchangeID: exchangeID, nodes: []map[string]any{node}}
			}
		case "system":
			if exchangeID != "" {
				if current != nil && current.kind == "assistant" && sameTranscriptStep(current.exchangeID, exchangeID) {
					current.nodes = append(current.nodes, node)
				} else {
					flush()
					current = &exchangeStep{kind: "assistant", exchangeID: exchangeID, nodes: []map[string]any{node}}
				}
			} else {
				flush()
				steps = append(steps, exchangeStep{kind: "system", exchangeID: "", nodes: []map[string]any{node}})
			}
		default:
			flush()
			steps = append(steps, exchangeStep{kind: "system", exchangeID: exchangeID, nodes: []map[string]any{node}})
		}
	}

	flush()
	return steps
}

func buildTranscriptUserTurn(step exchangeStep, allTimestamps *[]time.Time) *minitrace.Turn {
	for _, node := range step.nodes {
		message := mapValue(node["message"])
		if message == nil {
			continue
		}
		timestampPtr, _ := recordTranscriptTimestamp(message, allTimestamps)
		modelSlug := getModelSlug(message)
		text := transcriptVisibleText(message)
		if strings.TrimSpace(text) == "" {
			continue
		}
		turn := minitrace.BuildTurn(0, timestampPtr, "user", ptr("human"), text)
		turn.InputChannel = ptr("user_input")
		if modelSlug != "" {
			turn.Model = &modelSlug
		}
		if metadata := buildTranscriptFrameworkMetadata(message, "text", modelSlug, 0); metadata != nil {
			turn.FrameworkMetadata = metadata
		}
		return &turn
	}
	return nil
}

func buildTranscriptSystemTurn(step exchangeStep, allTimestamps *[]time.Time) *minitrace.Turn {
	for _, node := range step.nodes {
		message := mapValue(node["message"])
		if message == nil {
			continue
		}
		text := transcriptVisibleText(message)
		if strings.TrimSpace(text) == "" {
			continue
		}
		timestampPtr, _ := recordTranscriptTimestamp(message, allTimestamps)
		modelSlug := getModelSlug(message)
		turn := minitrace.BuildTurn(0, timestampPtr, "system", ptr("system"), text)
		turn.InputChannel = ptr("system_prompt")
		if modelSlug != "" {
			turn.Model = &modelSlug
		}
		if metadata := buildTranscriptFrameworkMetadata(message, transcriptContentType(message), modelSlug, 0); metadata != nil {
			turn.FrameworkMetadata = metadata
		}
		return &turn
	}
	return nil
}

func buildTranscriptAssistantStep(step exchangeStep, emittingTurnIndex int, allTimestamps *[]time.Time) (*minitrace.Turn, []minitrace.ToolCall, []minitrace.Annotation) {
	textParts := []string{}
	thinkingParts := []string{}
	toolCalls := []minitrace.ToolCall{}
	annotations := []minitrace.Annotation{}
	toolNodes := []map[string]any{}
	codeNodes := []map[string]any{}
	lastTimestamp := (*string)(nil)
	lastModel := ""
	contentTypes := []string{}

	for _, node := range step.nodes {
		message := mapValue(node["message"])
		if message == nil {
			continue
		}
		timestampPtr, _ := recordTranscriptTimestamp(message, allTimestamps)
		if timestampPtr != nil {
			lastTimestamp = timestampPtr
		}

		modelSlug := getModelSlug(message)
		if modelSlug != "" {
			lastModel = modelSlug
		}

		role := stringValuePath(message, "author", "role")
		contentType := transcriptContentType(message)
		if contentType != "" {
			contentTypes = append(contentTypes, contentType)
		}

		switch role {
		case "assistant":
			switch contentType {
			case "thoughts", "reasoning_recap":
				if text := transcriptArtifactText(message); strings.TrimSpace(text) != "" {
					thinkingParts = append(thinkingParts, text)
				}
			case "code":
				codeNodes = append(codeNodes, message)
			default:
				if text := transcriptVisibleText(message); strings.TrimSpace(text) != "" {
					textParts = append(textParts, text)
				}
			}
		case "tool":
			toolNodes = append(toolNodes, message)
		case "system":
			if text := transcriptVisibleText(message); strings.TrimSpace(text) != "" {
				annotations = append(annotations, minitrace.BuildAnnotation(
					fmt.Sprintf("ann-chatgpt-system-%d", len(annotations)),
					"adapter",
					"session",
					"session",
					"observation",
					"Embedded ChatGPT system node preserved as annotation",
					text,
					[]string{"chatgpt", "system", "transcript"},
					nil,
				))
			}
		}
	}

	if len(codeNodes) > 0 || len(toolNodes) > 0 {
		extracted, toolAnnotations := buildTranscriptToolCalls(step.exchangeID, codeNodes, toolNodes, lastTimestamp, emittingTurnIndex)
		toolCalls = append(toolCalls, extracted...)
		annotations = append(annotations, toolAnnotations...)
	}

	if len(textParts) == 0 && len(thinkingParts) == 0 && len(toolCalls) == 0 {
		return nil, toolCalls, annotations
	}

	content := joinNonEmpty(textParts, "\n\n")
	turn := minitrace.BuildTurn(0, lastTimestamp, "assistant", ptr("model"), content)
	if lastModel != "" {
		turn.Model = &lastModel
	}
	if len(textParts) == 0 && len(thinkingParts) > 0 {
		turn.ContentType = ptr("reasoning")
	} else if len(contentTypes) > 0 {
		mapped := mapContentType(contentTypes[len(contentTypes)-1])
		if mapped != "" {
			turn.ContentType = &mapped
		}
	}
	if len(thinkingParts) > 0 {
		thinking := joinNonEmpty(thinkingParts, "\n\n")
		if thinking != "" {
			turn.Thinking = &thinking
		}
	}
	turn.Streaming.WasStreamed = true
	turn.FrameworkMetadata = map[string]any{
		"step_exchange_id": step.exchangeID,
		"content_types":    uniqueStrings(contentTypes),
		"tool_call_count":  len(toolCalls),
	}
	return &turn, toolCalls, annotations
}

func buildTranscriptToolCalls(exchangeID string, codeNodes, toolNodes []map[string]any, timestamp *string, emittingTurnIndex int) ([]minitrace.ToolCall, []minitrace.Annotation) {
	toolCalls := []minitrace.ToolCall{}
	annotations := []minitrace.Annotation{}
	usedToolNodes := map[int]struct{}{}

	for codeIndex, codeNode := range codeNodes {
		toolName := firstNonEmpty(stringValue(codeNode["recipient"]), stringValuePath(codeNode, "author", "name"))
		argumentsText := transcriptCodeText(codeNode)
		arguments, rawArguments := parseTranscriptArguments(argumentsText)

		matchIndexes := []int{}
		for idx, toolNode := range toolNodes {
			if _, ok := usedToolNodes[idx]; ok {
				continue
			}
			if toolName == "" || toolName == stringValuePath(toolNode, "author", "name") {
				matchIndexes = append(matchIndexes, idx)
			}
		}

		if toolName == "" && len(matchIndexes) > 0 {
			toolName = stringValuePath(toolNodes[matchIndexes[0]], "author", "name")
		}
		if toolName == "" {
			toolName = fmt.Sprintf("unknown-%d", codeIndex)
		}

		resultPayloads := make([]map[string]any, 0, len(matchIndexes))
		success := true
		for _, idx := range matchIndexes {
			usedToolNodes[idx] = struct{}{}
			toolNode := toolNodes[idx]
			payload := map[string]any{
				"content":  transcriptVisibleText(toolNode),
				"metadata": mapValue(toolNode["metadata"]),
			}
			if role := stringValuePath(toolNode, "author", "name"); role != "" {
				payload["tool_name"] = role
			}
			resultPayloads = append(resultPayloads, payload)
			if errText := transcriptToolError(toolNode); errText != "" {
				success = false
			}
		}

		var result any
		if len(resultPayloads) > 0 {
			result = resultPayloads
		}

		outputError := transcriptResultError(resultPayloads)
		operationType := classifyTranscriptOperation(toolName)
		contentOrigin := classifyTranscriptContentOrigin(toolName)
		turnIndex := emittingTurnIndex
		toolCallID := fmt.Sprintf("chatgpt-tool-%d", codeIndex)
		if strings.TrimSpace(exchangeID) != "" {
			toolCallID = fmt.Sprintf("chatgpt-tool-%s-%d", exchangeID, codeIndex)
		}
		toolCall := minitrace.BuildToolCall(
			toolCallID,
			&turnIndex,
			timestamp,
			toolName,
			operationType,
			nil,
			nil,
			arguments,
			success,
			result,
			optionalString(outputError),
			nil,
			map[string]any{
				"code_message":       codeNode,
				"tool_node_count":    len(matchIndexes),
				"raw_arguments_text": rawArguments,
			},
			nil,
			contentOrigin,
			nil,
		)
		toolCalls = append(toolCalls, toolCall)
	}

	for idx, toolNode := range toolNodes {
		if _, ok := usedToolNodes[idx]; ok {
			continue
		}
		metadata, _ := json.Marshal(mapValue(toolNode["metadata"]))
		annotations = append(annotations, minitrace.BuildAnnotation(
			fmt.Sprintf("ann-chatgpt-orphan-tool-%d", len(annotations)),
			"adapter",
			"session",
			"session",
			"observation",
			"Orphan ChatGPT transcript tool node",
			string(metadata),
			[]string{"chatgpt", "tool", "orphan"},
			nil,
		))
	}

	return toolCalls, annotations
}

func shouldSkipTranscriptNode(message map[string]any) bool {
	role := stringValuePath(message, "author", "role")
	if role == "system" && boolValuePath(message, "metadata", "is_visually_hidden_from_conversation") {
		return true
	}
	if role == "system" {
		if weight, ok := floatValue(message["weight"]); ok && weight == 0 {
			if strings.TrimSpace(transcriptVisibleText(message)) == "" {
				return true
			}
		}
	}
	return false
}

func sameTranscriptStep(left, right string) bool {
	if left == "" || right == "" {
		return left == right
	}
	return left == right
}

func transcriptVisibleText(message map[string]any) string {
	content := mapValue(message["content"])
	if content == nil {
		return ""
	}
	contentType := stringValue(content["content_type"])
	switch contentType {
	case "code":
		return stringValue(content["text"])
	case "reasoning_recap":
		if recap := stringValue(content["content"]); recap != "" {
			return recap
		}
	case "thoughts":
		if summary := stringValue(content["summary"]); summary != "" {
			return summary
		}
	case "tether_browsing_display":
		return stringValue(content["result"])
	}

	text, _, _ := extractContentText(message)
	if text != "" {
		return text
	}

	if parts := listValue(content["parts"]); len(parts) > 0 {
		ret := []string{}
		for _, part := range parts {
			switch typed := part.(type) {
			case string:
				ret = append(ret, typed)
			case map[string]any:
				if value := stringValue(typed["text"]); value != "" {
					ret = append(ret, value)
				}
				if value := stringValue(typed["content"]); value != "" {
					ret = append(ret, value)
				}
			}
		}
		return joinNonEmpty(ret, "\n")
	}

	return firstNonEmpty(
		stringValue(content["text"]),
		stringValue(content["content"]),
		stringValue(content["result"]),
	)
}

func transcriptArtifactText(message map[string]any) string {
	content := mapValue(message["content"])
	if content == nil {
		return ""
	}
	return firstNonEmpty(
		stringValue(content["text"]),
		stringValue(content["content"]),
		stringValue(content["summary"]),
		transcriptVisibleText(message),
	)
}

func transcriptCodeText(message map[string]any) string {
	content := mapValue(message["content"])
	if content == nil {
		return ""
	}
	return firstNonEmpty(stringValue(content["text"]), transcriptVisibleText(message))
}

func transcriptContentType(message map[string]any) string {
	return stringValuePath(message, "content", "content_type")
}

func buildTranscriptFrameworkMetadata(message map[string]any, contentType, modelSlug string, imageCount int) map[string]any {
	metadata := buildTurnFrameworkMetadata(message, contentType, modelSlug, imageCount)
	msgMetadata := mapValue(message["metadata"])
	if msgMetadata != nil {
		if metadata == nil {
			metadata = map[string]any{}
		}
		for _, key := range []string{
			"turn_exchange_id",
			"thinking_effort",
			"reasoning_status",
			"reasoning_title",
			"request_id",
			"resolved_model_slug",
			"default_model_slug",
		} {
			if value, ok := msgMetadata[key]; ok && value != nil && value != "" {
				metadata[key] = value
			}
		}
	}
	return metadata
}

func parseTranscriptArguments(raw string) (any, string) {
	if strings.TrimSpace(raw) == "" {
		return nil, ""
	}
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
		return parsed, raw
	}
	return map[string]any{"raw": raw}, raw
}

func classifyTranscriptOperation(toolName string) string {
	switch strings.ToLower(toolName) {
	case "web.run", "file_search":
		return "READ"
	default:
		return "OTHER"
	}
}

func classifyTranscriptContentOrigin(toolName string) *string {
	switch strings.ToLower(toolName) {
	case "web.run":
		return ptr("web")
	case "file_search":
		return ptr("database")
	default:
		return nil
	}
}

func transcriptToolError(message map[string]any) string {
	metadata := mapValue(message["metadata"])
	if metadata == nil {
		return ""
	}
	for _, key := range []string{"error", "error_message", "failure_reason"} {
		if value := stringValue(metadata[key]); value != "" {
			return value
		}
	}
	return ""
}

func transcriptResultError(resultPayloads []map[string]any) string {
	for _, payload := range resultPayloads {
		if metadata, ok := payload["metadata"].(map[string]any); ok {
			for _, key := range []string{"error", "error_message", "failure_reason"} {
				if value := stringValue(metadata[key]); value != "" {
					return value
				}
			}
		}
	}
	return ""
}

func transcriptPrimaryModel(nodes []map[string]any) string {
	if slug := mostRecentAssistantModel(nodes); slug != "" {
		return slug
	}
	for i := len(nodes) - 1; i >= 0; i-- {
		message := mapValue(nodes[i]["message"])
		if message == nil {
			continue
		}
		if modelSlug := getModelSlug(message); modelSlug != "" {
			return modelSlug
		}
	}
	return ""
}

func recordTranscriptTimestamp(message map[string]any, allTimestamps *[]time.Time) (*string, time.Time) {
	if ts, ok := minitrace.SafeFromTimestamp(message["create_time"], false); ok {
		*allTimestamps = append(*allTimestamps, ts)
		formatted := minitrace.FormatTimestamp(ts)
		return &formatted, ts
	}
	return nil, time.Time{}
}

func uniqueTranscriptToolNames(toolCalls []minitrace.ToolCall) []string {
	names := make([]string, 0, len(toolCalls))
	seen := map[string]struct{}{}
	for _, toolCall := range toolCalls {
		if _, ok := seen[toolCall.ToolName]; ok {
			continue
		}
		seen[toolCall.ToolName] = struct{}{}
		names = append(names, toolCall.ToolName)
	}
	sort.Strings(names)
	return names
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	ret := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ret = append(ret, value)
	}
	return ret
}

func expandHomePath(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return filepath.Clean(path), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
}
