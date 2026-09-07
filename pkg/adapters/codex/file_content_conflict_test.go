package codex

import "testing"

func TestSamePathDifferentNativeContentsRemainUncertain(t *testing.T) {
	var records []map[string]any
	for _, content := range []string{"first", "second", "first"} {
		records = append(records, map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "item": map[string]any{"type": "FileChange", "id": "same", "status": "completed", "changes": map[string]any{"/same": map[string]any{"type": "add", "content": content}}}}})
	}
	session := convertMessageTestRecords(t, records)
	if len(session.ToolCalls) != 1 {
		t.Fatal("duplicate lifecycle count")
	}
	call := session.ToolCalls[0]
	if call.Output.Success != nil || call.Input.FileTargets[0].Success != nil {
		t.Fatal("content conflict asserted success")
	}
	sources := mapValue(call.FrameworkMetadata)["file_change_sources"].([]map[string]any)
	if sources[0]["changes_hash"] == sources[1]["changes_hash"] || sources[0]["changes_hash"] != sources[2]["changes_hash"] {
		t.Fatal("content provenance lost")
	}
}
