package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	apiv1 "github.com/go-go-golems/go-minitrace/gen/proto/go_go_golems/minitrace/api/v1"
	"github.com/go-go-golems/go-minitrace/pkg/annotate"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
	"github.com/go-go-golems/go-minitrace/pkg/minitracejs"
	"google.golang.org/protobuf/encoding/protojson"
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

	target := newTestQueryTarget(t, filepath.Join(archiveRoot, "active", "*", "*.minitrace.json"))
	server := NewServer(target, &ServeSettings{}, map[string]string{}, nil, nil)

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
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t, "phase1-query-failure")
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	target := newTestQueryTarget(t, filepath.Join(archiveRoot, "active", "*", "*.minitrace.json"))
	server := NewServer(target, &ServeSettings{}, map[string]string{}, nil, nil)

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
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t, "phase1-query-readonly")
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	target := newTestQueryTarget(t, filepath.Join(archiveRoot, "active", "*", "*.minitrace.json"))
	server := NewServer(target, &ServeSettings{}, map[string]string{}, nil, nil)

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
	if payload.Error == nil || !strings.Contains(payload.Error.Message, "only SELECT and WITH queries are allowed") {
		t.Fatalf("expected read-only validation error, got %+v", payload.Error)
	}
}

func TestLegacySessionRoutesReturnNotFound(t *testing.T) {
	server := NewServer(nil, &ServeSettings{DevMode: true}, map[string]string{}, nil, nil)

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/sessions"},
		{method: http.MethodGet, path: "/api/sessions/phase2-detail"},
		{method: http.MethodGet, path: "/api/sessions/phase2-summary/summary"},
		{method: http.MethodGet, path: "/api/sessions/phase3-blocks/blocks"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, nil)
			response := httptest.NewRecorder()

			server.mux.ServeHTTP(response, request)

			if response.Code != http.StatusNotFound {
				t.Fatalf("expected 404, got %d with body %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestHandleGetSessionsV2ReturnsEnvelope(t *testing.T) {
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t, "phase4-sessions-v2")
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	target := newTestQueryTarget(t, filepath.Join(archiveRoot, "active", "*", "*.minitrace.json"))
	server := NewServer(target, &ServeSettings{}, map[string]string{}, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v2/sessions", nil)
	response := httptest.NewRecorder()

	server.handleGetSessionsV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.ListSessionsResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal list sessions v2: %v", err)
	}
	if payload.Meta.GetSchemaVersion() != apiSchemaVersion {
		t.Fatalf("unexpected schema version %d", payload.Meta.GetSchemaVersion())
	}
	if len(payload.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(payload.Sessions))
	}
	if payload.Sessions[0].GetId() != "phase4-sessions-v2" {
		t.Fatalf("unexpected session id %q", payload.Sessions[0].GetId())
	}
	if payload.Sessions[0].GetEnvironment().GetModel() != "gpt-5" {
		t.Fatalf("unexpected model %q", payload.Sessions[0].GetEnvironment().GetModel())
	}
	if !strings.Contains(response.Body.String(), "schemaVersion") {
		t.Fatalf("expected camelCase schemaVersion in response: %s", response.Body.String())
	}
}

