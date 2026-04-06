package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
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

	index, err := buildSessionIndex([]string{filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")})
	if err != nil {
		t.Fatalf("buildSessionIndex returned error: %v", err)
	}

	path, ok := index["phase1-index"]
	if !ok {
		t.Fatalf("expected session ID to be indexed, got keys %#v", index)
	}
	if !strings.HasSuffix(path, minitrace.SanitizeID("phase1-index")+".minitrace.json") {
		t.Fatalf("unexpected indexed path %q", path)
	}
}

func TestBuildSessionIndexUsesCanonicalSessionIDFromJSON(t *testing.T) {
	archiveRoot := t.TempDir()
	sessionID := "phase1/index:unsafe'session"
	session := buildFixtureSession(t, sessionID)
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	index, err := buildSessionIndex([]string{filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")})
	if err != nil {
		t.Fatalf("buildSessionIndex returned error: %v", err)
	}

	path, ok := index[sessionID]
	if !ok {
		t.Fatalf("expected canonical session ID %q to be indexed, got keys %#v", sessionID, index)
	}
	if !strings.HasSuffix(path, minitrace.SanitizeID(sessionID)+".minitrace.json") {
		t.Fatalf("expected sanitized filename suffix, got %q", path)
	}
}

func TestBuildSessionIndexSupportsMultipleArchiveGlobs(t *testing.T) {
	archiveRoot1 := t.TempDir()
	archiveRoot2 := t.TempDir()
	session1 := buildFixtureSession(t, "phase1-index-a")
	session2 := buildFixtureSession(t, "phase1-index-b")
	if _, err := minitrace.WriteSession(session1, archiveRoot1); err != nil {
		t.Fatalf("WriteSession session1 returned error: %v", err)
	}
	if _, err := minitrace.WriteSession(session2, archiveRoot2); err != nil {
		t.Fatalf("WriteSession session2 returned error: %v", err)
	}

	index, err := buildSessionIndex([]string{
		filepath.Join(archiveRoot1, "active", "*", "*.minitrace.json"),
		filepath.Join(archiveRoot2, "active", "*", "*.minitrace.json"),
	})
	if err != nil {
		t.Fatalf("buildSessionIndex returned error: %v", err)
	}
	if len(index) != 2 {
		t.Fatalf("expected 2 indexed sessions, got %d", len(index))
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
		ArchiveGlobs: []string{filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")},
		TableName:    "sessions_base",
	}); err != nil {
		t.Fatalf("LoadArchive returned error: %v", err)
	}

	server := NewServer(conn, &ServeSettings{TableName: "sessions_base"}, map[string]string{}, nil, nil)

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

	server := NewServer(conn, &ServeSettings{TableName: "sessions_base"}, map[string]string{}, nil, nil)

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

func TestHandleExecuteQueryRejectsNonReadOnlyStatements(t *testing.T) {
	ctx := context.Background()
	db, conn, err := queryengine.OpenConnection(ctx, ":memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned error: %v", err)
	}
	defer func() { _ = conn.Close() }()
	defer func() { _ = db.Close() }()

	server := NewServer(conn, &ServeSettings{TableName: "sessions_base"}, map[string]string{}, nil, nil)

	request := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(`{"sql":"CREATE TABLE injected(id INTEGER)"}`))
	response := httptest.NewRecorder()

	server.handleExecuteQuery(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d with body %s", response.Code, response.Body.String())
	}

	var payload QueryResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if payload.Error == nil || !strings.Contains(payload.Error.Message, "read-only") {
		t.Fatalf("expected read-only validation error, got %+v", payload.Error)
	}
}

func TestHandleGetSessionsReturnsNormalizedSummaries(t *testing.T) {
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t, "phase2-sessions")
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
		ArchiveGlobs: []string{filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")},
		TableName:    "sessions_base",
	}); err != nil {
		t.Fatalf("LoadArchive returned error: %v", err)
	}

	server := NewServer(conn, &ServeSettings{TableName: "sessions_base"}, map[string]string{}, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	response := httptest.NewRecorder()

	server.handleGetSessions(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload []SessionSummaryResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshaling summaries: %v", err)
	}
	if len(payload) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(payload))
	}
	if payload[0].ID != "phase2-sessions" {
		t.Fatalf("unexpected session ID %q", payload[0].ID)
	}
	if payload[0].Environment.Model != "gpt-5" {
		t.Fatalf("unexpected model %q", payload[0].Environment.Model)
	}
	if payload[0].OperationalContext.WorkingDirectory != "/tmp/project" {
		t.Fatalf("unexpected workdir %q", payload[0].OperationalContext.WorkingDirectory)
	}
}

