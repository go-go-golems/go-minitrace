package minitracedb

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

// CorpusManifest is an explicit preflight of normalized archive identities.
// It never chooses a winner for a divergent session ID or modifies ingestion.
type CorpusManifest struct {
	Entries        []CorpusManifestEntry `json:"entries"`
	Sessions       []CorpusSessionGroup  `json:"sessions"`
	ConflictCount  int                   `json:"conflictCount"`
	DuplicateCount int                   `json:"duplicateCount"`
}
type CorpusManifestEntry struct {
	Path              string  `json:"path"`
	SessionID         string  `json:"sessionId"`
	OriginalSessionID *string `json:"originalSessionId"`
	SourceFormat      string  `json:"sourceFormat"`
	SourceFingerprint *string `json:"sourceFingerprint"`
	SHA256            string  `json:"sha256"`
	SizeBytes         int     `json:"sizeBytes"`
	DuplicateOf       *int    `json:"duplicateOf"`
	Conflict          bool    `json:"conflict"`
}
type CorpusSessionGroup struct {
	SessionID       string `json:"sessionId"`
	EntryIndexes    []int  `json:"entryIndexes"`
	ContentVariants int    `json:"contentVariants"`
	Conflict        bool   `json:"conflict"`
}

// BuildCorpusManifest hashes and decodes the same bytes for each supplied path.
// DuplicateOf is the zero-based first entry with byte-identical content. Conflict
// means the normalized session ID has multiple byte hashes (including metadata
// or whitespace differences); it is intentionally conservative, not semantic
// equivalence. Original/native identity is reported, never silently substituted.
func BuildCorpusManifest(paths []string) (CorpusManifest, error) {
	manifest := CorpusManifest{Entries: []CorpusManifestEntry{}, Sessions: []CorpusSessionGroup{}}
	byHash := map[string]int{}
	groups := map[string][]int{}
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return CorpusManifest{}, err
		}
		body, err := os.ReadFile(abs)
		if err != nil {
			return CorpusManifest{}, fmt.Errorf("manifest read %s: %w", abs, err)
		}
		var header struct {
			ID         string               `json:"id"`
			Provenance minitrace.Provenance `json:"provenance"`
		}
		if err = json.Unmarshal(body, &header); err != nil {
			return CorpusManifest{}, fmt.Errorf("manifest decode %s: %w", abs, err)
		}
		if header.ID == "" {
			return CorpusManifest{}, fmt.Errorf("manifest %s: missing normalized session id", abs)
		}
		sum := sha256.Sum256(body)
		hash := hex.EncodeToString(sum[:])
		index := len(manifest.Entries)
		entry := CorpusManifestEntry{Path: abs, SessionID: header.ID, OriginalSessionID: header.Provenance.OriginalSessionID, SourceFormat: header.Provenance.SourceFormat, SourceFingerprint: header.Provenance.SourceFingerprint, SHA256: hash, SizeBytes: len(body)}
		if first, ok := byHash[hash]; ok {
			entry.DuplicateOf = &first
			manifest.DuplicateCount++
		} else {
			byHash[hash] = index
		}
		manifest.Entries = append(manifest.Entries, entry)
		groups[header.ID] = append(groups[header.ID], index)
	}
	ids := make([]string, 0, len(groups))
	for id := range groups {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		indexes := groups[id]
		variants := map[string]bool{}
		for _, index := range indexes {
			variants[manifest.Entries[index].SHA256] = true
		}
		conflict := len(variants) > 1
		if conflict {
			manifest.ConflictCount++
			for _, index := range indexes {
				manifest.Entries[index].Conflict = true
			}
		}
		manifest.Sessions = append(manifest.Sessions, CorpusSessionGroup{SessionID: id, EntryIndexes: indexes, ContentVariants: len(variants), Conflict: conflict})
	}
	return manifest, nil
}