func TestHandleGetSessionSummaryV2ReturnsEnvelopeWithoutBlocks(t *testing.T) {
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t, "phase4-summary-v2")
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	index, err := buildSessionIndex([]string{filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")})
	if err != nil {
		t.Fatalf("buildSessionIndex returned error: %v", err)
	}

	server := NewServer(nil, &ServeSettings{}, index, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v2/sessions/phase4-summary-v2/summary", nil)
	request.SetPathValue("id", "phase4-summary-v2")
	response := httptest.NewRecorder()

	server.handleGetSessionSummaryV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.GetSessionSummaryResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal summary v2: %v", err)
	}
	if payload.Session.GetId() != "phase4-summary-v2" {
		t.Fatalf("unexpected session id %q", payload.Session.GetId())
	}
	if payload.Session.GetProvenance().GetSourceFormat() != "fixture" {
		t.Fatalf("unexpected source format %q", payload.Session.GetProvenance().GetSourceFormat())
	}
	if strings.Contains(response.Body.String(), "blocks") {
		t.Fatalf("summary response should not include blocks: %s", response.Body.String())
	}
}

func TestHandleGetSessionBlocksV2ReturnsEnvelope(t *testing.T) {
	archiveRoot := t.TempDir()
	session := buildTwoBlockSession(t, "phase4-blocks-v2")
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	index, err := buildSessionIndex([]string{filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")})
	if err != nil {
		t.Fatalf("buildSessionIndex returned error: %v", err)
	}

	server := NewServer(nil, &ServeSettings{}, index, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v2/sessions/phase4-blocks-v2/blocks", nil)
	request.SetPathValue("id", "phase4-blocks-v2")
	response := httptest.NewRecorder()

	server.handleGetSessionBlocksV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.GetSessionBlocksResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal blocks v2: %v", err)
	}
	if len(payload.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(payload.Blocks))
	}
	if payload.Blocks[0].GetArtifacts().GetDiaryWrites() != 1 {
		t.Fatalf("expected 1 diary write, got %d", payload.Blocks[0].GetArtifacts().GetDiaryWrites())
	}
	gap := payload.Blocks[1].GetGapMinutes()
	if gap < 9.9 || gap > 10.1 {
		t.Fatalf("expected ~10 minute gap, got %f", gap)
	}
}

func TestHandleGetSessionV2ReturnsDetailEnvelope(t *testing.T) {
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t, "phase4-detail-v2")
	event := minitrace.BuildEvent("event-1", session.Turns[1].Timestamp, "compaction", "Compaction", "Context compacted", map[string]any{"type": "compaction"})
	event.ToolCallID = &session.ToolCalls[0].ID
	event.AttachmentID = stringPtr("attachment-1")
	event.FrameworkMetadata = map[string]any{"source": "test"}
	attachment := minitrace.BuildAttachment("attachment-1", session.Turns[1].Timestamp, "image", "screenshot.png", "image/png", map[string]any{"type": "attachment"})
	attachment.Path = "screenshots/screenshot.png"
	attachment.ToolCallID = &session.ToolCalls[0].ID
	attachment.EventID = &event.ID
	session.Events = []minitrace.Event{event}
	session.Attachments = []minitrace.Attachment{attachment}
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	index, err := buildSessionIndex([]string{filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")})
	if err != nil {
		t.Fatalf("buildSessionIndex returned error: %v", err)
	}

	server := NewServer(nil, &ServeSettings{}, index, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v2/sessions/phase4-detail-v2", nil)
	request.SetPathValue("id", "phase4-detail-v2")
	response := httptest.NewRecorder()

	server.handleGetSessionV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.GetSessionDetailResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal detail v2: %v", err)
	}
	if payload.Session.GetId() != "phase4-detail-v2" {
		t.Fatalf("unexpected session id %q", payload.Session.GetId())
	}
	if len(payload.Session.GetBlocks()) != 1 {
		t.Fatalf("expected 1 block, got %d", len(payload.Session.GetBlocks()))
	}
	if payload.Session.GetBlocks()[0].GetTurns()[1].GetToolCallsInTurn()[0].GetToolName() != "exec_command" {
		t.Fatalf("unexpected tool name %q", payload.Session.GetBlocks()[0].GetTurns()[1].GetToolCallsInTurn()[0].GetToolName())
	}
	if len(payload.Session.GetEvents()) != 1 || payload.Session.GetEvents()[0].GetKind() != "compaction" {
		t.Fatalf("expected compaction event, got %+v", payload.Session.GetEvents())
	}
	if payload.Session.GetEvents()[0].GetAttachmentId() != "attachment-1" {
		t.Fatalf("expected event attachment link, got %+v", payload.Session.GetEvents()[0])
	}
	if len(payload.Session.GetAttachments()) != 1 || payload.Session.GetAttachments()[0].GetKind() != "image" {
		t.Fatalf("expected image attachment, got %+v", payload.Session.GetAttachments())
	}
	if payload.Session.GetAttachments()[0].GetEventId() != "event-1" {
		t.Fatalf("expected attachment event link, got %+v", payload.Session.GetAttachments()[0])
	}
}

func TestHandleCreateAndGetSessionAnnotationsV2(t *testing.T) {
	ctx := context.Background()
	outputDir := t.TempDir()
	store, err := annotate.Open(ctx, outputDir)
	if err != nil {
		t.Fatalf("annotate.Open returned error: %v", err)
	}
	defer func() { _ = store.Close() }()

	server := NewServer(nil, &ServeSettings{}, map[string]string{}, store, map[string]string{})

	createReq := &apiv1.CreateAnnotationRequest{
		ScopeType:      apiv1.AnnotationScopeType_ANNOTATION_SCOPE_TYPE_TURN,
		TargetId:       "1",
		Category:       apiv1.AnnotationCategory_ANNOTATION_CATEGORY_OBSERVATION,
		Title:          "Investigate this turn",
		Detail:         "Initial note",
		Tags:           []string{"triage", "ui"},
		TaxonomyMast:   []string{"M-OBS"},
		Classification: stringPtr("candidate"),
	}
	createBody, err := protojson.Marshal(createReq)
	if err != nil {
		t.Fatalf("protojson.Marshal createReq: %v", err)
	}
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v2/sessions/sess-v2-ann/annotations", strings.NewReader(string(createBody)))
	createRequest.SetPathValue("id", "sess-v2-ann")
	createResponse := httptest.NewRecorder()

	server.handleCreateAnnotationV2(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d with body %s", createResponse.Code, createResponse.Body.String())
	}

	var created apiv1.Annotation
	if err := protojson.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("protojson.Unmarshal created annotation: %v", err)
	}
	if created.GetScope().GetType() != apiv1.AnnotationScopeType_ANNOTATION_SCOPE_TYPE_TURN {
		t.Fatalf("unexpected scope type %v", created.GetScope().GetType())
	}
	if created.GetContent().GetCategory() != apiv1.AnnotationCategory_ANNOTATION_CATEGORY_OBSERVATION {
		t.Fatalf("unexpected category %v", created.GetContent().GetCategory())
	}
	if len(created.GetContent().GetTags()) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(created.GetContent().GetTags()))
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/v2/sessions/sess-v2-ann/annotations", nil)
	getRequest.SetPathValue("id", "sess-v2-ann")
	getResponse := httptest.NewRecorder()

	server.handleGetSessionAnnotationsV2(getResponse, getRequest)

	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", getResponse.Code, getResponse.Body.String())
	}

	var payload apiv1.GetSessionAnnotationsResponse
	if err := protojson.Unmarshal(getResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal get session annotations v2: %v", err)
	}
	if payload.GetSessionId() != "sess-v2-ann" {
		t.Fatalf("unexpected session id %q", payload.GetSessionId())
	}
	if payload.GetCount() != 1 {
		t.Fatalf("expected count=1, got %d", payload.GetCount())
	}
	if payload.GetAnnotations()[0].GetContent().GetTitle() != "Investigate this turn" {
		t.Fatalf("unexpected title %q", payload.GetAnnotations()[0].GetContent().GetTitle())
	}
}

func TestHandleListAnnotationsV2ReturnsIntentionalRows(t *testing.T) {
	ctx := context.Background()
	outputDir := t.TempDir()
	store, err := annotate.Open(ctx, outputDir)
	if err != nil {
		t.Fatalf("annotate.Open returned error: %v", err)
	}
	defer func() { _ = store.Close() }()

	ann := minitrace.Annotation{
		ID:        "ann-v2-list",
		Timestamp: minitrace.FormatTimestamp(time.Now().UTC()),
		Annotator: "user",
		Scope:     minitrace.AnnotationScope{Type: "tool_call", TargetID: "call-123"},
		Content: minitrace.AnnotationContent{
			Category: "ai-failure",
			Title:    "Bad output",
			Detail:   "Tool failed",
			Tags:     []string{"triage"},
		},
		TaxonomyMappings: minitrace.TaxonomyMappings{
			Minitrace: []string{"F-01"},
		},
	}
	if err := store.AddAnnotation(ctx, ann, "sess-v2-list"); err != nil {
		t.Fatalf("AddAnnotation returned error: %v", err)
	}

	server := NewServer(nil, &ServeSettings{}, map[string]string{}, store, map[string]string{})
	request := httptest.NewRequest(http.MethodGet, "/api/v2/annotations", nil)
	response := httptest.NewRecorder()

	server.handleListAnnotationsV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.ListAnnotationsResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal list annotations v2: %v", err)
	}
	if len(payload.GetAnnotations()) != 1 {
		t.Fatalf("expected 1 row, got %d", len(payload.GetAnnotations()))
	}
	row := payload.GetAnnotations()[0]
	if row.GetSessionId() != "sess-v2-list" {
		t.Fatalf("unexpected session id %q", row.GetSessionId())
	}
	if row.GetScopeType() != apiv1.AnnotationScopeType_ANNOTATION_SCOPE_TYPE_TOOL_CALL {
		t.Fatalf("unexpected scope type %v", row.GetScopeType())
	}
	if !strings.Contains(response.Body.String(), "sessionId") {
		t.Fatalf("expected camelCase sessionId field in response: %s", response.Body.String())
	}
}

