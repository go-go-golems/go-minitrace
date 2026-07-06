package minitracedb

import (
	"context"
	"database/sql"
	"path/filepath"
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

func TestQueryRunnerAllowsSelectFollowedByNewline(t *testing.T) {
	runner := setupQueryRunner(t)
	rows, err := runner.Query(context.Background(), "SELECT\n  session_id\nFROM sessions")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 1 || rows[0]["session_id"] != "s1" {
		t.Fatalf("unexpected rows %#v", rows)
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

func TestQueryRunnerSQLiteCatalogDenialSuggestsSchemaIntrospection(t *testing.T) {
	runner := setupQueryRunner(t)
	result, err := runner.QueryResult(context.Background(), `SELECT * FROM sqlite_master`)
	if err != nil {
		t.Fatalf("QueryResult error: %v", err)
	}
	if !strings.Contains(result.Error, "disallowed") {
		t.Fatalf("expected disallowed object error, got %#v", result)
	}
	if !strings.Contains(result.Error, "db.schema()") || !strings.Contains(result.Error, "db.tables()") {
		t.Fatalf("expected schema introspection hint mentioning db.schema() and db.tables(), got %q", result.Error)
	}
}

func TestQueryRunnerRejectsQuotedDisallowedObjects(t *testing.T) {
	runner := setupQueryRunner(t)
	queries := []string{
		`SELECT name FROM "sqlite_master"`,
		"SELECT name FROM `sqlite_master`",
		`SELECT name FROM [sqlite_master]`,
	}
	for _, query := range queries {
		result, err := runner.QueryResult(context.Background(), query)
		if err != nil {
			t.Fatalf("QueryResult error: %v", err)
		}
		if result.Error == "" {
			t.Fatalf("expected disallowed object error for %q, got %#v", query, result)
		}
	}
}

func TestQueryRunnerAllowsQuotedAllowedObjects(t *testing.T) {
	runner := setupQueryRunner(t)
	rows, err := runner.Query(context.Background(), `SELECT session_id FROM "sessions"`)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 1 || rows[0]["session_id"] != "s1" {
		t.Fatalf("unexpected rows %#v", rows)
	}
}

func TestQueryRunnerAllowsCTEAliases(t *testing.T) {
	runner := setupQueryRunner(t)
	rows, err := runner.Query(context.Background(), `WITH recent AS (SELECT session_id FROM sessions) SELECT session_id FROM recent`)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 1 || rows[0]["session_id"] != "s1" {
		t.Fatalf("unexpected rows %#v", rows)
	}
}

func TestQueryRunnerRejectsDisallowedObjectsInsideCTE(t *testing.T) {
	runner := setupQueryRunner(t)
	result, err := runner.QueryResult(context.Background(), `WITH catalog AS (SELECT name FROM sqlite_master) SELECT name FROM catalog`)
	if err != nil {
		t.Fatalf("QueryResult error: %v", err)
	}
	if result.Error == "" {
		t.Fatalf("expected disallowed object error, got %#v", result)
	}
}

func TestQueryRunnerAllowsSchemaQualifiedAllowedObjects(t *testing.T) {
	runner := setupQueryRunner(t)
	rows, err := runner.Query(context.Background(), `SELECT session_id FROM main.sessions`)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 1 || rows[0]["session_id"] != "s1" {
		t.Fatalf("unexpected rows %#v", rows)
	}
}

func TestQueryRunnerAttachmentAllowlistIsSchemaAware(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.sqlite")
	annoPath := filepath.Join(dir, "annotations.db")

	mainDB, err := OpenSQLiteFile(ctx, mainPath)
	if err != nil {
		t.Fatalf("OpenSQLiteFile main: %v", err)
	}
	if err := CreateSchema(ctx, mainDB); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	if _, err := mainDB.ExecContext(ctx, `INSERT INTO sessions(session_id, title) VALUES ('main-session', 'main row')`); err != nil {
		t.Fatalf("insert main session: %v", err)
	}
	if err := mainDB.Close(); err != nil {
		t.Fatalf("close main db: %v", err)
	}

	annoDB, err := sql.Open("sqlite3", annoPath)
	if err != nil {
		t.Fatalf("open annotation fixture: %v", err)
	}
	_, err = annoDB.ExecContext(ctx, `
		CREATE TABLE annotations (id TEXT PRIMARY KEY, detail TEXT);
		INSERT INTO annotations VALUES ('a1', 'expected annotation row');
		CREATE TABLE sync_state (session_id TEXT PRIMARY KEY);
		CREATE TABLE sessions (secret TEXT);
		INSERT INTO sessions VALUES ('attached schema leak sentinel');
	`)
	if err != nil {
		t.Fatalf("seed annotation fixture: %v", err)
	}
	if err := annoDB.Close(); err != nil {
		t.Fatalf("close annotation fixture: %v", err)
	}

	attachedDB, err := OpenSQLiteReadOnlyAttached(ctx, mainPath, map[string]string{AnnotationsAttachSchema: annoPath})
	if err != nil {
		t.Fatalf("OpenSQLiteReadOnlyAttached: %v", err)
	}
	target, err := NewDBQueryTarget(attachedDB, AllowedObjectNamesWithLiveAnnotations(), QueryOptions{MaxRows: 10})
	if err != nil {
		_ = attachedDB.Close()
		t.Fatalf("NewDBQueryTarget: %v", err)
	}
	defer func() { _ = target.Close() }()

	rows, err := target.Query(ctx, `SELECT detail FROM anno.annotations`)
	if err != nil {
		t.Fatalf("query anno.annotations: %v", err)
	}
	if len(rows) != 1 || rows[0]["detail"] != "expected annotation row" {
		t.Fatalf("unexpected annotation rows %#v", rows)
	}

	rows, err = target.Query(ctx, `SELECT title FROM main.sessions`)
	if err != nil {
		t.Fatalf("query main.sessions: %v", err)
	}
	if len(rows) != 1 || rows[0]["title"] != "main row" {
		t.Fatalf("unexpected main rows %#v", rows)
	}

	result, err := target.QueryResult(ctx, `SELECT secret FROM anno.sessions`)
	if err != nil {
		t.Fatalf("QueryResult anno.sessions: %v", err)
	}
	if result.Error == "" || !strings.Contains(result.Error, "anno.sessions") {
		t.Fatalf("expected anno.sessions to be denied with schema-qualified error, got %#v", result)
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
