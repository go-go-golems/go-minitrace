package minitracedb

import (
	"context"
	"database/sql"
	"testing"
)

func TestMemorySettingsOnEveryConnection(t *testing.T) {
	ctx := context.Background()
	db, err := OpenSQLiteMemory(ctx, "memory_settings")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Holding both connections forces database/sql to establish a second one.
	for i := 0; i < 2; i++ {
		conn, err := db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		checkPragma(t, conn, "PRAGMA temp_store", 2)
		checkPragma(t, conn, "PRAGMA foreign_keys", 1)
	}
}

func checkPragma(t *testing.T, conn *sql.Conn, query string, want int) {
	t.Helper()
	var got int
	if err := conn.QueryRowContext(context.Background(), query).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s = %d, want %d", query, got, want)
	}
}
