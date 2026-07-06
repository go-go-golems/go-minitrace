package serve

import (
	"fmt"
	"net/http"

	apiv1 "github.com/go-go-golems/go-minitrace/gen/proto/go_go_golems/minitrace/api/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

const apiSchemaVersion = 1

// sessionListSQL is a plain projection over the normalized sessions table
// (columns are already flattened), joined with the metrics table for the
// token totals the session browser displays.
const sessionListSQL = `
SELECT
  s.session_id,
  s.title,
  s.summary,
  s.classification,
  s.started_at,
  s.ended_at,
  s.duration_seconds,
  s.active_duration_seconds,
  s.hour_of_day,
  s.day_of_week,
  s.turn_count,
  s.tool_call_count,
  m.total_input_tokens,
  m.total_output_tokens,
  m.total_cache_read_tokens,
  s.agent_framework,
  s.model,
  s.working_directory,
  s.autonomy_level,
  s.sandbox
FROM sessions AS s
LEFT JOIN metrics AS m USING (session_id)
ORDER BY s.started_at
`

func (s *Server) handleGetSessionsV2(w http.ResponseWriter, r *http.Request) {
	if s.queryTarget == nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: "query target is not initialized"},
		})
		return
	}

	result, err := s.queryTarget.QueryResult(r.Context(), sessionListSQL)
	if err == nil && result.Error != "" {
		err = fmt.Errorf("%s", result.Error)
	}
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

	payload := &apiv1.ListSessionsResponse{
		Meta: &apiv1.ApiMeta{SchemaVersion: apiSchemaVersion},
	}
	for _, row := range result.Rows {
		payload.Sessions = append(payload.Sessions, protoSessionSummary(sessionSummaryFromRow(row)))
	}

	writeProtoJSON(w, http.StatusOK, payload)
}

// sessionSummaryFromRow maps one normalized sessions-list row (as returned by
// the sandboxed query runner) onto the session summary response shape used by
// the v2 API.
func sessionSummaryFromRow(row map[string]any) SessionSummaryResponse {
	return SessionSummaryResponse{
		ID:             cellString(row["session_id"]),
		Title:          cellString(row["title"]),
		Summary:        cellNullableString(row["summary"]),
		Classification: cellString(row["classification"]),
		Timing: SessionTimingResponse{
			StartedAt:             cellString(row["started_at"]),
			EndedAt:               cellNullableString(row["ended_at"]),
			DurationSeconds:       cellFloat(row["duration_seconds"]),
			ActiveDurationSeconds: cellFloat(row["active_duration_seconds"]),
			HourOfDay:             cellInt(row["hour_of_day"]),
			DayOfWeek:             cellInt(row["day_of_week"]),
		},
		Metrics: SessionMetricsResponse{
			TurnCount:            cellInt(row["turn_count"]),
			ToolCallCount:        cellInt(row["tool_call_count"]),
			TotalInputTokens:     cellOptionalInt(row["total_input_tokens"]),
			TotalOutputTokens:    cellOptionalInt(row["total_output_tokens"]),
			TotalCacheReadTokens: cellOptionalInt(row["total_cache_read_tokens"]),
		},
		Environment: SessionEnvironmentResponse{
			AgentFramework: cellString(row["agent_framework"]),
			Model:          cellString(row["model"]),
		},
		OperationalContext: SessionOperationalContextResponse{
			WorkingDirectory: cellString(row["working_directory"]),
			AutonomyLevel:    cellString(row["autonomy_level"]),
			Sandbox:          cellOptionalBool(row["sandbox"]),
		},
	}
}

func cellString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return fmt.Sprint(value)
	}
}

func cellNullableString(value any) *string {
	if value == nil {
		return nil
	}
	v := cellString(value)
	return &v
}

