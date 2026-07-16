package query

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

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
