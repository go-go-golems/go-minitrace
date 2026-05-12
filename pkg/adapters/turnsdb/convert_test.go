package turnsdb

import (
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

const testSchema = `
CREATE TABLE turns (
    conv_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    turn_id TEXT NOT NULL,
    turn_created_at_ms INTEGER NOT NULL,
    turn_metadata_json TEXT NOT NULL DEFAULT '{}',
    turn_data_json TEXT NOT NULL DEFAULT '{}',
    runtime_key TEXT NOT NULL DEFAULT '',
    inference_id TEXT NOT NULL DEFAULT '',
    updated_at_ms INTEGER NOT NULL,
    PRIMARY KEY (conv_id, session_id, turn_id)
);
CREATE TABLE blocks (
    block_id TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    hash_algorithm TEXT NOT NULL DEFAULT 'sha256-canonical-json-v1',
    kind TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL DEFAULT '{}',
    block_metadata_json TEXT NOT NULL DEFAULT '{}',
    first_seen_at_ms INTEGER NOT NULL,
    PRIMARY KEY (block_id, content_hash)
);
CREATE TABLE turn_block_membership (
    conv_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    turn_id TEXT NOT NULL,
    phase TEXT NOT NULL,
    snapshot_created_at_ms INTEGER NOT NULL,
    ordinal INTEGER NOT NULL,
    block_id TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    PRIMARY KEY (conv_id, session_id, turn_id, phase, snapshot_created_at_ms, ordinal)
);
`

func TestChooseCanonicalPrefersFinal(t *testing.T) {
	dbPath := createTestDB(t)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	insertTurn(t, db, "conv-1", "sess-1", "turn-1", 1000)
	insertBlock(t, db, "b1", "h1", "system", "system", `{"text":"sys"}`, `{}`)
	insertBlock(t, db, "b2", "h2", "user", "user", `{"text":"hello"}`, `{}`)
	insertBlock(t, db, "b3", "h3", "llm_text", "assistant", `{"text":"hi"}`, `{}`)

	insertMembership(t, db, "conv-1", "sess-1", "turn-1", "post_inference", 1010, 0, "b1", "h1")
	insertMembership(t, db, "conv-1", "sess-1", "turn-1", "post_inference", 1010, 1, "b2", "h2")
	insertMembership(t, db, "conv-1", "sess-1", "turn-1", "post_inference", 1010, 2, "b3", "h3")
	insertMembership(t, db, "conv-1", "sess-1", "turn-1", "final", 1020, 0, "b1", "h1")
	insertMembership(t, db, "conv-1", "sess-1", "turn-1", "final", 1020, 1, "b2", "h2")
	insertMembership(t, db, "conv-1", "sess-1", "turn-1", "final", 1020, 2, "b3", "h3")

	ro, err := connectReadOnly(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ro.Close() }()

	turns, err := loadCanonicalTurns(ro, "conv-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	if turns[0].Phase != "final" {
		t.Fatalf("expected final phase, got %s", turns[0].Phase)
	}
	if len(turns[0].Blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(turns[0].Blocks))
	}
	if turns[0].Blocks[2].Kind != "llm_text" {
		t.Fatalf("expected llm_text block, got %s", turns[0].Blocks[2].Kind)
	}
}

