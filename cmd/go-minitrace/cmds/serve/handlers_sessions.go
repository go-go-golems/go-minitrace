package serve

import (
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	queryengine "github.com/go-go-golems/go-minitrace/pkg/query"
	"github.com/pkg/errors"
)

type SessionSummaryResponse struct {
	ID                 string                            `json:"id"`
	Title              string                            `json:"title"`
	Summary            *string                           `json:"summary"`
	Classification     string                            `json:"classification"`
	Timing             SessionTimingResponse             `json:"timing"`
	Metrics            SessionMetricsResponse            `json:"metrics"`
	Environment        SessionEnvironmentResponse        `json:"environment"`
	OperationalContext SessionOperationalContextResponse `json:"operational_context"`
}

type SessionSummaryDetailResponse struct {
	SessionSummaryResponse
	Provenance SessionProvenanceResponse `json:"provenance"`
}

type SessionDetailResponse struct {
	SessionSummaryDetailResponse
	Blocks []SessionBlock `json:"blocks"`
}

type SessionTimingResponse struct {
	StartedAt             string  `json:"started_at"`
	EndedAt               *string `json:"ended_at"`
	DurationSeconds       float64 `json:"duration_seconds"`
	ActiveDurationSeconds float64 `json:"active_duration_seconds"`
	HourOfDay             int     `json:"hour_of_day"`
	DayOfWeek             int     `json:"day_of_week"`
}

type SessionMetricsResponse struct {
	TurnCount            int  `json:"turn_count"`
	ToolCallCount        int  `json:"tool_call_count"`
	TotalInputTokens     *int `json:"total_input_tokens,omitempty"`
	TotalOutputTokens    *int `json:"total_output_tokens,omitempty"`
	TotalCacheReadTokens *int `json:"total_cache_read_tokens,omitempty"`
}

type SessionEnvironmentResponse struct {
	AgentFramework string `json:"agent_framework"`
	Model          string `json:"model"`
}

type SessionOperationalContextResponse struct {
	WorkingDirectory string `json:"working_directory"`
	AutonomyLevel    string `json:"autonomy_level,omitempty"`
	Sandbox          *bool  `json:"sandbox,omitempty"`
}

type SessionProvenanceResponse struct {
	SourceFormat      string `json:"source_format"`
	SourcePath        string `json:"source_path"`
	OriginalSessionID string `json:"original_session_id"`
	ConvertedAt       string `json:"converted_at"`
}

type BadgeType string

type SessionBlock struct {
	BlockNum    int            `json:"block_num"`
	UserTurnIdx int            `json:"user_turn_idx"`
	UserTs      string         `json:"user_ts"`
	UserContent string         `json:"user_content"`
	AgentTurns  int            `json:"agent_turns"`
	ToolCalls   int            `json:"tool_calls"`
	GapMinutes  *float64       `json:"gap_minutes"`
	Turns       []TurnResponse `json:"turns"`
	Artifacts   BlockArtifacts `json:"artifacts"`
}

type TurnResponse struct {
	Idx             int                `json:"idx"`
	Role            string             `json:"role"`
	Source          string             `json:"source"`
	Content         string             `json:"content"`
	Timestamp       string             `json:"timestamp"`
	ToolCallsInTurn []ToolCallResponse `json:"tool_calls_in_turn"`
}

type ToolCallResponse struct {
	ID            string         `json:"id"`
	ToolName      string         `json:"tool_name"`
	Timestamp     string         `json:"timestamp"`
	OperationType string         `json:"operation_type"`
	Input         ToolCallInput  `json:"input"`
	Output        ToolCallOutput `json:"output"`
	Badges        []BadgeType    `json:"badges"`
}

type ToolCallInput struct {
	Command   string         `json:"command,omitempty"`
	Arguments map[string]any `json:"arguments,omitempty"`
	FilePath  string         `json:"file_path,omitempty"`
}

type ToolCallOutput struct {
	Success    bool    `json:"success"`
	Result     *string `json:"result"`
	Error      *string `json:"error"`
	DurationMs int     `json:"duration_ms"`
	Truncated  bool    `json:"truncated"`
}

type BlockArtifacts struct {
	Commits        []string `json:"commits"`
	TicketsCreated []string `json:"tickets_created"`
	DocsAdded      []string `json:"docs_added"`
	DiaryWrites    int      `json:"diary_writes"`
}

