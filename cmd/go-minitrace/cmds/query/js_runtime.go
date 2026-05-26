package query

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/dop251/goja"
	noderequire "github.com/dop251/goja_nodejs/require"
	fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	glazedvalues "github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
	"github.com/go-go-golems/go-minitrace/pkg/minitracejs"
	queryengine "github.com/go-go-golems/go-minitrace/pkg/query"
)

func RunJSCommandIntoProcessor(
	ctx context.Context,
	catalog *minitracecmd.Catalog,
	command *minitracecmd.MinitraceCommand,
	runtimeSettings *MinitraceQueryRuntimeSettings,
	vals *glazedvalues.Values,
	overrides map[string]any,
	conn *sql.Conn,
	gp middlewares.Processor,
) error {
	if command == nil || command.JS == nil {
		return fmt.Errorf("js command metadata is missing")
	}
	if strings.TrimSpace(command.JS.OutputMode) == jsverbs.OutputModeText {
		return fmt.Errorf("js text output mode is not supported in minitrace query commands yet")
	}
	if catalog == nil {
		return fmt.Errorf("catalog is nil")
	}
	root, ok := catalog.SourceRoots[command.SourceRoot]
	if !ok {
		return fmt.Errorf("source root %q not found for js command %s", command.SourceRoot, command.Name)
	}

	registry, err := jsverbs.ScanFS(root.FS, root.RootDir)
	if err != nil {
		return err
	}
	verb, err := findRegistryVerb(registry, command)
	if err != nil {
		return err
	}
	parsedValues, err := valuesWithOverrides(vals, command.Schema, overrides)
	if err != nil {
		return err
	}

	factory, err := gggengine.NewBuilder().
		WithRequireOptions(noderequire.WithLoader(registry.RequireLoader())).
		UseModuleMiddleware(gggengine.Pipeline()).
		WithModules(
			gggengine.NativeModuleSpec{
				ModuleID:   "minitrace-runtime",
				ModuleName: minitracejs.ModuleName,
				Loader: minitracejs.NewLoader(ctx, conn, command.Name, minitracejs.RuntimeSettings{
					ArchiveGlob:   runtimeSettings.ArchiveGlob,
					DBPath:        runtimeSettings.DBPath,
					TableName:     runtimeSettings.TableName,
					PersistLoaded: runtimeSettings.PersistLoaded,
				}),
			},
		).Build()
	if err != nil {
		return err
	}

	runtime, err := factory.NewRuntime(gggengine.WithStartupContext(ctx), gggengine.WithLifetimeContext(ctx))
	if err != nil {
		return err
	}
	defer func() { _ = runtime.Close(context.Background()) }()

	result, err := registry.InvokeInRuntime(ctx, runtime, verb, parsedValues)
	if err != nil {
		return err
	}
	return emitJSResult(ctx, gp, result)
}

func findRegistryVerb(registry *jsverbs.Registry, command *minitracecmd.MinitraceCommand) (*jsverbs.VerbSpec, error) {
	if registry == nil {
		return nil, fmt.Errorf("js registry is nil")
	}
	for _, verb := range registry.Verbs() {
		if verb == nil || verb.File == nil {
			continue
		}
		if verb.File.ModulePath == command.JS.ModulePath && verb.FunctionName == command.JS.FunctionName {
			return verb, nil
		}
	}
	return nil, fmt.Errorf("js verb %s#%s not found in scanned registry", command.JS.ModulePath, command.JS.FunctionName)
}

