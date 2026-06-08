package minitracedb

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	ImporterVersion  = "minitracedb-importer-v1"
	ConverterVersion = "minitracedb-auto-converter-v1"
)

type CacheKeyOptions struct {
	SchemaVersion    string `json:"schemaVersion"`
	ImporterVersion  string `json:"importerVersion"`
	ConverterVersion string `json:"converterVersion"`
	Backend          string `json:"backend"`
	Storage          string `json:"storage"`
	AutoConvert      bool   `json:"autoConvert"`
}

type SourceFingerprint struct {
	Kind        string `json:"kind"`
	Name        string `json:"name,omitempty"`
	Path        string `json:"path,omitempty"`
	SizeBytes   int64  `json:"sizeBytes"`
	ModTimeUTC  string `json:"modTimeUtc,omitempty"`
	SHA256      string `json:"sha256"`
	Fingerprint string `json:"fingerprint"`
}

type CacheKey struct {
	Key     string              `json:"key"`
	Options CacheKeyOptions     `json:"options"`
	Sources []SourceFingerprint `json:"sources"`
}

func DefaultCacheKeyOptions() CacheKeyOptions {
	return CacheKeyOptions{SchemaVersion: SchemaVersion, ImporterVersion: ImporterVersion, ConverterVersion: ConverterVersion, Backend: "sqlite", Storage: "memory"}
}

func FingerprintFile(path string) (SourceFingerprint, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return SourceFingerprint{}, fmt.Errorf("read source for fingerprint %s: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return SourceFingerprint{}, fmt.Errorf("stat source for fingerprint %s: %w", path, err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	fp := fingerprintPayload("file", filepath.Base(path), absPath, payload)
	fp.SizeBytes = info.Size()
	fp.ModTimeUTC = info.ModTime().UTC().Format(time.RFC3339Nano)
	return fp, nil
}

func FingerprintContent(name string, payload []byte) SourceFingerprint {
	return fingerprintPayload("content", name, "", payload)
}

func ComputeCacheKey(sources []SourceFingerprint, opts CacheKeyOptions) (CacheKey, error) {
	if opts.SchemaVersion == "" {
		opts.SchemaVersion = SchemaVersion
	}
	if opts.ImporterVersion == "" {
		opts.ImporterVersion = ImporterVersion
	}
	if opts.ConverterVersion == "" {
		opts.ConverterVersion = ConverterVersion
	}
	if opts.Backend == "" {
		opts.Backend = "sqlite"
	}
	if opts.Storage == "" {
		opts.Storage = "memory"
	}
	stableSources := append([]SourceFingerprint(nil), sources...)
	sort.SliceStable(stableSources, func(i, j int) bool {
		if stableSources[i].Kind != stableSources[j].Kind {
			return stableSources[i].Kind < stableSources[j].Kind
		}
		if stableSources[i].Path != stableSources[j].Path {
			return stableSources[i].Path < stableSources[j].Path
		}
		if stableSources[i].Name != stableSources[j].Name {
			return stableSources[i].Name < stableSources[j].Name
		}
		return stableSources[i].SHA256 < stableSources[j].SHA256
	})
	payload := struct {
		Options CacheKeyOptions     `json:"options"`
		Sources []SourceFingerprint `json:"sources"`
	}{Options: opts, Sources: stableSources}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return CacheKey{}, fmt.Errorf("encode cache key payload: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return CacheKey{Key: "mtdb-" + hex.EncodeToString(sum[:]), Options: opts, Sources: stableSources}, nil
}

func fingerprintPayload(kind, name, path string, payload []byte) SourceFingerprint {
	sum := sha256.Sum256(payload)
	sha := hex.EncodeToString(sum[:])
	return SourceFingerprint{Kind: kind, Name: name, Path: path, SizeBytes: int64(len(payload)), SHA256: sha, Fingerprint: kind + ":sha256:" + sha}
}