func TestHandleGetSessionSummaryReturnsMetadataWithoutBlocks(t *testing.T) {
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t, "phase2-summary")
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	index, err := buildSessionIndex([]string{filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")})
	if err != nil {
		t.Fatalf("buildSessionIndex returned error: %v", err)
	}

	ctx := context.Background()
	db, conn, err := queryengine.OpenConnection(ctx, ":memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned error: %v", err)
	}
	defer func() { _ = conn.Close() }()
	defer func() { _ = db.Close() }()

	server := NewServer(conn, &ServeSettings{TableName: "sessions_base"}, index, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/sessions/phase2-summary/summary", nil)
	request.SetPathValue("id", "phase2-summary")
	response := httptest.NewRecorder()

	server.handleGetSessionSummary(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload SessionSummaryDetailResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshaling summary detail: %v", err)
	}
	if payload.ID != "phase2-summary" {
		t.Fatalf("unexpected session ID %q", payload.ID)
	}
	if payload.Title != "Fixture Session" {
		t.Fatalf("unexpected session title %q", payload.Title)
	}
	if payload.Provenance.SourceFormat != "fixture" {
		t.Fatalf("unexpected source format %q", payload.Provenance.SourceFormat)
	}
	if strings.Contains(response.Body.String(), "\"blocks\"") {
		t.Fatalf("summary response should not include blocks: %s", response.Body.String())
	}
}

func TestHandleGetSessionReturnsDetailWithBlocks(t *testing.T) {
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t, "phase2-detail")
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	index, err := buildSessionIndex([]string{filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")})
	if err != nil {
		t.Fatalf("buildSessionIndex returned error: %v", err)
	}

	ctx := context.Background()
	db, conn, err := queryengine.OpenConnection(ctx, ":memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned error: %v", err)
	}
	defer func() { _ = conn.Close() }()
	defer func() { _ = db.Close() }()

	server := NewServer(conn, &ServeSettings{TableName: "sessions_base"}, index, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/sessions/phase2-detail", nil)
	request.SetPathValue("id", "phase2-detail")
	response := httptest.NewRecorder()

	server.handleGetSession(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload SessionDetailResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshaling detail: %v", err)
	}
	if payload.ID != "phase2-detail" {
		t.Fatalf("unexpected session ID %q", payload.ID)
	}
	if len(payload.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(payload.Blocks))
	}
	if payload.Blocks[0].ToolCalls != 1 {
		t.Fatalf("expected 1 tool call in block, got %d", payload.Blocks[0].ToolCalls)
	}
	if payload.Blocks[0].Artifacts.Commits[0] != "feat: fixture" {
		t.Fatalf("expected commit artifact, got %#v", payload.Blocks[0].Artifacts.Commits)
	}
	if len(payload.Blocks[0].Turns) != 2 {
		t.Fatalf("expected 2 turns in block, got %d", len(payload.Blocks[0].Turns))
	}
	if len(payload.Blocks[0].Turns[1].ToolCallsInTurn) != 1 {
		t.Fatalf("expected 1 tool call in assistant turn, got %d", len(payload.Blocks[0].Turns[1].ToolCallsInTurn))
	}
	if payload.Blocks[0].Turns[1].ToolCallsInTurn[0].ToolName != "exec_command" {
		t.Fatalf("unexpected tool name %q", payload.Blocks[0].Turns[1].ToolCallsInTurn[0].ToolName)
	}
	if payload.Blocks[0].Turns[1].ToolCallsInTurn[0].Badges[0] != BadgeCommit {
		t.Fatalf("expected commit badge, got %#v", payload.Blocks[0].Turns[1].ToolCallsInTurn[0].Badges)
	}
}

func TestHandleGetSessionBlocksReturnsGapsAndArtifacts(t *testing.T) {
	archiveRoot := t.TempDir()
	session := buildTwoBlockSession(t, "phase3-blocks")
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	index, err := buildSessionIndex([]string{filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")})
	if err != nil {
		t.Fatalf("buildSessionIndex returned error: %v", err)
	}

	ctx := context.Background()
	db, conn, err := queryengine.OpenConnection(ctx, ":memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned error: %v", err)
	}
	defer func() { _ = conn.Close() }()
	defer func() { _ = db.Close() }()

	server := NewServer(conn, &ServeSettings{TableName: "sessions_base"}, index, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/sessions/phase3-blocks/blocks", nil)
	request.SetPathValue("id", "phase3-blocks")
	response := httptest.NewRecorder()

	server.handleGetSessionBlocks(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload []SessionBlock
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshaling blocks: %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(payload))
	}
	if payload[0].Artifacts.DiaryWrites != 1 {
		t.Fatalf("expected 1 diary write, got %d", payload[0].Artifacts.DiaryWrites)
	}
	if payload[1].GapMinutes == nil || *payload[1].GapMinutes < 9.9 || *payload[1].GapMinutes > 10.1 {
		t.Fatalf("expected ~10 minute gap, got %+v", payload[1].GapMinutes)
	}
}

func TestHandleGetPresetsReturnsBuiltInAndExternalQueries(t *testing.T) {
	presetDir1 := t.TempDir()
	presetDir2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(presetDir1, "custom.sql"), []byte("-- custom preset\nSELECT 1;"), 0o644); err != nil {
		t.Fatalf("writing custom preset: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(presetDir2, "analysis"), 0o755); err != nil {
		t.Fatalf("creating nested preset dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(presetDir2, "analysis", "extra.sql"), []byte("-- extra preset\nSELECT 2;"), 0o644); err != nil {
		t.Fatalf("writing extra preset: %v", err)
	}

	server := NewServer(nil, &ServeSettings{
		TableName: "sessions_base",
		PresetDir: []string{presetDir1, presetDir2},
	}, map[string]string{}, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/presets", nil)
	response := httptest.NewRecorder()

	server.handleGetPresets(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload []SavedQuery
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshaling presets: %v", err)
	}
	if len(payload) == 0 {
		t.Fatal("expected presets to be returned")
	}
	foundBuiltIn := false
	foundCustom := false
	foundExtra := false
	for _, query := range payload {
		if query.Name == "session-list" && query.Folder == "core" && query.Readonly {
			foundBuiltIn = true
		}
		if query.Name == "custom" && query.Description == "custom preset" && query.Readonly {
			foundCustom = true
		}
		if query.Path == "analysis/extra.sql" && query.Description == "extra preset" && query.Readonly {
			foundExtra = true
		}
	}
	if !foundBuiltIn {
		t.Fatalf("expected built-in session-list preset in %#v", payload)
	}
	if !foundCustom {
		t.Fatalf("expected custom external preset in %#v", payload)
	}
	if !foundExtra {
		t.Fatalf("expected extra external preset in %#v", payload)
	}
}

func TestQueryCRUDValidatesPathsAndPersistsQueries(t *testing.T) {
	queryDir1 := t.TempDir()
	queryDir2 := t.TempDir()
	if err := os.MkdirAll(filepath.Join(queryDir2, "shared"), 0o755); err != nil {
		t.Fatalf("creating second query dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(queryDir2, "shared", "team.sql"), []byte("-- team query\nSELECT 9;"), 0o644); err != nil {
		t.Fatalf("writing second-root query: %v", err)
	}
	server := NewServer(nil, &ServeSettings{
		TableName: "sessions_base",
		QueryDir:  []string{queryDir1, queryDir2},
	}, map[string]string{}, nil, nil)

	saveRequest := httptest.NewRequest(http.MethodPost, "/api/queries", strings.NewReader(`{"name":"wesen-os-filter","folder":"saved/analysis","description":"saved query","sql":"SELECT 1;"}`))
	saveResponse := httptest.NewRecorder()
	server.handleSaveQuery(saveResponse, saveRequest)

	if saveResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d with body %s", saveResponse.Code, saveResponse.Body.String())
	}

	var saved SavedQuery
	if err := json.Unmarshal(saveResponse.Body.Bytes(), &saved); err != nil {
		t.Fatalf("unmarshaling saved query: %v", err)
	}
	if saved.Path != "saved/analysis/wesen-os-filter.sql" {
		t.Fatalf("unexpected saved path %q", saved.Path)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/queries", nil)
	getResponse := httptest.NewRecorder()
	server.handleGetQueries(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", getResponse.Code, getResponse.Body.String())
	}
	var queries []SavedQuery
	if err := json.Unmarshal(getResponse.Body.Bytes(), &queries); err != nil {
		t.Fatalf("unmarshaling queries: %v", err)
	}
	foundSaved := false
	foundSecondRoot := false
	for _, query := range queries {
		if query.Path == "saved/analysis/wesen-os-filter.sql" {
			foundSaved = true
		}
		if query.Path == "shared/team.sql" && query.Description == "team query" {
			foundSecondRoot = true
		}
	}
	if !foundSaved || !foundSecondRoot {
		t.Fatalf("expected queries from both roots, got %#v", queries)
	}

	updateRequest := httptest.NewRequest(http.MethodPut, "/api/queries/saved/analysis/wesen-os-filter.sql", strings.NewReader(`{"description":"updated query","sql":"SELECT 2;"}`))
	updateRequest.SetPathValue("path", "saved/analysis/wesen-os-filter.sql")
	updateResponse := httptest.NewRecorder()
	server.handleUpdateQuery(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", updateResponse.Code, updateResponse.Body.String())
	}

	updateSecondRootRequest := httptest.NewRequest(http.MethodPut, "/api/queries/shared/team.sql", strings.NewReader(`{"description":"updated team query","sql":"SELECT 10;"}`))
	updateSecondRootRequest.SetPathValue("path", "shared/team.sql")
	updateSecondRootResponse := httptest.NewRecorder()
	server.handleUpdateQuery(updateSecondRootResponse, updateSecondRootRequest)
	if updateSecondRootResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 for second-root update, got %d with body %s", updateSecondRootResponse.Code, updateSecondRootResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/queries/saved/analysis/wesen-os-filter.sql", nil)
	deleteRequest.SetPathValue("path", "saved/analysis/wesen-os-filter.sql")
	deleteResponse := httptest.NewRecorder()
	server.handleDeleteQuery(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d with body %s", deleteResponse.Code, deleteResponse.Body.String())
	}

	traversalRequest := httptest.NewRequest(http.MethodPost, "/api/queries", strings.NewReader(`{"name":"bad","folder":"../escape","description":"x","sql":"SELECT 1;"}`))
	traversalResponse := httptest.NewRecorder()
	server.handleSaveQuery(traversalResponse, traversalRequest)
	if traversalResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for traversal, got %d with body %s", traversalResponse.Code, traversalResponse.Body.String())
	}

	absoluteUpdateRequest := httptest.NewRequest(http.MethodPut, "/api/queries/tmp/evil.sql", strings.NewReader(`{"description":"x","sql":"SELECT 1;"}`))
	absoluteUpdateRequest.SetPathValue("path", "/tmp/evil.sql")
	absoluteUpdateResponse := httptest.NewRecorder()
	server.handleUpdateQuery(absoluteUpdateResponse, absoluteUpdateRequest)
	if absoluteUpdateResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for absolute path update, got %d with body %s", absoluteUpdateResponse.Code, absoluteUpdateResponse.Body.String())
	}

	if _, err := os.Stat(filepath.Join(queryDir1, "saved", "analysis", "wesen-os-filter.sql")); !os.IsNotExist(err) {
		t.Fatalf("expected saved query to be deleted from first root, stat err=%v", err)
	}
	secondRootContent, err := os.ReadFile(filepath.Join(queryDir2, "shared", "team.sql"))
	if err != nil {
		t.Fatalf("reading updated second-root query: %v", err)
	}
	if !strings.Contains(string(secondRootContent), "updated team query") {
		t.Fatalf("expected second-root file to be updated, got %q", string(secondRootContent))
	}
}

func TestSpaHandlerFallsBackToIndexHTML(t *testing.T) {
	handler := spaHandler(fstest.MapFS{
		"index.html":        {Data: []byte("<html>index</html>")},
		"static/app.js":     {Data: []byte("console.log('ok')")},
		"static/styles.css": {Data: []byte("body{}")},
	})

	indexRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	indexResponse := httptest.NewRecorder()
	handler.ServeHTTP(indexResponse, indexRequest)
	if indexResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 for index, got %d", indexResponse.Code)
	}
	if !strings.Contains(indexResponse.Body.String(), "index") {
		t.Fatalf("expected index content, got %q", indexResponse.Body.String())
	}

	assetRequest := httptest.NewRequest(http.MethodGet, "/static/app.js", nil)
	assetResponse := httptest.NewRecorder()
	handler.ServeHTTP(assetResponse, assetRequest)
	if assetResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 for asset, got %d", assetResponse.Code)
	}
	if !strings.Contains(assetResponse.Body.String(), "console.log") {
		t.Fatalf("expected asset content, got %q", assetResponse.Body.String())
	}

	routeRequest := httptest.NewRequest(http.MethodGet, "/sessions/abc", nil)
	routeResponse := httptest.NewRecorder()
	handler.ServeHTTP(routeResponse, routeRequest)
	if routeResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 for SPA fallback, got %d", routeResponse.Code)
	}
	if !strings.Contains(routeResponse.Body.String(), "index") {
		t.Fatalf("expected SPA fallback to index, got %q", routeResponse.Body.String())
	}
}