func cellInt(value any) int {
	switch typed := value.(type) {
	case nil:
		return 0
	case int64:
		return int(typed)
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func cellOptionalInt(value any) *int {
	if value == nil {
		return nil
	}
	v := cellInt(value)
	return &v
}

func cellFloat(value any) float64 {
	switch typed := value.(type) {
	case nil:
		return 0
	case float64:
		return typed
	case int64:
		return float64(typed)
	case int:
		return float64(typed)
	default:
		return 0
	}
}

func cellOptionalBool(value any) *bool {
	switch typed := value.(type) {
	case nil:
		return nil
	case bool:
		v := typed
		return &v
	case int64:
		v := typed != 0
		return &v
	case int:
		v := typed != 0
		return &v
	case float64:
		v := typed != 0
		return &v
	default:
		return nil
	}
}

func (s *Server) handleGetSessionSummaryV2(w http.ResponseWriter, r *http.Request) {
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

	payload := &apiv1.GetSessionSummaryResponse{
		Meta:    &apiv1.ApiMeta{SchemaVersion: apiSchemaVersion},
		Session: protoSessionSummaryDetail(normalizeSessionSummaryDetail(session)),
	}
	writeProtoJSON(w, http.StatusOK, payload)
}

func (s *Server) handleGetSessionBlocksV2(w http.ResponseWriter, r *http.Request) {
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

	blocks, err := protoSessionBlocks(buildSessionBlocks(session))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to normalize protobuf session blocks")
		return
	}

	payload := &apiv1.GetSessionBlocksResponse{
		Meta:   &apiv1.ApiMeta{SchemaVersion: apiSchemaVersion},
		Blocks: blocks,
	}
	writeProtoJSON(w, http.StatusOK, payload)
}

func (s *Server) handleGetSessionV2(w http.ResponseWriter, r *http.Request) {
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

	detail, err := protoSessionDetail(normalizeSessionDetail(session))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to normalize protobuf session detail")
		return
	}

	payload := &apiv1.GetSessionDetailResponse{
		Meta:    &apiv1.ApiMeta{SchemaVersion: apiSchemaVersion},
		Session: detail,
	}
	writeProtoJSON(w, http.StatusOK, payload)
}

func protoSessionSummary(summary SessionSummaryResponse) *apiv1.SessionSummary {
	return &apiv1.SessionSummary{
		Id:                 summary.ID,
		Title:              summary.Title,
		Summary:            summary.Summary,
		Classification:     summary.Classification,
		Timing:             protoSessionTiming(summary.Timing),
		Metrics:            protoSessionMetrics(summary.Metrics),
		Environment:        protoSessionEnvironment(summary.Environment),
		OperationalContext: protoOperationalContext(summary.OperationalContext),
	}
}

func protoSessionSummaryDetail(detail SessionSummaryDetailResponse) *apiv1.SessionSummaryDetail {
	return &apiv1.SessionSummaryDetail{
		Id:                 detail.ID,
		Title:              detail.Title,
		Summary:            detail.Summary,
		Classification:     detail.Classification,
		Timing:             protoSessionTiming(detail.Timing),
		Metrics:            protoSessionMetrics(detail.Metrics),
		Environment:        protoSessionEnvironment(detail.Environment),
		OperationalContext: protoOperationalContext(detail.OperationalContext),
		Provenance:         protoSessionProvenance(detail.Provenance),
		Events:             protoSessionEvents(detail.Events),
		Attachments:        protoSessionAttachments(detail.Attachments),
	}
}

func protoSessionDetail(detail SessionDetailResponse) (*apiv1.SessionDetail, error) {
	blocks, err := protoSessionBlocks(detail.Blocks)
	if err != nil {
		return nil, err
	}

	return &apiv1.SessionDetail{
		Id:                 detail.ID,
		Title:              detail.Title,
		Summary:            detail.Summary,
		Classification:     detail.Classification,
		Timing:             protoSessionTiming(detail.Timing),
		Metrics:            protoSessionMetrics(detail.Metrics),
		Environment:        protoSessionEnvironment(detail.Environment),
		OperationalContext: protoOperationalContext(detail.OperationalContext),
		Provenance:         protoSessionProvenance(detail.Provenance),
		Blocks:             blocks,
		Events:             protoSessionEvents(detail.Events),
		Attachments:        protoSessionAttachments(detail.Attachments),
	}, nil
}

func protoSessionEvents(events []SessionEventResponse) []*apiv1.SessionEvent {
	ret := make([]*apiv1.SessionEvent, 0, len(events))
	for _, event := range events {
		ret = append(ret, protoSessionEvent(event))
	}
	return ret
}

func protoSessionEvent(event SessionEventResponse) *apiv1.SessionEvent {
	return &apiv1.SessionEvent{
		Id:                 event.ID,
		Timestamp:          event.Timestamp,
		TurnIndex:          clampOptionalIntToUint32(event.TurnIndex),
		Ordinal:            clampOptionalIntToUint32(event.Ordinal),
		Kind:               event.Kind,
		Role:               event.Role,
		ToolCallId:         event.ToolCallID,
		AnnotationId:       event.AnnotationID,
		AttachmentId:       event.AttachmentID,
		Title:              event.Title,
		Summary:            event.Summary,
		Text:               event.Text,
		Severity:           event.Severity,
		CollapsedByDefault: event.CollapsedByDefault,
		FrameworkMetadata:  protoStruct(event.FrameworkMetadata),
	}
}

func protoSessionAttachments(attachments []SessionAttachmentResponse) []*apiv1.SessionAttachment {
	ret := make([]*apiv1.SessionAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		ret = append(ret, protoSessionAttachment(attachment))
	}
	return ret
}

