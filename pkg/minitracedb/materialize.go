package minitracedb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

type MaterializeOptions struct {
	SourcePath string
}

func LoadSessionFile(path string) (*minitrace.Session, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read minitrace session %s: %w", path, err)
	}
	var session minitrace.Session
	if err := json.Unmarshal(payload, &session); err != nil {
		return nil, fmt.Errorf("decode minitrace session %s: %w", path, err)
	}
	if session.ID == "" {
		return nil, fmt.Errorf("decode minitrace session %s: missing id", path)
	}
	return &session, nil
}

func MaterializeSession(ctx context.Context, db *sql.DB, session *minitrace.Session, opts MaterializeOptions) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin materialize transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if err := insertSession(ctx, tx, session, opts); err != nil {
		return err
	}
	for _, turn := range session.Turns {
		if err := insertTurn(ctx, tx, session.ID, turn); err != nil {
			return err
		}
		if err := insertTurnEvent(ctx, tx, session.ID, turn); err != nil {
			return err
		}
		for ordinal, toolCallID := range turn.ToolCallsInTurn {
			if _, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO turn_tool_calls(session_id, turn_index, tool_call_id, ordinal) VALUES (?, ?, ?, ?)`, session.ID, turn.Index, toolCallID, ordinal); err != nil {
				return fmt.Errorf("insert turn_tool_calls %s/%d/%s: %w", session.ID, turn.Index, toolCallID, err)
			}
		}
	}
	for _, toolCall := range session.ToolCalls {
		if err := insertToolCall(ctx, tx, session.ID, toolCall); err != nil {
			return err
		}
		if err := insertToolCallEvent(ctx, tx, session.ID, toolCall); err != nil {
			return err
		}
		for ordinal, target := range toolCall.EffectiveFileTargets() {
			columns := []string{"session_id", "tool_call_id", "path", "operation_type", "tool_name", "success", "turn_index", "target_ordinal", "evidence_kind", "evidence_status", "cwd", "resolved", "native_path", "source_reference"}
			if err := insertRow(ctx, tx, "files", columns, session.ID, toolCall.ID, target.Path, target.OperationType, toolCall.ToolName, nullableBoolPointer(target.Success), nullableIntPointer(toolCall.EmittingTurnIndex), ordinal, target.EvidenceKind, target.Status, nullableStringValue(target.CWD), boolInt(target.Resolved), nullableStringValue(target.NativePath), nullableStringValue(target.SourceReference)); err != nil {
				return fmt.Errorf("insert file %s/%s: %w", session.ID, target.Path, err)
			}
		}
	}
	for i, annotation := range session.Annotations {
		if err := insertAnnotation(ctx, tx, session.ID, annotation, i); err != nil {
			return err
		}
		if err := insertAnnotationEvent(ctx, tx, session.ID, annotation, i); err != nil {
			return err
		}
	}
	for i, event := range session.Events {
		if err := insertExplicitEvent(ctx, tx, session.ID, event, i); err != nil {
			return err
		}
	}
	for i, attachment := range session.Attachments {
		if err := insertAttachment(ctx, tx, session.ID, attachment, i); err != nil {
			return err
		}
	}
	if err := insertHandover(ctx, tx, session.ID, "received", session.Handover.Received); err != nil {
		return err
	}
	if err := insertHandover(ctx, tx, session.ID, "produced", session.Handover.Produced); err != nil {
		return err
	}
	if err := insertMetrics(ctx, tx, session); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit materialize transaction: %w", err)
	}
	committed = true
	return nil
}

func insertSession(ctx context.Context, tx *sql.Tx, session *minitrace.Session, opts MaterializeOptions) error {
	sourcePath := opts.SourcePath
	if sourcePath == "" && session.Provenance.SourcePath != nil {
		sourcePath = *session.Provenance.SourcePath
	}
	columns := []string{
		"session_id", "schema_version", "profile", "scenario_id", "quality", "title", "summary", "classification",
		"source_format", "source_path", "converted_at", "converter_version", "original_session_id",
		"for_research", "needs_cleaning", "contains_error", "contains_pii", "category_json",
		"model", "model_version", "temperature", "tools_enabled_json", "system_prompt", "agent_framework", "agent_version", "platform_type", "provider_hint",
		"working_directory", "git_branch", "git_ref", "autonomy_level", "sandbox", "framework_config_json",
		"privacy_level", "duration_seconds", "active_duration_seconds", "started_at", "ended_at", "hour_of_day", "day_of_week",
		"guidance_variant", "permission_level", "condition_custom_json",
		"project_id", "predecessor_session", "concurrent_sessions", "human_attention",
		"outcome_success", "outcome_partial", "failure_codes_json", "outcome_notes",
		"turn_count", "tool_call_count", "read_count", "modify_count", "create_count", "execute_count", "raw_json",
	}
	args := []any{
		session.ID, session.SchemaVersion, session.Profile, nullableString(session.ScenarioID), nullableString(session.Quality), nullableString(session.Title), nullableString(session.Summary), session.Classification,
		session.Provenance.SourceFormat, nullableStringValue(sourcePath), nullableStringValue(session.Provenance.ConvertedAt), nullableStringValue(session.Provenance.ConverterVersion), nullableString(session.Provenance.OriginalSessionID),
		boolInt(session.Flags.ForResearch), boolInt(session.Flags.NeedsCleaning), boolInt(session.Flags.ContainsError), boolInt(session.Flags.ContainsPII), jsonNullable(session.Flags.Category),
		nullableString(session.Environment.Model), nullableString(session.Environment.ModelVersion), nullableFloat(session.Environment.Temperature), jsonNullable(session.Environment.ToolsEnabled), nullableString(session.Environment.SystemPrompt), nullableString(session.Environment.AgentFramework), nullableString(session.Environment.AgentVersion), nullableString(session.Environment.PlatformType), nullableString(session.Environment.ProviderHint),
		nullableString(session.OperationalContext.WorkingDirectory), nullableString(session.OperationalContext.GitBranch), nullableString(session.OperationalContext.GitRef), nullableString(session.OperationalContext.AutonomyLevel), nullableBoolPointer(session.OperationalContext.Sandbox), jsonNullable(session.OperationalContext.FrameworkConfig),
		nullableStringValue(session.Timing.PrivacyLevel), nullableFloat(session.Timing.DurationSeconds), nullableFloat(session.Timing.ActiveDurationSeconds), nullableString(session.Timing.StartedAt), nullableString(session.Timing.EndedAt), nullableIntPointer(session.Timing.HourOfDay), nullableIntPointer(session.Timing.DayOfWeek),
		conditionString(session.Condition, "guidance"), conditionString(session.Condition, "permission"), conditionJSON(session.Condition),
		nullableString(session.Coordination.ProjectID), nullableString(session.Coordination.PredecessorSession), nullableIntPointer(session.Coordination.ConcurrentSessions), nullableStringValue(session.Coordination.HumanAttention),
		outcomeSuccess(session.Outcome), outcomePartial(session.Outcome), outcomeFailureCodes(session.Outcome), outcomeNotes(session.Outcome),
		session.Metrics.TurnCount, session.Metrics.ToolCallCount, session.Metrics.ReadCount, session.Metrics.ModifyCount, session.Metrics.CreateCount, session.Metrics.ExecuteCount, mustJSON(session),
	}
	if err := insertRow(ctx, tx, "sessions", columns, args...); err != nil {
		return fmt.Errorf("insert session %s: %w", session.ID, err)
	}
	return nil
}

func insertTurn(ctx context.Context, tx *sql.Tx, sessionID string, turn minitrace.Turn) error {
	columns := []string{
		"session_id", "turn_index", "timestamp", "role", "source", "model", "content_type", "input_channel", "content",
		"framework_metadata_json", "thinking", "intent_requested", "intent_inferred", "intent_proactive", "was_streamed", "stream_log",
		"input_tokens", "output_tokens", "cache_read_tokens", "cache_creation_tokens", "reasoning_tokens", "tool_tokens", "raw_json",
	}
	args := []any{
		sessionID, turn.Index, nullableString(turn.Timestamp), turn.Role, nullableString(turn.Source), nullableString(turn.Model), nullableString(turn.ContentType), nullableString(turn.InputChannel), turn.Content,
		jsonNullable(turn.FrameworkMetadata), nullableString(turn.Thinking), intentBool(turn.IntentMarkers, "requested"), intentBool(turn.IntentMarkers, "inferred"), intentBool(turn.IntentMarkers, "proactive"), boolInt(turn.Streaming.WasStreamed), nullableString(turn.Streaming.StreamLog),
		usageInt(turn.Usage, "input"), usageInt(turn.Usage, "output"), usageInt(turn.Usage, "cache_read"), usageInt(turn.Usage, "cache_creation"), usageInt(turn.Usage, "reasoning"), usageInt(turn.Usage, "tool"), mustJSON(turn),
	}
	if err := insertRow(ctx, tx, "turns", columns, args...); err != nil {
		return fmt.Errorf("insert turn %s/%d: %w", sessionID, turn.Index, err)
	}
	return nil
}

func insertToolCall(ctx context.Context, tx *sql.Tx, sessionID string, toolCall minitrace.ToolCall) error {
	columns := []string{
		"session_id", "tool_call_id", "emitting_turn_index", "timestamp", "tool_name", "operation_type", "file_path",
		"command", "justification", "arguments_json", "success", "outcome_status", "result", "error", "exit_code", "duration_ms", "truncated",
		"full_bytes", "full_hash", "full_reference", "redacted", "content_origin", "position_in_session", "tools_before_json", "time_since_last_user",
		"framework_metadata_json", "spawned_agent_type", "spawned_agent_task_scope", "spawned_agent_sub_session_id", "spawned_agent_outcome_summary", "raw_json", "record_kind",
	}
	args := []any{
		sessionID, toolCall.ID, nullableIntPointer(toolCall.EmittingTurnIndex), nullableString(toolCall.Timestamp), toolCall.ToolName, toolCall.OperationType, nullableString(toolCall.Input.FilePath),
		nullableString(toolCall.Input.Command), nullableString(toolCall.Input.Justification), jsonNullable(toolCall.Input.Arguments), nullableBoolPointer(toolCall.Output.Success), string(toolCall.Output.OutcomeStatus()), nullableString(toolCall.Output.Result), nullableString(toolCall.Output.Error), nullableIntPointer(toolCall.Output.ExitCode), nullableIntPointer(toolCall.Output.DurationMS), boolInt(toolCall.Output.Truncated),
		nullableIntPointer(toolCall.Output.FullBytes), nullableString(toolCall.Output.FullHash), nullableString(toolCall.Output.FullReference), nullableBoolPointer(toolCall.Output.Redacted), nullableString(toolCall.Output.ContentOrigin), nullableFloat(toolCall.Context.PositionInSession), jsonNullable(toolCall.Context.ToolsBefore), nullableFloat(toolCall.Context.TimeSinceLastUser),
		jsonNullable(toolCall.FrameworkMetadata), spawnedAgentString(toolCall.SpawnedAgent, "type"), spawnedAgentString(toolCall.SpawnedAgent, "scope"), spawnedAgentString(toolCall.SpawnedAgent, "sub_session"), spawnedAgentString(toolCall.SpawnedAgent, "outcome"), mustJSON(toolCall), toolCall.EffectiveRecordKind(),
	}
	if err := insertRow(ctx, tx, "tool_calls", columns, args...); err != nil {
		return fmt.Errorf("insert tool call %s/%s: %w", sessionID, toolCall.ID, err)
	}
	return nil
}

func insertAnnotation(ctx context.Context, tx *sql.Tx, sessionID string, annotation minitrace.Annotation, ordinal int) error {
	annotationID := annotation.ID
	if annotationID == "" {
		annotationID = fmt.Sprintf("annotation-%06d", ordinal)
	}
	columns := []string{
		"session_id", "annotation_id", "timestamp", "annotator", "scope_type", "target_id", "category", "tags_json", "title", "detail",
		"minitrace_taxonomy_json", "mast_taxonomy_json", "toolemu_taxonomy_json", "classification", "raw_json",
	}
	args := []any{
		sessionID, annotationID, nullableStringValue(annotation.Timestamp), nullableStringValue(annotation.Annotator), annotation.Scope.Type, annotation.Scope.TargetID, annotation.Content.Category, jsonNullable(annotation.Content.Tags), annotation.Content.Title, annotation.Content.Detail,
		jsonNullable(annotation.TaxonomyMappings.Minitrace), jsonNullable(annotation.TaxonomyMappings.Mast), jsonNullable(annotation.TaxonomyMappings.Toolemu), nullableString(annotation.Classification), mustJSON(annotation),
	}
	if err := insertRow(ctx, tx, "annotations", columns, args...); err != nil {
		return fmt.Errorf("insert annotation %s/%s: %w", sessionID, annotationID, err)
	}
	return nil
}

func insertHandover(ctx context.Context, tx *sql.Tx, sessionID, direction string, handover *minitrace.HandoverDocument) error {
	if handover == nil {
		return nil
	}
	columns := []string{"session_id", "direction", "from_session", "to_session", "document", "state_description", "raw_json"}
	args := []any{sessionID, direction, nullableString(handover.FromSession), nullableString(handover.ToSession), handover.Document, nullableString(handover.StateDescription), mustJSON(handover)}
	if err := insertRow(ctx, tx, "handovers", columns, args...); err != nil {
		return fmt.Errorf("insert handover %s/%s: %w", sessionID, direction, err)
	}
	return nil
}

func insertTurnEvent(ctx context.Context, tx *sql.Tx, sessionID string, turn minitrace.Turn) error {
	eventID := fmt.Sprintf("turn-%06d", turn.Index)
	title := turn.Role
	if turn.ContentType != nil && *turn.ContentType != "" {
		title = title + " / " + *turn.ContentType
	}
	_, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO events(session_id, event_id, turn_index, ordinal, kind, role, title, summary, text, severity, collapsed_by_default, raw_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, eventID, turn.Index, 0, "turn", turn.Role, title, summarizeText(turn.Content, 120), turn.Content, "info", 0, mustJSON(turn))
	if err != nil {
		return fmt.Errorf("insert turn event %s/%d: %w", sessionID, turn.Index, err)
	}
	return nil
}

func insertToolCallEvent(ctx context.Context, tx *sql.Tx, sessionID string, toolCall minitrace.ToolCall) error {
	turnIndex := nullableIntPointer(toolCall.EmittingTurnIndex)
	ordinal := 0
	if toolCall.EmittingTurnIndex != nil {
		ordinal = 1
	}
	eventID := "tool-" + toolCall.ID
	severity := "info"
	if toolCall.Output.Failed() {
		severity = "error"
	}
	summary := firstNonEmptyPointer(toolCall.Input.FilePath, toolCall.Input.Command, toolCall.Output.Error, toolCall.Output.Result)
	_, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO events(session_id, event_id, turn_index, ordinal, kind, role, tool_call_id, title, summary, text, severity, collapsed_by_default, raw_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, eventID, turnIndex, ordinal, "tool_call", "tool", toolCall.ID, toolCall.ToolName+" / "+toolCall.OperationType, summarizeText(summary, 120), firstNonEmptyPointer(toolCall.Output.Result, toolCall.Output.Error), severity, 1, mustJSON(toolCall))
	if err != nil {
		return fmt.Errorf("insert tool call event %s/%s: %w", sessionID, toolCall.ID, err)
	}
	return nil
}

func insertAnnotationEvent(ctx context.Context, tx *sql.Tx, sessionID string, annotation minitrace.Annotation, ordinal int) error {
	annotationID := annotation.ID
	if annotationID == "" {
		annotationID = fmt.Sprintf("annotation-%06d", ordinal)
	}
	eventID := "annotation-" + annotationID
	_, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO events(session_id, event_id, ordinal, kind, role, annotation_id, title, summary, text, severity, collapsed_by_default, raw_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, eventID, ordinal, "annotation", "annotation", annotationID, annotation.Content.Title, annotation.Content.Category, annotation.Content.Detail, "info", 1, mustJSON(annotation))
	if err != nil {
		return fmt.Errorf("insert annotation event %s/%s: %w", sessionID, annotationID, err)
	}
	return nil
}

func insertExplicitEvent(ctx context.Context, tx *sql.Tx, sessionID string, event minitrace.Event, ordinal int) error {
	eventID := event.ID
	if eventID == "" {
		eventID = fmt.Sprintf("event-%06d", ordinal)
	}
	eventOrdinal := any(ordinal)
	if event.Ordinal != nil {
		eventOrdinal = *event.Ordinal
	}
	kind := event.Kind
	if kind == "" {
		kind = "source_event"
	}
	severity := event.Severity
	if severity == "" {
		severity = "info"
	}
	columns := []string{
		"session_id", "event_id", "timestamp", "turn_index", "ordinal", "kind", "role", "tool_call_id", "annotation_id", "attachment_id",
		"title", "summary", "text", "severity", "collapsed_by_default", "framework_metadata_json", "raw_json",
	}
	args := []any{
		sessionID, eventID, nullableString(event.Timestamp), nullableIntPointer(event.TurnIndex), eventOrdinal, kind, nullableStringValue(event.Role), nullableString(event.ToolCallID), nullableString(event.AnnotationID), nullableString(event.AttachmentID),
		nullableStringValue(event.Title), nullableStringValue(event.Summary), nullableStringValue(event.Text), severity, boolInt(event.CollapsedByDefault), jsonNullable(event.FrameworkMetadata), mustJSON(event),
	}
	if err := insertRow(ctx, tx, "events", columns, args...); err != nil {
		return fmt.Errorf("insert explicit event %s/%s: %w", sessionID, eventID, err)
	}
	return nil
}

func insertAttachment(ctx context.Context, tx *sql.Tx, sessionID string, attachment minitrace.Attachment, ordinal int) error {
	attachmentID := attachment.ID
	if attachmentID == "" {
		attachmentID = fmt.Sprintf("attachment-%06d", ordinal)
	}
	kind := attachment.Kind
	if kind == "" {
		kind = "attachment"
	}
	columns := []string{
		"session_id", "attachment_id", "timestamp", "kind", "name", "media_type", "path", "url", "size_bytes", "hash", "content_ref", "text_preview",
		"turn_index", "tool_call_id", "event_id", "framework_metadata_json", "raw_json",
	}
	args := []any{
		sessionID, attachmentID, nullableString(attachment.Timestamp), kind, nullableStringValue(attachment.Name), nullableStringValue(attachment.MediaType), nullableStringValue(attachment.Path), nullableStringValue(attachment.URL), nullableIntPointer(attachment.SizeBytes), nullableStringValue(attachment.Hash), nullableStringValue(attachment.ContentRef), nullableStringValue(attachment.TextPreview),
		nullableIntPointer(attachment.TurnIndex), nullableString(attachment.ToolCallID), nullableString(attachment.EventID), jsonNullable(attachment.FrameworkMetadata), mustJSON(attachment),
	}
	if err := insertRow(ctx, tx, "attachments", columns, args...); err != nil {
		return fmt.Errorf("insert attachment %s/%s: %w", sessionID, attachmentID, err)
	}
	return nil
}

func insertMetrics(ctx context.Context, tx *sql.Tx, session *minitrace.Session) error {
	columns := []string{
		"session_id", "turn_count", "tool_call_count", "read_count", "modify_count", "create_count", "execute_count", "delegate_count",
		"read_ratio", "time_to_first_action", "idle_ratio", "total_input_tokens", "total_output_tokens", "total_cache_read_tokens", "total_cache_creation_tokens",
		"total_reasoning_tokens", "total_tool_tokens", "session_cost", "subagent_count", "subagent_tool_calls", "model_switches", "unique_models",
		"median_response_tokens", "max_response_tokens", "raw_json",
	}
	args := []any{
		session.ID, session.Metrics.TurnCount, session.Metrics.ToolCallCount, session.Metrics.ReadCount, session.Metrics.ModifyCount, session.Metrics.CreateCount, session.Metrics.ExecuteCount, session.Metrics.DelegateCount,
		nullableFloat(session.Metrics.ReadRatio), nullableFloat(session.Metrics.TimeToFirstAction), nullableFloat(session.Metrics.IdleRatio), nullableIntPointer(session.Metrics.TotalInputTokens), nullableIntPointer(session.Metrics.TotalOutputTokens), nullableIntPointer(session.Metrics.TotalCacheReadTokens), nullableIntPointer(session.Metrics.TotalCacheCreationTokens),
		nullableIntPointer(session.Metrics.TotalReasoningTokens), nullableIntPointer(session.Metrics.TotalToolTokens), nullableFloat(session.Metrics.SessionCost), session.Metrics.SubagentCount, session.Metrics.SubagentToolCalls, nullableIntPointer(session.Metrics.ModelSwitches), nullableIntPointer(session.Metrics.UniqueModels),
		nullableIntPointer(session.Metrics.MedianResponseTokens), nullableIntPointer(session.Metrics.MaxResponseTokens), mustJSON(session.Metrics),
	}
	if err := insertRow(ctx, tx, "metrics", columns, args...); err != nil {
		return fmt.Errorf("insert metrics %s: %w", session.ID, err)
	}
	return nil
}

func insertRow(ctx context.Context, tx *sql.Tx, table string, columns []string, args ...any) error {
	if len(columns) != len(args) {
		return fmt.Errorf("insert %s has %d columns but %d values", table, len(columns), len(args))
	}
	if err := validateInsertTarget(table, columns); err != nil {
		return err
	}
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	// #nosec G201 -- table and column identifiers are fixed schema-owned names validated by validateInsertTarget; all row values remain parameterized.
	stmt := fmt.Sprintf("INSERT OR REPLACE INTO %s(%s) VALUES (%s)", table, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	_, err := tx.ExecContext(ctx, stmt, args...)
	return err
}

func validateInsertTarget(table string, columns []string) error {
	if table == "" || !isSQLIdentifier(table) {
		return fmt.Errorf("invalid insert table %q", table)
	}
	for _, column := range columns {
		if column == "" || !isSQLIdentifier(column) {
			return fmt.Errorf("invalid insert column %q for table %s", column, table)
		}
	}
	return nil
}

func isSQLIdentifier(v string) bool {
	for i, r := range v {
		if r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return v != ""
}

func nullableString(v *string) any {
	if v == nil || *v == "" {
		return nil
	}
	return *v
}

func nullableStringValue(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func nullableIntPointer(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableFloat(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableBoolPointer(v *bool) any {
	if v == nil {
		return nil
	}
	return boolInt(*v)
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func usageInt(usage *minitrace.Usage, field string) any {
	if usage == nil {
		return nil
	}
	switch field {
	case "input":
		return nullableIntPointer(usage.InputTokens)
	case "output":
		return nullableIntPointer(usage.OutputTokens)
	case "cache_read":
		return nullableIntPointer(usage.CacheReadTokens)
	case "cache_creation":
		return nullableIntPointer(usage.CacheCreationTokens)
	case "reasoning":
		return nullableIntPointer(usage.ReasoningTokens)
	case "tool":
		return nullableIntPointer(usage.ToolTokens)
	default:
		return nil
	}
}

func intentBool(markers *minitrace.IntentMarkers, field string) any {
	if markers == nil {
		return nil
	}
	switch field {
	case "requested":
		return boolInt(markers.Requested)
	case "inferred":
		return boolInt(markers.Inferred)
	case "proactive":
		return boolInt(markers.Proactive)
	default:
		return nil
	}
}

func conditionString(condition *minitrace.Condition, field string) any {
	if condition == nil {
		return nil
	}
	switch field {
	case "guidance":
		return nullableString(condition.GuidanceVariant)
	case "permission":
		return nullableString(condition.PermissionLevel)
	default:
		return nil
	}
}

func conditionJSON(condition *minitrace.Condition) any {
	if condition == nil {
		return nil
	}
	return jsonNullable(condition.Custom)
}

func outcomeSuccess(outcome *minitrace.Outcome) any {
	if outcome == nil || outcome.Success == nil {
		return nil
	}
	return boolInt(*outcome.Success)
}

func outcomePartial(outcome *minitrace.Outcome) any {
	if outcome == nil {
		return nil
	}
	return boolInt(outcome.Partial)
}

func outcomeFailureCodes(outcome *minitrace.Outcome) any {
	if outcome == nil {
		return nil
	}
	return jsonNullable(outcome.FailureCodes)
}

func outcomeNotes(outcome *minitrace.Outcome) any {
	if outcome == nil {
		return nil
	}
	return nullableString(outcome.OutcomeNotes)
}

func spawnedAgentString(agent *minitrace.SpawnedAgent, field string) any {
	if agent == nil {
		return nil
	}
	switch field {
	case "type":
		return nullableStringValue(agent.AgentType)
	case "scope":
		return nullableStringValue(agent.TaskScope)
	case "sub_session":
		return nullableString(agent.SubSessionID)
	case "outcome":
		return nullableStringValue(agent.OutcomeSummary)
	default:
		return nil
	}
}

func firstNonEmptyPointer(values ...*string) string {
	for _, value := range values {
		if value != nil && *value != "" {
			return *value
		}
	}
	return ""
}

func summarizeText(value string, maxChars int) string {
	if maxChars > 0 && len(value) > maxChars {
		return value[:maxChars]
	}
	return value
}

func jsonNullable(v any) any {
	if v == nil {
		return nil
	}
	payload, err := json.Marshal(v)
	if err != nil || string(payload) == "null" {
		return nil
	}
	return string(payload)
}

func mustJSON(v any) string {
	payload, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(payload)
}