func buildFixtureSession(t *testing.T, sessionID string) *minitrace.Session {
	t.Helper()

	ts := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	formatted := minitrace.FormatTimestamp(ts)
	source := "human"
	assistantSource := "agent"
	durationMS := 42

	toolCall := minitrace.BuildToolCall(
		toolCallID(sessionID),
		intPtr(1),
		&formatted,
		"exec_command",
		"execute",
		nil,
		nil,
		map[string]any{"cmd": `git commit -m "feat: fixture"`},
		true,
		"/tmp/project\n",
		nil,
		&durationMS,
		nil,
		nil,
		nil,
		nil,
	)

	session := minitrace.BuildSessionSkeleton(sessionID, "codex", "fixture", "test")
	session.Title = stringPtr("Fixture Session")
	session.Summary = stringPtr("Fixture summary")
	session.Environment.Model = stringPtr("gpt-5")
	session.OperationalContext.WorkingDirectory = stringPtr("/tmp/project")
	session.OperationalContext.AutonomyLevel = stringPtr("default")
	session.OperationalContext.Sandbox = boolPtr(true)
	session.Turns = []minitrace.Turn{
		minitrace.BuildTurn(0, &formatted, "user", &source, "hello from fixture"),
		{
			Index:             1,
			Timestamp:         &formatted,
			Role:              "assistant",
			Source:            &assistantSource,
			Content:           "running pwd",
			ToolCallsInTurn:   []string{toolCall.ID},
			Streaming:         minitrace.Streaming{},
			FrameworkMetadata: nil,
		},
	}
	session.ToolCalls = []minitrace.ToolCall{toolCall}
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

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func toolCallID(sessionID string) string {
	return sessionID + "-tool-1"
}

func buildTwoBlockSession(t *testing.T, sessionID string) *minitrace.Session {
	t.Helper()

	ts1 := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	ts2 := ts1.Add(10 * time.Minute)
	userSource := "human"
	assistantSource := "agent"

	ts1Formatted := minitrace.FormatTimestamp(ts1)
	ts2Formatted := minitrace.FormatTimestamp(ts2)
	durationMS := 5
	diaryPath := "/tmp/project/ttmp/2026/04/01/reference/01-diary.md"

	diaryToolCall := minitrace.BuildToolCall(
		sessionID+"-diary-tool",
		intPtr(1),
		&ts1Formatted,
		"apply_patch",
		"modify",
		&diaryPath,
		nil,
		map[string]any{"cmd": "apply_patch diary"},
		true,
		"updated diary",
		nil,
		&durationMS,
		nil,
		nil,
		nil,
		nil,
	)

	session := minitrace.BuildSessionSkeleton(sessionID, "codex", "fixture", "test")
	session.Title = stringPtr("Two Block Fixture")
	session.Turns = []minitrace.Turn{
		minitrace.BuildTurn(0, &ts1Formatted, "user", &userSource, "first request"),
		{
			Index:             1,
			Timestamp:         &ts1Formatted,
			Role:              "assistant",
			Source:            &assistantSource,
			Content:           "wrote the diary",
			ToolCallsInTurn:   []string{diaryToolCall.ID},
			Streaming:         minitrace.Streaming{},
			FrameworkMetadata: nil,
		},
		minitrace.BuildTurn(2, &ts2Formatted, "user", &userSource, "second request"),
	}
	session.ToolCalls = []minitrace.ToolCall{diaryToolCall}
	session.Timing = minitrace.ComputeTiming([]time.Time{ts1, ts2})
	quality := minitrace.AssignQualityTier(session.Turns, session.ToolCalls)
	session.Quality = &quality
	session.Metrics = minitrace.ComputeMetrics(session.Turns, session.ToolCalls, session.Timing, 0, nil)
	return &session
}
