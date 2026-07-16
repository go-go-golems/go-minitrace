package validate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
)

const (
	CheckArchive  = "archive"
	CheckManifest = "manifest"
	CheckSource   = "source"
	CheckReceipt  = "receipt"
)

// DetectArchiveRoot accepts an archive root or any path below its active tree.
func DetectArchiveRoot(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		absolute = filepath.Dir(absolute)
	}
	for current := absolute; ; current = filepath.Dir(current) {
		if info, statErr := os.Stat(filepath.Join(current, "active")); statErr == nil && info.IsDir() {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return "", errors.Errorf("no minitrace archive root found from %s", path)
}

type archiveEntry struct {
	ID            string `json:"id"`
	Period        string `json:"period"`
	Path          string `json:"file_path"`
	FileSizeBytes int64  `json:"file_size_bytes"`
}

type periodManifestDocument struct {
	Period   string         `json:"period"`
	Sessions []archiveEntry `json:"sessions"`
}

type rootManifestDocument struct {
	Periods []struct {
		Period string `json:"period"`
		Path   string `json:"path"`
	} `json:"periods"`
}

// ValidateArchive runs native integrity checks over session archives,
// manifests, source fingerprints, and conversion receipts. An empty checks
// list enables every check.
func ValidateArchive(path string, checks []string) ([]Finding, error) {
	for _, check := range checks {
		if check != CheckArchive && check != CheckManifest && check != CheckSource && check != CheckReceipt {
			return nil, errors.Errorf("unknown archive validation check %q", check)
		}
	}
	root, err := DetectArchiveRoot(path)
	if err != nil {
		return nil, err
	}
	enabled := enabledChecks(checks)
	findings := make([]Finding, 0)
	archives := map[string]string{}
	periodByID := map[string]string{}

	err = filepath.WalkDir(filepath.Join(root, "active"), func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".minitrace.json") {
			return nil
		}
		payload, readErr := os.ReadFile(filePath)
		if readErr != nil {
			findings = append(findings, finding("archive-read-error", filePath, "", readErr.Error()))
			return nil
		}
		var session minitrace.Session
		if decodeErr := json.Unmarshal(payload, &session); decodeErr != nil {
			findings = append(findings, finding("archive-json-invalid", filePath, "", decodeErr.Error()))
			return nil
		}
		archives[session.ID] = filePath
		period := filepath.Base(filepath.Dir(filePath))
		periodByID[session.ID] = period
		if enabled[CheckArchive] {
			expectedName := minitrace.SanitizeID(session.ID) + ".minitrace.json"
			if entry.Name() != expectedName {
				findings = append(findings, finding("archive-filename-id-mismatch", filePath, session.ID, "expected "+expectedName))
			}
			expectedPeriod := "unknown"
			if session.Timing.StartedAt != nil && len(*session.Timing.StartedAt) >= 7 {
				expectedPeriod = minitrace.SanitizePeriod((*session.Timing.StartedAt)[:7])
			}
			if period != expectedPeriod {
				findings = append(findings, finding("archive-period-mismatch", filePath, session.ID, "expected period "+expectedPeriod))
			}
		}
		if enabled[CheckSource] && session.Provenance.SourceFingerprint != nil && session.Provenance.SourcePath != nil {
			if sourcePayload, sourceErr := os.ReadFile(*session.Provenance.SourcePath); sourceErr == nil {
				sum := sha256.Sum256(sourcePayload)
				actual := hex.EncodeToString(sum[:])
				if actual != *session.Provenance.SourceFingerprint {
					findings = append(findings, finding("source-fingerprint-mismatch", filePath, session.ID, "source bytes no longer match recorded fingerprint"))
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "walking archive")
	}

	if enabled[CheckManifest] {
		findings = append(findings, validateManifests(root, archives, periodByID)...)
	}
	if enabled[CheckReceipt] {
		findings = append(findings, validateReceipts(root)...)
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path == findings[j].Path {
			return findings[i].Code < findings[j].Code
		}
		return findings[i].Path < findings[j].Path
	})
	return findings, nil
}

func enabledChecks(checks []string) map[string]bool {
	ret := map[string]bool{}
	if len(checks) == 0 {
		for _, check := range []string{CheckArchive, CheckManifest, CheckSource, CheckReceipt} {
			ret[check] = true
		}
		return ret
	}
	for _, check := range checks {
		ret[check] = true
	}
	return ret
}

func finding(code, path, sessionID, detail string) Finding {
	return Finding{Code: code, Severity: SeverityError, Path: path, SessionID: sessionID, Detail: detail}
}

func validateManifests(root string, archives, periodByID map[string]string) []Finding {
	findings := []Finding{}
	rootPath := filepath.Join(root, "manifest.json")
	payload, err := os.ReadFile(rootPath)
	if err != nil {
		return append(findings, finding("root-manifest-missing", rootPath, "", err.Error()))
	}
	var rootManifest rootManifestDocument
	if err := json.Unmarshal(payload, &rootManifest); err != nil {
		return append(findings, finding("root-manifest-invalid", rootPath, "", err.Error()))
	}
	manifestIDs := map[string]bool{}
	for _, period := range rootManifest.Periods {
		manifestPath := filepath.Join(root, filepath.FromSlash(period.Path))
		periodPayload, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			findings = append(findings, finding("period-manifest-missing", manifestPath, "", readErr.Error()))
			continue
		}
		var manifest periodManifestDocument
		if decodeErr := json.Unmarshal(periodPayload, &manifest); decodeErr != nil {
			findings = append(findings, finding("period-manifest-invalid", manifestPath, "", decodeErr.Error()))
			continue
		}
		if manifest.Period != period.Period {
			findings = append(findings, finding("period-manifest-period-mismatch", manifestPath, "", "root and period manifest disagree"))
		}
		for _, session := range manifest.Sessions {
			manifestIDs[session.ID] = true
			archivePath, ok := archives[session.ID]
			if !ok {
				findings = append(findings, finding("orphan-manifest-entry", manifestPath, session.ID, "manifest entry has no archive"))
				continue
			}
			if periodByID[session.ID] != period.Period {
				findings = append(findings, finding("manifest-period-mismatch", manifestPath, session.ID, "archive is in a different period"))
			}
			if info, statErr := os.Stat(archivePath); statErr == nil && session.FileSizeBytes != info.Size() {
				findings = append(findings, finding("manifest-size-mismatch", manifestPath, session.ID, "recorded file size differs from archive"))
			}
		}
	}
	for id, path := range archives {
		if !manifestIDs[id] {
			findings = append(findings, finding("orphan-archive", path, id, "archive is absent from manifests"))
		}
	}
	return findings
}

func validateReceipts(root string) []Finding {
	findings := []Finding{}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || filepath.Ext(path) != ".json" || strings.HasSuffix(path, ".minitrace.json") || entry.Name() == "manifest.json" {
			return nil
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var header struct {
			Schema     string `json:"schema"`
			StartedAt  string `json:"started_at"`
			FinishedAt string `json:"finished_at"`
			Complete   *bool  `json:"complete"`
		}
		if json.Unmarshal(payload, &header) != nil || header.Schema != "go-minitrace-conversion-run-v1" {
			return nil
		}
		if header.StartedAt == "" || header.FinishedAt == "" || header.Complete == nil {
			findings = append(findings, finding("conversion-receipt-invalid", path, "", "receipt lacks timestamps or completion state"))
		}
		return nil
	})
	return findings
}
