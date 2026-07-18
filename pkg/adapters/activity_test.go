package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanJSONLLastTimestampUsesLatestValidCandidate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	content := "{\"timestamp\":\"2026-04-02T10:00:00Z\"}\nnot-json\n{\"timestamp\":\"2026-04-01T10:00:00Z\"}\n{\"timestamp\":\"invalid\"}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	latest, err := ScanJSONLLastTimestamp(path, func(record map[string]any) []string {
		timestamp, _ := record["timestamp"].(string)
		return []string{timestamp}
	})
	if err != nil {
		t.Fatalf("ScanJSONLLastTimestamp returned error: %v", err)
	}
	if latest != "2026-04-02T10:00:00Z" {
		t.Fatalf("expected latest valid timestamp, got %q", latest)
	}
}
