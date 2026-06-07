package minitracedb

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	sqlite3 "github.com/mattn/go-sqlite3"
)

type QueryOptions struct {
	MaxRows        int           `json:"maxRows"`
	MaxColumns     int           `json:"maxColumns"`
	MaxCellChars   int           `json:"maxCellChars"`
	Timeout        time.Duration `json:"timeout"`
	RequireOrderBy bool          `json:"requireOrderBy"`
}

type QueryResult struct {
	Columns   []string         `json:"columns"`
	Rows      []map[string]any `json:"rows"`
	Count     int              `json:"count"`
	Truncated bool             `json:"truncated,omitempty"`
	Error     string           `json:"error,omitempty"`
}

type QueryRunner struct {
	db             *sql.DB
	allowedObjects map[string]struct{}
	opts           QueryOptions
}

func DefaultQueryOptions() QueryOptions {
	return QueryOptions{MaxRows: 1000, MaxColumns: 128, MaxCellChars: 4000, Timeout: 5 * time.Second}
}

func WithDefaultQueryOptions(opts QueryOptions) QueryOptions {
	ret := opts
	def := DefaultQueryOptions()
	if ret.MaxRows <= 0 {
		ret.MaxRows = def.MaxRows
	}
	if ret.MaxColumns <= 0 {
		ret.MaxColumns = def.MaxColumns
	}
	if ret.MaxCellChars <= 0 {
		ret.MaxCellChars = def.MaxCellChars
	}
	if ret.Timeout <= 0 {
		ret.Timeout = def.Timeout
	}
	return ret
}

func NewQueryRunner(db *sql.DB, allowedObjects []string, opts QueryOptions) (*QueryRunner, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	allowed := map[string]struct{}{}
	for _, object := range allowedObjects {
		object = normalizeObjectName(object)
		if object != "" {
			allowed[object] = struct{}{}
		}
	}
	return &QueryRunner{db: db, allowedObjects: allowed, opts: WithDefaultQueryOptions(opts)}, nil
}

func (r *QueryRunner) Query(ctx context.Context, sqlText string, args ...any) ([]map[string]any, error) {
	result, err := r.QueryResult(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("%s", result.Error)
	}
	return result.Rows, nil
}

func (r *QueryRunner) QueryOne(ctx context.Context, sqlText string, args ...any) (map[string]any, error) {
	rows, err := r.Query(ctx, sqlText, args...)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0], nil
}

func (r *QueryRunner) QueryResult(ctx context.Context, sqlText string, args ...any) (QueryResult, error) {
	if r == nil || r.db == nil {
		return QueryResult{Error: "db is not initialized"}, nil
	}
	normalized, err := validateReadOnlySQLiteQuery(sqlText, r.allowedObjects, r.opts)
	if err != nil {
		return QueryResult{Error: err.Error()}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	qctx, cancel := context.WithTimeout(ctx, r.opts.Timeout)
	defer cancel()

	conn, err := r.db.Conn(qctx)
	if err != nil {
		return QueryResult{Error: err.Error()}, nil
	}
	defer func() { _ = conn.Close() }()

	if err := setSQLiteAuthorizer(conn, newReadOnlyAuthorizer()); err != nil {
		return QueryResult{Error: err.Error()}, nil
	}
	defer func() { _ = setSQLiteAuthorizer(conn, nil) }()

	if err := ensureReadonlyPreparedQuery(conn, normalized); err != nil {
		return QueryResult{Error: err.Error()}, nil
	}

	rows, err := conn.QueryContext(qctx, normalized, flattenArgs(args)...)
	if err != nil {
		return QueryResult{Error: err.Error()}, nil
	}
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	if err != nil {
		return QueryResult{Error: err.Error()}, nil
	}
	if len(cols) > r.opts.MaxColumns {
		return QueryResult{Error: fmt.Sprintf("query returns %d columns; max is %d", len(cols), r.opts.MaxColumns)}, nil
	}
	out := QueryResult{Columns: cols, Rows: make([]map[string]any, 0)}
	for rows.Next() {
		if out.Count >= r.opts.MaxRows {
			out.Truncated = true
			break
		}
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return QueryResult{Error: err.Error()}, nil
		}
		row := map[string]any{}
		for i, col := range cols {
			row[col] = NormalizeCell(values[i], r.opts.MaxCellChars)
		}
		out.Rows = append(out.Rows, row)
		out.Count++
	}
	if err := rows.Err(); err != nil {
		return QueryResult{Error: err.Error()}, nil
	}
	return out, nil
}