func TestFirstSeenBlocksSkipsPreviouslySeenBlocks(t *testing.T) {
	previous := []Block{
		{ID: "b1", ContentHash: "sys-hash", Kind: "system", Role: "system", Payload: map[string]any{"text": "sys"}, Metadata: map[string]any{}},
		{ID: "b2", Kind: "user", Role: "user", Payload: map[string]any{"text": "<currentMode>old</currentMode>"}, Metadata: map[string]any{}},
		{ID: "b3", Kind: "user", Role: "user", Payload: map[string]any{"text": "hello"}, Metadata: map[string]any{}},
		{ID: "b4", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "hi"}, Metadata: map[string]any{}},
	}
	current := []Block{
		{ID: "b1", ContentHash: "sys-hash", Kind: "system", Role: "system", Payload: map[string]any{"text": "sys"}, Metadata: map[string]any{}},
		{ID: "b3", Kind: "user", Role: "user", Payload: map[string]any{"text": "hello"}, Metadata: map[string]any{}},
		{ID: "b4", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "hi"}, Metadata: map[string]any{}},
		{ID: "b5", Kind: "user", Role: "user", Payload: map[string]any{"text": "new question"}, Metadata: map[string]any{}},
		{ID: "b6", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "new answer"}, Metadata: map[string]any{}},
	}

	seen := map[string]struct{}{}
	if _, err := firstSeenBlocks(previous, seen); err != nil {
		t.Fatalf("firstSeenBlocks previous returned error: %v", err)
	}
	newBlocks, err := firstSeenBlocks(current, seen)
	if err != nil {
		t.Fatalf("firstSeenBlocks current returned error: %v", err)
	}
	if len(newBlocks) != 2 {
		t.Fatalf("expected 2 new blocks, got %d", len(newBlocks))
	}
	if text := newBlocks[0].Payload["text"]; text != "new question" {
		t.Fatalf("unexpected first delta text: %v", text)
	}
	if text := newBlocks[1].Payload["text"]; text != "new answer" {
		t.Fatalf("unexpected second delta text: %v", text)
	}
}

func TestFirstSeenBlocksFailsWhenBlockIdentityIsMissing(t *testing.T) {
	_, err := firstSeenBlocks([]Block{{Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "hi"}}}, map[string]struct{}{})
	if err == nil {
		t.Fatalf("expected missing block_id to fail")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "missing block_id") {
		t.Fatalf("expected missing block_id error, got %v", err)
	}

	_, err = firstSeenBlocks([]Block{{ID: "tc-block", Kind: "tool_call", Role: "assistant", Payload: map[string]any{"name": "bash"}}}, map[string]struct{}{})
	if err == nil {
		t.Fatalf("expected missing tool payload id to fail")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "missing payload id") {
		t.Fatalf("expected missing payload id error, got %v", err)
	}

	_, err = firstSeenBlocks([]Block{{ID: "sys-block", Kind: "system", Role: "system", Payload: map[string]any{"text": "sys"}}}, map[string]struct{}{})
	if err == nil {
		t.Fatalf("expected missing system content_hash to fail")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "missing content_hash") {
		t.Fatalf("expected missing content_hash error, got %v", err)
	}
}

func TestFirstSeenBlocksTreatsRegeneratedSystemBlockIDsAsSamePrompt(t *testing.T) {
	previous := []Block{
		{ID: "sys-turn-1", ContentHash: "same-system-prompt", Kind: "system", Role: "system", Payload: map[string]any{"text": "sys"}, Metadata: map[string]any{}},
		{ID: "user-1", Kind: "user", Role: "user", Payload: map[string]any{"text": "first"}, Metadata: map[string]any{}},
	}
	current := []Block{
		{ID: "sys-turn-2", ContentHash: "same-system-prompt", Kind: "system", Role: "system", Payload: map[string]any{"text": "sys"}, Metadata: map[string]any{}},
		{ID: "user-2", Kind: "user", Role: "user", Payload: map[string]any{"text": "second"}, Metadata: map[string]any{}},
	}

	seen := map[string]struct{}{}
	if _, err := firstSeenBlocks(previous, seen); err != nil {
		t.Fatalf("firstSeenBlocks previous returned error: %v", err)
	}
	newBlocks, err := firstSeenBlocks(current, seen)
	if err != nil {
		t.Fatalf("firstSeenBlocks current returned error: %v", err)
	}
	if got := len(newBlocks); got != 1 {
		t.Fatalf("expected only the new user block, got %d: %#v", got, newBlocks)
	}
	if got := newBlocks[0].Payload["text"]; got != "second" {
		t.Fatalf("unexpected delta text: %v", got)
	}
}