func (s *Server) handleGetSessions(w http.ResponseWriter, r *http.Request) {
	sqlText, err := buildSessionListSQL(s.tableName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: err.Error()},
		})
		return
	}

	rows, err := s.conn.QueryContext(r.Context(), sqlText)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: errors.Wrap(err, "querying sessions").Error()},
		})
		return
	}
	defer func() { _ = rows.Close() }()

	summaries := make([]SessionSummaryResponse, 0)
	for rows.Next() {
		values := make([]any, 8)
		scanArgs := make([]any, len(values))
		for i := range scanArgs {
			scanArgs[i] = &values[i]
		}
		if err := rows.Scan(scanArgs...); err != nil {
			writeJSON(w, http.StatusInternalServerError, QueryResponse{
				Columns:    []string{},
				Rows:       []map[string]any{},
				DurationMS: 0,
				RowCount:   0,
				Error:      &QueryError{Message: errors.Wrap(err, "scanning session summary row").Error()},
			})
			return
		}

		summary, err := sessionSummaryFromValues(values)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, QueryResponse{
				Columns:    []string{},
				Rows:       []map[string]any{},
				DurationMS: 0,
				RowCount:   0,
				Error:      &QueryError{Message: err.Error()},
			})
			return
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: errors.Wrap(err, "iterating session summaries").Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, summaries)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	session, err := loadSessionByID(s.sessionIndex, sessionID)
	if err != nil {
		if isSessionNotFound(err) {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, normalizeSessionDetail(session))
}

func (s *Server) handleGetSessionSummary(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	session, err := loadSessionByID(s.sessionIndex, sessionID)
	if err != nil {
		if isSessionNotFound(err) {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, normalizeSessionSummaryDetail(session))
}

func (s *Server) handleGetSessionBlocks(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	session, err := loadSessionByID(s.sessionIndex, sessionID)
	if err != nil {
		if isSessionNotFound(err) {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, buildSessionBlocks(session))
}

func buildSessionListSQL(tableName string) (string, error) {
	if err := queryengine.ValidateIdentifier(tableName); err != nil {
		return "", err
	}
	return fmt.Sprintf(`
SELECT
  id,
  title,
  summary,
  classification,
  timing,
  metrics,
  environment,
  operational_context
FROM %s
ORDER BY timing->>'started_at';
`, tableName), nil
}

func sessionSummaryFromValues(values []any) (SessionSummaryResponse, error) {
	if len(values) != 8 {
		return SessionSummaryResponse{}, errors.Errorf("expected 8 session summary values, got %d", len(values))
	}

	timing := minitrace.Timing{}
	if err := decodeJSONColumn(values[4], &timing); err != nil {
		return SessionSummaryResponse{}, errors.Wrap(err, "decoding timing column")
	}
	metrics := minitrace.Metrics{}
	if err := decodeJSONColumn(values[5], &metrics); err != nil {
		return SessionSummaryResponse{}, errors.Wrap(err, "decoding metrics column")
	}
	environment := minitrace.Environment{}
	if err := decodeJSONColumn(values[6], &environment); err != nil {
		return SessionSummaryResponse{}, errors.Wrap(err, "decoding environment column")
	}
	operationalContext := minitrace.OperationalContext{}
	if err := decodeJSONColumn(values[7], &operationalContext); err != nil {
		return SessionSummaryResponse{}, errors.Wrap(err, "decoding operational_context column")
	}

	return SessionSummaryResponse{
		ID:                 scalarString(values[0]),
		Title:              scalarString(values[1]),
		Summary:            nullableString(values[2]),
		Classification:     scalarString(values[3]),
		Timing:             normalizeTiming(timing),
		Metrics:            normalizeMetrics(metrics),
		Environment:        normalizeEnvironment(environment),
		OperationalContext: normalizeOperationalContext(operationalContext),
	}, nil
}

func normalizeSessionSummaryDetail(session minitrace.Session) SessionSummaryDetailResponse {
	return SessionSummaryDetailResponse{
		SessionSummaryResponse: SessionSummaryResponse{
			ID:                 session.ID,
			Title:              stringValue(session.Title),
			Summary:            session.Summary,
			Classification:     session.Classification,
			Timing:             normalizeTiming(session.Timing),
			Metrics:            normalizeMetrics(session.Metrics),
			Environment:        normalizeEnvironment(session.Environment),
			OperationalContext: normalizeOperationalContext(session.OperationalContext),
		},
		Provenance: normalizeProvenance(session.Provenance),
	}
}

func normalizeSessionDetail(session minitrace.Session) SessionDetailResponse {
	return SessionDetailResponse{
		SessionSummaryDetailResponse: normalizeSessionSummaryDetail(session),
		Blocks:                       buildSessionBlocks(session),
	}
}

func normalizeTurn(turn minitrace.Turn, tcByID map[string]minitrace.ToolCall) TurnResponse {
	toolCalls := make([]ToolCallResponse, 0, len(turn.ToolCallsInTurn))
	for _, toolCallID := range turn.ToolCallsInTurn {
		toolCall, ok := tcByID[toolCallID]
		if !ok {
			continue
		}
		toolCalls = append(toolCalls, normalizeToolCall(toolCall))
	}

	return TurnResponse{
		Idx:             turn.Index,
		Role:            turn.Role,
		Source:          stringValue(turn.Source),
		Content:         turn.Content,
		Timestamp:       stringValue(turn.Timestamp),
		ToolCallsInTurn: toolCalls,
	}
}

func normalizeToolCall(toolCall minitrace.ToolCall) ToolCallResponse {
	return ToolCallResponse{
		ID:            toolCall.ID,
		ToolName:      toolCall.ToolName,
		Timestamp:     stringValue(toolCall.Timestamp),
		OperationType: toolCall.OperationType,
		Input: ToolCallInput{
			Command:   stringValue(toolCall.Input.Command),
			Arguments: normalizeArguments(toolCall.Input.Arguments),
			FilePath:  stringValue(toolCall.Input.FilePath),
		},
		Output: ToolCallOutput{
			Success:    toolCall.Output.Success,
			Result:     toolCall.Output.Result,
			Error:      toolCall.Output.Error,
			DurationMs: intValue(toolCall.Output.DurationMS),
			Truncated:  toolCall.Output.Truncated,
		},
		Badges: DetectBadges(toolCall),
	}
}

func normalizeTiming(timing minitrace.Timing) SessionTimingResponse {
	return SessionTimingResponse{
		StartedAt:             stringValue(timing.StartedAt),
		EndedAt:               timing.EndedAt,
		DurationSeconds:       floatValue(timing.DurationSeconds),
		ActiveDurationSeconds: floatValue(timing.ActiveDurationSeconds),
		HourOfDay:             intValue(timing.HourOfDay),
		DayOfWeek:             intValue(timing.DayOfWeek),
	}
}

func normalizeMetrics(metrics minitrace.Metrics) SessionMetricsResponse {
	return SessionMetricsResponse{
		TurnCount:            metrics.TurnCount,
		ToolCallCount:        metrics.ToolCallCount,
		TotalInputTokens:     metrics.TotalInputTokens,
		TotalOutputTokens:    metrics.TotalOutputTokens,
		TotalCacheReadTokens: metrics.TotalCacheReadTokens,
	}
}

func normalizeEnvironment(environment minitrace.Environment) SessionEnvironmentResponse {
	return SessionEnvironmentResponse{
		AgentFramework: stringValue(environment.AgentFramework),
		Model:          stringValue(environment.Model),
	}
}

func normalizeOperationalContext(operationalContext minitrace.OperationalContext) SessionOperationalContextResponse {
	return SessionOperationalContextResponse{
		WorkingDirectory: stringValue(operationalContext.WorkingDirectory),
		AutonomyLevel:    stringValue(operationalContext.AutonomyLevel),
		Sandbox:          operationalContext.Sandbox,
	}
}

func normalizeProvenance(provenance minitrace.Provenance) SessionProvenanceResponse {
	return SessionProvenanceResponse{
		SourceFormat:      provenance.SourceFormat,
		SourcePath:        stringValue(provenance.SourcePath),
		OriginalSessionID: stringValue(provenance.OriginalSessionID),
		ConvertedAt:       provenance.ConvertedAt,
	}
}

func decodeJSONColumn(raw any, dest any) error {
	switch typed := raw.(type) {
	case nil:
		return nil
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil
		}
		return json.Unmarshal([]byte(text), dest)
	case []byte:
		text := strings.TrimSpace(string(typed))
		if text == "" {
			return nil
		}
		return json.Unmarshal([]byte(text), dest)
	default:
		payload, err := json.Marshal(typed)
		if err != nil {
			return err
		}
		return json.Unmarshal(payload, dest)
	}
}

func scalarString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []byte:
		return string(typed)
	case sql.NullString:
		if typed.Valid {
			return typed.String
		}
		return ""
	default:
		return fmt.Sprint(value)
	}
}

