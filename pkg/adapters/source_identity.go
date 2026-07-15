package adapters

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// SourceIdentity describes the native source selected for conversion. It is
// intentionally adapter-neutral: adapters establish NativeSessionID, lineage,
// and role from their own source format, while this package owns normalized
// path and byte-level fingerprint evidence.
type ConversionWarning struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	RecordIndex int    `json:"record_index"`
}

type SourceIdentity struct {
	NativeSessionID  string              `json:"native_session_id,omitempty"`
	ParentSessionID  string              `json:"parent_session_id,omitempty"`
	SourcePath       string              `json:"source_path"`
	SourceFormat     string              `json:"source_format,omitempty"`
	WorkingDirectory string              `json:"working_directory,omitempty"`
	Role             string              `json:"role,omitempty"`
	IdentityBasis    string              `json:"identity_basis,omitempty"`
	SHA256           string              `json:"sha256,omitempty"`
	SizeBytes        int64               `json:"size_bytes,omitempty"`
	Warnings         []ConversionWarning `json:"warnings,omitempty"`
}

// FingerprintSource returns byte-level evidence for one source file. It does
// not inspect a framework-specific header; callers supply those fields after
// parsing their source format.
func FingerprintSource(path string) (string, int64, string, error) {
	normalizedPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", 0, "", fmt.Errorf("normalize source path %q: %w", path, err)
	}
	payload, err := os.ReadFile(normalizedPath)
	if err != nil {
		return "", 0, normalizedPath, fmt.Errorf("read source %q: %w", normalizedPath, err)
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), int64(len(payload)), normalizedPath, nil
}