func TestConvertConversationSnapshotsLinksToolCallsAndNormalizesBlankText(t *testing.T) {
	snapshots := []CanonicalTurnSnapshot{
		{
			ConvID:              "conv-tools",
			SessionID:           "sess-1",
			TurnID:              "turn-1",
			TurnCreatedAtMS:     1000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-1",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 1010,
			Blocks: []Block{
				{ID: "b1", ContentHash: "sys-hash", Kind: "system", Role: "system", Payload: map[string]any{"text": "You are an assistant"}, Metadata: map[string]any{}},
				{ID: "b2", Kind: "user", Role: "user", Payload: map[string]any{"text": "hello"}, Metadata: map[string]any{}},
				{ID: "b3", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "\n"}, Metadata: map[string]any{}},
				{ID: "tc1-block", Kind: "tool_call", Role: "assistant", Payload: map[string]any{"id": "tc-1", "name": "bash", "args": map[string]any{"command": "echo hi"}}, Metadata: map[string]any{"stream_seq": float64(1)}},
				{ID: "tu1-block", Kind: "tool_use", Role: "tool", Payload: map[string]any{"id": "tc-1", "result": "hi\n"}, Metadata: map[string]any{}},
			},
		},
	}

	session, err := convertConversationSnapshots(snapshots, "/tmp/fake.db")
	if err != nil {
		t.Fatalf("convertConversationSnapshots returned error: %v", err)
	}
	if got := len(session.ToolCalls); got != 1 {
		t.Fatalf("expected 1 tool call, got %d", got)
	}
	if !session.ToolCalls[0].Output.Success {
		t.Fatalf("expected successful tool call, got %+v", session.ToolCalls[0].Output)
	}
	if session.ToolCalls[0].Output.Result == nil || *session.ToolCalls[0].Output.Result != "hi\n" {
		t.Fatalf("unexpected tool result: %+v", session.ToolCalls[0].Output.Result)
	}
	if got := session.Turns[2].Content; got != "\n" {
		t.Fatalf("expected blank text payload to unwrap to newline, got %q", got)
	}
	if got := session.Turns[2].ToolCallsInTurn; len(got) != 1 || got[0] != "tc-1" {
		t.Fatalf("expected assistant turn to link tc-1, got %#v", got)
	}
}

func TestConvertConversationSnapshotsDoesNotDuplicateNonToolBlocksWhenMetadataChanges(t *testing.T) {
	snapshots := []CanonicalTurnSnapshot{
		{
			ConvID:              "conv-metadata",
			SessionID:           "sess-1",
			TurnID:              "turn-1",
			TurnCreatedAtMS:     1000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-1",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 1010,
			Blocks: []Block{
				{ID: "b1", ContentHash: "sys-hash", Kind: "system", Role: "system", Payload: map[string]any{"text": "You are an assistant"}, Metadata: map[string]any{}},
				{ID: "b2", Kind: "user", Role: "user", Payload: map[string]any{"text": "hello"}, Metadata: map[string]any{"seen": float64(1)}},
				{ID: "b3", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "hi"}, Metadata: map[string]any{}},
			},
		},
		{
			ConvID:              "conv-metadata",
			SessionID:           "sess-1",
			TurnID:              "turn-2",
			TurnCreatedAtMS:     2000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-2",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 2010,
			Blocks: []Block{
				{ID: "b1", ContentHash: "sys-hash", Kind: "system", Role: "system", Payload: map[string]any{"text": "You are an assistant"}, Metadata: map[string]any{}},
				{ID: "b2", Kind: "user", Role: "user", Payload: map[string]any{"text": "hello"}, Metadata: map[string]any{"seen": float64(2)}},
				{ID: "b3", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "hi"}, Metadata: map[string]any{}},
				{ID: "b4", Kind: "user", Role: "user", Payload: map[string]any{"text": "again"}, Metadata: map[string]any{}},
			},
		},
	}

	session, err := convertConversationSnapshots(snapshots, "/tmp/fake.db")
	if err != nil {
		t.Fatalf("convertConversationSnapshots returned error: %v", err)
	}
	if got := len(session.Turns); got != 4 {
		t.Fatalf("expected only one new non-tool turn despite metadata churn, got %d", got)
	}
	if got := session.Turns[3].Content; got != "again" {
		t.Fatalf("expected only new turn content, got %q", got)
	}
}