func nullableString(value any) *string {
	switch typed := value.(type) {
	case nil:
		return nil
	case sql.NullString:
		if !typed.Valid {
			return nil
		}
		v := typed.String
		return &v
	case string:
		v := typed
		return &v
	case []byte:
		v := string(typed)
		return &v
	default:
		v := fmt.Sprint(value)
		return &v
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func floatValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func normalizeArguments(arguments any) map[string]any {
	if arguments == nil {
		return nil
	}
	if argMap, ok := arguments.(map[string]any); ok {
		return argMap
	}
	return map[string]any{"value": arguments}
}

var errSessionNotFound = errors.New("session not found")

func loadSessionByID(index map[string]string, sessionID string) (minitrace.Session, error) {
	sessionPath, ok := index[sessionID]
	if !ok {
		return minitrace.Session{}, errSessionNotFound
	}
	payload, err := os.ReadFile(sessionPath)
	if err != nil {
		return minitrace.Session{}, errors.Wrap(err, "reading session file")
	}
	session := minitrace.Session{}
	if err := json.Unmarshal(payload, &session); err != nil {
		return minitrace.Session{}, errors.Wrap(err, "unmarshaling session file")
	}
	return session, nil
}

func isSessionNotFound(err error) bool {
	return stderrors.Is(err, errSessionNotFound)
}
