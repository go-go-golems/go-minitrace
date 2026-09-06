package codex

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func appendCodexFileChanges(records []map[string]any, calls []minitrace.ToolCall) []minitrace.ToolCall {
	byNative := map[string]int{}
	used := map[string]bool{}
	for _, call := range calls {
		used[call.ID] = true
	}
	for line, record := range records {
		payload := mapValue(record["payload"])
		item := mapValue(payload["item"])
		if record["type"] != "event_msg" || item["type"] != "FileChange" || (payload["type"] != "item_started" && payload["type"] != "item_completed") {
			continue
		}
		nativeID := stringValue(item["id"])
		missingID := nativeID == ""
		if missingID {
			nativeID = fmt.Sprintf("anonymous-line-%d", line+1)
		}
		identityKey := fmt.Sprintf("%t:%s", missingID, nativeID)
		index, exists := byNative[identityKey]
		if !exists {
			id := "codex-file-change:" + nativeID
			for suffix := 1; used[id]; suffix++ {
				id = fmt.Sprintf("codex-file-change:%s:line-%d:%d", nativeID, line+1, suffix)
			}
			used[id] = true
			index = len(calls)
			byNative[identityKey] = index
			calls = append(calls, minitrace.ToolCall{ID: id, RecordKind: minitrace.RecordKindFileChange, Timestamp: optionalString(stringValue(record["timestamp"])), ToolName: "apply_patch", OperationType: "MODIFY", Input: minitrace.ToolCallInput{FileTargets: []minitrace.FileTarget{}}, Output: minitrace.ToolCallOutput{Status: minitrace.ToolOutcomePending}, FrameworkMetadata: map[string]any{"record_kind": minitrace.RecordKindFileChange, "native_file_change_id": nativeID, "source_line": line + 1, "turn_id": payload["turn_id"], "thread_id": payload["thread_id"], "parent_association": "unknown", "turn_association": "unknown"}})
		}
		call := &calls[index]
		metadata := mapValue(call.FrameworkMetadata)
		if missingID {
			delete(metadata, "native_file_change_id")
			metadata["file_change_identity_kind"] = "source_line"
			addCodexFileDiagnostic(call, "missing_file_change_identity")
		}
		sources, _ := metadata["file_change_sources"].([]map[string]any)
		metadata["file_change_sources"] = append(sources, map[string]any{"source_line": line + 1, "event_type": payload["type"], "status": item["status"], "started_at_ms": payload["started_at_ms"], "completed_at_ms": payload["completed_at_ms"]})
		if payload["type"] != "item_completed" {
			continue
		}
		status := stringValue(item["status"])
		if previous, ok := metadata["completed_status"].(string); ok && previous != status {
			metadata["file_change_conflict"] = true
		}
		metadata["completed_status"] = status
		targets := []minitrace.FileTarget{}
		changes := mapValue(item["changes"])
		if changes == nil {
			addCodexFileDiagnostic(call, "unsupported_file_changes_shape")
		}
		paths := make([]string, 0, len(changes))
		for path := range changes {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			change := mapValue(changes[path])
			operation := ""
			switch stringValue(change["type"]) {
			case "add":
				operation = "NEW"
			case "update":
				operation = "MODIFY"
			case "delete":
				operation = "DELETE"
			default:
				addCodexFileDiagnostic(call, "unsupported_file_change_operation")
				continue
			}
			if path == "" {
				addCodexFileDiagnostic(call, "invalid_file_change_path")
				continue
			}
			reference := fmt.Sprintf("line:%d", line+1)
			target := codexFileTarget(path, operation, "native_file_change", stringValue(item["cwd"]), reference)
			if move := stringValue(change["move_path"]); move != "" {
				target.OperationType = "DELETE"
				targets = append(targets, target)
				target = codexFileTarget(move, "NEW", "native_file_change", stringValue(item["cwd"]), reference)
			}
			targets = append(targets, target)
		}
		// Compare structural payloads without source references or outcome fields.
		signature := make([]string, 0, len(targets))
		for _, target := range targets {
			signature = append(signature, target.Path+"\x00"+target.OperationType)
		}
		if previous, ok := metadata["change_signature"].([]string); ok && !reflect.DeepEqual(previous, signature) {
			metadata["file_change_conflict"] = true
		}
		metadata["change_signature"] = signature
		call.Output.Success = nil
		call.Output.Status = minitrace.ToolOutcomeUnknown
		switch status {
		case "completed":
			call.Output.SetSuccess(true)
		case "failed":
			call.Output.SetSuccess(false)
		case "cancelled", "canceled":
			call.Output.Status = minitrace.ToolOutcomeCancelled
		default:
			addCodexFileDiagnostic(call, "unknown_file_change_status")
		}
		if metadata["file_change_conflict"] == true {
			addCodexFileDiagnostic(call, "conflicting_file_change_evidence")
			call.Output.Success = nil
			call.Output.Status = minitrace.ToolOutcomeUnknown
			// Keep prior targets as attempts too, rather than losing conflicting paths.
			seen := map[string]bool{}
			for _, target := range targets {
				seen[target.Path+"\x00"+target.OperationType] = true
			}
			for _, target := range call.Input.FileTargets {
				if !seen[target.Path+"\x00"+target.OperationType] {
					targets = append(targets, target)
				}
			}
		}
		for i := range targets {
			targets[i].Status = "attempted"
			targets[i].Success = nil
			if call.Output.Succeeded() {
				targets[i].Status = "confirmed"
				success := true
				targets[i].Success = &success
			}
		}
		call.Input.FileTargets = targets
		if len(targets) > 0 {
			call.Input.FilePath = ptr(targets[0].Path)
		}
	}
	sort.SliceStable(calls, func(i, j int) bool {
		a, _ := codexInteger(mapValue(calls[i].FrameworkMetadata)["source_line"])
		b, _ := codexInteger(mapValue(calls[j].FrameworkMetadata)["source_line"])
		return a < b
	})
	return calls
}
