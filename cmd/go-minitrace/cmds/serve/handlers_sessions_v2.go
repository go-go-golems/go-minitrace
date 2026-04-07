package serve

import (
	"net/http"

	apiv1 "github.com/go-go-golems/go-minitrace/gen/proto/go_go_golems/minitrace/api/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

const apiSchemaVersion = 1

func (s *Server) handleGetSessionsV2(w http.ResponseWriter, r *http.Request) {
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
			Error:      &QueryError{Message: err.Error()},
		})
		return
	}
	defer func() { _ = rows.Close() }()

	payload := &apiv1.ListSessionsResponse{
		Meta: &apiv1.ApiMeta{SchemaVersion: apiSchemaVersion},
	}

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
				Error:      &QueryError{Message: err.Error()},
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

		payload.Sessions = append(payload.Sessions, protoSessionSummary(summary))
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: err.Error()},
		})
		return
	}

	writeProtoJSON(w, http.StatusOK, payload)
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
	}, nil
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
		BlockNum:    uint32(block.BlockNum),
		UserTurnIdx: int32(block.UserTurnIdx),
		UserTs:      block.UserTs,
		UserContent: block.UserContent,
		AgentTurns:  uint32(block.AgentTurns),
		ToolCalls:   uint32(block.ToolCalls),
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

func protoTurn(turn TurnResponse) (*apiv1.Turn, error) {
	toolCalls, err := protoToolCalls(turn.ToolCallsInTurn)
	if err != nil {
		return nil, err
	}
	return &apiv1.Turn{
		Idx:             uint32(turn.Idx),
		Role:            turn.Role,
		Source:          turn.Source,
		Content:         turn.Content,
		Timestamp:       turn.Timestamp,
		ToolCallsInTurn: toolCalls,
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
		DurationMs: uint32(output.DurationMs),
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
		HourOfDay:             uint32(timing.HourOfDay),
		DayOfWeek:             uint32(timing.DayOfWeek),
	}
}

func protoSessionMetrics(metrics SessionMetricsResponse) *apiv1.SessionMetrics {
	return &apiv1.SessionMetrics{
		TurnCount:            uint32(metrics.TurnCount),
		ToolCallCount:        uint32(metrics.ToolCallCount),
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
		DiaryWrites:    uint32(artifacts.DiaryWrites),
	}
}

func protoOptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func protoOptionalUint32(value *int) *uint32 {
	if value == nil {
		return nil
	}
	v := uint32(*value)
	return &v
}

func protoOptionalFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}
