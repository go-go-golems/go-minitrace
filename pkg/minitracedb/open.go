package minitracedb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	sqlite3 "github.com/mattn/go-sqlite3"
)

func OpenSQLiteMemory(ctx context.Context, prefix string) (*sql.DB, error) {
	name, err := uniqueMemoryName(prefix)
	if err != nil {
		return nil, err
	}
	dsn := "file:" + name + "?mode=memory&cache=shared"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite memory: %w", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys=ON;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
	}
	return db, nil
}

func OpenSQLiteFile(ctx context.Context, path string) (*sql.DB, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("sqlite path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite file: %w", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys=ON;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
	}
	return db, nil
}

func OpenSQLiteReadOnly(ctx context.Context, path string) (*sql.DB, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("sqlite path is required")
	}
	dsn := "file:" + filepath.ToSlash(path) + "?mode=ro&immutable=1"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite read-only: %w", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite read-only: %w", err)
	}
	return db, nil
}

var attachDriverSeq atomic.Int64

// OpenSQLiteReadOnlyAttached opens the SQLite database at path read-only and
// ATTACHes each auxiliary database under its schema name on every pooled
// connection (via a per-call driver ConnectHook, so the attachment survives
// connection churn). The attach happens at connection setup time, before the
// sandbox authorizer is installed by QueryRunner, so read-only queries can
// reference the attached tables (e.g. anno.annotations) live.
//
// The main database is opened with mode=ro (attached databases inherit the
// connection's read-only mode), and deliberately without immutable=1 so that
// live auxiliary databases (WAL-mode annotation stores) read correctly.
func OpenSQLiteReadOnlyAttached(ctx context.Context, path string, attachments map[string]string) (*sql.DB, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("sqlite path is required")
	}
	type attachment struct {
		schema string
		path   string
	}
	attachList := make([]attachment, 0, len(attachments))
	for schema, dbPath := range attachments {
		schema = strings.TrimSpace(schema)
		dbPath = strings.TrimSpace(dbPath)
		if schema == "" || dbPath == "" {
			return nil, fmt.Errorf("attachment schema and path are required")
		}
		attachList = append(attachList, attachment{schema: schema, path: dbPath})
	}

	driverName := fmt.Sprintf("sqlite3_minitrace_attach_%d", attachDriverSeq.Add(1))
	sql.Register(driverName, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			for _, att := range attachList {
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

	dsn := "file:" + filepath.ToSlash(path) + "?mode=ro"
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite read-only with attachments: %w", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite read-only with attachments: %w", err)
	}
	return db, nil
}

func uniqueMemoryName(prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "minitrace"
	}
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("randomize sqlite memory name: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(b), nil
}
