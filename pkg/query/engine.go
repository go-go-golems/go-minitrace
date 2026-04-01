package query

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
)

type LoadOptions struct {
	ArchiveGlob   string
	TableName     string
	PersistLoaded bool
}

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func OpenConnection(ctx context.Context, dbPath string) (*sql.DB, *sql.Conn, error) {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("opening duckdb database %q: %w", dbPath, err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	conn, err := db.Conn(ctx)
	if err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("opening duckdb connection: %w", err)
	}
	return db, conn, nil
}

func LoadArchive(ctx context.Context, conn *sql.Conn, opts LoadOptions) error {
	if err := ValidateIdentifier(opts.TableName); err != nil {
		return err
	}
	if strings.TrimSpace(opts.ArchiveGlob) == "" {
		return fmt.Errorf("archive glob is required")
	}

	loadSQL := BuildLoadSQL(opts)
	if _, err := conn.ExecContext(ctx, loadSQL); err != nil {
		return fmt.Errorf("loading archive into duckdb: %w", err)
	}
	return nil
}

func BuildLoadSQL(opts LoadOptions) string {
	tableKeyword := "TEMP TABLE"
	if opts.PersistLoaded {
		tableKeyword = "TABLE"
	}

	return fmt.Sprintf(`
CREATE OR REPLACE %s %s AS
SELECT *
FROM read_json(
  %s,
  columns = {
    id: 'VARCHAR',
    title: 'VARCHAR',
    summary: 'VARCHAR',
    classification: 'VARCHAR',
    profile: 'VARCHAR',
    provenance: 'JSON',
    flags: 'JSON',
    environment: 'JSON',
    operational_context: 'JSON',
    timing: 'JSON',
    turns: 'JSON[]',
    tool_calls: 'JSON[]',
    annotations: 'JSON[]',
    metrics: 'JSON'
  },
  ignore_errors = true
);
`, tableKeyword, opts.TableName, quoteSQLString(opts.ArchiveGlob))
}

func ResolveSQL(presetName string, inlineSQL string, sqlFile string, tableName string) (string, error) {
	if err := ValidateIdentifier(tableName); err != nil {
		return "", err
	}

	if strings.TrimSpace(presetName) != "" {
		return ResolvePresetSQL(strings.TrimSpace(presetName), tableName)
	}
	if strings.TrimSpace(inlineSQL) != "" {
		return inlineSQL, nil
	}
	if strings.TrimSpace(sqlFile) != "" {
		payload, err := os.ReadFile(sqlFile)
		if err != nil {
			return "", fmt.Errorf("reading sql file %q: %w", sqlFile, err)
		}
		return string(payload), nil
	}
	return "", fmt.Errorf("no query source specified")
}

func RunIntoProcessor(ctx context.Context, conn *sql.Conn, sqlText string, gp middlewares.Processor) error {
	rows, err := conn.QueryContext(ctx, sqlText)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("reading query columns: %w", err)
	}

	for rows.Next() {
		values := make([]any, len(columns))
		scanArgs := make([]any, len(columns))
		for i := range scanArgs {
			scanArgs[i] = &values[i]
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return fmt.Errorf("scanning query row: %w", err)
		}

		rowParts := make([]types.MapRowPair, 0, len(columns))
		for i, column := range columns {
			rowParts = append(rowParts, types.MRP(column, NormalizeValue(values[i])))
		}
		if err := gp.AddRow(ctx, types.NewRow(rowParts...)); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating query rows: %w", err)
	}
	return nil
}

func NormalizeValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(typed)
	case time.Time:
		return typed.UTC().Format(time.RFC3339Nano)
	default:
		return typed
	}
}

func ValidateIdentifier(identifier string) error {
	if !identifierPattern.MatchString(identifier) {
		return fmt.Errorf("invalid identifier %q", identifier)
	}
	return nil
}

func quoteSQLString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
