package minitrace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
)

type SessionIndexEntry struct {
	ID             string
	Period         string
	Profile        string
	Title          *string
	Classification string
	Quality        string
	StartedAt      *string
	Duration       *float64
	Model          *string
	AgentFramework *string
	TurnCount      int
	ToolCallCount  int
	FileSizeBytes  int64
	SourceFormat   string
	Flags          Flags
	FilePath       string
}

func WriteSession(session *Session, outputDir string) (*SessionIndexEntry, error) {
	if session == nil {
		return nil, errors.New("session is required")
	}

	period := "unknown"
	if session.Timing.StartedAt != nil && len(*session.Timing.StartedAt) >= 7 {
		period = SanitizePeriod((*session.Timing.StartedAt)[:7])
	}

	if session.Provenance.SourcePath != nil {
		normalized := NormalizePath(*session.Provenance.SourcePath)
		session.Provenance.SourcePath = &normalized
	}

	dir := filepath.Join(outputDir, "active", period)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, errors.Wrap(err, "creating session output directory")
	}

	fileName := SanitizeID(session.ID) + ".minitrace.json"
	filePath := filepath.Join(dir, fileName)
	payload, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, "marshaling session")
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(filePath, payload, 0o644); err != nil {
		return nil, errors.Wrap(err, "writing session file")
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "stating written session file")
	}

	quality := ""
	if session.Quality != nil {
		quality = *session.Quality
	}

	return &SessionIndexEntry{
		ID:             session.ID,
		Period:         period,
		Profile:        session.Profile,
		Title:          session.Title,
		Classification: session.Classification,
		Quality:        quality,
		StartedAt:      session.Timing.StartedAt,
		Duration:       session.Timing.DurationSeconds,
		Model:          session.Environment.Model,
		AgentFramework: session.Environment.AgentFramework,
		TurnCount:      session.Metrics.TurnCount,
		ToolCallCount:  session.Metrics.ToolCallCount,
		FileSizeBytes:  info.Size(),
		SourceFormat:   session.Provenance.SourceFormat,
		Flags:          session.Flags,
		FilePath:       filePath,
	}, nil
}

func WriteManifests(sessionIndex []*SessionIndexEntry, outputDir string) error {
	byPeriod := map[string][]*SessionIndexEntry{}
	for _, entry := range sessionIndex {
		if entry == nil {
			continue
		}
		period := SanitizePeriod(entry.Period)
		byPeriod[period] = append(byPeriod[period], entry)
	}

	type periodSession struct {
		ID             string   `json:"id"`
		SchemaVersion  string   `json:"schema_version"`
		Profile        string   `json:"profile"`
		Title          *string  `json:"title"`
		Classification string   `json:"classification"`
		Quality        string   `json:"quality"`
		StartedAt      *string  `json:"started_at"`
		Duration       *float64 `json:"duration_seconds"`
		Model          *string  `json:"model"`
		AgentFramework string   `json:"agent_framework"`
		TurnCount      int      `json:"turn_count"`
		ToolCallCount  int      `json:"tool_call_count"`
		FilePath       string   `json:"file_path"`
		FileSizeBytes  int64    `json:"file_size_bytes"`
		SourceFormat   string   `json:"source_format"`
		Flags          Flags    `json:"flags"`
	}

	type periodManifest struct {
		Version     string          `json:"version"`
		Period      string          `json:"period"`
		GeneratedAt string          `json:"generated_at"`
		Sessions    []periodSession `json:"sessions"`
	}

	type rootPeriod struct {
		Period       string `json:"period"`
		Path         string `json:"path"`
		SessionCount int    `json:"session_count"`
	}

	type rootManifest struct {
		Version     string       `json:"version"`
		GeneratedAt string       `json:"generated_at"`
		Periods     []rootPeriod `json:"periods"`
		Statistics  struct {
			TotalSessions    int                `json:"total_sessions"`
			ByProfile        map[string]int     `json:"by_profile"`
			ByQuality        map[string]int     `json:"by_quality"`
			ByClassification map[string]int     `json:"by_classification"`
			DateRange        map[string]*string `json:"date_range"`
		} `json:"statistics"`
	}

	periods := make([]string, 0, len(byPeriod))
	for period := range byPeriod {
		periods = append(periods, period)
	}
	sort.Strings(periods)

	root := rootManifest{
		Version:     "minitrace-manifest-v2",
		GeneratedAt: FormatTimestamp(NowUTC()),
		Periods:     []rootPeriod{},
	}
	root.Statistics.ByProfile = map[string]int{}
	root.Statistics.ByQuality = map[string]int{}
	root.Statistics.ByClassification = map[string]int{}
	root.Statistics.DateRange = map[string]*string{
		"earliest": nil,
		"latest":   nil,
	}

	for _, entries := range byPeriod {
		for _, entry := range entries {
			root.Statistics.TotalSessions++
			root.Statistics.ByProfile[entry.Profile]++
			root.Statistics.ByQuality[entry.Quality]++
			root.Statistics.ByClassification[entry.Classification]++

			if entry.StartedAt == nil {
				continue
			}
			if root.Statistics.DateRange["earliest"] == nil || *entry.StartedAt < *root.Statistics.DateRange["earliest"] {
				root.Statistics.DateRange["earliest"] = entry.StartedAt
			}
			if root.Statistics.DateRange["latest"] == nil || *entry.StartedAt > *root.Statistics.DateRange["latest"] {
				root.Statistics.DateRange["latest"] = entry.StartedAt
			}
		}
	}

	for _, period := range periods {
		entries := byPeriod[period]
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].ID < entries[j].ID
		})

		manifest := periodManifest{
			Version:     "minitrace-manifest-v2",
			Period:      period,
			GeneratedAt: FormatTimestamp(NowUTC()),
			Sessions:    make([]periodSession, 0, len(entries)),
		}
		for _, entry := range entries {
			framework := "unknown"
			if entry.AgentFramework != nil && *entry.AgentFramework != "" {
				framework = *entry.AgentFramework
			}
			manifest.Sessions = append(manifest.Sessions, periodSession{
				ID:             entry.ID,
				SchemaVersion:  SchemaVersion,
				Profile:        entry.Profile,
				Title:          entry.Title,
				Classification: entry.Classification,
				Quality:        entry.Quality,
				StartedAt:      entry.StartedAt,
				Duration:       entry.Duration,
				Model:          entry.Model,
				AgentFramework: framework,
				TurnCount:      entry.TurnCount,
				ToolCallCount:  entry.ToolCallCount,
				FilePath:       filepath.Base(entry.FilePath),
				FileSizeBytes:  entry.FileSizeBytes,
				SourceFormat:   entry.SourceFormat,
				Flags:          entry.Flags,
			})
		}

		dir := filepath.Join(outputDir, "active", period)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return errors.Wrap(err, "creating period manifest directory")
		}
		payload, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return errors.Wrap(err, "marshaling period manifest")
		}
		payload = append(payload, '\n')
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), payload, 0o644); err != nil {
			return errors.Wrap(err, "writing period manifest")
		}

		root.Periods = append(root.Periods, rootPeriod{
			Period:       period,
			Path:         filepath.ToSlash(filepath.Join("active", period, "manifest.json")),
			SessionCount: len(entries),
		})
	}

	rootPayload, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshaling root manifest")
	}
	rootPayload = append(rootPayload, '\n')
	if err := os.WriteFile(filepath.Join(outputDir, "manifest.json"), rootPayload, 0o644); err != nil {
		return errors.Wrap(err, "writing root manifest")
	}
	return nil
}