func TestConvertConversationSnapshotsDoesNotDuplicateToolCallsWhenMetadataChanges(t *testing.T) {
	snapshots := []CanonicalTurnSnapshot{
		{
			ConvID:              "conv-tools",
			SessionID:           "sess-1",
			TurnID:              "turn-1",
			TurnCreatedAtMS:     1000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-1",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 1010,
			Blocks: []Block{
				{ID: "b1", ContentHash: "sys-hash", Kind: "system", Role: "system", Payload: map[string]any{"text": "You are an assistant"}, Metadata: map[string]any{}},
				{ID: "b2", Kind: "user", Role: "user", Payload: map[string]any{"text": "hello"}, Metadata: map[string]any{}},
				{ID: "b3", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "running tool"}, Metadata: map[string]any{}},
				{ID: "tc1-block", Kind: "tool_call", Role: "assistant", Payload: map[string]any{"id": "tc-1", "name": "bash", "args": map[string]any{"command": "echo hi"}}, Metadata: map[string]any{"stream_seq": float64(1)}},
				{ID: "tu1-block", Kind: "tool_use", Role: "tool", Payload: map[string]any{"id": "tc-1", "result": "hi\n"}, Metadata: map[string]any{}},
			},
		},
		{
			ConvID:              "conv-tools",
			SessionID:           "sess-1",
			TurnID:              "turn-2",
			TurnCreatedAtMS:     2000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-2",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 2010,
			Blocks: []Block{
				{ID: "b1", ContentHash: "sys-hash", Kind: "system", Role: "system", Payload: map[string]any{"text": "You are an assistant"}, Metadata: map[string]any{}},
				{ID: "b2", Kind: "user", Role: "user", Payload: map[string]any{"text": "hello"}, Metadata: map[string]any{}},
				{ID: "b3", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "running tool"}, Metadata: map[string]any{}},
				{ID: "tc1-block", Kind: "tool_call", Role: "assistant", Payload: map[string]any{"id": "tc-1", "name": "bash", "args": map[string]any{"command": "echo hi"}}, Metadata: map[string]any{"stream_seq": float64(99)}},
				{ID: "tu1-block", Kind: "tool_use", Role: "tool", Payload: map[string]any{"id": "tc-1", "result": "hi\n"}, Metadata: map[string]any{}},
				{ID: "b4", Kind: "user", Role: "user", Payload: map[string]any{"text": "again"}, Metadata: map[string]any{}},
				{ID: "b5", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "done"}, Metadata: map[string]any{}},
			},
		},
	}

	session, err := convertConversationSnapshots(snapshots, "/tmp/fake.db")
	if err != nil {
		t.Fatalf("convertConversationSnapshots returned error: %v", err)
	}
	if got := len(session.ToolCalls); got != 1 {
		t.Fatalf("expected 1 deduplicated tool call, got %d: %#v", got, session.ToolCalls)
	}
	if !session.ToolCalls[0].Output.Success {
		t.Fatalf("expected successful tool call not to be downgraded by metadata churn, got %+v", session.ToolCalls[0].Output)
	}
	if session.ToolCalls[0].EmittingTurnIndex == nil || *session.ToolCalls[0].EmittingTurnIndex != 2 {
		t.Fatalf("expected original emitting turn index 2, got %+v", session.ToolCalls[0].EmittingTurnIndex)
	}
	if got := session.Turns[2].ToolCallsInTurn; len(got) != 1 || got[0] != "tc-1" {
		t.Fatalf("expected only original assistant turn to link tc-1, got %#v", got)
	}
}

