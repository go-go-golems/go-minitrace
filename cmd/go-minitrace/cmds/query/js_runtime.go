package query

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"strings"

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
