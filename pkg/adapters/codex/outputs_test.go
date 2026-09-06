package codex

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestCodexTypedOutputDecoding(t *testing.T) {
	tests := []struct {
		name   string
		output any
		text   string
		code   *int
	}{
		{"raw JSON stdout", `{"value":42}`, `{"value":42}`, nil},
		{"unrecognized metadata", `{"output":"data","metadata":{"other":1}}`, `{"output":"data","metadata":{"other":1}}`, nil},
		{"plain stdout phrase", "Process exited with code 0", "Process exited with code 0", nil},
		{"known metadata", `{"output":"hello","metadata":{"exit_code":0,"duration_seconds":0.25}}`, "hello", ptr(0)},
		{"typed chunk", []any{map[string]any{"type": "input_text", "text": `{"chunk_id":"test","output":"failed","exit_code":7,"wall_time_seconds":0.1}`}}, "failed", ptr(7)},
		{"typed fulfilled chunk", []any{map[string]any{"type": "input_text", "text": `{"status":"fulfilled","value":{"chunk_id":"test","output":"ok","exit_code":0}}`}}, "ok", ptr(0)},
		{"renderer envelope", "Wall time: 0.25 seconds\nProcess exited with code 0\nOutput:\nhello", "hello", ptr(0)},
		{"malformed wall time", "Wall time: \nProcess exited with code 1\nOutput:\nerror", "error", ptr(1)},
		{"invalid exit type", `{"output":"data","metadata":{"exit_code":"zero"}}`, `{"output":"data","metadata":{"exit_code":"zero"}}`, nil},
		{"image block", []any{map[string]any{"type": "input_image", "image_url": "data:image/png;base64,SECRET"}}, "[image output]", nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			call := buildCodexResponseToolCall("call", nil, "turn", map[string]any{"name": "exec_command"})
			applyCodexFunctionOutput(&call, test.output)
			if call.Output.Result == nil || *call.Output.Result != test.text {
				t.Fatalf("got result %v, want %q", call.Output.Result, test.text)
			}
			if test.code == nil {
				if call.Output.Success != nil || call.Output.ExitCode != nil {
					t.Fatalf("invented outcome: %#v", call.Output)
				}
			} else if call.Output.ExitCode == nil || *call.Output.ExitCode != *test.code || call.Output.Succeeded() != (*test.code == 0) {
				t.Fatalf("lost outcome: %#v", call.Output)
			}
		})
	}
}

func TestCodexWrapperChildrenRemainIndependent(t *testing.T) {
	blocks := []any{
		map[string]any{"type": "input_text", "text": `{"chunk_id":"a","output":"ok","exit_code":0}`},
		map[string]any{"type": "input_text", "text": `{"chunk_id":"b","output":"error","exit_code":9}`},
	}
	for _, name := range []string{"exec", "exec_command"} {
		call := buildCodexResponseToolCall("call", nil, "turn", map[string]any{"name": name})
		applyCodexFunctionOutput(&call, blocks)
		if call.Output.Success != nil || call.Output.ExitCode != nil || *call.Output.Result != "ok\nerror" {
			t.Fatalf("multiple outputs wrongly flattened to one outcome: %#v", call.Output)
		}
		evidence := mapValue(call.FrameworkMetadata)["output_blocks"].([]codexOutputEvidence)
		if len(evidence) != 2 || *evidence[0].ExitCode != 0 || *evidence[1].ExitCode != 9 {
			t.Fatalf("independent child outcomes lost: %#v", evidence)
		}
	}
	call := buildCodexResponseToolCall("wrapper", nil, "turn", map[string]any{"name": "exec"})
	applyCodexFunctionOutput(&call, blocks[:1])
	if call.Output.Success != nil || call.Output.ExitCode != nil {
		t.Fatal("a single printed child outcome is not the wrapper's own outcome")
	}
}

func TestCodexOutputBeforeCallAndTruncation(t *testing.T) {
	text := strings.Repeat("x", minitrace.TruncateLimit+1000)
	envelope, err := json.Marshal(map[string]any{"output": text, "metadata": map[string]any{"exit_code": 7, "duration_seconds": 0.5}})
	if err != nil {
		t.Fatal(err)
	}
	session := convertMessageTestRecords(t, []map[string]any{
		{"type": "response_item", "payload": map[string]any{"type": "function_call_output", "call_id": "c", "output": []any{map[string]any{"type": "input_text", "text": string(envelope)}}}},
		{"type": "response_item", "payload": map[string]any{"type": "function_call", "call_id": "c", "name": "exec_command", "arguments": `{"cmd":"test"}`}},
		{"type": "response_item", "payload": map[string]any{"type": "function_call", "call_id": "pending", "name": "exec_command", "arguments": `{"cmd":"unknown"}`}},
	})
	out := session.ToolCalls[0].Output
	if !out.Failed() || out.ExitCode == nil || *out.ExitCode != 7 || out.DurationMS == nil || *out.DurationMS != 500 {
		t.Fatalf("metadata lost before truncation: %#v", out)
	}
	if !out.Truncated || out.FullBytes == nil || *out.FullBytes != len(text) || out.FullHash == nil {
		t.Fatalf("missing full output evidence: %#v", out)
	}
	if out.FullReference == nil || *out.FullReference != "/synthetic/session.jsonl#L1" {
		t.Fatalf("output source reference must name the output line: %v", out.FullReference)
	}
	if pending := session.ToolCalls[1].Output; pending.Success != nil || pending.OutcomeStatus() != minitrace.ToolOutcomePending {
		t.Fatalf("missing output became success: %#v", pending)
	}
}