func TestHandleUpdateAndDeleteAnnotationV2(t *testing.T) {
	ctx := context.Background()
	outputDir := t.TempDir()
	store, err := annotate.Open(ctx, outputDir)
	if err != nil {
		t.Fatalf("annotate.Open returned error: %v", err)
	}
	defer func() { _ = store.Close() }()

	ann := minitrace.Annotation{
		ID:        "ann-v2-update",
		Timestamp: minitrace.FormatTimestamp(time.Now().UTC()),
		Annotator: "user",
		Scope:     minitrace.AnnotationScope{Type: "session", TargetID: "sess-v2-update"},
		Content: minitrace.AnnotationContent{
			Category: "observation",
			Title:    "Original",
			Detail:   "Old detail",
			Tags:     []string{"old"},
		},
		TaxonomyMappings: minitrace.TaxonomyMappings{
			Minitrace: []string{"OLD"},
		},
	}
	if err := store.AddAnnotation(ctx, ann, "sess-v2-update"); err != nil {
		t.Fatalf("AddAnnotation returned error: %v", err)
	}

	server := NewServer(nil, &ServeSettings{}, map[string]string{}, store, map[string]string{})
	newCategory := apiv1.AnnotationCategory_ANNOTATION_CATEGORY_QUESTION
	updateReq := &apiv1.UpdateAnnotationRequest{
		Title:             stringPtr("Updated"),
		Detail:            stringPtr("New detail"),
		Category:          &newCategory,
		Tags:              &apiv1.StringList{Values: []string{}},
		TaxonomyMinitrace: &apiv1.StringList{Values: []string{"NEW"}},
	}
	updateBody, err := protojson.Marshal(updateReq)
	if err != nil {
		t.Fatalf("protojson.Marshal updateReq: %v", err)
	}
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/v2/annotations/ann-v2-update", strings.NewReader(string(updateBody)))
	updateRequest.SetPathValue("annId", "ann-v2-update")
	updateResponse := httptest.NewRecorder()

	server.handleUpdateAnnotationV2(updateResponse, updateRequest)

	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", updateResponse.Code, updateResponse.Body.String())
	}

	stored, err := store.GetAnnotationsForSession(ctx, "sess-v2-update")
	if err != nil {
		t.Fatalf("GetAnnotationsForSession returned error: %v", err)
	}
	if stored[0].Content.Title != "Updated" {
		t.Fatalf("unexpected updated title %q", stored[0].Content.Title)
	}
	if len(stored[0].Content.Tags) != 0 {
		t.Fatalf("expected tags to be cleared, got %#v", stored[0].Content.Tags)
	}
	if stored[0].Content.Category != "question" {
		t.Fatalf("unexpected updated category %q", stored[0].Content.Category)
	}
	if len(stored[0].TaxonomyMappings.Minitrace) != 1 || stored[0].TaxonomyMappings.Minitrace[0] != "NEW" {
		t.Fatalf("unexpected taxonomy %#v", stored[0].TaxonomyMappings.Minitrace)
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v2/annotations/ann-v2-update", nil)
	deleteRequest.SetPathValue("annId", "ann-v2-update")
	deleteResponse := httptest.NewRecorder()

	server.handleDeleteAnnotationV2(deleteResponse, deleteRequest)

	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", deleteResponse.Code, deleteResponse.Body.String())
	}

	storedAfterDelete, err := store.GetAnnotationsForSession(ctx, "sess-v2-update")
	if err != nil {
		t.Fatalf("GetAnnotationsForSession after delete returned error: %v", err)
	}
	if len(storedAfterDelete) != 0 {
		t.Fatalf("expected 0 annotations after delete, got %d", len(storedAfterDelete))
	}
}

func TestHandleSyncAnnotationsV2ReturnsStructuredReport(t *testing.T) {
	ctx := context.Background()
	archiveRoot := t.TempDir()
	outputDir := t.TempDir()
	session := buildFixtureSession(t, "sess-v2-sync")
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}
	index, err := buildSessionIndex([]string{filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")})
	if err != nil {
		t.Fatalf("buildSessionIndex returned error: %v", err)
	}
	store, err := annotate.Open(ctx, outputDir)
	if err != nil {
		t.Fatalf("annotate.Open returned error: %v", err)
	}
	defer func() { _ = store.Close() }()

	ann := minitrace.Annotation{
		ID:        "ann-v2-sync",
		Timestamp: minitrace.FormatTimestamp(time.Now().UTC()),
		Annotator: "user",
		Scope:     minitrace.AnnotationScope{Type: "session", TargetID: "sess-v2-sync"},
		Content: minitrace.AnnotationContent{
			Category: "success",
			Title:    "Synced annotation",
		},
	}
	if err := store.AddAnnotation(ctx, ann, "sess-v2-sync"); err != nil {
		t.Fatalf("AddAnnotation returned error: %v", err)
	}

	server := NewServer(nil, &ServeSettings{}, map[string]string{}, store, index)
	syncReq := &apiv1.SyncAnnotationsRequest{SessionId: stringPtr("sess-v2-sync")}
	syncBody, err := protojson.Marshal(syncReq)
	if err != nil {
		t.Fatalf("protojson.Marshal syncReq: %v", err)
	}
	syncRequest := httptest.NewRequest(http.MethodPost, "/api/v2/annotations/sync", strings.NewReader(string(syncBody)))
	syncResponse := httptest.NewRecorder()

	server.handleSyncAnnotationsV2(syncResponse, syncRequest)

	if syncResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", syncResponse.Code, syncResponse.Body.String())
	}

	var payload apiv1.SyncAnnotationsResponse
	if err := protojson.Unmarshal(syncResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal sync annotations v2: %v", err)
	}
	if len(payload.GetSynced()) != 1 || payload.GetSynced()[0] != "sess-v2-sync" {
		t.Fatalf("unexpected synced report %#v", payload.GetSynced())
	}

	sessionPath := index["sess-v2-sync"]
	payloadBytes, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatalf("reading synced session file: %v", err)
	}
	var syncedSession minitrace.Session
	if err := json.Unmarshal(payloadBytes, &syncedSession); err != nil {
		t.Fatalf("json.Unmarshal synced session: %v", err)
	}
	if len(syncedSession.Annotations) != 1 || syncedSession.Annotations[0].Content.Title != "Synced annotation" {
		t.Fatalf("unexpected synced annotations %#v", syncedSession.Annotations)
	}
}

