package query

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"
)

//go:embed presets/**/*.sql
var presetFS embed.FS

type PresetEntry struct {
	Name   string
	Folder string
	Path   string
	File   string
}

var (
	presetIndexOnce    sync.Once
	presetEntriesCache []PresetEntry
	presetByIDCache    map[string]PresetEntry
	presetByNameCache  map[string][]PresetEntry
)

func initPresetIndex() {
	presetEntries := make([]PresetEntry, 0)
	presetByID := map[string]PresetEntry{}
	presetByName := map[string][]PresetEntry{}

	err := fs.WalkDir(presetFS, "presets", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(filePath), ".sql") {
			return nil
		}

		relPath := strings.TrimPrefix(filePath, "presets/")
		folder := path.Dir(relPath)
		if folder == "." {
			folder = ""
		}
		name := strings.TrimSuffix(path.Base(relPath), ".sql")
		id := strings.TrimSuffix(relPath, ".sql")
		entry := PresetEntry{
			Name:   name,
			Folder: folder,
			Path:   relPath,
			File:   filePath,
		}
		presetEntries = append(presetEntries, entry)
		presetByID[id] = entry
		presetByName[name] = append(presetByName[name], entry)
		return nil
	})
	if err != nil {
		panic(fmt.Errorf("walking embedded presets: %w", err))
	}

	sort.Slice(presetEntries, func(i, j int) bool {
		return presetEntries[i].Path < presetEntries[j].Path
	})

	presetEntriesCache = presetEntries
	presetByIDCache = presetByID
	presetByNameCache = presetByName
}

func presetIndex() ([]PresetEntry, map[string]PresetEntry, map[string][]PresetEntry) {
	presetIndexOnce.Do(initPresetIndex)
	return presetEntriesCache, presetByIDCache, presetByNameCache
}

func ListPresetEntries() []PresetEntry {
	entries, _, _ := presetIndex()
	return append([]PresetEntry(nil), entries...)
}

func ListPresets() []string {
	entries, _, _ := presetIndex()
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, strings.TrimSuffix(entry.Path, ".sql"))
	}
	return ids
}

func ResolvePresetSQL(name string, tableName string) (string, error) {
	_, presetByID, presetByName := presetIndex()

	entry, ok := presetByID[name]
	if !ok {
		matches := presetByName[name]
		switch len(matches) {
		case 1:
			entry = matches[0]
		case 0:
			return "", fmt.Errorf("unknown preset %q (available: %s)", name, strings.Join(ListPresets(), ", "))
		default:
			options := make([]string, 0, len(matches))
			for _, match := range matches {
				options = append(options, strings.TrimSuffix(match.Path, ".sql"))
			}
			sort.Strings(options)
			return "", fmt.Errorf("ambiguous preset %q (matches: %s)", name, strings.Join(options, ", "))
		}
	}

	payload, err := presetFS.ReadFile(path.Clean(entry.File))
	if err != nil {
		return "", fmt.Errorf("reading embedded preset %q: %w", name, err)
	}

	return strings.ReplaceAll(string(payload), "{{TABLE_NAME}}", tableName), nil
}
