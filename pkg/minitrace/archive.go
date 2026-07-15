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

type CollisionPolicy string

const (
	// CollisionError rejects distinct content for an existing archive ID.
	CollisionError CollisionPolicy = "error"
	// CollisionReplace permits a deliberate destructive replacement.
	CollisionReplace CollisionPolicy = "replace"
)

// WriteSession publishes one archive with the safe default collision policy.
func WriteSession(session *Session, outputDir string) (*SessionIndexEntry, error) {
	return WriteSessionWithCollisionPolicy(session, outputDir, CollisionError)
}

// WriteSessionWithCollisionPolicy writes one archive. Matching non-empty
// source fingerprints are idempotent; all other existing destinations require
// an explicit replacement policy so independent sources cannot silently share
// one archive ID.
func WriteSessionWithCollisionPolicy(session *Session, outputDir string, policy CollisionPolicy) (*SessionIndexEntry, error) {
	if session == nil {
		return nil, errors.New("session is required")
	}
	if policy != CollisionError && policy != CollisionReplace {
		return nil, errors.Errorf("unsupported collision policy %q", policy)
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
	filePath := filepath.Join(dir, SanitizeID(session.ID)+".minitrace.json")

	if existing, info, err := readExistingSession(filePath); err == nil {
		if sameSourceFingerprint(existing, session) {
			return sessionIndexEntry(existing, period, filePath, info.Size()), nil
		}
		if policy != CollisionReplace {
			return nil, errors.Errorf("archive collision for session %q at %s; use explicit replacement only after verifying source provenance", session.ID, filePath)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	payload, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, "marshaling session")
	}
	payload = append(payload, '\n')
	tempFile, err := os.CreateTemp(dir, ".minitrace-*.tmp")
	if err != nil {
		return nil, errors.Wrap(err, "creating temporary session file")
	}
	tempPath := tempFile.Name()
	defer func() { _ = os.Remove(tempPath) }()
	if _, err := tempFile.Write(payload); err != nil {
		_ = tempFile.Close()
		return nil, errors.Wrap(err, "writing temporary session file")
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return nil, errors.Wrap(err, "syncing temporary session file")
	}
	if err := tempFile.Close(); err != nil {
		return nil, errors.Wrap(err, "closing temporary session file")
	}
	if err := os.Rename(tempPath, filePath); err != nil {
		return nil, errors.Wrap(err, "publishing session file")
	}
	if err := syncDirectory(dir); err != nil {
		return nil, err
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "stating written session file")
	}
	return sessionIndexEntry(session, period, filePath, info.Size()), nil
}

func syncDirectory(dir string) error {
	directory, err := os.Open(dir)
	if err != nil {
		return errors.Wrap(err, "opening archive directory for sync")
	}
	defer func() { _ = directory.Close() }()
	if err := directory.Sync(); err != nil {
		return errors.Wrap(err, "syncing archive directory")
	}
	return nil
}

func readExistingSession(path string) (*Session, os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, err
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, errors.Wrap(err, "reading existing session archive")
	}
	var session Session
	if err := json.Unmarshal(payload, &session); err != nil {
		return nil, nil, errors.Wrap(err, "decoding existing session archive")
	}
	return &session, info, nil
}

func sameSourceFingerprint(existing, candidate *Session) bool {
	if existing == nil || candidate == nil || existing.Provenance.SourceFingerprint == nil || candidate.Provenance.SourceFingerprint == nil {
		return false
	}
	return *existing.Provenance.SourceFingerprint != "" && *existing.Provenance.SourceFingerprint == *candidate.Provenance.SourceFingerprint
}

func sessionIndexEntry(session *Session, period, filePath string, fileSizeBytes int64) *SessionIndexEntry {
	quality := ""
	if session.Quality != nil {
		quality = *session.Quality
	}
	return &SessionIndexEntry{ID: session.ID, Period: period, Profile: session.Profile, Title: session.Title, Classification: session.Classification, Quality: quality, StartedAt: session.Timing.StartedAt, Duration: session.Timing.DurationSeconds, Model: session.Environment.Model, AgentFramework: session.Environment.AgentFramework, TurnCount: session.Metrics.TurnCount, ToolCallCount: session.Metrics.ToolCallCount, FileSizeBytes: fileSizeBytes, SourceFormat: session.Provenance.SourceFormat, Flags: session.Flags, FilePath: filePath}
}