func TestLegacyAnnotationRoutesReturnNotFound(t *testing.T) {
	server := NewServer(nil, &ServeSettings{DevMode: true}, map[string]string{}, nil, nil)

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/sessions/sess-v2-ann/annotations"},
		{method: http.MethodPost, path: "/api/sessions/sess-v2-ann/annotations"},
		{method: http.MethodGet, path: "/api/annotations"},
		{method: http.MethodPut, path: "/api/annotations/ann-v2-update"},
		{method: http.MethodDelete, path: "/api/annotations/ann-v2-update"},
		{method: http.MethodPost, path: "/api/annotations/sync"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, nil)
			response := httptest.NewRecorder()

			server.mux.ServeHTTP(response, request)

			if response.Code != http.StatusNotFound {
				t.Fatalf("expected 404, got %d with body %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestLegacyPresetAndQueryRoutesReturnNotFound(t *testing.T) {
	server := NewServer(nil, &ServeSettings{DevMode: true}, map[string]string{}, nil, nil)

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/presets"},
		{method: http.MethodGet, path: "/api/queries"},
		{method: http.MethodPost, path: "/api/queries"},
		{method: http.MethodPut, path: "/api/queries/saved/analysis/wesen-os-filter.sql"},
		{method: http.MethodDelete, path: "/api/queries/saved/analysis/wesen-os-filter.sql"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, nil)
			response := httptest.NewRecorder()

			server.mux.ServeHTTP(response, request)

			if response.Code != http.StatusNotFound {
				t.Fatalf("expected 404, got %d with body %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestHandleGetPresetsV2ReturnsEnvelopeAndQueries(t *testing.T) {
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
		PresetDir: []string{presetDir1, presetDir2},
	}, map[string]string{}, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v2/presets", nil)
	response := httptest.NewRecorder()

	server.handleGetPresetsV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.ListPresetsResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal presets v2: %v", err)
	}
	if payload.GetMeta().GetSchemaVersion() != 1 {
		t.Fatalf("unexpected schema version %d", payload.GetMeta().GetSchemaVersion())
	}
	foundBuiltIn := false
	foundCustom := false
	foundExtra := false
	for _, query := range payload.GetPresets() {
		if query.GetName() == "session-list" && query.GetFolder() == "core/overview" && query.GetPath() == "core/overview/session-list.sql" && query.GetReadonly() {
			foundBuiltIn = true
		}
		if query.GetName() == "custom" && query.GetDescription() == "custom preset" && query.GetReadonly() {
			foundCustom = true
		}
		if query.GetPath() == "analysis/extra.sql" && query.GetDescription() == "extra preset" && query.GetReadonly() {
			foundExtra = true
		}
	}
	if !foundBuiltIn || !foundCustom || !foundExtra {
		t.Fatalf("expected built-in/custom/extra presets in %#v", payload.GetPresets())
	}
}

func TestHandleGetQueryCommandsV2ReturnsEmbeddedCatalog(t *testing.T) {
	server := NewServer(nil, &ServeSettings{}, map[string]string{}, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v2/query-commands", nil)
	response := httptest.NewRecorder()

	server.handleGetQueryCommandsV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.ListQueryCommandsResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal query commands v2: %v", err)
	}
	if payload.GetMeta().GetSchemaVersion() != 1 {
		t.Fatalf("unexpected schema version %d", payload.GetMeta().GetSchemaVersion())
	}

	foundSessionList := false
	foundAlias := false
	for _, command := range payload.GetCommands() {
		if command.GetPath() == "overview/session-list.sql" {
			foundSessionList = true
			if command.GetKind() != apiv1.QueryCommandKind_QUERY_COMMAND_KIND_VERB {
				t.Fatalf("session-list kind = %v, want verb", command.GetKind())
			}
			if len(command.GetFlags()) == 0 {
				t.Fatalf("session-list should expose flags")
			}
			if !strings.Contains(command.GetRawSql(), "FROM sessions s") {
				t.Fatalf("session-list raw_sql missing template body: %q", command.GetRawSql())
			}
			if command.GetRawSqlPath() != "overview/session-list.sql" {
				t.Fatalf("session-list raw_sql_path = %q, want overview/session-list.sql", command.GetRawSqlPath())
			}
		}
		if command.GetPath() == "overview/aliases/codex-framework-summary.alias.yaml" {
			foundAlias = true
			if command.GetKind() != apiv1.QueryCommandKind_QUERY_COMMAND_KIND_ALIAS {
				t.Fatalf("alias kind = %v, want alias", command.GetKind())
			}
			if command.GetAliasFor() != "framework-summary" {
				t.Fatalf("aliasFor = %q, want framework-summary", command.GetAliasFor())
			}
			if len(command.GetFlags()) == 0 {
				t.Fatalf("alias should expose target flags for form rendering")
			}
			if command.GetRawSqlPath() != "overview/framework-summary.sql" {
				t.Fatalf("alias raw_sql_path = %q, want overview/framework-summary.sql", command.GetRawSqlPath())
			}
			if !strings.Contains(command.GetRawSql(), "GROUP BY framework") {
				t.Fatalf("alias raw_sql should expose target template body: %q", command.GetRawSql())
			}
		}
	}
	if !foundSessionList || !foundAlias {
		t.Fatalf("expected embedded verb and alias commands, got %#v", payload.GetCommands())
	}
}

func TestHandleGetQueryCommandsV2UsesConfiguredSourceRoots(t *testing.T) {
	repo := t.TempDir()
	content := `/* sqleton
name: session-list
short: Override session list from external repository
*/
SELECT 99 AS answer FROM {{TABLE_NAME}};`
	if err := os.WriteFile(filepath.Join(repo, "session-list.sql"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := NewServer(nil, &ServeSettings{}, map[string]string{}, nil, nil)
	server.commandSourceRoots = minitracecmd.SourceRootsFromPaths([]string{repo})
	request := httptest.NewRequest(http.MethodGet, "/api/v2/query-commands", nil)
	response := httptest.NewRecorder()

	server.handleGetQueryCommandsV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.ListQueryCommandsResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal query commands v2: %v", err)
	}

	foundOverride := false
	for _, command := range payload.GetCommands() {
		if command.GetPath() == "session-list.sql" {
			foundOverride = true
			if command.GetShortDescription() != "Override session list from external repository" {
				t.Fatalf("shortDescription = %q, want override description", command.GetShortDescription())
			}
		}
	}
	if !foundOverride {
		t.Fatalf("expected overridden session-list in %#v", payload.GetCommands())
	}
}

func TestHandleExecuteQueryCommandV2RenderOnlyReturnsRenderedSQL(t *testing.T) {
	server := NewServer(nil, &ServeSettings{}, map[string]string{}, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/v2/query-commands/overview/framework-summary.sql/execute", strings.NewReader(`{"values":{"framework":["codex"]},"renderOnly":true}`))
	request.SetPathValue("path", "overview/framework-summary.sql/execute")
	response := httptest.NewRecorder()

	server.handleExecuteQueryCommandV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.ExecuteQueryCommandResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal execute query command render-only: %v", err)
	}
	if !strings.Contains(payload.GetRenderedSql(), "FROM sessions s") {
		t.Fatalf("rendered_sql missing normalized sessions table: %q", payload.GetRenderedSql())
	}
	if !strings.Contains(payload.GetRenderedSql(), "IN ('codex')") {
		t.Fatalf("rendered_sql missing framework filter: %q", payload.GetRenderedSql())
	}
}

func TestHandleExecuteQueryCommandV2RenderOnlyHydratesSQLDefaults(t *testing.T) {
	server := NewServer(nil, &ServeSettings{}, map[string]string{}, nil, nil)
	server.commandSourceRoots = minitracecmd.SourceRootsFromPaths([]string{checkedInQueryRepositoryRoot(t, "mixed-sql-js-showcase")})
	request := httptest.NewRequest(http.MethodPost, "/api/v2/query-commands/overview/framework-summary.sql/execute", strings.NewReader(`{"renderOnly":true}`))
	request.SetPathValue("path", "overview/framework-summary.sql/execute")
	response := httptest.NewRecorder()

	server.handleExecuteQueryCommandV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.ExecuteQueryCommandResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal execute query command render-only defaults: %v", err)
	}
	if strings.Contains(payload.GetRenderedSql(), "<no value>") {
		t.Fatalf("rendered_sql still contains missing placeholder: %q", payload.GetRenderedSql())
	}
	if !strings.Contains(payload.GetRenderedSql(), "LIMIT 10") {
		t.Fatalf("rendered_sql missing hydrated default limit: %q", payload.GetRenderedSql())
	}
}

func TestHandleExecuteQueryCommandV2ExecutesAliasAgainstLoadedArchive(t *testing.T) {
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t, "phase5-query-command")
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	archiveGlob := filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")
	target := newTestQueryTarget(t, archiveGlob)
	server := NewServer(target, &ServeSettings{ArchiveGlob: []string{archiveGlob}}, map[string]string{}, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/v2/query-commands/overview/aliases/codex-framework-summary.alias.yaml/execute", strings.NewReader(`{}`))
	request.SetPathValue("path", "overview/aliases/codex-framework-summary.alias.yaml/execute")
	response := httptest.NewRecorder()

	server.handleExecuteQueryCommandV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.ExecuteQueryCommandResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal execute query command response: %v", err)
	}
	if payload.GetRowCount() != 1 {
		t.Fatalf("expected row_count=1, got %d", payload.GetRowCount())
	}
	if !strings.Contains(payload.GetRenderedSql(), "IN ('codex')") {
		t.Fatalf("rendered_sql missing alias-provided framework filter: %q", payload.GetRenderedSql())
	}
	if got := payload.GetRows()[0].GetFields()["framework"].GetStringValue(); got != "codex" {
		t.Fatalf("expected framework codex, got %q", got)
	}
}

func TestHandleExecuteQueryCommandV2SQLUsesServerQueryTargetLimits(t *testing.T) {
	archiveRoot := t.TempDir()
	for _, id := range []string{"query-command-limit-a", "query-command-limit-b", "query-command-limit-c"} {
		if _, err := minitrace.WriteSession(buildFixtureSession(t, id), archiveRoot); err != nil {
			t.Fatalf("WriteSession returned error: %v", err)
		}
	}

	archiveGlob := filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	target, err := minitracejs.NewArchiveQueryTarget(context.Background(), []string{archiveGlob}, serveQueryOptions(2, 30000))
	if err != nil {
		t.Fatalf("NewArchiveQueryTarget returned error: %v", err)
	}
	t.Cleanup(func() { _ = target.Close() })

	server := NewServer(target, &ServeSettings{ArchiveGlob: []string{archiveGlob}}, map[string]string{}, nil, nil)
	server.commandSourceRoots = []minitracecmd.SourceRoot{{
		Name: "test-root",
		FS: fstest.MapFS{
			"queries/all-sessions.sql": &fstest.MapFile{Data: []byte(`/* sqleton
name: all-sessions
short: List all sessions without an explicit SQL LIMIT
*/
SELECT id FROM {{TABLE_NAME}} ORDER BY id;`)},
		},
		RootDir:  "queries",
		Readonly: true,
	}}

	request := httptest.NewRequest(http.MethodPost, "/api/v2/query-commands/all-sessions.sql/execute", strings.NewReader(`{}`))
	request.SetPathValue("path", "all-sessions.sql/execute")
	response := httptest.NewRecorder()

	server.handleExecuteQueryCommandV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.ExecuteQueryCommandResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal execute query command response: %v", err)
	}
	if payload.GetRowCount() != 2 {
		t.Fatalf("expected query command to honor server max rows=2, got row_count=%d rows=%#v", payload.GetRowCount(), payload.GetRows())
	}
}

