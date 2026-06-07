package minitracedb

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupQueryRunner(t *testing.T) *QueryRunner {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := CreateSchema(context.Background(), db); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO sessions(session_id, title, tool_call_count) VALUES ('s1', 'hello world', 3)`); err != nil {
		t.Fatalf("insert fixture: %v", err)
	}
	runner, err := NewQueryRunner(db, AllowedTableNames(), QueryOptions{MaxRows: 10, MaxColumns: 10, MaxCellChars: 5})
	if err != nil {
		t.Fatalf("NewQueryRunner: %v", err)
	}
	return runner
}

func TestQueryRunnerSelect(t *testing.T) {
	runner := setupQueryRunner(t)
	rows, err := runner.Query(context.Background(), `SELECT session_id, tool_call_count FROM sessions ORDER BY session_id`)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 1 || rows[0]["session_id"] != "s1" {
		t.Fatalf("unexpected rows %#v", rows)
	}
}

func TestQueryRunnerQueryOne(t *testing.T) {
	runner := setupQueryRunner(t)
	row, err := runner.QueryOne(context.Background(), `SELECT session_id FROM sessions ORDER BY session_id`)
	if err != nil {
		t.Fatalf("QueryOne: %v", err)
	}
	if row["session_id"] != "s1" {
		t.Fatalf("unexpected row %#v", row)
	}
}

func TestQueryRunnerRejectsWrites(t *testing.T) {
	runner := setupQueryRunner(t)
	result, err := runner.QueryResult(context.Background(), `INSERT INTO sessions(session_id) VALUES ('bad')`)
	if err != nil {
		t.Fatalf("QueryResult error: %v", err)
	}
	if !strings.Contains(result.Error, "only SELECT and WITH") {
		t.Fatalf("expected readonly error, got %#v", result)
	}
}

func TestQueryRunnerRejectsMultipleStatements(t *testing.T) {
	runner := setupQueryRunner(t)
	result, err := runner.QueryResult(context.Background(), `SELECT 1; SELECT 2`)
	if err != nil {
		t.Fatalf("QueryResult error: %v", err)
	}
	if !strings.Contains(result.Error, "multiple SQL") {
		t.Fatalf("expected multi statement error, got %#v", result)
	}
}

func TestQueryRunnerRejectsDisallowedObjects(t *testing.T) {
	runner := setupQueryRunner(t)
	result, err := runner.QueryResult(context.Background(), `SELECT name FROM sqlite_master`)
	if err != nil {
		t.Fatalf("QueryResult error: %v", err)
	}
	if !strings.Contains(result.Error, "disallowed") {
		t.Fatalf("expected disallowed object error, got %#v", result)
	}
}

func TestQueryRunnerTruncatesCells(t *testing.T) {
	runner := setupQueryRunner(t)
	row, err := runner.QueryOne(context.Background(), `SELECT title FROM sessions`)
	if err != nil {
		t.Fatalf("QueryOne: %v", err)
	}
	if row["title"] != "hello" {
		t.Fatalf("expected truncated title, got %#v", row["title"])
	}
}
