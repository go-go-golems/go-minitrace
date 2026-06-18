package minitracejs

import (
	"strings"
	"testing"
)

func TestImportBuilderConvertSupportsCopilotJSONL(t *testing.T) {
	jsonl := strings.Join([]string{
		`{"type":"session.start","id":"ev-1","timestamp":"2026-06-17T10:00:00Z","data":{"sessionId":"copilot-import","copilotVersion":"1.0.0"}}`,
		`{"type":"user.message","id":"ev-2","timestamp":"2026-06-17T10:00:01Z","data":{"content":"Inspect README.md","interactionId":"int-1"}}`,
		`{"type":"assistant.message","id":"ev-3","timestamp":"2026-06-17T10:00:02Z","data":{"content":"I inspected it.","model":"gpt-test","turnId":"turn-1"}}`,
	}, "\n")

	builder := NewImportBuilder()
	builder.content = jsonl
	builder.name = "copilot-events.jsonl"
	converted, err := builder.Converted()
	if err != nil {
		t.Fatalf("Converted: %v", err)
	}
	if converted.SessionID != "copilot-import" || converted.Adapter != "copilot" || converted.Format != "copilot-jsonl" {
		t.Fatalf("unexpected converted summary: %+v", converted)
	}
	if converted.TurnCount != 2 {
		t.Fatalf("expected 2 turns, got %d", converted.TurnCount)
	}
}

func TestImportBuilderPreviewSummarizesConvertedPiSession(t *testing.T) {
	jsonl := strings.Join([]string{
		`{"type":"session","id":"pi-preview","version":3,"timestamp":"2026-06-10T12:00:00Z","cwd":"/tmp/project"}`,
		`{"type":"model_change","provider":"claude-agent-sdk","modelId":"claude-opus-4-6","timestamp":"2026-06-10T12:00:01Z"}`,
		`{"type":"message","message":{"role":"user","timestamp":"2026-06-10T12:00:02Z","content":[{"type":"text","text":"Please inspect the repo."}]}}`,
		`{"type":"message","message":{"role":"assistant","timestamp":"2026-06-10T12:00:03Z","model":"claude-opus-4-6","provider":"claude-agent-sdk","content":[{"type":"thinking","thinking":"Need to list files."},{"type":"toolCall","id":"call-1","name":"bash","arguments":{"command":"ls -la"}}]}}`,
		`{"type":"message","message":{"role":"toolResult","timestamp":"2026-06-10T12:00:04Z","toolCallId":"call-1","toolName":"bash","isError":false,"content":"total 0"}}`,
	}, "\n")

	builder := NewImportBuilder()
	builder.content = jsonl
	builder.name = "pi-preview.jsonl"
	preview, err := builder.Preview()
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if preview.SessionID != "pi-preview" || preview.Adapter != "pi" || preview.Format != "pi-jsonl" {
		t.Fatalf("unexpected identity: %+v", preview)
	}
	if preview.TurnCount != 2 || preview.ToolCallCount != 1 {
		t.Fatalf("unexpected counts: turns=%d tools=%d", preview.TurnCount, preview.ToolCallCount)
	}
	if preview.EventCount != 1 || preview.EventCounts["model_change"] != 1 || len(preview.SampleEvents) != 1 {
		t.Fatalf("unexpected event preview: count=%d counts=%+v samples=%+v", preview.EventCount, preview.EventCounts, preview.SampleEvents)
	}
	if !preview.HasThinking {
		t.Fatalf("expected thinking signal")
	}
	if preview.RoleCounts["user"] != 1 || preview.RoleCounts["assistant"] != 1 {
		t.Fatalf("unexpected role counts: %+v", preview.RoleCounts)
	}
	if preview.ToolCounts["bash"] != 1 {
		t.Fatalf("unexpected tool counts: %+v", preview.ToolCounts)
	}
	if len(preview.SampleTurns) != 2 || len(preview.SampleTools) != 1 {
		t.Fatalf("unexpected samples: turns=%d tools=%d", len(preview.SampleTurns), len(preview.SampleTools))
	}
	if preview.SampleTools[0].Command != "ls -la" || !preview.SampleTools[0].HasResult {
		t.Fatalf("unexpected tool sample: %+v", preview.SampleTools[0])
	}
}
