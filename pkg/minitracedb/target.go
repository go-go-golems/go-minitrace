package minitracedb

import (
	"context"
	"database/sql"
)

// QueryTarget is the single query seam over the normalized minitrace SQLite
// database. Every consumer (JS runtime, SQL command files, `query run`, serve)
// executes read-only SQL through this interface, so all paths share the same
// sandbox, limits, and structured error envelope.
type QueryTarget interface {
	Query(ctx context.Context, sqlText string, args ...any) ([]map[string]any, error)
	QueryOne(ctx context.Context, sqlText string, args ...any) (map[string]any, error)
	QueryResult(ctx context.Context, sqlText string, args ...any) (QueryResult, error)
	Schema() SchemaDescriptor
	Tables() []TableDescriptor
	Close() error
}

// dbQueryTarget adapts an already-open *sql.DB (typically a read-only cached
// normalized database, possibly with auxiliary databases attached) to the
// QueryTarget interface using the standard sandboxed QueryRunner.
type dbQueryTarget struct {
	db     *sql.DB
	runner *QueryRunner
	schema SchemaDescriptor
}

// NewDBQueryTarget wraps db in the sandboxed QueryRunner and exposes it as a
// QueryTarget. Closing the target closes db.
func NewDBQueryTarget(db *sql.DB, allowedObjects []string, opts QueryOptions) (QueryTarget, error) {
	runner, err := NewQueryRunner(db, allowedObjects, opts)
	if err != nil {
		return nil, err
	}
	return &dbQueryTarget{db: db, runner: runner, schema: Schema()}, nil
}

func (t *dbQueryTarget) Query(ctx context.Context, sqlText string, args ...any) ([]map[string]any, error) {
	return t.runner.Query(ctx, sqlText, args...)
}

func (t *dbQueryTarget) QueryOne(ctx context.Context, sqlText string, args ...any) (map[string]any, error) {
	return t.runner.QueryOne(ctx, sqlText, args...)
}

func (t *dbQueryTarget) QueryResult(ctx context.Context, sqlText string, args ...any) (QueryResult, error) {
	return t.runner.QueryResult(ctx, sqlText, args...)
}

func (t *dbQueryTarget) Schema() SchemaDescriptor { return t.schema }

func (t *dbQueryTarget) Tables() []TableDescriptor {
	return append([]TableDescriptor(nil), t.schema.Tables...)
}

func (t *dbQueryTarget) Close() error {
	if t == nil || t.db == nil {
		return nil
	}
	return t.db.Close()
}
