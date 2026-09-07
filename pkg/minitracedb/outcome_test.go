package minitracedb

import (
	"context"
	"database/sql"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestMaterializeNullableToolOutcomes(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	session := minitrace.BuildSessionSkeleton("outcomes", "codex", "synthetic", "test")
	statuses := []minitrace.ToolOutcomeStatus{
		minitrace.ToolOutcomeUnknown, minitrace.ToolOutcomePending, minitrace.ToolOutcomeCancelled,
		minitrace.ToolOutcomeSucceeded, minitrace.ToolOutcomeFailed,
	}
	path := "attempt.txt"
	for _, status := range statuses {
		output := minitrace.ToolCallOutput{Status: status}
		if status == minitrace.ToolOutcomeSucceeded || status == minitrace.ToolOutcomeFailed {
			output.SetSuccess(status == minitrace.ToolOutcomeSucceeded)
		}
		session.ToolCalls = append(session.ToolCalls, minitrace.ToolCall{
			ID: string(status), ToolName: "exec_command", OperationType: "EXECUTE",
			Input: minitrace.ToolCallInput{FilePath: &path}, Output: output,
		})
	}
	if err := MaterializeSession(ctx, db, &session, MaterializeOptions{}); err != nil {
		t.Fatal(err)
	}
	for _, status := range statuses {
		var success, fileSuccess sql.NullInt64
		var actual string
		if err := db.QueryRowContext(ctx, `SELECT success, outcome_status FROM tool_calls WHERE tool_call_id=?`, string(status)).Scan(&success, &actual); err != nil {
			t.Fatal(err)
		}
		if actual != string(status) {
			t.Errorf("status=%s, want %s", actual, status)
		}
		known := status == minitrace.ToolOutcomeSucceeded || status == minitrace.ToolOutcomeFailed
		if success.Valid != known {
			t.Errorf("%s success=%+v, known=%v", status, success, known)
		}
		if known && (success.Int64 == 1) != (status == minitrace.ToolOutcomeSucceeded) {
			t.Errorf("wrong binary outcome for %s: %+v", status, success)
		}
		if err := db.QueryRowContext(ctx, `SELECT success FROM files WHERE tool_call_id=?`, string(status)).Scan(&fileSuccess); err != nil {
			t.Fatal(err)
		}
		if fileSuccess != success {
			t.Errorf("file projection changed outcome for %s: %+v vs %+v", status, fileSuccess, success)
		}
	}
	var failures int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM tool_calls WHERE success=0`).Scan(&failures); err != nil {
		t.Fatal(err)
	}
	if failures != 1 {
		t.Fatalf("unknown outcomes counted as failures: %d", failures)
	}
	var errorEvents int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM events WHERE severity='error'`).Scan(&errorEvents); err != nil {
		t.Fatal(err)
	}
	if errorEvents != 1 {
		t.Fatalf("unknown outcomes rendered as error events: %d", errorEvents)
	}
}