func TestConvertConversationSnapshotsAttachesLeadingToolsToNextAssistant(t *testing.T) {
	snapshots := []CanonicalTurnSnapshot{
		{
			ConvID:              "conv-leading-tool",
			SessionID:           "sess-1",
			TurnID:              "turn-1",
			TurnCreatedAtMS:     1000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-1",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 1010,
			Blocks: []Block{
				{ID: "tc1-block", Kind: "tool_call", Role: "assistant", Payload: map[string]any{"id": "tc-1", "name": "lookup", "args": map[string]any{"q": "x"}}, Metadata: map[string]any{}},
				{ID: "tu1-block", Kind: "tool_use", Role: "tool", Payload: map[string]any{"id": "tc-1", "result": "result"}, Metadata: map[string]any{}},
				{ID: "b1", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "I found it."}, Metadata: map[string]any{}},
			},
		},
	}

	session, err := convertConversationSnapshots(snapshots, "/tmp/fake.db")
	if err != nil {
		t.Fatalf("convertConversationSnapshots returned error: %v", err)
	}
	if got := session.Turns[0].ToolCallsInTurn; len(got) != 1 || got[0] != "tc-1" {
		t.Fatalf("expected following assistant turn to link tc-1, got %#v", got)
	}
	if session.ToolCalls[0].EmittingTurnIndex == nil || *session.ToolCalls[0].EmittingTurnIndex != 0 {
		t.Fatalf("expected tool call emitting turn 0, got %+v", session.ToolCalls[0].EmittingTurnIndex)
	}
}

func TestConvertConversationSnapshotsDoesNotAttachNewTurnLeadingToolToPreviousAssistant(t *testing.T) {
	snapshots := []CanonicalTurnSnapshot{
		{
			ConvID:              "conv-leading-tool-after-previous",
			SessionID:           "sess-1",
			TurnID:              "turn-1",
			TurnCreatedAtMS:     1000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-1",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 1010,
			Blocks: []Block{
				{ID: "sys-1", ContentHash: "same-system", Kind: "system", Role: "system", Payload: map[string]any{"text": "You are an assistant"}, Metadata: map[string]any{}},
				{ID: "u1", Kind: "user", Role: "user", Payload: map[string]any{"text": "first"}, Metadata: map[string]any{}},
				{ID: "a1", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "first answer"}, Metadata: map[string]any{}},
			},
		},
		{
			ConvID:              "conv-leading-tool-after-previous",
			SessionID:           "sess-1",
			TurnID:              "turn-2",
			TurnCreatedAtMS:     2000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-2",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 2010,
			Blocks: []Block{
				{ID: "sys-2", ContentHash: "same-system", Kind: "system", Role: "system", Payload: map[string]any{"text": "You are an assistant"}, Metadata: map[string]any{}},
				{ID: "u2", Kind: "user", Role: "user", Payload: map[string]any{"text": "second"}, Metadata: map[string]any{}},
				{ID: "tc2-block", Kind: "tool_call", Role: "assistant", Payload: map[string]any{"id": "tc-2", "name": "lookup", "args": map[string]any{"q": "second"}}, Metadata: map[string]any{}},
				{ID: "tu2-block", Kind: "tool_use", Role: "tool", Payload: map[string]any{"id": "tc-2", "result": "result"}, Metadata: map[string]any{}},
				{ID: "a2", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "second answer"}, Metadata: map[string]any{}},
			},
		},
	}

	session, err := convertConversationSnapshots(snapshots, "/tmp/fake.db")
	if err != nil {
		t.Fatalf("convertConversationSnapshots returned error: %v", err)
	}
	if got := session.Turns[2].ToolCallsInTurn; len(got) != 0 {
		t.Fatalf("expected previous assistant to have no new-turn tools, got %#v", got)
	}
	if got := session.Turns[4].ToolCallsInTurn; len(got) != 1 || got[0] != "tc-2" {
		t.Fatalf("expected following assistant turn to link tc-2, got %#v", got)
	}
	if session.ToolCalls[0].EmittingTurnIndex == nil || *session.ToolCalls[0].EmittingTurnIndex != 4 {
		t.Fatalf("expected tool call emitting turn 4, got %+v", session.ToolCalls[0].EmittingTurnIndex)
	}
}

