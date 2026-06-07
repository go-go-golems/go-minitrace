package minitracedb

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestLoadSessionContentAutoConvertsPiJSONL(t *testing.T) {
	loaded, err := LoadSessionContentAuto(jsonl(t, []map[string]any{
		{"type": "session", "id": "pi-1", "version": 3, "timestamp": "2026-03-29T12:00:00Z", "cwd": "/tmp/project"},
		{"type": "message", "timestamp": "2026-03-29T12:00:01Z", "message": map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "Read app.go"}}}},
	}), LoadOptions{SourceName: "pi.jsonl", AutoConvert: true})
	if err != nil {
		t.Fatalf("LoadSessionContentAuto: %v", err)
	}
	if loaded.Format != "pi-jsonl" || loaded.Adapter != "pi" || loaded.Session.ID != "pi-1" {
		t.Fatalf("unexpected load result: %#v", loaded)
	}
}

func TestLoadSessionContentAutoConvertsCodexJSONL(t *testing.T) {
	loaded, err := LoadSessionContentAuto(jsonl(t, []map[string]any{
		{"type": "session_meta", "payload": map[string]any{"id": "codex-1", "cwd": "/tmp/project"}},
		{"type": "event_msg", "timestamp": "2026-03-29T12:00:00Z", "payload": map[string]any{"type": "user_message", "message": "hello"}},
	}), LoadOptions{SourceName: "codex.jsonl", AutoConvert: true})
	if err != nil {
		t.Fatalf("LoadSessionContentAuto: %v", err)
	}
	if loaded.Format != "codex-jsonl" || loaded.Adapter != "codex" || loaded.Session.ID != "codex-1" {
		t.Fatalf("unexpected load result: %#v", loaded)
	}
}

func TestLoadSessionContentAutoConvertsClaudeCodeJSONL(t *testing.T) {
	loaded, err := LoadSessionContentAuto(jsonl(t, []map[string]any{
		{"type": "system", "timestamp": "2026-03-29T12:00:00Z", "cwd": "/tmp/project", "message": map[string]any{"content": "system"}},
		{"type": "user", "timestamp": "2026-03-29T12:00:01Z", "message": map[string]any{"content": "Read app.go"}},
	}), LoadOptions{SourceName: "claude.jsonl", AutoConvert: true})
	if err != nil {
		t.Fatalf("LoadSessionContentAuto: %v", err)
	}
	if loaded.Format != "claude-code-jsonl" || loaded.Adapter != "claude-code" || loaded.Session.ID != "claude" {
		t.Fatalf("unexpected load result: format=%s adapter=%s id=%s", loaded.Format, loaded.Adapter, loaded.Session.ID)
	}
}

func TestLoadSessionContentAutoRequiresAutoConvertForJSONL(t *testing.T) {
	_, err := LoadSessionContentAuto(jsonl(t, []map[string]any{{"type": "session", "id": "pi-1"}}), LoadOptions{SourceName: "pi.jsonl"})
	if err == nil {
		t.Fatalf("expected disabled autoconvert error")
	}
}

func jsonl(t *testing.T, records []map[string]any) []byte {
	t.Helper()
	var b bytes.Buffer
	for _, record := range records {
		payload, err := json.Marshal(record)
		if err != nil {
			t.Fatalf("marshal record: %v", err)
		}
		b.Write(payload)
		b.WriteByte('\n')
	}
	return b.Bytes()
}