// WriteManifests writes the root and per-period manifests describing the
// whole archive directory. It first rescans existing session files under
// outputDir (active/*/*.minitrace.json) so sessions written by earlier
// invocations stay listed, then merges in the current invocation's session
// index; the current invocation wins on session ID collisions.
func WriteManifests(sessionIndex []*SessionIndexEntry, outputDir string) error {
	merged := map[string]*SessionIndexEntry{}
	for _, entry := range scanExistingSessionIndex(outputDir) {
		merged[entry.ID] = entry
	}
	for _, entry := range sessionIndex {
		if entry == nil {
			continue
		}
		merged[entry.ID] = entry
	}

	byPeriod := map[string][]*SessionIndexEntry{}
	for _, entry := range merged {
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

// manifestSessionFields is the slim projection of a session file holding only
// the fields the manifests need; rescanning stays cheap by decoding just
// these fields instead of the full session.
type manifestSessionFields struct {
	ID             string  `json:"id"`
	Profile        string  `json:"profile"`
	Title          *string `json:"title"`
	Classification string  `json:"classification"`
	Quality        *string `json:"quality"`
	Flags          Flags   `json:"flags"`
	Timing         struct {
		StartedAt       *string  `json:"started_at"`
		DurationSeconds *float64 `json:"duration_seconds"`
	} `json:"timing"`
	Environment struct {
		Model          *string `json:"model"`
		AgentFramework *string `json:"agent_framework"`
	} `json:"environment"`
	Metrics struct {
		TurnCount     int `json:"turn_count"`
		ToolCallCount int `json:"tool_call_count"`
	} `json:"metrics"`
	Provenance struct {
		SourceFormat string `json:"source_format"`
	} `json:"provenance"`
}

// scanExistingSessionIndex rebuilds SessionIndexEntry values from the session
// files already present under outputDir/active/<period>/. Unreadable or
// invalid files are skipped with a warning so a broken session never blocks
// manifest generation.
func scanExistingSessionIndex(outputDir string) []*SessionIndexEntry {
	pattern := filepath.Join(outputDir, "active", "*", "*.minitrace.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		log.Warn().Err(err).Str("pattern", pattern).Msg("skipping manifest rescan: invalid glob pattern")
		return nil
	}
	sort.Strings(matches)

	entries := make([]*SessionIndexEntry, 0, len(matches))
	for _, path := range matches {
		entry, err := readSessionIndexEntry(path)
		if err != nil {
			log.Warn().Err(err).Str("path", path).Msg("skipping unreadable session file during manifest rescan")
			continue
		}
		entries = append(entries, entry)
	}
	return entries
}

func readSessionIndexEntry(path string) (*SessionIndexEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "reading session file")
	}
	var fields manifestSessionFields
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, errors.Wrap(err, "decoding session file")
	}
	if fields.ID == "" {
		return nil, errors.New("session file has no id")
	}

	quality := ""
	if fields.Quality != nil {
		quality = *fields.Quality
	}
	return &SessionIndexEntry{
		ID:             fields.ID,
		Period:         SanitizePeriod(filepath.Base(filepath.Dir(path))),
		Profile:        fields.Profile,
		Title:          fields.Title,
		Classification: fields.Classification,
		Quality:        quality,
		StartedAt:      fields.Timing.StartedAt,
		Duration:       fields.Timing.DurationSeconds,
		Model:          fields.Environment.Model,
		AgentFramework: fields.Environment.AgentFramework,
		TurnCount:      fields.Metrics.TurnCount,
		ToolCallCount:  fields.Metrics.ToolCallCount,
		FileSizeBytes:  int64(len(data)),
		SourceFormat:   fields.Provenance.SourceFormat,
		Flags:          fields.Flags,
		FilePath:       path,
	}, nil
}
