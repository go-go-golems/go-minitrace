package codex

import (
	"reflect"
	"testing"
)

func TestLiteralShellTargets(t *testing.T) {
	cases := []struct {
		script     string
		want       []shellTarget
		diagnostic bool
	}{
		{`printf first > first.txt; printf second >> 'second file.txt'; exit 7`, []shellTarget{{"first.txt", "MODIFY"}, {"second file.txt", "MODIFY"}}, false},
		{`printf 'quoted > not-a-path' > real`, []shellTarget{{"real", "MODIFY"}}, false},
		{"printf ok # > comment\ncat < input", []shellTarget{{"input", "READ"}}, false},
		{`printf ok > 'literal-$name'`, []shellTarget{{"literal-$name", "MODIFY"}}, false},
		{`printf ok > escaped\ space`, []shellTarget{{"escaped space", "MODIFY"}}, false},
		{`if false; then echo > never; fi`, nil, true},
		{"cat <<'EOF'\necho > body\nEOF", nil, true},
		{`echo > "$dynamic"`, nil, true},
		{`cat <(echo > hidden)`, nil, true},
		{`false && echo > skipped`, nil, true},
		{`cd elsewhere; echo > misplaced`, nil, true},
		{`eval 'cd elsewhere'; echo > misplaced`, nil, true},
		{`echo > output 2>&1`, nil, true},
		{`echo 'unclosed > false`, nil, true},
		{`rg 'needle' search-root`, nil, false},
	}
	for _, c := range cases {
		got, diagnostic := literalShellTargets(c.script)
		if !reflect.DeepEqual(got, c.want) || (diagnostic != "") != c.diagnostic {
			t.Errorf("%q: targets=%v diagnostic=%q", c.script, got, diagnostic)
		}
	}
}

func TestCodexStructuralTargetsAndNoOpaqueWrapperInference(t *testing.T) {
	command := executionRecord("item_completed", "command", "completed", ptr(0))
	item := mapValue(mapValue(command["payload"])["item"])
	item["command"] = []any{"/bin/sh", "-lc", "printf x > first; printf y >> second"}
	wrapper := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "call_id": "wrapper", "name": "exec", "input": `if(false){tools.exec_command({cmd:"echo > invented"})}`}}
	patch := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "call_id": "patch", "name": "apply_patch", "input": "*** Begin Patch\n*** Add File: one\n+x\n*** Update File: two\n@@\n-x\n+y\n*** Delete File: three\n*** End Patch"}}
	session := convertMessageTestRecords(t, []map[string]any{wrapper, patch, command})
	for _, call := range session.ToolCalls {
		switch call.ID {
		case "wrapper":
			if len(call.Input.FileTargets) != 0 || call.Input.FilePath != nil {
				t.Fatal("opaque wrapper created file evidence")
			}
		case "patch":
			if len(call.Input.FileTargets) != 3 {
				t.Fatal("patch lost targets")
			}
			for _, target := range call.Input.FileTargets {
				if target.Resolved || target.Success != nil || target.Status != "attempted" {
					t.Fatalf("patch inferred cwd/effect: %#v", target)
				}
			}
		default:
			if len(call.Input.FileTargets) != 2 {
				t.Fatal("execution redirects lost")
			}
			for _, target := range call.Input.FileTargets {
				if !target.Resolved || target.CWD != "/workspace/test" || target.Success != nil || target.Status != "attempted" {
					t.Fatalf("shell success leaked into file effect: %#v", target)
				}
			}
		}
	}
}

func TestNativeFileChangesAreDeduplicatedAndConflictsStayAttempts(t *testing.T) {
	record := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "turn_id": "turn", "item": map[string]any{"type": "FileChange", "id": "change", "status": "completed", "changes": map[string]any{
		"/cwd/old":   map[string]any{"type": "update", "move_path": "/cwd/new"},
		"/cwd/added": map[string]any{"type": "add", "content": "not copied into metadata"},
	}}}}
	session := convertMessageTestRecords(t, []map[string]any{record, record})
	if len(session.ToolCalls) != 1 || len(session.ToolCalls[0].Input.FileTargets) != 3 {
		t.Fatal("native file notification duplicated or move target lost")
	}
	call := session.ToolCalls[0]
	if call.EmittingTurnIndex != nil || call.RecordKind != "file_change" {
		t.Fatal("native file record falsely linked/classified")
	}
	for _, target := range call.Input.FileTargets {
		if target.Status != "confirmed" || target.Success == nil || !*target.Success {
			t.Fatalf("native effect not confirmed: %#v", target)
		}
	}
	conflict := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "item": map[string]any{"type": "FileChange", "id": "change", "status": "failed", "changes": map[string]any{"/cwd/other": map[string]any{"type": "delete"}}}}}
	session = convertMessageTestRecords(t, []map[string]any{record, conflict, record})
	call = session.ToolCalls[0]
	if call.Output.Success != nil || len(call.Input.FileTargets) != 4 {
		t.Fatal("conflicting file evidence lost")
	}
	for _, target := range call.Input.FileTargets {
		if target.Success != nil || target.Status != "attempted" {
			t.Fatal("conflict became confirmed effect")
		}
	}
}