func TestHandleExecuteQueryCommandV2RenderOnlyUsesCallerOverrideThenAliasThenCommandDefaults(t *testing.T) {
	server := NewServer(nil, &ServeSettings{}, map[string]string{}, nil, nil)
	server.commandSourceRoots = []minitracecmd.SourceRoot{{
		Name: "test-root",
		FS: fstest.MapFS{
			"queries/overview/framework-summary.sql": &fstest.MapFile{Data: []byte(`/* sqleton
name: framework-summary
short: Summarize sessions by framework
flags:
  - name: framework
    type: stringList
    help: Restrict to specific agent frameworks
  - name: limit
    type: int
    default: 10
    help: Maximum number of rows to return
*/
SELECT
  environment->>'agent_framework' AS framework,
  COUNT(*) AS session_count
FROM {{TABLE_NAME}}
WHERE 1=1
{{ if .framework -}}
AND environment->>'agent_framework' IN ({{ .framework | sqlStringIn }})
{{ end -}}
GROUP BY 1
ORDER BY session_count DESC
LIMIT {{ .limit }};`)},
			"queries/overview/aliases/custom-framework-summary.alias.yaml": &fstest.MapFile{Data: []byte(`name: custom-framework-summary
short: Summary using alias defaults
aliasFor: framework-summary
flags:
  framework:
    - pi
`)},
		},
		RootDir:  "queries",
		Readonly: true,
	}}

	request := httptest.NewRequest(http.MethodPost, "/api/v2/query-commands/overview/aliases/custom-framework-summary.alias.yaml/execute", strings.NewReader(`{"values":{"framework":["codex"]},"renderOnly":true}`))
	request.SetPathValue("path", "overview/aliases/custom-framework-summary.alias.yaml/execute")
	response := httptest.NewRecorder()

	server.handleExecuteQueryCommandV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.ExecuteQueryCommandResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal precedence render-only response: %v", err)
	}
	if !strings.Contains(payload.GetRenderedSql(), "IN ('codex')") {
		t.Fatalf("rendered_sql missing caller override framework: %q", payload.GetRenderedSql())
	}
	if strings.Contains(payload.GetRenderedSql(), "IN ('pi')") {
		t.Fatalf("rendered_sql should not keep alias framework once caller overrides it: %q", payload.GetRenderedSql())
	}
	if !strings.Contains(payload.GetRenderedSql(), "LIMIT 10") {
		t.Fatalf("rendered_sql missing command default limit: %q", payload.GetRenderedSql())
	}
}

