package annotate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// AttachAnnotationsToDuckDB installs sqlite_scanner, loads it, and attaches
// the annotations SQLite database to the DuckDB connection.
//
// This makes annotations directly queryable as SQLite tables from DuckDB SQL:
//
//	SELECT * FROM annotations JOIN sessions_base ...
//
// Annotations are live — DuckDB reads the SQLite file directly on each query.
// No refresh is needed after writes.
func AttachAnnotationsToDuckDB(ctx context.Context, conn *sql.Conn, outputDir string) error {
	// Install and load the sqlite_scanner extension (built into DuckDB).
	if _, err := conn.ExecContext(ctx, "INSTALL sqlite_scanner"); err != nil {
		return fmt.Errorf("installing sqlite_scanner: %w", err)
	}
	if _, err := conn.ExecContext(ctx, "LOAD sqlite_scanner"); err != nil {
		return fmt.Errorf("loading sqlite_scanner: %w", err)
	}

	// Resolve output directory to absolute path.
	absDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("resolving output dir: %w", err)
	}
	dbPath := filepath.Join(absDir, "annotations.db")

	// If annotations.db doesn't exist yet, skip attachment.
	// The DB is created on first write via the Store.
	if _, err := os.Stat(dbPath); err != nil {
		return nil
	}

	// Attach the SQLite DB to DuckDB.
	// overwrite=true allows re-attaching if already open.
	// Named parameter syntax is required: overwrite => true, not positional.
	if _, err := conn.ExecContext(ctx,
		"CALL sqlite_attach($1, overwrite => true)",
		dbPath,
	); err != nil {
		return fmt.Errorf("attaching annotations.db: %w", err)
	}

	return nil
}