func valuesWithOverrides(vals *glazedvalues.Values, commandSchema *schema.Schema, overrides map[string]any) (*glazedvalues.Values, error) {
	if vals == nil {
		vals = glazedvalues.New()
	}
	if commandSchema == nil || len(overrides) == 0 {
		return vals.Clone(), nil
	}

	ret := vals.Clone()
	sectionsByField := map[string]schema.Section{}
	commandSchema.ForEach(func(_ string, section schema.Section) {
		if section == nil {
			return
		}
		section.GetDefinitions().ForEach(func(def *fields.Definition) {
			if def == nil {
				return
			}
			if _, ok := sectionsByField[def.Name]; ok {
				return
			}
			sectionsByField[def.Name] = section
		})
	})

	for name, value := range overrides {
		section, ok := sectionsByField[name]
		if !ok || section == nil {
			continue
		}
		definition, ok := section.GetDefinitions().Get(name)
		if !ok || definition == nil {
			continue
		}
		sectionValues := ret.GetOrCreate(section)
		fieldValue, ok := sectionValues.Fields.Get(name)
		if !ok || fieldValue == nil {
			fieldValue = &fields.FieldValue{Definition: definition.Clone()}
			sectionValues.Fields.Set(name, fieldValue)
		}
		if err := fieldValue.Update(value); err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func minitraceModuleLoader(
	ctx context.Context,
	conn *sql.Conn,
	command *minitracecmd.MinitraceCommand,
	runtimeSettings *MinitraceQueryRuntimeSettings,
) noderequire.ModuleLoader {
	return func(vm *goja.Runtime, moduleObj *goja.Object) {
		exports := moduleObj.Get("exports").(*goja.Object)
		_ = exports.Set("query", func(sqlText string, args ...any) ([]map[string]any, error) {
			return minitraceQuery(ctx, conn, sqlText, args...)
		})
		_ = exports.Set("queryOne", func(sqlText string, args ...any) (map[string]any, error) {
			rows, err := minitraceQuery(ctx, conn, sqlText, args...)
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
		_ = runtimeObj.Set("commandName", command.Name)
		_ = exports.Set("runtime", runtimeObj)

		sqlObj := vm.NewObject()
		_ = sqlObj.Set("string", func(value any) (string, error) { return jsSQLString(value) })
		_ = sqlObj.Set("stringIn", func(value any) (string, error) { return jsSQLStringIn(value) })
		_ = sqlObj.Set("like", func(value any) (string, error) { return jsSQLLike(value) })
		_ = exports.Set("sql", sqlObj)
	}
}

func minitraceQuery(ctx context.Context, conn *sql.Conn, sqlText string, args ...any) ([]map[string]any, error) {
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

func emitJSResult(ctx context.Context, gp middlewares.Processor, result any) error {
	if result == nil {
		return nil
	}
	if row, ok := toRow(result); ok {
		return gp.AddRow(ctx, row)
	}
	value := reflect.ValueOf(result)
	if value.IsValid() && (value.Kind() == reflect.Slice || value.Kind() == reflect.Array) {
		for i := 0; i < value.Len(); i++ {
			item := value.Index(i).Interface()
			if row, ok := toRow(item); ok {
				if err := gp.AddRow(ctx, row); err != nil {
					return err
				}
				continue
			}
			if err := gp.AddRow(ctx, types.NewRow(types.MRP("value", item))); err != nil {
				return err
			}
		}
		return nil
	}
	return gp.AddRow(ctx, types.NewRow(types.MRP("value", result)))
}

func toRow(value any) (types.Row, bool) {
	switch v := value.(type) {
	case map[string]any:
		return types.NewRowFromMap(v), true
	case map[string]string:
		row := types.NewRow()
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			row.Set(key, v[key])
		}
		return row, true
	default:
		rv := reflect.ValueOf(value)
		if !rv.IsValid() || rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
			return nil, false
		}
		row := types.NewRow()
		keys := []string{}
		valuesByKey := map[string]any{}
		iter := rv.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			keys = append(keys, key)
			valuesByKey[key] = iter.Value().Interface()
		}
		sort.Strings(keys)
		for _, key := range keys {
			row.Set(key, valuesByKey[key])
		}
		return row, true
	}
}

func jsSQLString(value any) (string, error) {
	s, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("sql.string expects string, got %T", value)
	}
	return "'" + strings.ReplaceAll(s, "'", "''") + "'", nil
}

func jsSQLLike(value any) (string, error) {
	s, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("sql.like expects string, got %T", value)
	}
	return "'%" + strings.ReplaceAll(s, "'", "''") + "%'", nil
}

func jsSQLStringIn(value any) (string, error) {
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