func TestHandleExecuteQueryCommandV2ExecutesJSCommandAgainstLoadedArchive(t *testing.T) {
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t, "phase5-js-query-command")
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	archiveGlob := filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")
	target := newTestQueryTarget(t, archiveGlob)
	server := NewServer(target, &ServeSettings{ArchiveGlob: []string{archiveGlob}}, map[string]string{}, nil, nil)
	server.commandSourceRoots = []minitracecmd.SourceRoot{{
		Name: "test-root",
		FS: fstest.MapFS{
			"queries/overview/session-list.js": &fstest.MapFile{Data: []byte(`
__section__("filters", {
  fields: {
    limit: { type: "int", default: 10 }
  }
});

function sessionList(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().Build();
  return db.query(` + "`" + `
    SELECT session_id AS id
    FROM sessions
    LIMIT ${filters.limit}
  ` + "`" + `);
}

__verb__("sessionList", {
  name: "session-list",
  short: "List sessions",
  fields: {
    filters: { bind: "filters" }
  }
});
`)},
		},
		RootDir:  "queries",
		Readonly: true,
	}}

	request := httptest.NewRequest(http.MethodPost, "/api/v2/query-commands/overview/session-list/execute", strings.NewReader(`{"values":{"limit":1}}`))
	request.SetPathValue("path", "overview/session-list/execute")
	response := httptest.NewRecorder()

	server.handleExecuteQueryCommandV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.ExecuteQueryCommandResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal execute js query command response: %v", err)
	}
	if payload.GetRowCount() != 1 {
		t.Fatalf("expected row_count=1, got %d", payload.GetRowCount())
	}
	if got := payload.GetRows()[0].GetFields()["id"].GetStringValue(); got != "phase5-js-query-command" {
		t.Fatalf("expected returned id phase5-js-query-command, got %q", got)
	}
}

func TestHandleExecuteQueryCommandV2ExecutesCheckedInJSShowcaseAlias(t *testing.T) {
	server := newLoadedQueryCommandServer(t, checkedInQueryRepositoryRoot(t, "js-showcase"),
		fixtureSessionForQueryRepository(t, "phase6-js-alias-a", "/tmp/workspaces/alpha/project"),
		fixtureSessionForQueryRepository(t, "phase6-js-alias-b", "/tmp/workspaces/beta/project"),
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v2/query-commands/analysis/aliases/focus-top-workspaces.alias.yaml/execute", strings.NewReader(`{}`))
	request.SetPathValue("path", "analysis/aliases/focus-top-workspaces.alias.yaml/execute")
	response := httptest.NewRecorder()

	server.handleExecuteQueryCommandV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.ExecuteQueryCommandResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal execute checked-in js alias response: %v", err)
	}
	if payload.GetRowCount() == 0 {
		t.Fatalf("expected alias to return rows, got %d", payload.GetRowCount())
	}
	if got := payload.GetRows()[0].GetFields()["workspace_slug"].GetStringValue(); got == "" {
		t.Fatalf("expected non-empty workspace_slug in %#v", payload.GetRows()[0].GetFields())
	}
}

func TestHandleExecuteQueryCommandV2ExecutesCheckedInMixedShowcaseSQL(t *testing.T) {
	server := newLoadedQueryCommandServer(t, checkedInQueryRepositoryRoot(t, "mixed-sql-js-showcase"),
		fixtureSessionForQueryRepository(t, "phase6-mixed-sql", "/tmp/workspaces/mixed/project"),
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v2/query-commands/overview/framework-summary.sql/execute", strings.NewReader(`{}`))
	request.SetPathValue("path", "overview/framework-summary.sql/execute")
	response := httptest.NewRecorder()

	server.handleExecuteQueryCommandV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.ExecuteQueryCommandResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal execute checked-in mixed sql response: %v", err)
	}
	if payload.GetRowCount() != 1 {
		t.Fatalf("expected row_count=1, got %d", payload.GetRowCount())
	}
	if got := payload.GetRows()[0].GetFields()["framework"].GetStringValue(); got != "codex" {
		t.Fatalf("expected framework codex, got %q", got)
	}
}

func TestHandleExecuteQueryCommandV2ExecutesCheckedInMixedShowcaseJSWithDefaults(t *testing.T) {
	server := newLoadedQueryCommandServer(t, checkedInQueryRepositoryRoot(t, "mixed-sql-js-showcase"),
		fixtureSessionForQueryRepository(t, "phase6-mixed-js-default-a", "/tmp/workspaces/mixed/a"),
		fixtureSessionForQueryRepository(t, "phase6-mixed-js-default-b", "/tmp/workspaces/mixed/b"),
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v2/query-commands/overview/session-tools/session-list/execute", strings.NewReader(`{}`))
	request.SetPathValue("path", "overview/session-tools/session-list/execute")
	response := httptest.NewRecorder()

	server.handleExecuteQueryCommandV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.ExecuteQueryCommandResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal execute checked-in mixed js defaults response: %v", err)
	}
	if payload.GetRowCount() != 2 {
		t.Fatalf("expected row_count=2, got %d", payload.GetRowCount())
	}
}

