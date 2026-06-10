package minitracedb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeCacheKeyIsDeterministicAndVersioned(t *testing.T) {
	one := FingerprintContent("one.jsonl", []byte(`{"type":"session","id":"one"}`))
	two := FingerprintContent("two.jsonl", []byte(`{"type":"session","id":"two"}`))
	opts := DefaultCacheKeyOptions()
	opts.AutoConvert = true

	keyA, err := ComputeCacheKey([]SourceFingerprint{one, two}, opts)
	if err != nil {
		t.Fatalf("ComputeCacheKey A: %v", err)
	}
	keyB, err := ComputeCacheKey([]SourceFingerprint{two, one}, opts)
	if err != nil {
		t.Fatalf("ComputeCacheKey B: %v", err)
	}
	if keyA.Key != keyB.Key {
		t.Fatalf("cache key should be source-order stable: %s != %s", keyA.Key, keyB.Key)
	}
	if keyA.Options.SchemaVersion != SchemaVersion || keyA.Options.ImporterVersion != ImporterVersion || keyA.Options.ConverterVersion != ConverterVersion {
		t.Fatalf("cache key options did not include current versions: %#v", keyA.Options)
	}
	if len(keyA.Sources) != 2 {
		t.Fatalf("len(sources) = %d, want 2", len(keyA.Sources))
	}

	changedOpts := opts
	changedOpts.SchemaVersion = "normalized-sqlite-v3"
	keyC, err := ComputeCacheKey([]SourceFingerprint{one, two}, changedOpts)
	if err != nil {
		t.Fatalf("ComputeCacheKey C: %v", err)
	}
	if keyC.Key == keyA.Key {
		t.Fatalf("schema version change should change cache key: %s", keyA.Key)
	}

	changedSource := FingerprintContent("one.jsonl", []byte(`{"type":"session","id":"changed"}`))
	keyD, err := ComputeCacheKey([]SourceFingerprint{changedSource, two}, opts)
	if err != nil {
		t.Fatalf("ComputeCacheKey D: %v", err)
	}
	if keyD.Key == keyA.Key {
		t.Fatalf("source content change should change cache key: %s", keyA.Key)
	}
}

func TestFingerprintFileIncludesContentAndFileMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.minitrace.json")
	if err := os.WriteFile(path, []byte(`{"id":"session-1"}`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	fp, err := FingerprintFile(path)
	if err != nil {
		t.Fatalf("FingerprintFile: %v", err)
	}
	if fp.Kind != "file" || fp.Name != "session.minitrace.json" || fp.Path == "" {
		t.Fatalf("unexpected file fingerprint identity: %#v", fp)
	}
	if fp.SizeBytes <= 0 || fp.SHA256 == "" || fp.Fingerprint == "" || fp.ModTimeUTC == "" {
		t.Fatalf("file fingerprint missing metadata: %#v", fp)
	}
}