func TestConvertConversationSnapshotsAttachesInterleavedToolsToNearestAssistant(t *testing.T) {
	snapshots := []CanonicalTurnSnapshot{
		{
			ConvID:              "conv-interleaved",
			SessionID:           "sess-1",
			TurnID:              "turn-1",
			TurnCreatedAtMS:     1000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-1",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 1010,
			Blocks: []Block{
				{ID: "b1", ContentHash: "sys-hash", Kind: "system", Role: "system", Payload: map[string]any{"text": "You are an assistant"}, Metadata: map[string]any{}},
				{ID: "b2", Kind: "user", Role: "user", Payload: map[string]any{"text": "inspect inventory"}, Metadata: map[string]any{}},
				{ID: "b3", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "First I will list coins."}, Metadata: map[string]any{}},
				{ID: "tc1-block", Kind: "tool_call", Role: "assistant", Payload: map[string]any{"id": "tc-1", "name": "list_coins", "args": map[string]any{"kind": "gold"}}, Metadata: map[string]any{}},
				{ID: "tu1-block", Kind: "tool_use", Role: "tool", Payload: map[string]any{"id": "tc-1", "result": "coins"}, Metadata: map[string]any{}},
				{ID: "b4", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "Now I will price them."}, Metadata: map[string]any{}},
				{ID: "tc2-block", Kind: "tool_call", Role: "assistant", Payload: map[string]any{"id": "tc-2", "name": "price_coins", "args": map[string]any{"currency": "USD"}}, Metadata: map[string]any{}},
				{ID: "tu2-block", Kind: "tool_use", Role: "tool", Payload: map[string]any{"id": "tc-2", "result": "prices"}, Metadata: map[string]any{}},
				{ID: "b5", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "Done."}, Metadata: map[string]any{}},
			},
		},
	}

	session, err := convertConversationSnapshots(snapshots, "/tmp/fake.db")
	if err != nil {
		t.Fatalf("convertConversationSnapshots returned error: %v", err)
	}
	if got := len(session.ToolCalls); got != 2 {
		t.Fatalf("expected 2 tool calls, got %d", got)
	}
	if got := session.Turns[2].ToolCallsInTurn; len(got) != 1 || got[0] != "tc-1" {
		t.Fatalf("expected first assistant segment to link tc-1, got %#v", got)
	}
	if got := session.Turns[3].ToolCallsInTurn; len(got) != 1 || got[0] != "tc-2" {
		t.Fatalf("expected second assistant segment to link tc-2, got %#v", got)
	}
	if got := session.Turns[4].ToolCallsInTurn; len(got) != 0 {
		t.Fatalf("expected final assistant segment to have no tool links, got %#v", got)
	}
}

func TestConvertConversationSnapshotsPreservesNonCumulativeStoredTurns(t *testing.T) {
	snapshots := []CanonicalTurnSnapshot{
		{
			ConvID:              "conv-noncumulative",
			SessionID:           "sess-1",
			TurnID:              "turn-1",
			TurnCreatedAtMS:     1000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-1",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 1010,
			Blocks: []Block{
				{ID: "sys-1", ContentHash: "same-system", Kind: "system", Role: "system", Payload: map[string]any{"text": "You are an assistant"}, Metadata: map[string]any{}},
				{ID: "u1", Kind: "user", Role: "user", Payload: map[string]any{"text": "first question"}, Metadata: map[string]any{}},
				{ID: "a1", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "first answer"}, Metadata: map[string]any{}},
			},
		},
		{
			ConvID:              "conv-noncumulative",
			SessionID:           "sess-1",
			TurnID:              "turn-2",
			TurnCreatedAtMS:     2000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-2",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 2010,
			Blocks: []Block{
				{ID: "sys-2", ContentHash: "same-system", Kind: "system", Role: "system", Payload: map[string]any{"text": "You are an assistant"}, Metadata: map[string]any{}},
				{ID: "u2", Kind: "user", Role: "user", Payload: map[string]any{"text": "second question"}, Metadata: map[string]any{}},
				{ID: "a2", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "second answer"}, Metadata: map[string]any{}},
			},
		},
	}

	session, err := convertConversationSnapshots(snapshots, "/tmp/fake.db")
	if err != nil {
		t.Fatalf("convertConversationSnapshots returned error: %v", err)
	}
	contents := []string{}
	for _, turn := range session.Turns {
		contents = append(contents, turn.Content)
	}
	expected := []string{"You are an assistant", "first question", "first answer", "second question", "second answer"}
	if !reflect.DeepEqual(contents, expected) {
		t.Fatalf("unexpected contents:\ngot  %#v\nwant %#v", contents, expected)
	}
}

