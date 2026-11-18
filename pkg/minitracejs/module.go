package minitracejs

import (
	"context"
	"database/sql"

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
		relation := newRelationalBinding(vm)
		_ = exports.Set("from", relation.from)
		_ = exports.Set("param", relation.parameterBuilders())
		_ = exports.Set("fragment", relation.newFragment)
		_ = exports.Set("native", func() *goja.Object { return nativeEvidenceBuilderObject(ctx, vm) })
		_ = exports.Set("corpusManifest", func(paths []string) map[string]any { return corpusManifest(vm, paths) })
		_ = exports.Set("utf8ByteLength", func(text string) int { return len(text) })
		_ = exports.Set("importer", func() *goja.Object {
			return importBuilderObject(vm, NewImportBuilder())
		})
		_ = exports.Set("db", func() *goja.Object {
			builder := NewDBBuilderWithRuntime(ctx, runtimeSettings)
			builder.relation = relation
			return builderObject(vm, builder)
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

		_ = exports.Set("type", relation.typeBuilders())
		_ = exports.Set("sql", relation.sql)
	}
}
