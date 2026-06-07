package minitracedb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

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
		if toolCall.Input.FilePath != nil && *toolCall.Input.FilePath != "" {
			turnIndex := nullableIntPointer(toolCall.EmittingTurnIndex)
			if _, err := tx.ExecContext(ctx, `INSERT INTO files(session_id, tool_call_id, path, operation_type, tool_name, success, turn_index) VALUES (?, ?, ?, ?, ?, ?, ?)`, session.ID, toolCall.ID, *toolCall.Input.FilePath, toolCall.OperationType, toolCall.ToolName, boolInt(toolCall.Output.Success), turnIndex); err != nil {
				return fmt.Errorf("insert file %s/%s: %w", session.ID, *toolCall.Input.FilePath, err)
			}
		}
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
	raw := mustJSON(session)
	sourcePath := opts.SourcePath
	if sourcePath == "" && session.Provenance.SourcePath != nil {
		sourcePath = *session.Provenance.SourcePath
	}
	_, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO sessions(
		session_id, schema_version, profile, title, summary, classification, source_format, source_path, converted_at,
		model, agent_framework, working_directory, started_at, ended_at, turn_count, tool_call_count,
		read_count, modify_count, create_count, execute_count, contains_error, raw_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.SchemaVersion, session.Profile, nullableString(session.Title), nullableString(session.Summary), session.Classification,
		session.Provenance.SourceFormat, nullableStringValue(sourcePath), nullableStringValue(session.Provenance.ConvertedAt), nullableString(session.Environment.Model), nullableString(session.Environment.AgentFramework), nullableString(session.OperationalContext.WorkingDirectory), nullableString(session.Timing.StartedAt), nullableString(session.Timing.EndedAt),
		session.Metrics.TurnCount, session.Metrics.ToolCallCount, session.Metrics.ReadCount, session.Metrics.ModifyCount, session.Metrics.CreateCount, session.Metrics.ExecuteCount, boolInt(session.Flags.ContainsError), raw)
	if err != nil {
		return fmt.Errorf("insert session %s: %w", session.ID, err)
	}
	return nil
}

func insertTurn(ctx context.Context, tx *sql.Tx, sessionID string, turn minitrace.Turn) error {
	_, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO turns(
		session_id, turn_index, timestamp, role, source, model, content_type, input_channel, content,
		thinking, was_streamed, input_tokens, output_tokens, reasoning_tokens, raw_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, turn.Index, nullableString(turn.Timestamp), turn.Role, nullableString(turn.Source), nullableString(turn.Model), nullableString(turn.ContentType), nullableString(turn.InputChannel), turn.Content, nullableString(turn.Thinking), boolInt(turn.Streaming.WasStreamed), usageInt(turn.Usage, "input"), usageInt(turn.Usage, "output"), usageInt(turn.Usage, "reasoning"), mustJSON(turn))
	if err != nil {
		return fmt.Errorf("insert turn %s/%d: %w", sessionID, turn.Index, err)
	}
	return nil
}

func insertToolCall(ctx context.Context, tx *sql.Tx, sessionID string, toolCall minitrace.ToolCall) error {
	_, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO tool_calls(
		session_id, tool_call_id, emitting_turn_index, timestamp, tool_name, operation_type, file_path,
		command, justification, arguments_json, success, result, error, exit_code, duration_ms, truncated, raw_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, toolCall.ID, nullableIntPointer(toolCall.EmittingTurnIndex), nullableString(toolCall.Timestamp), toolCall.ToolName, toolCall.OperationType, nullableString(toolCall.Input.FilePath), nullableString(toolCall.Input.Command), nullableString(toolCall.Input.Justification), mustJSON(toolCall.Input.Arguments), boolInt(toolCall.Output.Success), nullableString(toolCall.Output.Result), nullableString(toolCall.Output.Error), nullableIntPointer(toolCall.Output.ExitCode), nullableIntPointer(toolCall.Output.DurationMS), boolInt(toolCall.Output.Truncated), mustJSON(toolCall))
	if err != nil {
		return fmt.Errorf("insert tool call %s/%s: %w", sessionID, toolCall.ID, err)
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
	if !toolCall.Output.Success {
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

func insertMetrics(ctx context.Context, tx *sql.Tx, session *minitrace.Session) error {
	_, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO metrics(
		session_id, turn_count, tool_call_count, read_count, modify_count, create_count, execute_count,
		delegate_count, read_ratio, time_to_first_action, total_input_tokens, total_output_tokens, session_cost, raw_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.Metrics.TurnCount, session.Metrics.ToolCallCount, session.Metrics.ReadCount, session.Metrics.ModifyCount, session.Metrics.CreateCount, session.Metrics.ExecuteCount, session.Metrics.DelegateCount, nullableFloat(session.Metrics.ReadRatio), nullableFloat(session.Metrics.TimeToFirstAction), nullableIntPointer(session.Metrics.TotalInputTokens), nullableIntPointer(session.Metrics.TotalOutputTokens), nullableFloat(session.Metrics.SessionCost), mustJSON(session.Metrics))
	if err != nil {
		return fmt.Errorf("insert metrics %s: %w", session.ID, err)
	}
	return nil
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
	case "reasoning":
		return nullableIntPointer(usage.ReasoningTokens)
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

func summarizeText(value string, max int) string {
	if max > 0 && len(value) > max {
		return value[:max]
	}
	return value
}

func mustJSON(v any) string {
	payload, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(payload)
}
