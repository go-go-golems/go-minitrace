package query

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/duckdb/duckdb-go/v2"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
)

type LoadOptions struct {
	ArchiveGlobs  []string
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
	resolvedFiles, err := ExpandArchiveGlobs(opts.ArchiveGlobs)
	if err != nil {
		return err
	}

	loadSQL := buildLoadSQLFromFiles(opts.TableName, opts.PersistLoaded, resolvedFiles)
	if _, err := conn.ExecContext(ctx, loadSQL); err != nil {
		return fmt.Errorf("loading archive into duckdb: %w", err)
	}
	return nil
}

func BuildLoadSQL(opts LoadOptions) string {
	resolvedFiles, err := ExpandArchiveGlobs(opts.ArchiveGlobs)
	if err != nil {
		panic(err)
	}
	return buildLoadSQLFromFiles(opts.TableName, opts.PersistLoaded, resolvedFiles)
}

func buildLoadSQLFromFiles(tableName string, persistLoaded bool, resolvedFiles []string) string {
	tableKeyword := "TEMP TABLE"
	if persistLoaded {
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
`, tableKeyword, tableName, quoteSQLStringList(resolvedFiles))
}

func ExpandArchiveGlobs(archiveGlobs []string) ([]string, error) {
	normalizedGlobs := normalizeStringList(archiveGlobs)
	if len(normalizedGlobs) == 0 {
		return nil, fmt.Errorf("archive glob is required")
	}

	resolvedFiles := make([]string, 0)
	seenFiles := make(map[string]struct{})

	for _, archiveGlob := range normalizedGlobs {
		files, err := filepath.Glob(archiveGlob)
		if err != nil {
			return nil, fmt.Errorf("expanding archive glob %q: %w", archiveGlob, err)
		}
		for _, filePath := range files {
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return nil, fmt.Errorf("resolving absolute path for %s: %w", filePath, err)
			}
			if _, ok := seenFiles[absPath]; ok {
				continue
			}
			seenFiles[absPath] = struct{}{}
			resolvedFiles = append(resolvedFiles, absPath)
		}
	}

	if len(resolvedFiles) == 0 {
		return nil, fmt.Errorf("archive globs matched no files: %s", strings.Join(normalizedGlobs, ", "))
	}

	return resolvedFiles, nil
}

func normalizeStringList(values []string) []string {
	ret := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ret = append(ret, value)
	}
	return ret
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
	case *big.Int:
		// DuckDB SUM() over integers returns *big.Int, which Goja maps to JS BigInt.
		// BigInt cannot participate in arithmetic with JS Number, so convert to int64.
		// This is safe for session metrics (counts, sums) which never exceed int64 range.
		if typed.IsInt64() {
			return typed.Int64()
		}
		// Fallback for truly huge values: convert to float64 (loses precision above 2^53)
		r, _ := new(big.Float).SetInt(typed).Float64()
		return r
	case duckdb.Decimal:
		return typed.Float64()
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

func quoteSQLStringList(values []string) string {
	if len(values) == 1 {
		return quoteSQLString(values[0])
	}

	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, quoteSQLString(value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
