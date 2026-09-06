package codex

import (
	"strconv"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

// A native turn can contain many messages. Keep native identity separate from
// the message index that minitrace assigns only after reconciliation.
type codexMessage struct {
	role            string
	content         string
	contentIdentity string
	id              string
	turnID          string
	timestamp       *string
	representation  string
	metadata        map[string]any
	sources         []map[string]any
	representations map[string]bool
}

func codexNativeTurnID(payload map[string]any, current string) string {
	passthrough := mapValue(payload["internal_chat_message_metadata_passthrough"])
	return firstNonEmpty(stringValue(payload["turn_id"]), stringValue(passthrough["turn_id"]), current)
}

// collectCodexMessages reconciles representations, never text from quoted
// histories. Its keys are source record indexes so the main parser can retain
// reasoning/usage chronology while emitting each logical message once.
func collectCodexMessages(records []map[string]any) map[int]*codexMessage {
	out := map[int]*codexMessage{}
	byID := map[string]*codexMessage{}
	currentTurn := ""
	var previous *codexMessage
	previousLine := -2
	for index, record := range records {
		payload := mapValue(record["payload"])
		kind := stringValue(record["type"])
		if kind == "turn_context" || (kind == "event_msg" && stringValue(payload["type"]) == "task_started") {
			currentTurn = codexNativeTurnID(payload, currentTurn)
		}
		message := decodeCodexMessage(record, index, currentTurn)
		if message == nil {
			continue
		}
		key := codexMessageKey(message.role, message.turnID, message.id)
		var existing *codexMessage
		if message.id != "" {
			existing = byID[key]
		}
		// A different-ID mirror is permitted only for adjacent, complementary
		// representations with an established shared native turn. Matching text
		// alone never deduplicates two genuine messages.
		if existing == nil && previous != nil && previousLine == index-1 &&
			message.turnID != "" && previous.turnID == message.turnID &&
			previous.role == message.role && previous.contentIdentity == message.contentIdentity &&
			!previous.representations[message.representation] &&
			(previous.representations["response_item/message"] || message.representation == "response_item/message") {
			existing = previous
		}
		if existing != nil {
			if existing.contentIdentity != message.contentIdentity {
				existing.metadata["fidelity_diagnostics"] = []string{"conflicting_message_content"}
			}
			existing.sources = append(existing.sources, message.sources...)
			existing.representations[message.representation] = true
			if message.representation == "response_item/message" && existing.representation != "response_item/message" {
				existing.content = message.content
				existing.contentIdentity = message.contentIdentity
				existing.id = message.id
				existing.timestamp = message.timestamp
				existing.representation = message.representation
			}
			for k, v := range message.metadata {
				if _, found := existing.metadata[k]; !found {
					existing.metadata[k] = v
				}
			}
			if message.id != "" {
				byID[key] = existing
			}
			previous = existing
		} else {
			out[index] = message
			if message.id != "" {
				byID[key] = message
			}
			previous = message
		}
		previousLine = index
	}
	return out
}

func codexMessageKey(role, turnID, id string) string {
	return role + "\x00" + turnID + "\x00" + id
}

func decodeCodexMessage(record map[string]any, index int, currentTurn string) *codexMessage {
	payload := mapValue(record["payload"])
	kind := stringValue(record["type"])
	payloadType := stringValue(payload["type"])
	body := payload
	role := ""
	representation := kind + "/" + payloadType
	content := ""
	switch {
	case kind == "response_item" && payloadType == "message":
		role = stringValue(payload["role"])
	case kind == "event_msg" && payloadType == "item_completed":
		body = mapValue(payload["item"])
		switch stringValue(body["type"]) {
		case "UserMessage":
			role = "user"
		case "AgentMessage":
			role = "assistant"
		default:
			return nil
		}
		representation += "/" + stringValue(body["type"])
	case kind == "event_msg" && payloadType == "user_message":
		role = "user"
		content = stringValue(payload["message"])
	case kind == "event_msg" && payloadType == "agent_message":
		role = "assistant"
		content = stringValue(payload["message"])
	default:
		return nil
	}
	nativeRole := role
	if role == "developer" {
		role = "system"
	}
	if role != "user" && role != "assistant" && role != "system" {
		return nil
	}
	turnID := codexNativeTurnID(payload, currentTurn)
	metadata := codexTurnMetadata(turnID, body, nil)
	if metadata == nil {
		metadata = map[string]any{}
	}
	contentIdentity := content
	if body["content"] != nil {
		var kinds []string
		content, kinds, contentIdentity = codexMessageText(body["content"])
		metadata["content_block_kinds"] = kinds
		for _, k := range kinds {
			if strings.Contains(strings.ToLower(k), "image") {
				metadata["has_image_signal"] = true
			}
		}
	}
	if nativeRole != role {
		metadata["native_role"] = nativeRole
	}
	source := map[string]any{"source_line": index + 1, "event_type": representation}
	id := stringValue(body["id"])
	if id != "" {
		source["native_message_id"] = id
	}
	if ordinal, ok := record["ordinal"]; ok {
		source["source_ordinal"] = ordinal
	}
	return &codexMessage{
		role: role, content: content, contentIdentity: contentIdentity, id: id, turnID: turnID,
		timestamp:      optionalString(stringValue(record["timestamp"])),
		representation: representation, metadata: metadata, sources: []map[string]any{source},
		representations: map[string]bool{representation: true},
	}
}

func codexMessageText(content any) (string, []string, string) {
	if text, ok := content.(string); ok {
		return text, []string{"text"}, text
	}
	var texts, kinds []string
	for _, value := range listValue(content) {
		block := mapValue(value)
		kind := stringValue(block["type"])
		kinds = append(kinds, kind)
		switch strings.ToLower(kind) {
		case "text", "input_text", "output_text", "summary_text":
			if text, ok := block["text"].(string); ok {
				texts = append(texts, text)
			}
		}
	}
	// Legacy event messages concatenate content blocks without separators.
	// Compare this exact native text, not whitespace-normalized prose, when
	// reconciling mirrors; the display retains visible block boundaries.
	return strings.Join(texts, "\n"), kinds, strings.Join(texts, "")
}

func (message *codexMessage) buildTurn(index int, model string, thinking []string) minitrace.Turn {
	source := "model"
	switch message.role {
	case "user":
		source = "human"
	case "system":
		source = "system"
	}
	turn := minitrace.BuildTurn(index, message.timestamp, message.role, &source, message.content)
	if message.role == "user" {
		turn.InputChannel = ptr("user_input")
	}
	if message.role == "assistant" {
		turn.Model = optionalString(model)
		for k, v := range attachCodexThinking(&turn, thinking) {
			message.metadata[k] = v
		}
	}
	message.metadata["native_message_id"] = message.id
	message.metadata["message_sources"] = message.sources
	turn.FrameworkMetadata = message.metadata
	return turn
}

// linkCodexMessageCalls accepts only explicit native message identity. Native
// turn membership alone cannot prove whether commentary or a final answer
// emitted a tool, and neither temporal proximity nor JS text is sufficient.
func linkCodexMessageCalls(turns []minitrace.Turn, calls []minitrace.ToolCall) {
	messages := map[string]int{}
	for index := range turns {
		turn := &turns[index]
		if turn.Role != "assistant" {
			continue
		}
		metadata := mapValue(turn.FrameworkMetadata)
		turnID := stringValue(metadata["turn_id"])
		for _, source := range metadata["message_sources"].([]map[string]any) {
			if id := stringValue(source["native_message_id"]); id != "" {
				messages[codexMessageKey("assistant", turnID, id)] = index
			}
		}
	}
	for index := range calls {
		call := &calls[index]
		metadata := mapValue(call.FrameworkMetadata)
		messageID := stringValue(metadata["emitting_message_id"])
		key := codexMessageKey("assistant", stringValue(metadata["turn_id"]), messageID)
		if turnIndex, ok := messages[key]; messageID != "" && ok {
			call.EmittingTurnIndex = ptr(turnIndex)
			turns[turnIndex].ToolCallsInTurn = append(turns[turnIndex].ToolCallsInTurn, call.ID)
			metadata["turn_association"] = "explicit_message_id"
		} else {
			call.EmittingTurnIndex = nil
			metadata["turn_association"] = "unknown"
			if messageID != "" {
				metadata["fidelity_diagnostics"] = []string{"unresolved_emitting_message_id"}
			}
		}
		call.FrameworkMetadata = metadata
	}
}

func codexCallSourceMetadata(payload map[string]any, index int) map[string]any {
	metadata := map[string]any{
		"source_line":      index + 1,
		"source_reference": "line:" + strconv.Itoa(index+1),
	}
	passthrough := mapValue(payload["internal_chat_message_metadata_passthrough"])
	if id := firstNonEmpty(stringValue(payload["message_id"]), stringValue(passthrough["message_id"])); id != "" {
		metadata["emitting_message_id"] = id
	}
	return metadata
}
