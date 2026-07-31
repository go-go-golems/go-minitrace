package minitracedb

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"strings"
	"time"
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
	db           *sql.DB
	allowedReads map[string]struct{}
	opts         QueryOptions
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

func NewQueryRunner(db *sql.DB, allowedReads []string, opts QueryOptions) (*QueryRunner, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	allowed := map[string]struct{}{}
	for _, object := range allowedReads {
		key := normalizeAllowedReadKey(object)
		if key != "" {
			allowed[key] = struct{}{}
		}
	}
	return &QueryRunner{db: db, allowedReads: allowed, opts: WithDefaultQueryOptions(opts)}, nil
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
	normalized, err := validateReadOnlySQLiteQuery(sqlText, r.opts)
	if err != nil {
		preview := strings.TrimSpace(sqlText)
		if len(preview) > 160 {
			preview = preview[:160]
		}
		if preview != "" {
			return QueryResult{Error: err.Error() + ": " + preview}, nil
		}
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

	authState := &queryAuthorizationState{}
	if err := setSQLiteAuthorizer(conn, newReadOnlyAuthorizer(r.allowedReads, authState)); err != nil {
		return QueryResult{Error: err.Error()}, nil
	}
	defer func() { _ = setSQLiteAuthorizer(conn, nil) }()

	if err := ensureReadonlyPreparedQuery(conn, normalized); err != nil {
		if authErr := authState.err(); authErr != nil {
			return QueryResult{Error: authErr.Error()}, nil
		}
		return QueryResult{Error: err.Error()}, nil
	}

	rows, err := conn.QueryContext(qctx, normalized, flattenArgs(args)...)
	if err != nil {
		if authErr := authState.err(); authErr != nil {
			return QueryResult{Error: authErr.Error()}, nil
		}
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
		if authErr := authState.err(); authErr != nil {
			return QueryResult{Error: authErr.Error()}, nil
		}
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

func validateReadOnlySQLiteQuery(sqlText string, opts QueryOptions) (string, error) {
	normalized, err := normalizeQuery(sqlText)
	if err != nil {
		return "", err
	}
	sanitized := stripSQLLiteralsAndComments(normalized)
	lower := strings.ToLower(strings.TrimSpace(sanitized))
	if !hasReadOnlyQueryPrefix(lower, "select") && !hasReadOnlyQueryPrefix(lower, "with") {
		return "", fmt.Errorf("only SELECT and WITH queries are allowed")
	}
	if opts.RequireOrderBy && strings.Contains(lower, " from ") && !strings.Contains(lower, " order by ") {
		return "", fmt.Errorf("query must include ORDER BY")
	}
	return normalized, nil
}

func hasReadOnlyQueryPrefix(lower, prefix string) bool {
	if lower == prefix {
		return true
	}
	if !strings.HasPrefix(lower, prefix) {
		return false
	}
	if len(lower) == len(prefix) {
		return true
	}
	return lower[len(prefix)] == ' ' || lower[len(prefix)] == '\n' || lower[len(prefix)] == '\t' || lower[len(prefix)] == '\r'
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

func normalizeObjectName(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func normalizeAllowedReadKey(v string) string {
	v = normalizeObjectName(v)
	if v == "" {
		return ""
	}
	if strings.Contains(v, ".") {
		parts := strings.SplitN(v, ".", 2)
		schema := strings.TrimSpace(parts[0])
		object := strings.TrimSpace(parts[1])
		if schema == "" || object == "" {
			return ""
		}
		return schema + "." + object
	}
	return "main." + v
}

func readKey(database, object string) string {
	database = normalizeObjectName(database)
	if database == "" {
		database = "main"
	}
	object = normalizeObjectName(object)
	if object == "" {
		return ""
	}
	return database + "." + object
}

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

type queryAuthorizationState struct {
	deniedOp        int
	deniedSchema    string
	deniedObject    string
	displayDeniedAs string
}

func (s *queryAuthorizationState) deny(op int, schema, object string) {
	if s == nil || s.deniedObject != "" {
		return
	}
	s.deniedOp = op
	s.deniedSchema = normalizeObjectName(schema)
	if s.deniedSchema == "" {
		s.deniedSchema = "main"
	}
	s.deniedObject = normalizeObjectName(object)
	if s.deniedObject == "" {
		s.deniedObject = normalizeObjectName(schema)
		s.deniedSchema = ""
	}
	if s.deniedSchema != "" && s.deniedSchema != "main" {
		s.displayDeniedAs = s.deniedSchema + "." + s.deniedObject
	} else {
		s.displayDeniedAs = s.deniedObject
	}
}

func (s *queryAuthorizationState) err() error {
	if s == nil || s.deniedObject == "" {
		return nil
	}
	display := s.displayDeniedAs
	if display == "" {
		display = s.deniedObject
	}
	if isDeniedReadOp(s.deniedOp) {
		if strings.HasPrefix(s.deniedObject, "sqlite_") {
			return fmt.Errorf("query references disallowed table/view %q; use db.schema() or db.tables() from JS to introspect the schema", display)
		}
		return fmt.Errorf("query references disallowed table/view %q", display)
	}
	return fmt.Errorf("query uses disallowed SQLite operation %d on %q", s.deniedOp, display)
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
