package minitracejs

import (
	"strings"
	"testing"
)

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
