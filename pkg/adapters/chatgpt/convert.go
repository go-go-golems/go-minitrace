package chatgpt

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
)

const (
	AdapterVersion  = "go-minitrace-chatgpt-adapter-dev"
	SourceFormat    = "chatgpt-export-v1"
	MaxDecompressed = 500 * 1024 * 1024
)

func StreamConversations(zipPath string, idFilter []string, fn func(map[string]any) error) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return errors.Wrap(err, "opening ChatGPT export ZIP")
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
		convID := firstNonEmpty(stringValue(conversation["conversation_id"]), stringValue(conversation["id"]))
		if matchesIDFilter(convID, idFilter) {
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
	convID := firstNonEmpty(stringValue(conv["conversation_id"]), stringValue(conv["id"]), "unknown")
	title := stringValue(conv["title"])
	mapping := mapValue(conv["mapping"])
	currentNode := stringValue(conv["current_node"])
	nodes := linearizeTree(mapping, currentNode)
	if len(nodes) == 0 {
		return nil, errors.Errorf("cannot linearize conversation %s", truncateID(convID, 12))
	}

	turns := make([]minitrace.Turn, 0, len(nodes))
	allTimestamps := []time.Time{}
	modelsSeen := []string{}
	modelSet := map[string]struct{}{}
	turnIndex := 0

	for _, node := range nodes {
		message := mapValue(node["message"])
		if message == nil {
			continue
		}

		role := stringValuePath(message, "author", "role")
		if role == "" {
			continue
		}
		if boolValuePath(message, "metadata", "is_visually_hidden_from_conversation") {
			continue
		}
		if role == "system" {
			if weight, ok := floatValue(message["weight"]); ok && weight == 0 {
				continue
			}
		}

		var timestampPtr *string
		if createdTime := message["create_time"]; createdTime != nil {
			if ts, ok := minitrace.SafeFromTimestamp(createdTime, false); ok {
				allTimestamps = append(allTimestamps, ts)
				formatted := minitrace.FormatTimestamp(ts)
				timestampPtr = &formatted
			}
		}

		text, imageCount, contentType := extractContentText(message)
		if text == "" && (contentType == "code" || contentType == "thoughts" || contentType == "reasoning_recap") {
			continue
		}

		modelSlug := getModelSlug(message)
		if modelSlug != "" {
			if _, ok := modelSet[modelSlug]; !ok {
				modelSet[modelSlug] = struct{}{}
				modelsSeen = append(modelsSeen, modelSlug)
			}
		}

		mtRole := "system"
		source := ptr("framework")
		var inputChannel *string
		switch role {
		case "system":
			mtRole = "system"
			source = ptr("system")
			inputChannel = ptr("system_prompt")
		case "user":
			mtRole = "user"
			source = ptr("human")
			inputChannel = ptr("user_input")
		case "assistant":
			mtRole = "assistant"
			source = ptr("model")
		case "tool":
			mtRole = "system"
			source = ptr("framework")
			inputChannel = ptr("tool_output")
		}

		turn := minitrace.BuildTurn(turnIndex, timestampPtr, mtRole, source, text)
		turn.InputChannel = inputChannel
		if modelSlug != "" {
			turn.Model = &modelSlug
		}
		if mappedContentType := mapContentType(contentType); mappedContentType != "" {
			turn.ContentType = &mappedContentType
		}
		if metadata := buildTurnFrameworkMetadata(message, contentType, modelSlug, imageCount); metadata != nil {
			turn.FrameworkMetadata = metadata
		}
		if role == "assistant" {
			turn.Streaming.WasStreamed = true
		}
		turns = append(turns, turn)
		turnIndex++
	}

	timing := minitrace.ComputeTiming(allTimestamps)
	quality := minitrace.AssignQualityTier(turns, nil)
	primaryModel := mostRecentAssistantModel(nodes)

	session := minitrace.BuildSessionSkeleton(convID, "chatgpt-web", SourceFormat, AdapterVersion)
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
	session.Quality = &quality
	session.Timing = timing
	session.Turns = turns
	session.ToolCalls = []minitrace.ToolCall{}
	session.Annotations = []minitrace.Annotation{}
	session.Metrics = minitrace.ComputeMetrics(turns, nil, timing, 0, nil)

	modelSwitches, uniqueModels := computeModelSwitches(nodes)
	session.Metrics.ModelSwitches = modelSwitches
	session.Metrics.UniqueModels = uniqueModels
	session.Flags.ContainsPII = true
	session.Flags.ForResearch = false
	session.Flags.NeedsCleaning = true
	session.Classification = "confidential"

	return &session, nil
}