func NormalizeCell(value any, maxChars int) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []byte:
		return truncateString(string(typed), maxChars)
	case string:
		return truncateString(typed, maxChars)
	case time.Time:
		return typed.UTC().Format(time.RFC3339Nano)
	case *big.Int:
		if typed.IsInt64() {
			return typed.Int64()
		}
		f, _ := new(big.Float).SetInt(typed).Float64()
		return f
	default:
		return typed
	}
}

func truncateString(s string, maxChars int) string {
	if maxChars > 0 && len(s) > maxChars {
		return s[:maxChars]
	}
	return s
}

func validateReadOnlySQLiteQuery(sqlText string, allowedObjects map[string]struct{}, opts QueryOptions) (string, error) {
	normalized, err := normalizeQuery(sqlText)
	if err != nil {
		return "", err
	}
	sanitized := stripSQLLiteralsAndComments(normalized)
	lower := strings.ToLower(strings.TrimSpace(sanitized))
	if !strings.HasPrefix(lower, "select ") && !strings.HasPrefix(lower, "with ") {
		return "", fmt.Errorf("only SELECT and WITH queries are allowed")
	}
	if opts.RequireOrderBy && strings.Contains(lower, " from ") && !strings.Contains(lower, " order by ") {
		return "", fmt.Errorf("query must include ORDER BY")
	}
	if len(allowedObjects) > 0 {
		for _, object := range referencedObjects(sanitized) {
			if _, ok := allowedObjects[normalizeObjectName(object)]; !ok {
				return "", fmt.Errorf("query references disallowed table/view %q", object)
			}
		}
	}
	return normalized, nil
}

func normalizeQuery(sqlText string) (string, error) {
	trimmed := strings.TrimSpace(sqlText)
	if trimmed == "" {
		return "", fmt.Errorf("sql is required")
	}
	if strings.HasSuffix(trimmed, ";") {
		trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, ";"))
	}
	sanitized := stripSQLLiteralsAndComments(trimmed)
	if strings.Contains(sanitized, ";") {
		return "", fmt.Errorf("multiple SQL statements are not allowed")
	}
	if strings.TrimSpace(trimmed) == "" {
		return "", fmt.Errorf("sql is required")
	}
	return trimmed, nil
}

var fromJoinObjectRe = regexp.MustCompile(`(?i)\b(from|join)\s+([a-zA-Z_][a-zA-Z0-9_\.]*)`)

func referencedObjects(sqlText string) []string {
	matches := fromJoinObjectRe.FindAllStringSubmatch(sqlText, -1)
	seen := map[string]struct{}{}
	var out []string
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		object := normalizeExtractedObject(match[2])
		if object == "" {
			continue
		}
		if _, ok := seen[object]; ok {
			continue
		}
		seen[object] = struct{}{}
		out = append(out, object)
	}
	return out
}

func normalizeExtractedObject(v string) string {
	v = strings.Trim(v, "`\"[] ")
	if dot := strings.LastIndex(v, "."); dot >= 0 && dot+1 < len(v) {
		v = v[dot+1:]
	}
	return v
}

