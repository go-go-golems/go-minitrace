package query

import (
	"embed"
	"fmt"
	"path"
	"sort"
	"strings"
)

//go:embed presets/*.sql
var presetFS embed.FS

var presetFiles = map[string]string{
	"annotations":              "presets/annotations.sql",
	"framework-summary":        "presets/framework-summary.sql",
	"read-ratio-distribution":  "presets/read-ratio-distribution.sql",
	"session-list":             "presets/session-list.sql",
	"timing-analysis":          "presets/timing-analysis.sql",
	"tool-operation-breakdown": "presets/tool-operation-breakdown.sql",
}

func ListPresets() []string {
	names := make([]string, 0, len(presetFiles))
	for name := range presetFiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func ResolvePresetSQL(name string, tableName string) (string, error) {
	fileName, ok := presetFiles[name]
	if !ok {
		return "", fmt.Errorf("unknown preset %q (available: %s)", name, strings.Join(ListPresets(), ", "))
	}

	payload, err := presetFS.ReadFile(path.Clean(fileName))
	if err != nil {
		return "", fmt.Errorf("reading embedded preset %q: %w", name, err)
	}

	return strings.ReplaceAll(string(payload), "{{TABLE_NAME}}", tableName), nil
}