func TestConvertConversationSnapshotsHandlesMixedCumulativeAndNonCumulativeStoredTurns(t *testing.T) {
	snapshots := []CanonicalTurnSnapshot{
		{
			ConvID:              "conv-mixed",
			SessionID:           "sess-1",
			TurnID:              "turn-1",
			TurnCreatedAtMS:     1000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-1",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 1010,
			Blocks: []Block{
				{ID: "sys-1", ContentHash: "same-system", Kind: "system", Role: "system", Payload: map[string]any{"text": "You are an assistant"}, Metadata: map[string]any{}},
				{ID: "u1", Kind: "user", Role: "user", Payload: map[string]any{"text": "first question"}, Metadata: map[string]any{}},
				{ID: "a1", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "first answer"}, Metadata: map[string]any{}},
			},
		},
		{
			ConvID:              "conv-mixed",
			SessionID:           "sess-1",
			TurnID:              "turn-2",
			TurnCreatedAtMS:     2000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-2",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 2010,
			Blocks: []Block{
				{ID: "sys-2", ContentHash: "same-system", Kind: "system", Role: "system", Payload: map[string]any{"text": "You are an assistant"}, Metadata: map[string]any{}},
				{ID: "u1", Kind: "user", Role: "user", Payload: map[string]any{"text": "first question"}, Metadata: map[string]any{}},
				{ID: "a1", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "first answer"}, Metadata: map[string]any{}},
				{ID: "u2", Kind: "user", Role: "user", Payload: map[string]any{"text": "second question"}, Metadata: map[string]any{}},
				{ID: "a2", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "second answer"}, Metadata: map[string]any{}},
			},
		},
		{
			ConvID:              "conv-mixed",
			SessionID:           "sess-1",
			TurnID:              "turn-3",
			TurnCreatedAtMS:     3000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-3",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 3010,
			Blocks: []Block{
				{ID: "sys-3", ContentHash: "same-system", Kind: "system", Role: "system", Payload: map[string]any{"text": "You are an assistant"}, Metadata: map[string]any{}},
				{ID: "u3", Kind: "user", Role: "user", Payload: map[string]any{"text": "third question"}, Metadata: map[string]any{}},
				{ID: "a3", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "third answer"}, Metadata: map[string]any{}},
			},
		},
	}

	session, err := convertConversationSnapshots(snapshots, "/tmp/fake.db")
	if err != nil {
		t.Fatalf("convertConversationSnapshots returned error: %v", err)
	}
	contents := []string{}
	for _, turn := range session.Turns {
		contents = append(contents, turn.Content)
	}
	expected := []string{"You are an assistant", "first question", "first answer", "second question", "second answer", "third question", "third answer"}
	if !reflect.DeepEqual(contents, expected) {
		t.Fatalf("unexpected contents:\ngot  %#v\nwant %#v", contents, expected)
	}
}

