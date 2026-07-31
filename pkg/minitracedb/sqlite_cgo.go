//go:build cgo

package minitracedb

// The cgo half of minitracedb: everything that needs mattn/go-sqlite3, which
// is a cgo binding.
//
// Why this file exists. Importers that only want the schema and the query
// presets — `CreateSchema`, `AllowedTableNames`, `ListPresetEntries`,
// `ResolvePresetSQL` — used to drag mattn/go-sqlite3 in with them, because it
// was imported by open.go and query.go directly. Building such a program with
// CGO_ENABLED=0 does not fail cleanly: mattn's non-cgo build compiles to a
// stub, so the symbols simply vanish and the compiler reports a screenful of
// `undefined: sqlite3.SQLITE_OK` from inside a dependency the importer never
// named. Confining the binding here means those importers build pure-Go, and
// the functions that genuinely need SQLite's C API keep their signatures and
// fail with one clear error instead.
//
// Everything here has a same-named counterpart in sqlite_nocgo.go.

import (
	"database/sql"
	"fmt"
	"strings"

	sqlite3 "github.com/mattn/go-sqlite3"
)

// registerAttachDriver registers a driver whose ConnectHook ATTACHes each
// auxiliary database, so the attachment survives connection churn.
func registerAttachDriver(driverName string, attachments []attachment) error {
	sql.Register(driverName, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			for _, att := range attachments {
				stmt := fmt.Sprintf(
					"ATTACH DATABASE '%s' AS \"%s\"",
					strings.ReplaceAll(att.path, "'", "''"),
					strings.ReplaceAll(att.schema, `"`, `""`),
				)
				if _, err := conn.Exec(stmt, nil); err != nil {
					return fmt.Errorf("attach %s as %s: %w", att.path, att.schema, err)
				}
			}
			return nil
		},
	})
	return nil
}

func setSQLiteAuthorizer(conn *sql.Conn, callback func(int, string, string, string) int) error {
	if conn == nil {
		return fmt.Errorf("sql connection is nil")
	}
	return conn.Raw(func(driverConn any) error {
		sqliteConn, ok := driverConn.(*sqlite3.SQLiteConn)
		if !ok {
			return fmt.Errorf("unexpected sqlite driver connection type %T", driverConn)
		}
		sqliteConn.RegisterAuthorizer(callback)
		return nil
	})
}

func ensureReadonlyPreparedQuery(conn *sql.Conn, sqlText string) error {
	return conn.Raw(func(driverConn any) error {
		sqliteConn, ok := driverConn.(*sqlite3.SQLiteConn)
		if !ok {
			return fmt.Errorf("unexpected sqlite driver connection type %T", driverConn)
		}
		stmtDriver, err := sqliteConn.Prepare(sqlText)
		if err != nil {
			return err
		}
		defer func() { _ = stmtDriver.Close() }()
		stmt, ok := stmtDriver.(*sqlite3.SQLiteStmt)
		if !ok {
			return fmt.Errorf("unexpected sqlite statement type %T", stmtDriver)
		}
		if !stmt.Readonly() {
			return fmt.Errorf("only read-only SELECT queries are allowed")
		}
		return nil
	})
}

// isDeniedReadOp reports whether an authorizer action code is SQLITE_READ,
// which gets a more helpful error message than the generic denial.
func isDeniedReadOp(op int) bool { return op == sqlite3.SQLITE_READ }

func newReadOnlyAuthorizer(allowedReads map[string]struct{}, state *queryAuthorizationState) func(int, string, string, string) int {
	return func(op int, object, _, database string) int {
		switch op {
		case sqlite3.SQLITE_SELECT, sqlite3.SQLITE_FUNCTION:
			return sqlite3.SQLITE_OK
		case sqlite3.SQLITE_READ:
			if len(allowedReads) == 0 {
				return sqlite3.SQLITE_OK
			}
			key := readKey(database, object)
			if _, ok := allowedReads[key]; ok {
				return sqlite3.SQLITE_OK
			}
			state.deny(op, database, object)
			return sqlite3.SQLITE_DENY
		case sqlite3.SQLITE_INSERT, sqlite3.SQLITE_UPDATE, sqlite3.SQLITE_DELETE, sqlite3.SQLITE_PRAGMA, sqlite3.SQLITE_ATTACH, sqlite3.SQLITE_DETACH, sqlite3.SQLITE_TRANSACTION, sqlite3.SQLITE_CREATE_INDEX, sqlite3.SQLITE_CREATE_TABLE, sqlite3.SQLITE_CREATE_TEMP_INDEX, sqlite3.SQLITE_CREATE_TEMP_TABLE, sqlite3.SQLITE_CREATE_TEMP_TRIGGER, sqlite3.SQLITE_CREATE_TEMP_VIEW, sqlite3.SQLITE_CREATE_TRIGGER, sqlite3.SQLITE_CREATE_VIEW, sqlite3.SQLITE_CREATE_VTABLE, sqlite3.SQLITE_DROP_INDEX, sqlite3.SQLITE_DROP_TABLE, sqlite3.SQLITE_DROP_TEMP_INDEX, sqlite3.SQLITE_DROP_TEMP_TABLE, sqlite3.SQLITE_DROP_TEMP_TRIGGER, sqlite3.SQLITE_DROP_TEMP_VIEW, sqlite3.SQLITE_DROP_TRIGGER, sqlite3.SQLITE_DROP_VIEW, sqlite3.SQLITE_DROP_VTABLE, sqlite3.SQLITE_ALTER_TABLE, sqlite3.SQLITE_REINDEX, sqlite3.SQLITE_ANALYZE, sqlite3.SQLITE_SAVEPOINT:
			state.deny(op, database, object)
			return sqlite3.SQLITE_DENY
		default:
			state.deny(op, database, object)
			return sqlite3.SQLITE_DENY
		}
	}
}