func linearizeTree(mapping map[string]any, currentNode string) []map[string]any {
	if currentNode == "" {
		return nil
	}

	path := []map[string]any{}
	visited := map[string]struct{}{}
	nodeID := currentNode

	for nodeID != "" {
		if _, ok := visited[nodeID]; ok {
			break
		}
		visited[nodeID] = struct{}{}
		node := mapValue(mapping[nodeID])
		if node == nil {
			break
		}
		path = append(path, node)
		nodeID = stringValue(node["parent"])
	}

	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

func extractContentText(message map[string]any) (string, int, string) {
	content := mapValue(message["content"])
	if content == nil {
		return "", 0, ""
	}
	contentType := stringValue(content["content_type"])
	parts := listValue(content["parts"])
	textParts := []string{}
	imageCount := 0

	for _, part := range parts {
		switch typed := part.(type) {
		case string:
			textParts = append(textParts, typed)
		case map[string]any:
			partType := stringValue(typed["content_type"])
			switch partType {
			case "image_asset_pointer":
				imageCount++
				asset := stringValue(typed["asset_pointer"])
				textParts = append(textParts, fmt.Sprintf("[image: %s]", asset))
			default:
				if text := stringValue(typed["text"]); text != "" {
					textParts = append(textParts, text)
				}
			}
		}
	}

	return joinNonEmpty(textParts, "\n"), imageCount, contentType
}

func getModelSlug(message map[string]any) string {
	return stringValuePath(message, "metadata", "model_slug")
}

func mapContentType(contentType string) string {
	switch contentType {
	case "text":
		return "text"
	case "multimodal_text":
		return "multimodal_text"
	case "code":
		return "code"
	case "thoughts", "reasoning_recap":
		return "reasoning"
	default:
		return ""
	}
}

func buildTurnFrameworkMetadata(message map[string]any, contentType, modelSlug string, imageCount int) map[string]any {
	metadata := map[string]any{}
	if contentType != "" && contentType != "text" {
		metadata["content_type"] = contentType
	}
	if modelSlug != "" {
		metadata["model_slug"] = modelSlug
	}
	if finishDetails := anyValuePath(message, "metadata", "finish_details"); finishDetails != nil {
		metadata["finish_details"] = finishDetails
	}
	if imageCount > 0 {
		metadata["image_count"] = imageCount
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func mostRecentAssistantModel(nodes []map[string]any) string {
	for i := len(nodes) - 1; i >= 0; i-- {
		message := mapValue(nodes[i]["message"])
		if message == nil {
			continue
		}
		if stringValuePath(message, "author", "role") != "assistant" {
			continue
		}
		if modelSlug := getModelSlug(message); modelSlug != "" {
			return modelSlug
		}
	}
	return ""
}

func computeModelSwitches(nodes []map[string]any) (*int, *int) {
	models := []string{}
	for _, node := range nodes {
		message := mapValue(node["message"])
		if message == nil || stringValuePath(message, "author", "role") != "assistant" {
			continue
		}
		if modelSlug := getModelSlug(message); modelSlug != "" {
			models = append(models, modelSlug)
		}
	}
	if len(models) < 2 {
		return nil, nil
	}

	unique := map[string]struct{}{}
	switches := 0
	for i, model := range models {
		unique[model] = struct{}{}
		if i > 0 && models[i-1] != model {
			switches++
		}
	}
	uniqueCount := len(unique)
	return &switches, &uniqueCount
}

func matchesIDFilter(convID string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, filter := range filters {
		if strings.HasPrefix(convID, filter) {
			return true
		}
	}
	return false
}

func CountRealMessages(conv map[string]any) int {
	mapping := mapValue(conv["mapping"])
	count := 0
	for _, rawNode := range mapping {
		node := mapValue(rawNode)
		if node == nil {
			continue
		}
		role := stringValuePath(mapValue(node["message"]), "author", "role")
		if role == "user" || role == "assistant" {
			count++
		}
	}
	return count
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
	default:
		return nil
	}
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

func stringValuePath(root map[string]any, keys ...string) string {
	value := anyValuePath(root, keys...)
	return stringValue(value)
}

func boolValuePath(root map[string]any, keys ...string) bool {
	value := anyValuePath(root, keys...)
	if typed, ok := value.(bool); ok {
		return typed
	}
	return false
}

func anyValuePath(root map[string]any, keys ...string) any {
	current := any(root)
	for _, key := range keys {
		next, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = next[key]
	}
	return current
}

func floatValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	default:
		return 0, false
	}
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

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func truncateID(value string, n int) string {
	if n <= 0 || len(value) <= n {
		return value
	}
	return value[:n]
}

func ptr[T any](value T) *T {
	return &value
}
