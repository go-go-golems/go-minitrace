package query

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/pkg/errors"
)

type archiveInventoryFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size_bytes"`
}

type archiveInventory struct {
	Globs        []string               `json:"globs"`
	Files        []archiveInventoryFile `json:"files"`
	Unmatched    []string               `json:"unmatched_globs"`
	InventorySHA string                 `json:"inventory_sha256"`
}

type queryProvenance struct {
	Kind       string `json:"kind"`
	Path       string `json:"path,omitempty"`
	Preset     string `json:"preset,omitempty"`
	SHA256     string `json:"sha256"`
	InlineText string `json:"inline_text,omitempty"`
}

type queryRunRecord struct {
	Schema       string           `json:"schema"`
	StartedAt    string           `json:"started_at"`
	FinishedAt   string           `json:"finished_at"`
	Status       string           `json:"status"`
	ErrorCode    string           `json:"error_code,omitempty"`
	Error        string           `json:"error,omitempty"`
	Query        queryProvenance  `json:"query"`
	Inventory    archiveInventory `json:"archive_inventory"`
	MaxRows      int              `json:"max_rows"`
	MaxCellChars int              `json:"max_cell_chars"`
	TimeoutMS    int              `json:"timeout_ms"`
	Columns      []string         `json:"columns"`
	RowCount     int              `json:"row_count"`
	Truncated    bool             `json:"truncated"`
}

func hashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func writeQueryRunRecord(path string, record queryRunRecord) error {
	if path == "" {
		return nil
	}
	record.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".query-run-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if _, err := temporary.Write(payload); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func resolveArchiveInventory(globs []string) (archiveInventory, error) {
	inventory := archiveInventory{Globs: append([]string(nil), globs...), Files: []archiveInventoryFile{}, Unmatched: []string{}}
	sort.Strings(inventory.Globs)
	seen := map[string]bool{}
	for _, pattern := range inventory.Globs {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return inventory, errors.Wrapf(err, "expanding archive glob %q", pattern)
		}
		if len(matches) == 0 {
			inventory.Unmatched = append(inventory.Unmatched, pattern)
		}
		for _, match := range matches {
			absolute, err := filepath.Abs(match)
			if err != nil {
				return inventory, err
			}
			absolute = filepath.Clean(absolute)
			if seen[absolute] {
				continue
			}
			seen[absolute] = true
			payload, err := os.ReadFile(absolute)
			if err != nil {
				return inventory, errors.Wrapf(err, "hashing archive %s", absolute)
			}
			sum := sha256.Sum256(payload)
			inventory.Files = append(inventory.Files, archiveInventoryFile{Path: absolute, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(payload))})
		}
	}
	sort.Slice(inventory.Files, func(i, j int) bool { return inventory.Files[i].Path < inventory.Files[j].Path })
	canonical, err := json.Marshal(struct {
		Files     []archiveInventoryFile `json:"files"`
		Unmatched []string               `json:"unmatched_globs"`
	}{inventory.Files, inventory.Unmatched})
	if err != nil {
		return inventory, err
	}
	sum := sha256.Sum256(canonical)
	inventory.InventorySHA = hex.EncodeToString(sum[:])
	return inventory, nil
}