func TestHandleExecuteQueryCommandV2ExecutesCheckedInMixedShowcaseJSAlias(t *testing.T) {
	server := newLoadedQueryCommandServer(t, checkedInQueryRepositoryRoot(t, "mixed-sql-js-showcase"),
		fixtureSessionForQueryRepository(t, "phase6-mixed-js-a", "/tmp/workspaces/mixed/a"),
		fixtureSessionForQueryRepository(t, "phase6-mixed-js-b", "/tmp/workspaces/mixed/b"),
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v2/query-commands/analysis/aliases/top-workspaces.alias.yaml/execute", strings.NewReader(`{}`))
	request.SetPathValue("path", "analysis/aliases/top-workspaces.alias.yaml/execute")
	response := httptest.NewRecorder()

	server.handleExecuteQueryCommandV2(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", response.Code, response.Body.String())
	}

	var payload apiv1.ExecuteQueryCommandResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal execute checked-in mixed js alias response: %v", err)
	}
	if payload.GetRowCount() == 0 {
		t.Fatalf("expected mixed js alias to return rows, got %d", payload.GetRowCount())
	}
	if got := payload.GetRows()[0].GetFields()["working_directory"].GetStringValue(); got == "" {
		t.Fatalf("expected non-empty working_directory in %#v", payload.GetRows()[0].GetFields())
	}
}

func TestHandleExecuteQueryCommandV2ReturnsNotFoundForUnknownCommand(t *testing.T) {
	server := NewServer(nil, &ServeSettings{}, map[string]string{}, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/v2/query-commands/missing.sql/execute", strings.NewReader(`{}`))
	request.SetPathValue("path", "missing.sql/execute")
	response := httptest.NewRecorder()

	server.handleExecuteQueryCommandV2(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d with body %s", response.Code, response.Body.String())
	}
}

func TestQueryCRUDV2ValidatesPathsAndPersistsQueries(t *testing.T) {
	queryDir1 := t.TempDir()
	queryDir2 := t.TempDir()
	if err := os.MkdirAll(filepath.Join(queryDir2, "shared"), 0o755); err != nil {
		t.Fatalf("creating second query dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(queryDir2, "shared", "team.sql"), []byte("-- team query\nSELECT 9;"), 0o644); err != nil {
		t.Fatalf("writing second-root query: %v", err)
	}
	server := NewServer(nil, &ServeSettings{
		QueryDir: []string{queryDir1, queryDir2},
	}, map[string]string{}, nil, nil)

	saveReq := &apiv1.SaveQueryRequest{
		Name:        "wesen-os-filter",
		Folder:      "saved/analysis",
		Description: "saved query",
		Sql:         "SELECT 1;",
	}
	saveBody, err := protojson.Marshal(saveReq)
	if err != nil {
		t.Fatalf("protojson.Marshal saveReq: %v", err)
	}
	saveRequest := httptest.NewRequest(http.MethodPost, "/api/v2/queries", strings.NewReader(string(saveBody)))
	saveResponse := httptest.NewRecorder()
	server.handleSaveQueryV2(saveResponse, saveRequest)

	if saveResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d with body %s", saveResponse.Code, saveResponse.Body.String())
	}

	var saved apiv1.SavedQuery
	if err := protojson.Unmarshal(saveResponse.Body.Bytes(), &saved); err != nil {
		t.Fatalf("protojson.Unmarshal saved query: %v", err)
	}
	if saved.GetPath() != "saved/analysis/wesen-os-filter.sql" {
		t.Fatalf("unexpected saved path %q", saved.GetPath())
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/v2/queries", nil)
	getResponse := httptest.NewRecorder()
	server.handleGetQueriesV2(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", getResponse.Code, getResponse.Body.String())
	}
	var queries apiv1.ListQueriesResponse
	if err := protojson.Unmarshal(getResponse.Body.Bytes(), &queries); err != nil {
		t.Fatalf("protojson.Unmarshal queries v2: %v", err)
	}
	foundSaved := false
	foundSecondRoot := false
	for _, query := range queries.GetQueries() {
		if query.GetPath() == "saved/analysis/wesen-os-filter.sql" {
			foundSaved = true
		}
		if query.GetPath() == "shared/team.sql" && query.GetDescription() == "team query" {
			foundSecondRoot = true
		}
	}
	if !foundSaved || !foundSecondRoot {
		t.Fatalf("expected queries from both roots, got %#v", queries.GetQueries())
	}

	updateReq := &apiv1.UpdateQueryRequest{Description: "updated query", Sql: "SELECT 2;"}
	updateBody, err := protojson.Marshal(updateReq)
	if err != nil {
		t.Fatalf("protojson.Marshal updateReq: %v", err)
	}
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/v2/queries/saved/analysis/wesen-os-filter.sql", strings.NewReader(string(updateBody)))
	updateRequest.SetPathValue("path", "saved/analysis/wesen-os-filter.sql")
	updateResponse := httptest.NewRecorder()
	server.handleUpdateQueryV2(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", updateResponse.Code, updateResponse.Body.String())
	}

	updateSecondRootReq := &apiv1.UpdateQueryRequest{Description: "updated team query", Sql: "SELECT 10;"}
	updateSecondRootBody, err := protojson.Marshal(updateSecondRootReq)
	if err != nil {
		t.Fatalf("protojson.Marshal updateSecondRootReq: %v", err)
	}
	updateSecondRootRequest := httptest.NewRequest(http.MethodPut, "/api/v2/queries/shared/team.sql", strings.NewReader(string(updateSecondRootBody)))
	updateSecondRootRequest.SetPathValue("path", "shared/team.sql")
	updateSecondRootResponse := httptest.NewRecorder()
	server.handleUpdateQueryV2(updateSecondRootResponse, updateSecondRootRequest)
	if updateSecondRootResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 for second-root update, got %d with body %s", updateSecondRootResponse.Code, updateSecondRootResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v2/queries/saved/analysis/wesen-os-filter.sql", nil)
	deleteRequest.SetPathValue("path", "saved/analysis/wesen-os-filter.sql")
	deleteResponse := httptest.NewRecorder()
	server.handleDeleteQueryV2(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", deleteResponse.Code, deleteResponse.Body.String())
	}
	var deleted apiv1.DeleteQueryResponse
	if err := protojson.Unmarshal(deleteResponse.Body.Bytes(), &deleted); err != nil {
		t.Fatalf("protojson.Unmarshal deleted query response: %v", err)
	}
	if deleted.GetStatus() != "deleted" || deleted.GetPath() != "saved/analysis/wesen-os-filter.sql" {
		t.Fatalf("unexpected delete response status=%q path=%q", deleted.GetStatus(), deleted.GetPath())
	}

	traversalSaveReq := &apiv1.SaveQueryRequest{Name: "bad", Folder: "../escape", Description: "x", Sql: "SELECT 1;"}
	traversalSaveBody, err := protojson.Marshal(traversalSaveReq)
	if err != nil {
		t.Fatalf("protojson.Marshal traversalSaveReq: %v", err)
	}
	traversalRequest := httptest.NewRequest(http.MethodPost, "/api/v2/queries", strings.NewReader(string(traversalSaveBody)))
	traversalResponse := httptest.NewRecorder()
	server.handleSaveQueryV2(traversalResponse, traversalRequest)
	if traversalResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for traversal, got %d with body %s", traversalResponse.Code, traversalResponse.Body.String())
	}

	absoluteUpdateReq := &apiv1.UpdateQueryRequest{Description: "x", Sql: "SELECT 1;"}
	absoluteUpdateBody, err := protojson.Marshal(absoluteUpdateReq)
	if err != nil {
		t.Fatalf("protojson.Marshal absoluteUpdateReq: %v", err)
	}
	absoluteUpdateRequest := httptest.NewRequest(http.MethodPut, "/api/v2/queries/tmp/evil.sql", strings.NewReader(string(absoluteUpdateBody)))
	absoluteUpdateRequest.SetPathValue("path", "/tmp/evil.sql")
	absoluteUpdateResponse := httptest.NewRecorder()
	server.handleUpdateQueryV2(absoluteUpdateResponse, absoluteUpdateRequest)
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

func checkedInQueryRepositoryRoot(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "testdata", "query-repositories", name))
}

