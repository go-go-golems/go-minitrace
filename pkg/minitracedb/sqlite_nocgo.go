//go:build !cgo

package minitracedb

// The non-cgo half of minitracedb. See sqlite_cgo.go for why the split exists.
//
// mattn/go-sqlite3 is a cgo binding, so with CGO_ENABLED=0 there is no SQLite
// here at all — not a slower one, none. Every function that needs the C API
// keeps its signature and returns one clear error, so a program that only uses
// the schema and preset helpers (CreateSchema, AllowedTableNames,
// ListPresetEntries, ResolvePresetSQL) builds and runs pure-Go, and a program
// that actually opens a database gets told why instead of failing to compile
// inside a dependency it never named.
//
// The `sqlite3` driver these helpers rely on is likewise unregistered under
// this build, so sql.Open("sqlite3", …) in open.go reports `unknown driver`.

import (
	"database/sql"
	"fmt"
)

const errNoCGO = "minitracedb needs SQLite through mattn/go-sqlite3, which is a cgo binding: rebuild with CGO_ENABLED=1"

func registerAttachDriver(_ string, _ []attachment) error {
	return fmt.Errorf("%s", errNoCGO)
}

func setSQLiteAuthorizer(_ *sql.Conn, _ func(int, string, string, string) int) error {
	return fmt.Errorf("%s", errNoCGO)
}

func ensureReadonlyPreparedQuery(_ *sql.Conn, _ string) error {
	return fmt.Errorf("%s", errNoCGO)
}

// isDeniedReadOp cannot consult SQLITE_READ without the binding. The denial
// message it selects is a nicety, and the authorizer never runs in this build
// anyway, so the generic branch is correct.
func isDeniedReadOp(_ int) bool { return false }

func newReadOnlyAuthorizer(_ map[string]struct{}, _ *queryAuthorizationState) func(int, string, string, string) int {
	return nil
}
