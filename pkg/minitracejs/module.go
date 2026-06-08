package minitracejs

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

const ModuleName = "minitrace"

type RuntimeSettings struct {
	ArchiveGlob   []string `json:"archiveGlob,omitempty"`
	DBPath        string   `json:"dbPath,omitempty"`
	TableName     string   `json:"tableName,omitempty"`
	PersistLoaded bool     `json:"persistLoaded,omitempty"`
}

func NewLoader(ctx context.Context, conn *sql.Conn, commandName string, runtimeSettings RuntimeSettings) require.ModuleLoader {
	_ = conn // Legacy host-table query access was removed; JS scripts should use mt.db().
	if ctx == nil {
		ctx = context.Background()
	}
	return func(vm *goja.Runtime, moduleObj *goja.Object) {
		exports := moduleObj.Get("exports").(*goja.Object)
		_ = exports.Set("importer", func() *goja.Object {
			return importBuilderObject(vm, NewImportBuilder())
		})
		_ = exports.Set("db", func() *goja.Object {
			return builderObject(vm, NewDBBuilderWithRuntime(ctx, runtimeSettings))
		})
		_ = exports.Set("sources", func() *goja.Object {
			return sourcesBuilderObject(vm, NewSourceSetBuilder())
		})
		_ = exports.Set("importPolicy", func() *goja.Object {
			return importPolicyBuilderObject(vm, NewImportPolicyBuilder())
		})
		_ = exports.Set("cache", func() *goja.Object {
			return cachePolicyBuilderObject(vm, NewCachePolicyBuilder())
		})
		_ = exports.Set("limits", func() *goja.Object {
			return queryLimitsBuilderObject(vm, NewQueryLimitsBuilder())
		})

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