func normalizeObjectName(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func stripSQLLiteralsAndComments(sqlText string) string {
	const (
		stateNormal = iota
		stateSingleQuote
		stateDoubleQuote
		stateBacktickQuote
		stateBracketQuote
		stateLineComment
		stateBlockComment
	)
	var b strings.Builder
	b.Grow(len(sqlText))
	state := stateNormal
	for i := 0; i < len(sqlText); i++ {
		ch := sqlText[i]
		next := byte(0)
		if i+1 < len(sqlText) {
			next = sqlText[i+1]
		}
		switch state {
		case stateNormal:
			switch {
			case ch == '\'':
				state = stateSingleQuote
				b.WriteByte(' ')
			case ch == '"':
				state = stateDoubleQuote
				b.WriteByte(' ')
			case ch == '`':
				state = stateBacktickQuote
				b.WriteByte(' ')
			case ch == '[':
				state = stateBracketQuote
				b.WriteByte(' ')
			case ch == '-' && next == '-':
				state = stateLineComment
				b.WriteString("  ")
				i++
			case ch == '/' && next == '*':
				state = stateBlockComment
				b.WriteString("  ")
				i++
			default:
				b.WriteByte(ch)
			}
		case stateSingleQuote:
			if ch == '\'' && next == '\'' {
				b.WriteString("  ")
				i++
				continue
			}
			if ch == '\'' {
				state = stateNormal
			}
			b.WriteByte(' ')
		case stateDoubleQuote:
			if ch == '"' && next == '"' {
				b.WriteString("  ")
				i++
				continue
			}
			if ch == '"' {
				state = stateNormal
			}
			b.WriteByte(' ')
		case stateBacktickQuote:
			if ch == '`' {
				state = stateNormal
			}
			b.WriteByte(' ')
		case stateBracketQuote:
			if ch == ']' {
				state = stateNormal
			}
			b.WriteByte(' ')
		case stateLineComment:
			if ch == '\n' {
				state = stateNormal
				b.WriteByte('\n')
			} else {
				b.WriteByte(' ')
			}
		case stateBlockComment:
			if ch == '*' && next == '/' {
				state = stateNormal
				b.WriteString("  ")
				i++
			} else if ch == '\n' {
				b.WriteByte('\n')
			} else {
				b.WriteByte(' ')
			}
		}
	}
	return b.String()
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

func newReadOnlyAuthorizer() func(int, string, string, string) int {
	return func(op int, _, _, _ string) int {
		switch op {
		case sqlite3.SQLITE_SELECT, sqlite3.SQLITE_READ, sqlite3.SQLITE_FUNCTION:
			return sqlite3.SQLITE_OK
		case sqlite3.SQLITE_INSERT, sqlite3.SQLITE_UPDATE, sqlite3.SQLITE_DELETE, sqlite3.SQLITE_PRAGMA, sqlite3.SQLITE_ATTACH, sqlite3.SQLITE_DETACH, sqlite3.SQLITE_TRANSACTION, sqlite3.SQLITE_CREATE_INDEX, sqlite3.SQLITE_CREATE_TABLE, sqlite3.SQLITE_CREATE_TEMP_INDEX, sqlite3.SQLITE_CREATE_TEMP_TABLE, sqlite3.SQLITE_CREATE_TEMP_TRIGGER, sqlite3.SQLITE_CREATE_TEMP_VIEW, sqlite3.SQLITE_CREATE_TRIGGER, sqlite3.SQLITE_CREATE_VIEW, sqlite3.SQLITE_CREATE_VTABLE, sqlite3.SQLITE_DROP_INDEX, sqlite3.SQLITE_DROP_TABLE, sqlite3.SQLITE_DROP_TEMP_INDEX, sqlite3.SQLITE_DROP_TEMP_TABLE, sqlite3.SQLITE_DROP_TEMP_TRIGGER, sqlite3.SQLITE_DROP_TEMP_VIEW, sqlite3.SQLITE_DROP_TRIGGER, sqlite3.SQLITE_DROP_VIEW, sqlite3.SQLITE_DROP_VTABLE:
			return sqlite3.SQLITE_DENY
		default:
			return sqlite3.SQLITE_OK
		}
	}
}

func flattenArgs(args []any) []any {
	ret := make([]any, 0, len(args))
	for _, arg := range args {
		if slice, ok := arg.([]any); ok {
			ret = append(ret, slice...)
			continue
		}
		ret = append(ret, arg)
	}
	return ret
}
