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
		Period       string `json:"period"`
		Path         string `json:"path"`
		SessionCount int    `json:"session_count"`
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
	archiveRoot, err := os.OpenRoot(root)
	if err != nil {
		return nil, errors.Wrap(err, "opening archive root")
	}
	defer func() { _ = archiveRoot.Close() }()
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
		relativePath, relativeErr := filepath.Rel(root, filePath)
		if relativeErr != nil {
			return relativeErr
		}
		payload, readErr := archiveRoot.ReadFile(relativePath)
		if readErr != nil {
			findings = append(findings, finding("archive-read-error", filePath, "", readErr.Error()))
			return nil
		}
		var session minitrace.Session
		if decodeErr := json.Unmarshal(payload, &session); decodeErr != nil {
			findings = append(findings, finding("archive-json-invalid", filePath, "", decodeErr.Error()))
			return nil
		}
		period := filepath.Base(filepath.Dir(filePath))
		if previousPath, duplicate := archives[session.ID]; duplicate {
			findings = append(findings, finding("duplicate-archive-id", filePath, session.ID, "also present at "+previousPath))
		} else {
			archives[session.ID] = filePath
			periodByID[session.ID] = period
		}
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
		if enabled[CheckSource] && session.Provenance.SourcePath != nil {
			if session.Provenance.SourceFingerprint == nil || *session.Provenance.SourceFingerprint == "" {
				findings = append(findings, Finding{Code: "source-fingerprint-missing", Severity: SeverityWarning, Path: filePath, SessionID: session.ID, Detail: "source path is recorded without a source fingerprint"})
			} else if sourcePayload, sourceErr := os.ReadFile(*session.Provenance.SourcePath); sourceErr == nil {
				sum := sha256.Sum256(sourcePayload)
				actual := hex.EncodeToString(sum[:])
				if actual != *session.Provenance.SourceFingerprint {
					findings = append(findings, finding("source-fingerprint-mismatch", filePath, session.ID, "source bytes no longer match recorded fingerprint"))
				}
			} else {
				findings = append(findings, Finding{Code: "source-unavailable", Severity: SeverityInfo, Path: filePath, SessionID: session.ID, Detail: sourceErr.Error()})
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
		findings = append(findings, validateReceipts(root, archiveRoot)...)
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
		if period.SessionCount != len(manifest.Sessions) {
			findings = append(findings, finding("root-manifest-count-mismatch", rootPath, "", "period count disagrees with period manifest"))
		}
		for _, session := range manifest.Sessions {
			manifestIDs[session.ID] = true
			archivePath, ok := archives[session.ID]
			if !ok {
				findings = append(findings, finding("orphan-manifest-entry", manifestPath, session.ID, "manifest entry has no archive"))
				continue
			}
			expectedFileName := filepath.Base(archivePath)
			if filepath.Base(session.Path) != session.Path || session.Path != expectedFileName {
				findings = append(findings, finding("manifest-path-mismatch", manifestPath, session.ID, "expected file_path "+expectedFileName))
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

type conversionReceiptDocument struct {
	Schema     string `json:"schema"`
	Adapter    string `json:"adapter"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
	OutputDir  string `json:"output_dir"`
	Complete   *bool  `json:"complete"`
	Outputs    []struct {
		SessionID string `json:"session_id"`
		Path      string `json:"path"`
		Status    string `json:"status"`
	} `json:"outputs"`
	Failures []struct {
		Stage string `json:"stage"`
		Error string `json:"error"`
	} `json:"failures"`
	Summary struct {
		Requested int `json:"requested"`
		Published int `json:"published"`
		Failed    int `json:"failed"`
	} `json:"summary"`
}

func validateReceipts(root string, archiveRoot *os.Root) []Finding {
	findings := []Finding{}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || filepath.Ext(path) != ".json" || strings.HasSuffix(path, ".minitrace.json") || entry.Name() == "manifest.json" {
			return nil
		}
		relativePath, relativeErr := filepath.Rel(root, path)
		if relativeErr != nil {
			return nil
		}
		payload, err := archiveRoot.ReadFile(relativePath)
		if err != nil {
			return nil
		}
		var receipt conversionReceiptDocument
		if json.Unmarshal(payload, &receipt) != nil || receipt.Schema != "go-minitrace-conversion-run-v1" {
			return nil
		}
		invalid := func(detail string) {
			findings = append(findings, finding("conversion-receipt-invalid", path, "", detail))
		}
		if receipt.Adapter == "" || receipt.OutputDir == "" || receipt.StartedAt == "" || receipt.FinishedAt == "" || receipt.Complete == nil {
			invalid("receipt lacks adapter, output directory, timestamps, or completion state")
		}
		if receipt.Summary.Published != len(receipt.Outputs) || receipt.Summary.Failed != len(receipt.Failures) {
			invalid("receipt summary does not reconcile with outputs and failures")
		}
		if receipt.Summary.Requested < receipt.Summary.Published+receipt.Summary.Failed {
			invalid("receipt requested count is smaller than terminal results")
		}
		if receipt.Complete != nil && *receipt.Complete && len(receipt.Failures) != 0 {
			invalid("complete receipt contains failures")
		}
		if receipt.Complete != nil && !*receipt.Complete && len(receipt.Failures) == 0 {
			invalid("incomplete receipt does not explain its failure")
		}
		for _, output := range receipt.Outputs {
			if output.SessionID == "" || (output.Status != "created" && output.Status != "unchanged" && output.Status != "replaced") {
				invalid("receipt output lacks session identity or has an unknown status")
				continue
			}
			outputPath := output.Path
			if !filepath.IsAbs(outputPath) {
				outputPath = filepath.Join(receipt.OutputDir, outputPath)
			}
			if _, err := os.Stat(outputPath); err != nil {
				findings = append(findings, finding("conversion-receipt-output-missing", path, output.SessionID, output.Path))
			}
		}
		for _, failure := range receipt.Failures {
			if failure.Stage == "" || failure.Error == "" {
				invalid("receipt failure lacks stage or error detail")
			}
		}
		return nil
	})
	return findings
}
