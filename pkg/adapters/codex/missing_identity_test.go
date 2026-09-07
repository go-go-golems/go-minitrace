package codex

import "testing"

func TestMissingIdentityCannotCollideWithLiteralNativeID(t *testing.T) {
	for _, kind := range []string{"CommandExecution", "FileChange"} {
		records := []map[string]any{}
		for _, id := range []string{"", "anonymous-line-1"} {
			item := map[string]any{"type": kind, "id": id, "status": "completed", "exit_code": 0, "command": []any{"/bin/sh", "-lc", "pwd"}, "changes": map[string]any{"/file": map[string]any{"type": "add"}}}
			records = append(records, map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "item": item}})
		}
		session := convertMessageTestRecords(t, records)
		if len(session.ToolCalls) != 2 || session.ToolCalls[0].ID == session.ToolCalls[1].ID {
			t.Fatalf("%s fallback collided with native identity", kind)
		}
		metadata := mapValue(session.ToolCalls[0].FrameworkMetadata)
		if metadata["native_execution_id"] != nil || metadata["native_file_change_id"] != nil {
			t.Fatalf("%s missing native identity fabricated", kind)
		}
	}
}
