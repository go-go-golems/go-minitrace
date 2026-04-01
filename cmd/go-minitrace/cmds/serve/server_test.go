package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	queryengine "github.com/go-go-golems/go-minitrace/pkg/query"
)

func TestBuildSessionIndexIndexesWrittenSessions(t *testing.T) {
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t, "phase1-index")
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	index, err := buildSessionIndex(filepath.Join(archiveRoot, "active", "*", "*.minitrace.json"))
	if err != nil {
		t.Fatalf("buildSessionIndex returned error: %v", err)
	}

	path, ok := index["phase1-index"]
	if !ok {
		t.Fatalf("expected session ID to be indexed, got keys %#v", index)
	}
	if !strings.HasSuffix(path, "phase1-index.minitrace.json") {
		t.Fatalf("unexpected indexed path %q", path)
	}
}

func TestHandleExecuteQueryReturnsStructuredRows(t *testing.T) {
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t, "phase1-query")
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	ctx := context.Background()
	db, conn, err := queryengine.OpenConnection(ctx, ":memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned error: %v", err)
	}
	defer func() { _ = conn.Close() }()
	defer func() { _ = db.Close() }()

	if err := queryengine.LoadArchive(ctx, conn, queryengine.LoadOptions{
		ArchiveGlob: filepath.Join(archiveRoot, "active", "*", "*.minitrace.json"),
		TableName:   "sessions_base",
	}); err != nil {
		t.Fatalf("LoadArchive returned error: %v", err)
	}

	server := NewServer(conn, &ServeSettings{TableName: "sessions_base"}, map[string]string{})

	request := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(`{"sql":"SELECT id FROM sessions_base"}`))
	response := httptest.NewRecorder()

	server.handleExecuteQuery(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload QueryResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if payload.Error != nil {
		t.Fatalf("expected no query error, got %+v", payload.Error)
	}
	if len(payload.Columns) != 1 || payload.Columns[0] != "id" {
		t.Fatalf("unexpected columns %#v", payload.Columns)
	}
	if payload.RowCount != 1 {
		t.Fatalf("expected row_count=1, got %d", payload.RowCount)
	}
	if got := payload.Rows[0]["id"]; got != "phase1-query" {
		t.Fatalf("expected phase1-query ID, got %#v", got)
	}
}

func TestHandleExecuteQueryReturnsStructuredSQLFailure(t *testing.T) {
	ctx := context.Background()
	db, conn, err := queryengine.OpenConnection(ctx, ":memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned error: %v", err)
	}
	defer func() { _ = conn.Close() }()
	defer func() { _ = db.Close() }()

	server := NewServer(conn, &ServeSettings{TableName: "sessions_base"}, map[string]string{})

	request := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(`{"sql":"SELECT missing FROM sessions_base"}`))
	response := httptest.NewRecorder()

	server.handleExecuteQuery(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d with body %s", response.Code, response.Body.String())
	}

	var payload QueryResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if payload.Error == nil || payload.Error.Message == "" {
		t.Fatalf("expected structured query error, got %+v", payload)
	}
	if payload.RowCount != 0 {
		t.Fatalf("expected row_count=0, got %d", payload.RowCount)
	}
}

func buildFixtureSession(t *testing.T, sessionID string) *minitrace.Session {
	t.Helper()

	ts := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	formatted := minitrace.FormatTimestamp(ts)
	source := "human"

	session := minitrace.BuildSessionSkeleton(sessionID, "codex", "fixture", "test")
	session.Title = stringPtr("Fixture Session")
	session.Summary = stringPtr("Fixture summary")
	session.Environment.Model = stringPtr("gpt-5")
	session.Turns = []minitrace.Turn{
		minitrace.BuildTurn(0, &formatted, "user", &source, "hello from fixture"),
	}
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