func protoSessionAttachment(attachment SessionAttachmentResponse) *apiv1.SessionAttachment {
	return &apiv1.SessionAttachment{
		Id:                attachment.ID,
		Timestamp:         attachment.Timestamp,
		Kind:              attachment.Kind,
		Name:              attachment.Name,
		MediaType:         attachment.MediaType,
		Path:              attachment.Path,
		Url:               attachment.URL,
		SizeBytes:         clampOptionalIntToUint32(attachment.SizeBytes),
		Hash:              attachment.Hash,
		ContentRef:        attachment.ContentRef,
		TextPreview:       attachment.TextPreview,
		TurnIndex:         clampOptionalIntToUint32(attachment.TurnIndex),
		ToolCallId:        attachment.ToolCallID,
		EventId:           attachment.EventID,
		FrameworkMetadata: protoStruct(attachment.FrameworkMetadata),
	}
}

func protoStruct(value map[string]any) *structpb.Struct {
	if len(value) == 0 {
		return nil
	}
	ret, err := structpb.NewStruct(value)
	if err != nil {
		return nil
	}
	return ret
}

func protoSessionBlocks(blocks []SessionBlock) ([]*apiv1.SessionBlock, error) {
	ret := make([]*apiv1.SessionBlock, 0, len(blocks))
	for _, block := range blocks {
		protoBlock, err := protoSessionBlock(block)
		if err != nil {
			return nil, err
		}
		ret = append(ret, protoBlock)
	}
	return ret, nil
}

func protoSessionBlock(block SessionBlock) (*apiv1.SessionBlock, error) {
	turns, err := protoTurns(block.Turns)
	if err != nil {
		return nil, err
	}
	return &apiv1.SessionBlock{
		BlockNum:    clampIntToUint32(block.BlockNum),
		UserTurnIdx: clampIntToInt32(block.UserTurnIdx),
		UserTs:      block.UserTs,
		UserContent: block.UserContent,
		AgentTurns:  clampIntToUint32(block.AgentTurns),
		ToolCalls:   clampIntToUint32(block.ToolCalls),
		GapMinutes:  protoOptionalFloat64(block.GapMinutes),
		Turns:       turns,
		Artifacts:   protoBlockArtifacts(block.Artifacts),
	}, nil
}

func protoTurns(turns []TurnResponse) ([]*apiv1.Turn, error) {
	ret := make([]*apiv1.Turn, 0, len(turns))
	for _, turn := range turns {
		protoTurn, err := protoTurn(turn)
		if err != nil {
			return nil, err
		}
		ret = append(ret, protoTurn)
	}
	return ret, nil
}

func protoTurnUsage(u *TurnUsageResponse) *apiv1.TurnUsage {
	if u == nil {
		return nil
	}
	ret := &apiv1.TurnUsage{}
	ret.InputTokens = clampOptionalIntToUint32(u.InputTokens)
	ret.OutputTokens = clampOptionalIntToUint32(u.OutputTokens)
	ret.CacheReadTokens = clampOptionalIntToUint32(u.CacheReadTokens)
	ret.ReasoningTokens = clampOptionalIntToUint32(u.ReasoningTokens)
	return ret
}

func protoTurn(turn TurnResponse) (*apiv1.Turn, error) {
	toolCalls, err := protoToolCalls(turn.ToolCallsInTurn)
	if err != nil {
		return nil, err
	}
	return &apiv1.Turn{
		Idx:             clampIntToUint32(turn.Idx),
		Role:            turn.Role,
		Source:          turn.Source,
		Content:         turn.Content,
		Timestamp:       turn.Timestamp,
		ToolCallsInTurn: toolCalls,
		Thinking:        turn.Thinking,
		Model:           turn.Model,
		Usage:           protoTurnUsage(turn.Usage),
	}, nil
}

func protoToolCalls(toolCalls []ToolCallResponse) ([]*apiv1.ToolCall, error) {
	ret := make([]*apiv1.ToolCall, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		protoToolCall, err := protoToolCall(toolCall)
		if err != nil {
			return nil, err
		}
		ret = append(ret, protoToolCall)
	}
	return ret, nil
}

func protoToolCall(toolCall ToolCallResponse) (*apiv1.ToolCall, error) {
	input, err := protoToolCallInput(toolCall.Input)
	if err != nil {
		return nil, err
	}
	return &apiv1.ToolCall{
		Id:            toolCall.ID,
		ToolName:      toolCall.ToolName,
		Timestamp:     toolCall.Timestamp,
		OperationType: toolCall.OperationType,
		Input:         input,
		Output:        protoToolCallOutput(toolCall.Output),
		Badges:        protoToolCallBadges(toolCall.Badges),
	}, nil
}