func TestConvertConversationSnapshotsEmitsOnlyNewTurnContent(t *testing.T) {
	snapshots := []CanonicalTurnSnapshot{
		{
			ConvID:              "conv-1",
			SessionID:           "sess-1",
			TurnID:              "turn-1",
			TurnCreatedAtMS:     1000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-1",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 1010,
			Blocks: []Block{
				{ID: "b1", ContentHash: "sys-hash", Kind: "system", Role: "system", Payload: map[string]any{"text": "You are an assistant"}, Metadata: map[string]any{}},
				{ID: "b2", Kind: "user", Role: "user", Payload: map[string]any{"text": "hello"}, Metadata: map[string]any{}},
				{ID: "b3", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "hi"}, Metadata: map[string]any{}},
			},
		},
		{
			ConvID:              "conv-1",
			SessionID:           "sess-1",
			TurnID:              "turn-2",
			TurnCreatedAtMS:     2000,
			RuntimeKey:          "gpt-5-nano",
			InferenceID:         "inf-2",
			TurnMetadata:        map[string]any{},
			TurnData:            map[string]any{},
			Phase:               "final",
			SnapshotCreatedAtMS: 2010,
			Blocks: []Block{
				{ID: "b1", ContentHash: "sys-hash", Kind: "system", Role: "system", Payload: map[string]any{"text": "You are an assistant"}, Metadata: map[string]any{}},
				{ID: "b2", Kind: "user", Role: "user", Payload: map[string]any{"text": "hello"}, Metadata: map[string]any{}},
				{ID: "b3", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "hi"}, Metadata: map[string]any{}},
				{ID: "b4", Kind: "user", Role: "user", Payload: map[string]any{"text": "again"}, Metadata: map[string]any{}},
				{ID: "b5", Kind: "llm_text", Role: "assistant", Payload: map[string]any{"text": "done"}, Metadata: map[string]any{}},
			},
		},
	}

	session, err := convertConversationSnapshots(snapshots, "/tmp/fake.db")
	if err != nil {
		t.Fatalf("convertConversationSnapshots returned error: %v", err)
	}
	if got := len(session.Turns); got != 5 {
		t.Fatalf("expected 5 turns, got %d", got)
	}
	contents := []string{}
	for _, turn := range session.Turns {
		contents = append(contents, turn.Content)
	}
	expected := []string{"You are an assistant", "hello", "hi", "again", "done"}
	for i := range expected {
		if contents[i] != expected[i] {
			t.Fatalf("unexpected content at %d: got %q want %q", i, contents[i], expected[i])
		}
	}
	if session.Environment.Model == nil || *session.Environment.Model != "gpt-5-nano" {
		t.Fatalf("expected gpt-5-nano, got %+v", session.Environment.Model)
	}
}

func createTestDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "turns.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(testSchema); err != nil {
		t.Fatalf("creating schema: %v", err)
	}
	return dbPath
}

func insertTurn(t *testing.T, db *sql.DB, convID, sessionID, turnID string, createdAtMS int64) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO turns(
		  conv_id, session_id, turn_id, turn_created_at_ms, turn_metadata_json,
		  turn_data_json, runtime_key, inference_id, updated_at_ms
		) VALUES (?, ?, ?, ?, '{}', '{}', 'gpt-5-nano', 'inf', ?)
	`, convID, sessionID, turnID, createdAtMS, createdAtMS); err != nil {
		t.Fatalf("insert turn: %v", err)
	}
}

func insertBlock(t *testing.T, db *sql.DB, blockID, contentHash, kind, role, payloadJSON, metadataJSON string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO blocks(
		  block_id, content_hash, kind, role, payload_json, block_metadata_json, first_seen_at_ms
		) VALUES (?, ?, ?, ?, ?, ?, 1)
	`, blockID, contentHash, kind, role, payloadJSON, metadataJSON); err != nil {
		t.Fatalf("insert block: %v", err)
	}
}

func insertMembership(t *testing.T, db *sql.DB, convID, sessionID, turnID, phase string, ts int64, ordinal int, blockID, contentHash string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO turn_block_membership(
		  conv_id, session_id, turn_id, phase, snapshot_created_at_ms, ordinal, block_id, content_hash
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, convID, sessionID, turnID, phase, ts, ordinal, blockID, contentHash); err != nil {
		t.Fatalf("insert membership: %v", err)
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
