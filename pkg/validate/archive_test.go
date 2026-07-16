package validate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func createArchiveFixture(t *testing.T) (string, *minitrace.Session, *minitrace.SessionIndexEntry) {
	t.Helper()
	root := t.TempDir()
	session := minitrace.BuildSessionSkeleton("session-one", "test", "test-v1", "test")
	startedAt := "2026-07-15T12:00:00Z"
	session.Timing.StartedAt = &startedAt
	entry, err := minitrace.WriteSession(&session, root)
	if err != nil {
		t.Fatalf("WriteSession: %v", err)
	}
	if err := minitrace.WriteManifests([]*minitrace.SessionIndexEntry{entry}, root); err != nil {
		t.Fatalf("WriteManifests: %v", err)
	}
	return root, &session, entry
}

func TestDetectArchiveRootFromSessionPath(t *testing.T) {
	root, _, entry := createArchiveFixture(t)
	detected, err := DetectArchiveRoot(entry.FilePath)
	if err != nil {
		t.Fatalf("DetectArchiveRoot: %v", err)
	}
	if detected != root {
		t.Fatalf("expected %s, got %s", root, detected)
	}
}

func TestValidateArchiveAcceptsConsistentArchive(t *testing.T) {
	root, _, _ := createArchiveFixture(t)
	findings, err := ValidateArchive(root, nil)
	if err != nil {
		t.Fatalf("ValidateArchive: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("unexpected findings: %+v", findings)
	}
}

func TestValidateArchiveFindsFilenamePeriodAndOrphanErrors(t *testing.T) {
	root, session, entry := createArchiveFixture(t)
	wrongDir := filepath.Join(root, "active", "2025-01")
	if err := os.MkdirAll(wrongDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wrongPath := filepath.Join(wrongDir, "wrong-name.minitrace.json")
	payload, _ := json.Marshal(session)
	if err := os.WriteFile(wrongPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(entry.FilePath); err != nil {
		t.Fatal(err)
	}
	findings, err := ValidateArchive(root, nil)
	if err != nil {
		t.Fatalf("ValidateArchive: %v", err)
	}
	codes := map[string]bool{}
	for _, finding := range findings {
		codes[finding.Code] = true
	}
	for _, expected := range []string{"archive-filename-id-mismatch", "archive-period-mismatch", "manifest-period-mismatch", "manifest-size-mismatch"} {
		if !codes[expected] {
			t.Fatalf("missing %s in %+v", expected, findings)
		}
	}
}

func TestValidateArchiveFindsDuplicateIDs(t *testing.T) {
	root, _, entry := createArchiveFixture(t)
	payload, err := os.ReadFile(entry.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	duplicateDir := filepath.Join(root, "active", "2025-01")
	if err := os.MkdirAll(duplicateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(duplicateDir, filepath.Base(entry.FilePath)), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	findings, err := ValidateArchive(root, []string{CheckArchive})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding.Code == "duplicate-archive-id" {
			return
		}
	}
	t.Fatalf("missing duplicate finding: %+v", findings)
}

func TestValidateArchiveChecksManifestFilePath(t *testing.T) {
	root, _, _ := createArchiveFixture(t)
	manifestPath := filepath.Join(root, "active", "2026-07", "manifest.json")
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatal(err)
	}
	sessions := manifest["sessions"].([]any)
	sessions[0].(map[string]any)["file_path"] = "missing.minitrace.json"
	payload, _ = json.Marshal(manifest)
	if err := os.WriteFile(manifestPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	findings, err := ValidateArchive(root, []string{CheckManifest})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding.Code == "manifest-path-mismatch" {
			return
		}
	}
	t.Fatalf("missing manifest path finding: %+v", findings)
}

func TestValidateArchiveChecksConversionReceipts(t *testing.T) {
	root, _, _ := createArchiveFixture(t)
	receipt := filepath.Join(root, "runs", "bad.json")
	if err := os.MkdirAll(filepath.Dir(receipt), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receipt, []byte(`{"schema":"go-minitrace-conversion-run-v1","complete":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := ValidateArchive(root, []string{CheckReceipt})
	if err != nil {
		t.Fatalf("ValidateArchive: %v", err)
	}
	foundInvalid := false
	for _, finding := range findings {
		foundInvalid = foundInvalid || finding.Code == "conversion-receipt-invalid"
	}
	if !foundInvalid {
		t.Fatalf("unexpected findings: %+v", findings)
	}
}
