package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFingerprintSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.jsonl")
	payload := []byte("fixture source\n")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("writing source: %v", err)
	}

	digest, size, normalizedPath, err := FingerprintSource(path)
	if err != nil {
		t.Fatalf("FingerprintSource returned error: %v", err)
	}
	if digest != "7cefb9aa217c81555befc729d7fa5d70dbc83bfe20d91eaac7e8af9aee481432" {
		t.Fatalf("digest = %q", digest)
	}
	if size != int64(len(payload)) {
		t.Fatalf("size = %d, want %d", size, len(payload))
	}
	if !filepath.IsAbs(normalizedPath) {
		t.Fatalf("normalized path = %q, want absolute", normalizedPath)
	}
}
