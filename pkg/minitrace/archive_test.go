package minitrace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type testRootManifest struct {
	Version string `json:"version"`
	Periods []struct {
		Period       string `json:"period"`
		Path         string `json:"path"`
		SessionCount int    `json:"session_count"`
	} `json:"periods"`
	Statistics struct {
		TotalSessions int `json:"total_sessions"`
	} `json:"statistics"`
}

type testPeriodManifest struct {
	Period   string `json:"period"`
	Sessions []struct {
		ID             string `json:"id"`
		AgentFramework string `json:"agent_framework"`
	} `json:"sessions"`
}

func writeTestSession(t *testing.T, outputDir, sessionID, framework, startedAt string) *SessionIndexEntry {
	t.Helper()
	session := BuildSessionSkeleton(sessionID, framework, framework+"-format-v1", "go-minitrace/test")
	session.Timing.StartedAt = ptr(startedAt)
	entry, err := WriteSession(&session, outputDir)
	if err != nil {
		t.Fatalf("WriteSession(%s) returned error: %v", sessionID, err)
	}
	return entry
}

func readRootManifest(t *testing.T, outputDir string) testRootManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(outputDir, "manifest.json"))
	if err != nil {
		t.Fatalf("reading root manifest: %v", err)
	}
	var root testRootManifest
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("decoding root manifest: %v", err)
	}
	return root
}

func readPeriodManifest(t *testing.T, outputDir, period string) testPeriodManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(outputDir, "active", period, "manifest.json"))
	if err != nil {
		t.Fatalf("reading period manifest %s: %v", period, err)
	}
	var manifest testPeriodManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decoding period manifest %s: %v", period, err)
	}
	return manifest
}

func TestWriteManifestsMergesSessionsFromEarlierInvocations(t *testing.T) {
	outputDir := t.TempDir()

	// First invocation: session A (codex, 2026-01).
	entryA := writeTestSession(t, outputDir, "session-a", "codex", "2026-01-05T10:00:00Z")
	if err := WriteManifests([]*SessionIndexEntry{entryA}, outputDir); err != nil {
		t.Fatalf("WriteManifests (first invocation) returned error: %v", err)
	}

	// Second invocation: session B (claude-code, 2026-02) only.
	entryB := writeTestSession(t, outputDir, "session-b", "claude-code", "2026-02-10T12:00:00Z")
	if err := WriteManifests([]*SessionIndexEntry{entryB}, outputDir); err != nil {
		t.Fatalf("WriteManifests (second invocation) returned error: %v", err)
	}

	root := readRootManifest(t, outputDir)
	if root.Statistics.TotalSessions != 2 {
		t.Fatalf("expected 2 total sessions after merge, got %d", root.Statistics.TotalSessions)
	}
	if len(root.Periods) != 2 {
		t.Fatalf("expected 2 periods after merge, got %+v", root.Periods)
	}
	periodCounts := map[string]int{}
	for _, period := range root.Periods {
		periodCounts[period.Period] = period.SessionCount
	}
	if periodCounts["2026-01"] != 1 || periodCounts["2026-02"] != 1 {
		t.Fatalf("expected one session per period, got %+v", periodCounts)
	}

	january := readPeriodManifest(t, outputDir, "2026-01")
	if len(january.Sessions) != 1 || january.Sessions[0].ID != "session-a" || january.Sessions[0].AgentFramework != "codex" {
		t.Fatalf("unexpected 2026-01 manifest sessions: %+v", january.Sessions)
	}
	february := readPeriodManifest(t, outputDir, "2026-02")
	if len(february.Sessions) != 1 || february.Sessions[0].ID != "session-b" || february.Sessions[0].AgentFramework != "claude-code" {
		t.Fatalf("unexpected 2026-02 manifest sessions: %+v", february.Sessions)
	}
}

func TestWriteManifestsCurrentInvocationWinsOnIDCollision(t *testing.T) {
	outputDir := t.TempDir()

	entryOld := writeTestSession(t, outputDir, "session-x", "codex", "2026-01-05T10:00:00Z")
	if err := WriteManifests([]*SessionIndexEntry{entryOld}, outputDir); err != nil {
		t.Fatalf("WriteManifests (first invocation) returned error: %v", err)
	}

	// Re-convert the same session in a new invocation with a different title.
	session := BuildSessionSkeleton("session-x", "codex", "codex-format-v1", "go-minitrace/test")
	session.Timing.StartedAt = ptr("2026-01-05T10:00:00Z")
	session.Title = ptr("updated title")
	entryNew, err := WriteSession(&session, outputDir)
	if err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}
	if err := WriteManifests([]*SessionIndexEntry{entryNew}, outputDir); err != nil {
		t.Fatalf("WriteManifests (second invocation) returned error: %v", err)
	}

	root := readRootManifest(t, outputDir)
	if root.Statistics.TotalSessions != 1 {
		t.Fatalf("expected 1 total session after collision merge, got %d", root.Statistics.TotalSessions)
	}
}

func TestWriteManifestsSkipsInvalidSessionFiles(t *testing.T) {
	outputDir := t.TempDir()

	entryA := writeTestSession(t, outputDir, "session-a", "codex", "2026-01-05T10:00:00Z")

	brokenDir := filepath.Join(outputDir, "active", "2026-03")
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatalf("creating broken period dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(brokenDir, "broken.minitrace.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("writing broken session file: %v", err)
	}

	if err := WriteManifests([]*SessionIndexEntry{entryA}, outputDir); err != nil {
		t.Fatalf("WriteManifests returned error despite broken session file: %v", err)
	}

	root := readRootManifest(t, outputDir)
	if root.Statistics.TotalSessions != 1 {
		t.Fatalf("expected broken session file to be skipped, got %d sessions", root.Statistics.TotalSessions)
	}
	for _, period := range root.Periods {
		if period.Period == "2026-03" {
			t.Fatalf("expected 2026-03 period with only a broken file to be absent, got %+v", root.Periods)
		}
	}
}