func protoToolCallInput(input ToolCallInput) (*apiv1.ToolCallInput, error) {
	ret := &apiv1.ToolCallInput{
		Command:  protoOptionalString(input.Command),
		FilePath: protoOptionalString(input.FilePath),
	}
	if len(input.Arguments) > 0 {
		arguments, err := structpb.NewStruct(input.Arguments)
		if err != nil {
			return nil, err
		}
		ret.Arguments = arguments
	}
	return ret, nil
}

func protoToolCallOutput(output ToolCallOutput) *apiv1.ToolCallOutput {
	return &apiv1.ToolCallOutput{
		Success:    output.Success,
		Result:     output.Result,
		Error:      output.Error,
		DurationMs: clampIntToUint32(output.DurationMs),
		Truncated:  output.Truncated,
	}
}

var protoToolCallBadgeMap = map[BadgeType]apiv1.ToolCallBadge{
	BadgeCommit:       apiv1.ToolCallBadge_TOOL_CALL_BADGE_COMMIT,
	BadgeTicketCreate: apiv1.ToolCallBadge_TOOL_CALL_BADGE_TICKET_CREATE,
	BadgeDocAdd:       apiv1.ToolCallBadge_TOOL_CALL_BADGE_DOC_ADD,
	BadgeDiaryWrite:   apiv1.ToolCallBadge_TOOL_CALL_BADGE_DIARY_WRITE,
	BadgeError:        apiv1.ToolCallBadge_TOOL_CALL_BADGE_ERROR,
}

func protoToolCallBadges(badges []BadgeType) []apiv1.ToolCallBadge {
	ret := make([]apiv1.ToolCallBadge, 0, len(badges))
	for _, badge := range badges {
		if protoBadge, ok := protoToolCallBadgeMap[badge]; ok {
			ret = append(ret, protoBadge)
			continue
		}
		ret = append(ret, apiv1.ToolCallBadge_TOOL_CALL_BADGE_UNSPECIFIED)
	}
	return ret
}

func protoSessionTiming(timing SessionTimingResponse) *apiv1.SessionTiming {
	return &apiv1.SessionTiming{
		StartedAt:             timing.StartedAt,
		EndedAt:               timing.EndedAt,
		DurationSeconds:       timing.DurationSeconds,
		ActiveDurationSeconds: timing.ActiveDurationSeconds,
		HourOfDay:             clampIntToUint32(timing.HourOfDay),
		DayOfWeek:             clampIntToUint32(timing.DayOfWeek),
	}
}

func protoSessionMetrics(metrics SessionMetricsResponse) *apiv1.SessionMetrics {
	return &apiv1.SessionMetrics{
		TurnCount:            clampIntToUint32(metrics.TurnCount),
		ToolCallCount:        clampIntToUint32(metrics.ToolCallCount),
		TotalInputTokens:     protoOptionalUint32(metrics.TotalInputTokens),
		TotalOutputTokens:    protoOptionalUint32(metrics.TotalOutputTokens),
		TotalCacheReadTokens: protoOptionalUint32(metrics.TotalCacheReadTokens),
	}
}

func protoSessionEnvironment(environment SessionEnvironmentResponse) *apiv1.SessionEnvironment {
	return &apiv1.SessionEnvironment{
		AgentFramework: environment.AgentFramework,
		Model:          environment.Model,
	}
}

func protoOperationalContext(operationalContext SessionOperationalContextResponse) *apiv1.SessionOperationalContext {
	return &apiv1.SessionOperationalContext{
		WorkingDirectory: operationalContext.WorkingDirectory,
		AutonomyLevel:    protoOptionalString(operationalContext.AutonomyLevel),
		Sandbox:          operationalContext.Sandbox,
	}
}

func protoSessionProvenance(provenance SessionProvenanceResponse) *apiv1.SessionProvenance {
	return &apiv1.SessionProvenance{
		SourceFormat:      provenance.SourceFormat,
		SourcePath:        provenance.SourcePath,
		OriginalSessionId: provenance.OriginalSessionID,
		ConvertedAt:       provenance.ConvertedAt,
	}
}

func protoBlockArtifacts(artifacts BlockArtifacts) *apiv1.BlockArtifacts {
	return &apiv1.BlockArtifacts{
		Commits:        append([]string(nil), artifacts.Commits...),
		TicketsCreated: append([]string(nil), artifacts.TicketsCreated...),
		DocsAdded:      append([]string(nil), artifacts.DocsAdded...),
		DiaryWrites:    clampIntToUint32(artifacts.DiaryWrites),
	}
}

func protoOptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func protoOptionalUint32(value *int) *uint32 {
	return clampOptionalIntToUint32(value)
}

func protoOptionalFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}
