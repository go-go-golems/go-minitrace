package query

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestResolvePresetSQLSubstitutesTableName(t *testing.T) {
	sqlText, err := ResolvePresetSQL("session-list", "custom_sessions")
	if err != nil {
		t.Fatalf("ResolvePresetSQL returned error: %v", err)
	}
	if sqlText == "" {
		t.Fatal("expected sql text")
	}
	if want := "FROM custom_sessions"; !contains(sqlText, want) {
		t.Fatalf("expected %q in sql text, got:\n%s", want, sqlText)
	}
}

func TestResolveSQLReadsSQLFile(t *testing.T) {
	dir := t.TempDir()
	sqlFile := filepath.Join(dir, "custom.sql")
	if err := os.WriteFile(sqlFile, []byte("SELECT 42 AS answer"), 0o644); err != nil {
		t.Fatalf("writing sql file: %v", err)
	}

	sqlText, err := ResolveSQL("", "", sqlFile, "sessions_base")
	if err != nil {
		t.Fatalf("ResolveSQL returned error: %v", err)
	}
	if sqlText != "SELECT 42 AS answer" {
		t.Fatalf("unexpected sql file content: %q", sqlText)
	}
}

func TestNormalizeValueConvertsBytesAndTime(t *testing.T) {
	if got := NormalizeValue([]byte("hello")); got != "hello" {
		t.Fatalf("expected hello, got %#v", got)
	}

	ts := time.Date(2026, 4, 1, 12, 0, 0, 123456000, time.UTC)
	got, ok := NormalizeValue(ts).(string)
	if !ok {
		t.Fatalf("expected normalized time string, got %T", NormalizeValue(ts))
	}
	if got == "" {
		t.Fatal("expected non-empty normalized time string")
	}
}

func TestLoadArchiveAndRunPreset(t *testing.T) {
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t)
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	ctx := context.Background()
	db, conn, err := OpenConnection(ctx, ":memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned error: %v", err)
	}
	defer func() { _ = conn.Close() }()
	defer func() { _ = db.Close() }()

	if err := LoadArchive(ctx, conn, LoadOptions{
		ArchiveGlobs: []string{filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")},
		TableName:    "sessions_base",
	}); err != nil {
		t.Fatalf("LoadArchive returned error: %v", err)
	}

	var loadedCount int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions_base").Scan(&loadedCount); err != nil {
		t.Fatalf("counting loaded rows: %v", err)
	}
	if loadedCount != 1 {
		t.Fatalf("expected loaded row count 1, got %d", loadedCount)
	}

	sqlText, err := ResolveSQL("session-list", "", "", "sessions_base")
	if err != nil {
		t.Fatalf("ResolveSQL returned error: %v", err)
	}

	processor := &collectingProcessor{}
	if err := RunIntoProcessor(ctx, conn, sqlText, processor); err != nil {
		t.Fatalf("RunIntoProcessor returned error: %v", err)
	}
	if err := processor.Close(ctx); err != nil {
		t.Fatalf("processor.Close returned error: %v", err)
	}

	if len(processor.rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(processor.rows))
	}

	row := processor.rows[0]
	if got, _ := row.Get("id"); got != "fixture-session" {
		t.Fatalf("expected fixture-session id, got %#v", got)
	}
	if got, _ := row.Get("framework"); got != "codex" {
		t.Fatalf("expected codex framework, got %#v", got)
	}
	if got, _ := row.Get("source_format"); got != "fixture" {
		t.Fatalf("expected fixture source_format, got %#v", got)
	}
}

func TestExpandArchiveGlobsDeduplicatesOverlappingMatches(t *testing.T) {
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t)
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	files, err := ExpandArchiveGlobs([]string{
		filepath.Join(archiveRoot, "active", "*", "*.minitrace.json"),
		filepath.Join(archiveRoot, "active", "2026-04", "*.minitrace.json"),
	})
	if err != nil {
		t.Fatalf("ExpandArchiveGlobs returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected deduplicated file list of length 1, got %d: %#v", len(files), files)
	}
}

func TestLoadArchiveSupportsMultipleArchiveGlobs(t *testing.T) {
	archiveRoot1 := t.TempDir()
	archiveRoot2 := t.TempDir()
	session1 := buildFixtureSession(t)
	session2 := buildFixtureSession(t)
	session2.ID = "fixture-session-2"
	if _, err := minitrace.WriteSession(session1, archiveRoot1); err != nil {
		t.Fatalf("WriteSession session1 returned error: %v", err)
	}
	if _, err := minitrace.WriteSession(session2, archiveRoot2); err != nil {
		t.Fatalf("WriteSession session2 returned error: %v", err)
	}

	ctx := context.Background()
	db, conn, err := OpenConnection(ctx, ":memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned error: %v", err)
	}
	defer func() { _ = conn.Close() }()
	defer func() { _ = db.Close() }()

	if err := LoadArchive(ctx, conn, LoadOptions{
		ArchiveGlobs: []string{
			filepath.Join(archiveRoot1, "active", "*", "*.minitrace.json"),
			filepath.Join(archiveRoot2, "active", "*", "*.minitrace.json"),
		},
		TableName: "sessions_base",
	}); err != nil {
		t.Fatalf("LoadArchive returned error: %v", err)
	}

	var loadedCount int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions_base").Scan(&loadedCount); err != nil {
		t.Fatalf("counting loaded rows: %v", err)
	}
	if loadedCount != 2 {
		t.Fatalf("expected loaded row count 2, got %d", loadedCount)
	}
}

func buildFixtureSession(t *testing.T) *minitrace.Session {
	t.Helper()

	ts := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	formatted := minitrace.FormatTimestamp(ts)
	turn := minitrace.BuildTurn(0, &formatted, "user", stringPtr("human"), "hello from fixture")

	session := minitrace.BuildSessionSkeleton("fixture-session", "codex", "fixture", "test")
	session.Title = stringPtr("Fixture Session")
	session.Environment.Model = stringPtr("gpt-5")
	session.Turns = []minitrace.Turn{turn}
	session.ToolCalls = []minitrace.ToolCall{}
	session.Annotations = []minitrace.Annotation{}
	session.Timing = minitrace.ComputeTiming([]time.Time{ts})
	quality := minitrace.AssignQualityTier(session.Turns, session.ToolCalls)
	session.Quality = &quality
	session.Metrics = minitrace.ComputeMetrics(session.Turns, session.ToolCalls, session.Timing, 0, nil)
	return &session
}

func stringPtr(value string) *string {
	return &value
}

func contains(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}

type collectingProcessor struct {
	rows []types.Row
}

func (p *collectingProcessor) AddRow(_ context.Context, row types.Row) error {
	p.rows = append(p.rows, row)
	return nil
}

func (p *collectingProcessor) Close(_ context.Context) error {
	return nil
}
