package minitracejs

import (
	"strings"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
)

func TestPreviewLoadedSessionWithOptionsPrivacyAndSampleLimit(t *testing.T) {
	session := minitrace.BuildSessionSkeleton("preview-options", "codex", "codex-session-jsonl-v1", "test")
	longContent := strings.Repeat("alpha ", 80)
	session.Turns = []minitrace.Turn{
		minitrace.BuildTurn(0, nil, "user", nil, longContent),
		minitrace.BuildTurn(1, nil, "assistant", nil, "second turn"),
	}
	command := "go test ./..."
	session.ToolCalls = []minitrace.ToolCall{
		minitrace.BuildToolCall("tool-1", nil, nil, "exec_command", "EXECUTE", nil, &command, map[string]any{"cmd": command}, true, "ok", nil, nil, nil, nil, nil, nil),
		minitrace.BuildToolCall("tool-2", nil, nil, "read_file", "READ", nil, nil, nil, true, "ok", nil, nil, nil, nil, nil, nil),
	}
	loaded := &minitracedb.LoadedSession{Session: &session, Format: "codex-jsonl", Adapter: "codex"}

	structural := PreviewLoadedSessionWithOptions(loaded, PreviewOptions{SampleLimit: 1, Privacy: "structural"})
	if structural.TurnCount != 2 || structural.ToolCallCount != 2 {
		t.Fatalf("expected full counts despite sample limit, got turns=%d tools=%d", structural.TurnCount, structural.ToolCallCount)
	}
	if len(structural.SampleTurns) != 1 || len(structural.SampleTools) != 1 {
		t.Fatalf("expected one sampled turn/tool, got turns=%d tools=%d", len(structural.SampleTurns), len(structural.SampleTools))
	}
	if structural.SampleTurns[0].Preview != "" || structural.SampleTools[0].Command != "" {
		t.Fatalf("expected structural privacy to suppress snippets, got turn=%q command=%q", structural.SampleTurns[0].Preview, structural.SampleTools[0].Command)
	}

	full := PreviewLoadedSessionWithOptions(loaded, PreviewOptions{SampleLimit: 2, Privacy: "full"})
	if len(full.SampleTurns) != 2 || len(full.SampleTools) != 2 {
		t.Fatalf("expected two sampled turns/tools, got turns=%d tools=%d", len(full.SampleTurns), len(full.SampleTools))
	}
	if full.SampleTurns[0].Preview != strings.TrimSpace(longContent) {
		t.Fatalf("expected full privacy to retain full text")
	}
	if full.SampleTools[0].Command != command {
		t.Fatalf("expected full privacy to retain command, got %q", full.SampleTools[0].Command)
	}
}