func newLoadedQueryCommandServer(t *testing.T, repoRoot string, sessions ...*minitrace.Session) *Server {
	t.Helper()

	archiveRoot := t.TempDir()
	for _, session := range sessions {
		if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
			t.Fatalf("WriteSession returned error: %v", err)
		}
	}

	archiveGlob := filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")
	target := newTestQueryTarget(t, archiveGlob)
	server := NewServer(target, &ServeSettings{ArchiveGlob: []string{archiveGlob}}, map[string]string{}, nil, nil)
	server.commandSourceRoots = minitracecmd.SourceRootsFromPaths([]string{repoRoot})
	return server
}

func fixtureSessionForQueryRepository(t *testing.T, sessionID, workingDirectory string) *minitrace.Session {
	t.Helper()
	session := buildFixtureSession(t, sessionID)
	session.OperationalContext.WorkingDirectory = stringPtr(workingDirectory)
	session.Title = stringPtr("Fixture " + sessionID)
	return session
}

// TestBuildServeQueryTargetAttachesLiveAnnotations verifies that serve's SQL
// surface reads the annotation store live via the same-engine ATTACH: rows
// written to annotations.db after startup are visible without a rebuild.
func TestBuildServeQueryTargetAttachesLiveAnnotations(t *testing.T) {
	ctx := context.Background()
	archiveRoot := t.TempDir()
	session := buildFixtureSession(t, "phase3-anno-live")
	if _, err := minitrace.WriteSession(session, archiveRoot); err != nil {
		t.Fatalf("WriteSession returned error: %v", err)
	}

	store, err := annotate.Open(ctx, archiveRoot)
	if err != nil {
		t.Fatalf("annotate.Open returned error: %v", err)
	}
	defer func() { _ = store.Close() }()
	firstAnn := minitrace.Annotation{
		ID:        "ann-live-001",
		Timestamp: minitrace.FormatTimestamp(time.Now().UTC()),
		Annotator: "tester",
		Scope:     minitrace.AnnotationScope{Type: "session", TargetID: "phase3-anno-live"},
		Content:   minitrace.AnnotationContent{Category: "observation", Title: "before startup"},
	}
	if err := store.AddAnnotation(ctx, firstAnn, "phase3-anno-live"); err != nil {
		t.Fatalf("AddAnnotation returned error: %v", err)
	}

	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	glob := filepath.Join(archiveRoot, "active", "*", "*.minitrace.json")
	target, attached, err := buildServeQueryTarget(ctx, []string{glob}, archiveRoot, serveQueryOptions(10000, 30000))
	if err != nil {
		t.Fatalf("buildServeQueryTarget returned error: %v", err)
	}
	defer func() { _ = target.Close() }()
	if !attached {
		t.Fatalf("expected annotations to be attached")
	}

	rows, err := target.Query(ctx, "SELECT title FROM anno.annotations ORDER BY title")
	if err != nil {
		t.Fatalf("querying anno.annotations: %v", err)
	}
	if len(rows) != 1 || rows[0]["title"] != "before startup" {
		t.Fatalf("unexpected annotation rows %#v", rows)
	}

	// Live read: a row added after the target was built must be visible.
	secondAnn := firstAnn
	secondAnn.ID = "ann-live-002"
	secondAnn.Content.Title = "after startup"
	if err := store.AddAnnotation(ctx, secondAnn, "phase3-anno-live"); err != nil {
		t.Fatalf("AddAnnotation returned error: %v", err)
	}
	rows, err = target.Query(ctx, "SELECT title FROM anno.annotations ORDER BY title")
	if err != nil {
		t.Fatalf("querying anno.annotations after write: %v", err)
	}
	if len(rows) != 2 || rows[1]["title"] != "before startup" || rows[0]["title"] != "after startup" {
		t.Fatalf("expected live annotation rows, got %#v", rows)
	}

	// The sandbox still denies everything outside the allowlist.
	result, err := target.QueryResult(ctx, "SELECT * FROM sqlite_master")
	if err != nil {
		t.Fatalf("QueryResult(sqlite_master) returned error: %v", err)
	}
	if result.Error == "" {
		t.Fatalf("expected sqlite_master to be denied")
	}

	// The normalized archive tables are still queryable on the same target.
	row, err := target.QueryOne(ctx, "SELECT COUNT(*) AS n FROM sessions")
	if err != nil {
		t.Fatalf("querying sessions: %v", err)
	}
	if row["n"] != int64(1) {
		t.Fatalf("expected 1 session, got %#v", row["n"])
	}
}

// newTestQueryTarget builds the normalized SQLite query target for the given
// archive globs, isolating the builder disk cache in a per-test directory.
func newTestQueryTarget(t *testing.T, archiveGlobs ...string) minitracedb.QueryTarget {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	target, err := minitracejs.NewArchiveQueryTarget(context.Background(), archiveGlobs, serveQueryOptions(10000, 30000))
	if err != nil {
		t.Fatalf("NewArchiveQueryTarget returned error: %v", err)
	}
	t.Cleanup(func() { _ = target.Close() })
	return target
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
