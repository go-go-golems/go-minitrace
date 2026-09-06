package serve

import (
	"encoding/json"
	stderrors "errors"
	"os"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
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
	Provenance  SessionProvenanceResponse   `json:"provenance"`
	Events      []SessionEventResponse      `json:"events"`
	Attachments []SessionAttachmentResponse `json:"attachments"`
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

type TurnUsageResponse struct {
	InputTokens     *int `json:"input_tokens,omitempty"`
	OutputTokens    *int `json:"output_tokens,omitempty"`
	CacheReadTokens *int `json:"cache_read_tokens,omitempty"`
	ReasoningTokens *int `json:"reasoning_tokens,omitempty"`
}

type SessionEventResponse struct {
	ID                 string         `json:"id"`
	Timestamp          string         `json:"timestamp"`
	TurnIndex          *int           `json:"turn_index,omitempty"`
	Ordinal            *int           `json:"ordinal,omitempty"`
	Kind               string         `json:"kind"`
	Role               string         `json:"role"`
	ToolCallID         *string        `json:"tool_call_id,omitempty"`
	AnnotationID       *string        `json:"annotation_id,omitempty"`
	AttachmentID       *string        `json:"attachment_id,omitempty"`
	Title              string         `json:"title"`
	Summary            string         `json:"summary"`
	Text               string         `json:"text"`
	Severity           string         `json:"severity"`
	CollapsedByDefault bool           `json:"collapsed_by_default"`
	FrameworkMetadata  map[string]any `json:"framework_metadata,omitempty"`
}

type SessionAttachmentResponse struct {
	ID                string         `json:"id"`
	Timestamp         string         `json:"timestamp"`
	Kind              string         `json:"kind"`
	Name              string         `json:"name"`
	MediaType         string         `json:"media_type"`
	Path              string         `json:"path"`
	URL               string         `json:"url"`
	SizeBytes         *int           `json:"size_bytes,omitempty"`
	Hash              string         `json:"hash"`
	ContentRef        string         `json:"content_ref"`
	TextPreview       string         `json:"text_preview"`
	TurnIndex         *int           `json:"turn_index,omitempty"`
	ToolCallID        *string        `json:"tool_call_id,omitempty"`
	EventID           *string        `json:"event_id,omitempty"`
	FrameworkMetadata map[string]any `json:"framework_metadata,omitempty"`
}

type TurnResponse struct {
	Idx             int                `json:"idx"`
	Role            string             `json:"role"`
	Source          string             `json:"source"`
	Content         string             `json:"content"`
	Timestamp       string             `json:"timestamp"`
	Thinking        *string            `json:"thinking,omitempty"`
	Model           *string            `json:"model,omitempty"`
	Usage           *TurnUsageResponse `json:"usage,omitempty"`
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
	Success    *bool   `json:"success"`
	Status     string  `json:"status"`
	ExitCode   *int    `json:"exit_code"`
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
		Provenance:  normalizeProvenance(session.Provenance),
		Events:      normalizeEvents(session.Events),
		Attachments: normalizeAttachments(session.Attachments),
	}
}

func normalizeSessionDetail(session minitrace.Session) SessionDetailResponse {
	return SessionDetailResponse{
		SessionSummaryDetailResponse: normalizeSessionSummaryDetail(session),
		Blocks:                       buildSessionBlocks(session),
	}
}

func normalizeEvents(events []minitrace.Event) []SessionEventResponse {
	ret := make([]SessionEventResponse, 0, len(events))
	for _, event := range events {
		ret = append(ret, SessionEventResponse{
			ID:                 event.ID,
			Timestamp:          stringValue(event.Timestamp),
			TurnIndex:          event.TurnIndex,
			Ordinal:            event.Ordinal,
			Kind:               event.Kind,
			Role:               event.Role,
			ToolCallID:         event.ToolCallID,
			AnnotationID:       event.AnnotationID,
			AttachmentID:       event.AttachmentID,
			Title:              event.Title,
			Summary:            event.Summary,
			Text:               event.Text,
			Severity:           event.Severity,
			CollapsedByDefault: event.CollapsedByDefault,
			FrameworkMetadata:  normalizeArguments(event.FrameworkMetadata),
		})
	}
	return ret
}

func normalizeAttachments(attachments []minitrace.Attachment) []SessionAttachmentResponse {
	ret := make([]SessionAttachmentResponse, 0, len(attachments))
	for _, attachment := range attachments {
		ret = append(ret, SessionAttachmentResponse{
			ID:                attachment.ID,
			Timestamp:         stringValue(attachment.Timestamp),
			Kind:              attachment.Kind,
			Name:              attachment.Name,
			MediaType:         attachment.MediaType,
			Path:              attachment.Path,
			URL:               attachment.URL,
			SizeBytes:         attachment.SizeBytes,
			Hash:              attachment.Hash,
			ContentRef:        attachment.ContentRef,
			TextPreview:       attachment.TextPreview,
			TurnIndex:         attachment.TurnIndex,
			ToolCallID:        attachment.ToolCallID,
			EventID:           attachment.EventID,
			FrameworkMetadata: normalizeArguments(attachment.FrameworkMetadata),
		})
	}
	return ret
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
		Thinking:        turn.Thinking,
		Model:           turn.Model,
		Usage:           normalizeUsage(turn.Usage),
		ToolCallsInTurn: toolCalls,
	}
}

func normalizeUsage(u *minitrace.Usage) *TurnUsageResponse {
	if u == nil {
		return nil
	}
	return &TurnUsageResponse{
		InputTokens:     u.InputTokens,
		OutputTokens:    u.OutputTokens,
		CacheReadTokens: u.CacheReadTokens,
		ReasoningTokens: u.ReasoningTokens,
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
			Status:     string(toolCall.Output.OutcomeStatus()),
			ExitCode:   toolCall.Output.ExitCode,
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
