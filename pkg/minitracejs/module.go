package minitracejs

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	queryengine "github.com/go-go-golems/go-minitrace/pkg/query"
)

const ModuleName = "minitrace"

type RuntimeSettings struct {
	ArchiveGlob   []string `json:"archiveGlob,omitempty"`
	DBPath        string   `json:"dbPath,omitempty"`
	TableName     string   `json:"tableName,omitempty"`
	PersistLoaded bool     `json:"persistLoaded,omitempty"`
}

func NewLoader(ctx context.Context, conn *sql.Conn, commandName string, runtimeSettings RuntimeSettings) require.ModuleLoader {
	if ctx == nil {
		ctx = context.Background()
	}
	return func(vm *goja.Runtime, moduleObj *goja.Object) {
		exports := moduleObj.Get("exports").(*goja.Object)
		_ = exports.Set("query", func(sqlText string, args ...any) ([]map[string]any, error) {
			return Query(ctx, conn, sqlText, args...)
		})
		_ = exports.Set("queryOne", func(sqlText string, args ...any) (map[string]any, error) {
			rows, err := Query(ctx, conn, sqlText, args...)
			if err != nil {
				return nil, err
			}
			if len(rows) == 0 {
				return nil, nil
			}
			return rows[0], nil
		})
		_ = exports.Set("tableName", runtimeSettings.TableName)

		runtimeObj := vm.NewObject()
		_ = runtimeObj.Set("tableName", runtimeSettings.TableName)
		_ = runtimeObj.Set("dbPath", runtimeSettings.DBPath)
		_ = runtimeObj.Set("archiveGlob", append([]string(nil), runtimeSettings.ArchiveGlob...))
		_ = runtimeObj.Set("persistLoaded", runtimeSettings.PersistLoaded)
		_ = runtimeObj.Set("commandName", commandName)
		_ = exports.Set("runtime", runtimeObj)

		sqlObj := vm.NewObject()
		_ = sqlObj.Set("string", func(value any) (string, error) { return SQLString(value) })
		_ = sqlObj.Set("stringIn", func(value any) (string, error) { return SQLStringIn(value) })
		_ = sqlObj.Set("like", func(value any) (string, error) { return SQLLike(value) })
		_ = exports.Set("sql", sqlObj)
	}
}

func Query(ctx context.Context, conn *sql.Conn, sqlText string, args ...any) ([]map[string]any, error) {
	if conn == nil {
		return nil, fmt.Errorf("minitrace query connection is nil")
	}
	if err := queryengine.ValidateReadOnlyQuery(sqlText); err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx, sqlText, flattenArgs(args)...)
	if err != nil {
		return nil, fmt.Errorf("executing js query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("reading js query columns: %w", err)
	}

	ret := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(columns))
		scanArgs := make([]any, len(columns))
		for i := range scanArgs {
			scanArgs[i] = &values[i]
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("scanning js query row: %w", err)
		}
		row := make(map[string]any, len(columns))
		for i, column := range columns {
			row[column] = queryengine.NormalizeValue(values[i])
		}
		ret = append(ret, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating js query rows: %w", err)
	}
	return ret, nil
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

func SQLString(value any) (string, error) {
	s, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("sql.string expects string, got %T", value)
	}
	return "'" + strings.ReplaceAll(s, "'", "''") + "'", nil
}

func SQLLike(value any) (string, error) {
	s, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("sql.like expects string, got %T", value)
	}
	return "'%" + strings.ReplaceAll(s, "'", "''") + "%'", nil
}

func SQLStringIn(value any) (string, error) {
	switch typed := value.(type) {
	case nil:
		return "", nil
	case []string:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, "'"+strings.ReplaceAll(item, "'", "''")+"'")
		}
		return strings.Join(parts, ", "), nil
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			s, ok := item.(string)
			if !ok {
				return "", fmt.Errorf("sql.stringIn expects string items, got %T", item)
			}
			parts = append(parts, "'"+strings.ReplaceAll(s, "'", "''")+"'")
		}
		return strings.Join(parts, ", "), nil
	case string:
		return "'" + strings.ReplaceAll(typed, "'", "''") + "'", nil
	default:
		return "", fmt.Errorf("sql.stringIn expects string or []string, got %T", value)
	}
}
