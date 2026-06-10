package minitracedb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSchemaContainsCoreTables(t *testing.T) {
	schema := Schema()
	if schema.Version != SchemaVersion {
		t.Fatalf("schema version = %q, want %q", schema.Version, SchemaVersion)
	}
	want := map[string]bool{
		"sessions":        false,
		"turns":           false,
		"tool_calls":      false,
		"turn_tool_calls": false,
		"files":           false,
		"metrics":         false,
		"attachments":     false,
		"events":          false,
	}
	for _, table := range schema.Tables {
		if _, ok := want[table.Name]; ok {
			want[table.Name] = true
		}
		if table.CreateSQL == "" {
			t.Fatalf("table %s has empty CreateSQL", table.Name)
		}
		if len(table.Columns) == 0 {
			t.Fatalf("table %s has no columns", table.Name)
		}
	}
	for table, seen := range want {
		if !seen {
			t.Fatalf("missing table %s", table)
		}
	}
}

func TestCreateSchemaCreatesTables(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := CreateSchema(context.Background(), db); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	for _, table := range AllowedTableNames() {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("expected table %s: %v", table, err)
		}
		if name != table {
			t.Fatalf("table name = %q, want %q", name, table)
		}
	}
}

func TestValidateAllowedTable(t *testing.T) {
	if !ValidateAllowedTable("sessions") || !ValidateAllowedTable("SESSIONS") {
		t.Fatalf("expected sessions to be allowed")
	}
	if ValidateAllowedTable("sqlite_master") {
		t.Fatalf("sqlite_master should not be an allowed minitrace table")
	}
}
