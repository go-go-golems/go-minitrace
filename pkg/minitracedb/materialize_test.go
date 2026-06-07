package minitracedb

import (
	"context"
	"database/sql"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	_ "github.com/mattn/go-sqlite3"
)

func TestMaterializeSessionPopulatesCoreTables(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	session := fixtureSession()
	if err := MaterializeSession(ctx, db, session, MaterializeOptions{SourcePath: "/tmp/session.minitrace.json"}); err != nil {
		t.Fatalf("MaterializeSession: %v", err)
	}
	assertCount(t, db, "sessions", 1)
	assertCount(t, db, "turns", 2)
	assertCount(t, db, "tool_calls", 1)
	assertCount(t, db, "turn_tool_calls", 1)
	assertCount(t, db, "files", 1)
	assertCount(t, db, "metrics", 1)
	assertCount(t, db, "events", 3)

	runner, err := NewQueryRunner(db, AllowedTableNames(), QueryOptions{})
	if err != nil {
		t.Fatalf("NewQueryRunner: %v", err)
	}
	row, err := runner.QueryOne(ctx, `SELECT tool_name, operation_type, file_path FROM tool_calls`)
	if err != nil {
		t.Fatalf("query tool calls: %v", err)
	}
	if row["tool_name"] != "Read" || row["operation_type"] != "read" || row["file_path"] != "app.go" {
		t.Fatalf("unexpected tool call row: %#v", row)
	}
}

func assertCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("count %s = %d, want %d", table, got, want)
	}
}

func fixtureSession() *minitrace.Session {
	session := minitrace.BuildSessionSkeleton("session-1", "pi", "minitrace-json-v1", "test")
	title := "Fixture session"
	session.Title = &title
	session.Turns = []minitrace.Turn{
		{Index: 0, Role: "user", Content: "Read app.go"},
		{Index: 1, Role: "assistant", Content: "Reading", ToolCallsInTurn: []string{"tool-1"}},
	}
	path := "app.go"
	result := "package main"
	emittingTurn := 1
	session.ToolCalls = []minitrace.ToolCall{{
		ID:                "tool-1",
		EmittingTurnIndex: &emittingTurn,
		ToolName:          "Read",
		OperationType:     "read",
		Input:             minitrace.ToolCallInput{FilePath: &path},
		Output:            minitrace.ToolCallOutput{Success: true, Result: &result},
	}}
	metrics := minitrace.ComputeMetrics(session.Turns, session.ToolCalls, session.Timing, len(session.Annotations), nil)
	session.Metrics = metrics
	return &session
}
