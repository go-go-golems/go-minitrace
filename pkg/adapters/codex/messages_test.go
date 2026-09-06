package codex

import (
	"reflect"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestPaginatedFidelityMessages(t *testing.T) {
	records, err := parseJSONLFile("testdata/paginated-fidelity.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	session := convertMessageTestRecords(t, records)
	if len(session.Turns) != 5 {
		t.Fatalf("want five reconciled messages, got %d", len(session.Turns))
	}
	want := []string{"continue", "Inspecting.\nThen testing.", "A child failed; some outcomes remain unknown.", "continue"}
	for i, text := range want {
		if session.Turns[i].Content != text {
			t.Errorf("turn %d: got %q, want %q", i, session.Turns[i].Content, text)
		}
	}
	for _, call := range session.ToolCalls {
		if call.EmittingTurnIndex != nil {
			t.Errorf("call %s has no explicit native message linkage, got %v", call.ID, *call.EmittingTurnIndex)
		}
	}
	metadata := mapValue(session.Turns[1].FrameworkMetadata)
	if metadata["native_message_id"] != "message-plan" || metadata["turn_id"] != "turn-1" {
		t.Fatalf("lost native message identity: %#v", metadata)
	}
	if len(metadata["message_sources"].([]map[string]any)) != 2 {
		t.Fatalf("want both mirror provenance records: %#v", metadata)
	}
	again := convertMessageTestRecords(t, records)
	if !reflect.DeepEqual(session.Turns, again.Turns) {
		t.Fatal("repeated conversion changed message identities/order/content")
	}
}

func TestCodexMessageIdentityRules(t *testing.T) {
	tests := []struct {
		name    string
		records []map[string]any
		want    int
	}{
		{"different ids same text same representation", []map[string]any{responseMessage("a", "turn", "user", "again"), responseMessage("b", "turn", "user", "again")}, 2},
		{"missing ids stay distinct", []map[string]any{responseMessage("", "turn", "user", "again"), responseMessage("", "turn", "user", "again")}, 2},
		{"same id repeated", []map[string]any{responseMessage("a", "turn", "assistant", "again"), responseMessage("a", "turn", "assistant", "again")}, 1},
		{"same id different turns", []map[string]any{responseMessage("a", "one", "user", "again"), responseMessage("a", "two", "user", "again")}, 2},
		{"adjacent different id mirrors", []map[string]any{responseMessage("a", "turn", "user", "again"), completedMessage("b", "turn", "UserMessage", "again")}, 1},
		{"adjacent mirrors reverse order", []map[string]any{completedMessage("a", "turn", "UserMessage", "again"), responseMessage("b", "turn", "user", "again")}, 1},
		{"no turn identity no heuristic merge", []map[string]any{responseMessage("a", "", "user", "again"), completedMessage("b", "", "UserMessage", "again")}, 2},
		{"nonadjacent different ids stay distinct", []map[string]any{responseMessage("a", "turn", "user", "again"), {"type": "world_state"}, completedMessage("b", "turn", "UserMessage", "again")}, 2},
		{"mirror followed by genuine repetition", []map[string]any{responseMessage("a", "turn", "user", "again"), completedMessage("b", "turn", "UserMessage", "again"), responseMessage("c", "turn", "user", "again")}, 2},
		{"fallback item only", []map[string]any{completedMessage("a", "turn", "AgentMessage", "hello")}, 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := convertMessageTestRecords(t, test.records)
			if len(session.Turns) != test.want {
				t.Fatalf("got %d messages, want %d", len(session.Turns), test.want)
			}
		})
	}
}

func TestCodexMessageConflictAndCanonicalSource(t *testing.T) {
	session := convertMessageTestRecords(t, []map[string]any{
		completedMessage("a", "turn", "AgentMessage", "item text"),
		responseMessage("a", "turn", "assistant", "response text"),
	})
	if len(session.Turns) != 1 || session.Turns[0].Content != "response text" {
		t.Fatalf("response representation must be canonical: %#v", session.Turns)
	}
	metadata := mapValue(session.Turns[0].FrameworkMetadata)
	if !reflect.DeepEqual(metadata["fidelity_diagnostics"], []string{"conflicting_message_content"}) {
		t.Fatalf("conflict not diagnosed: %#v", metadata)
	}
}

func TestCodexExplicitMessageLinkage(t *testing.T) {
	call := map[string]any{"type": "response_item", "payload": map[string]any{
		"type": "function_call", "name": "exec_command", "call_id": "c", "arguments": `{"cmd":"pwd"}`,
		"message_id": "emitter", "turn_id": "turn",
	}}
	session := convertMessageTestRecords(t, []map[string]any{
		responseMessage("user", "turn", "user", "test"), call,
		responseMessage("emitter", "turn", "assistant", "emitting message"),
		responseMessage("final", "turn", "assistant", "final answer"),
	})
	if index := session.ToolCalls[0].EmittingTurnIndex; index == nil || *index != 1 {
		t.Fatalf("want explicit emitter index 1, got %v", index)
	}
	if !reflect.DeepEqual(session.Turns[1].ToolCallsInTurn, []string{"c"}) || len(session.Turns[2].ToolCallsInTurn) != 0 {
		t.Fatalf("linkage must be reciprocal and not attributed to final answer: %#v", session.Turns)
	}
	mapValue(call["payload"])["message_id"] = "user"
	session = convertMessageTestRecords(t, []map[string]any{responseMessage("user", "turn", "user", "test"), call})
	if session.ToolCalls[0].EmittingTurnIndex != nil {
		t.Fatal("user message cannot be a tool emitter")
	}
}

func TestCodexMessageBlocksAndNativeRole(t *testing.T) {
	record := responseMessage("developer", "turn", "developer", "instruction")
	mapValue(record["payload"])["content"] = []any{
		map[string]any{"type": "input_text", "text": "instruction"},
		map[string]any{"type": "input_image", "image_url": "data:image/png;base64,PRIVATE"},
		map[string]any{"type": "output_text", "text": "more"},
	}
	session := convertMessageTestRecords(t, []map[string]any{record})
	turn := session.Turns[0]
	if turn.Role != "system" || turn.Content != "instruction\nmore" {
		t.Fatalf("unexpected content/role: %#v", turn)
	}
	metadata := mapValue(turn.FrameworkMetadata)
	if metadata["native_role"] != "developer" || metadata["has_image_signal"] != true {
		t.Fatalf("native role and image signal missing: %#v", metadata)
	}
	if _, ok := metadata["image_url"]; ok {
		t.Fatal("image payload must not be copied")
	}
}

func TestCodexLegacyConcatenatedBlockMirror(t *testing.T) {
	response := responseMessage("user", "turn", "user", "unused")
	mapValue(response["payload"])["content"] = []any{
		map[string]any{"type": "input_text", "text": "first\n"},
		map[string]any{"type": "input_text", "text": "second"},
	}
	records := []map[string]any{
		{"type": "event_msg", "payload": map[string]any{"type": "task_started", "turn_id": "turn"}},
		response,
		{"type": "event_msg", "payload": map[string]any{"type": "user_message", "message": "first\nsecond"}},
	}
	session := convertMessageTestRecords(t, records)
	if len(session.Turns) != 1 || session.Turns[0].Content != "first\n\nsecond" {
		t.Fatalf("native concatenation must reconcile without losing display boundaries: %#v", session.Turns)
	}
	// Whitespace normalization would falsely merge this non-mirror.
	mapValue(records[2]["payload"])["message"] = "first second"
	if got := len(convertMessageTestRecords(t, records).Turns); got != 2 {
		t.Fatalf("whitespace-different text must stay distinct, got %d", got)
	}
}

func TestCodexLegacyMirrorsAndUnknownAssociations(t *testing.T) {
	session := convertMessageTestRecords(t, []map[string]any{
		{"type": "event_msg", "payload": map[string]any{"type": "task_started", "turn_id": "turn"}},
		responseMessage("user", "turn", "user", "inspect"),
		{"type": "event_msg", "payload": map[string]any{"type": "user_message", "message": "inspect"}},
		{"type": "response_item", "payload": map[string]any{"type": "function_call", "name": "exec_command", "call_id": "call", "arguments": `{\"cmd\":\"pwd\"}`}},
		{"type": "event_msg", "payload": map[string]any{"type": "agent_message", "message": "done", "memory_citation": "retained"}},
		responseMessage("assistant", "turn", "assistant", "done"),
	})
	if len(session.Turns) != 2 || session.Turns[0].Content != "inspect" || session.Turns[1].Content != "done" {
		t.Fatalf("legacy mirrors not reconciled: %#v", session.Turns)
	}
	if mapValue(session.Turns[1].FrameworkMetadata)["memory_citation"] != "retained" {
		t.Fatal("canonical response discarded legacy metadata")
	}
	if session.ToolCalls[0].EmittingTurnIndex != nil {
		t.Fatal("a later final answer is not an established emitter")
	}
	for _, format := range []string{"session-jsonl-v1", "legacy-rollout-jsonl-v0"} {
		t.Run(format, func(t *testing.T) {
			call := map[string]any{"type": "function_call", "call_id": "orphan", "name": "exec_command", "arguments": `{\"cmd\":\"pwd\"}`}
			var records []map[string]any
			if format == "session-jsonl-v1" {
				records = []map[string]any{{"type": "response_item", "payload": call}}
			} else {
				records = []map[string]any{call}
			}
			session, err := ConvertRecords(records, "test", "/synthetic/session.jsonl", format)
			if err != nil {
				t.Fatal(err)
			}
			if len(session.ToolCalls) != 1 || session.ToolCalls[0].EmittingTurnIndex != nil {
				t.Fatalf("no-message source produced invalid association: %#v", session.ToolCalls)
			}
		})
	}
}

func convertMessageTestRecords(t *testing.T, records []map[string]any) *minitrace.Session {
	t.Helper()
	session, err := ConvertRecords(records, "test", "/synthetic/session.jsonl", "session-jsonl-v1")
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func responseMessage(id, turnID, role, text string) map[string]any {
	return map[string]any{"type": "response_item", "payload": map[string]any{
		"type": "message", "id": id, "role": role,
		"content": []any{map[string]any{"type": "input_text", "text": text}},
		"internal_chat_message_metadata_passthrough": map[string]any{"turn_id": turnID},
	}}
}

func completedMessage(id, turnID, kind, text string) map[string]any {
	return map[string]any{"type": "event_msg", "payload": map[string]any{
		"type": "item_completed", "turn_id": turnID,
		"item": map[string]any{"type": kind, "id": id, "content": []any{map[string]any{"type": "Text", "text": text}}},
	}}
}
