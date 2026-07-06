package query

import (
	"context"
	"fmt"
	"os"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/layout"
	glazedvalues "github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
	"github.com/go-go-golems/go-minitrace/pkg/minitracejs"
	"github.com/pkg/errors"
)

// SQLCommandTableName is the {{TABLE_NAME}} substitution used when rendering
// SQL command files. It points at the sessions_base compatibility view so
// legacy session-level command files keep working on the normalized SQLite
// database; new command files should reference the normalized tables directly.
const SQLCommandTableName = minitracedb.SessionsBaseCompatView

type MinitraceQueryRuntimeSettings struct {
	ArchiveGlob   []string `glazed:"archive-glob"`
	DBPath        string   `glazed:"db-path"`
	TableName     string   `glazed:"table-name"`
	PersistLoaded bool     `glazed:"persist-loaded"`
}

type MinitraceCatalogGlazeCommand struct {
	*cmds.CommandDescription
	command *minitracecmd.MinitraceCommand
	catalog *minitracecmd.Catalog
}

var _ cmds.GlazeCommand = &MinitraceCatalogGlazeCommand{}

func NewMinitraceCatalogGlazeCommand(command *minitracecmd.MinitraceCommand, catalog *minitracecmd.Catalog) (*MinitraceCatalogGlazeCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}
	queryRuntimeSection, err := NewQueryRuntimeSection()
	if err != nil {
		return nil, err
	}

	options := []cmds.CommandDescriptionOption{
		cmds.WithShort(command.Short),
		cmds.WithLong(command.Long),
		cmds.WithTags(command.Tags...),
		cmds.WithMetadata(command.Metadata),
		cmds.WithSections(glazedSection, commandSettingsSection, queryRuntimeSection),
	}
	if command.Schema != nil {
		options = append(options, cmds.WithSchema(command.Schema.Clone()))
	} else {
		options = append(options, cmds.WithFlags(command.Flags...), cmds.WithArguments(command.Arguments...))
	}
	if len(command.Layout) > 0 {
		options = append(options, cmds.WithLayout(&layout.Layout{Sections: command.Layout}))
	}

	desc := cmds.NewCommandDescription(command.Name, options...)
	return &MinitraceCatalogGlazeCommand{
		CommandDescription: desc,
		command:            command,
		catalog:            catalog,
	}, nil
}

func (c *MinitraceCatalogGlazeCommand) RunIntoGlazeProcessor(ctx context.Context, vals *glazedvalues.Values, gp middlewares.Processor) error {
	runtimeSettings := &MinitraceQueryRuntimeSettings{}
	if err := vals.DecodeSectionInto(QueryRuntimeSectionSlug, runtimeSettings); err != nil {
		return err
	}
	if len(runtimeSettings.ArchiveGlob) == 0 {
		return errors.New("archive-glob is required")
	}

	commandValues := collectCommandValues(vals, c.command)
	resolvedCommand, resolvedValues, err := minitracecmd.ResolveAliasCommand(c.catalog, c.command, commandValues)
	if err != nil {
		return err
	}

	switch resolvedCommand.Runtime {
	case minitracecmd.CommandRuntimeSQL, minitracecmd.CommandRuntimeUnknown:
		warnDeprecatedRuntimeSettings(runtimeSettings)
		sqlText, err := minitracecmd.RenderCommand(resolvedCommand, minitracecmd.RenderContext{
			TableName: SQLCommandTableName,
			Values:    resolvedValues,
		})
		if err != nil {
			return err
		}
		target, err := minitracejs.NewArchiveQueryTarget(ctx, runtimeSettings.ArchiveGlob, minitracedb.DefaultQueryOptions())
		if err != nil {
			return err
		}
		defer func() { _ = target.Close() }()
		return RunQueryTargetIntoProcessor(ctx, target, sqlText, gp)
	case minitracecmd.CommandRuntimeJS:
		return RunJSCommandIntoProcessor(ctx, c.catalog, resolvedCommand, runtimeSettings, vals, resolvedValues, gp)
	default:
		return errors.Errorf("unsupported command runtime %q", resolvedCommand.Runtime)
	}
}

// warnDeprecatedRuntimeSettings prints a one-line stderr warning when the
// legacy DuckDB runtime flags are explicitly set. SQL command files now run
// against the normalized SQLite database built from --archive-glob, so these
// flags no longer affect the SQL path (they are still passed to JS commands
// through mt.runtime).
func warnDeprecatedRuntimeSettings(runtimeSettings *MinitraceQueryRuntimeSettings) {
	if runtimeSettings == nil {
		return
	}
	if (runtimeSettings.DBPath != "" && runtimeSettings.DBPath != ":memory:") ||
		(runtimeSettings.TableName != "" && runtimeSettings.TableName != "sessions_base") ||
		runtimeSettings.PersistLoaded {
		fmt.Fprintln(os.Stderr, "Warning: --db-path, --table-name, and --persist-loaded are deprecated; SQL query commands now run against the normalized SQLite database built from --archive-glob.")
	}
}

// RunQueryTargetIntoProcessor executes sandboxed SQL on a minitracedb query
// target and emits the resulting rows into a Glazed processor, preserving the
// result column order.
func RunQueryTargetIntoProcessor(ctx context.Context, target minitracedb.QueryTarget, sqlText string, gp middlewares.Processor) error {
	result, err := target.QueryResult(ctx, sqlText)
	if err != nil {
		return err
	}
	if result.Error != "" {
		return errors.New(result.Error)
	}
	for _, row := range result.Rows {
		pairs := make([]types.MapRowPair, 0, len(result.Columns))
		for _, column := range result.Columns {
			pairs = append(pairs, types.MRP(column, row[column]))
		}
		if err := gp.AddRow(ctx, types.NewRow(pairs...)); err != nil {
			return err
		}
	}
	return nil
}

func collectCommandValues(vals *glazedvalues.Values, command *minitracecmd.MinitraceCommand) map[string]any {
	ret := map[string]any{}
	if vals == nil {
		return ret
	}

	if command != nil && command.Schema != nil {
		vals.ForEach(func(_ string, sectionValues *glazedvalues.SectionValues) {
			if sectionValues == nil {
				return
			}
			sectionValues.Fields.ForEach(func(_ string, fieldValue *fields.FieldValue) {
				if fieldValue == nil || fieldValue.Definition == nil {
					return
				}
				ret[fieldValue.Definition.Name] = fieldValue.Value
			})
		})
		return ret
	}

	defaultValues := vals.DefaultSectionValues()
	definitions := make([]*fields.Definition, 0, len(command.Flags)+len(command.Arguments))
	definitions = append(definitions, command.Flags...)
	definitions = append(definitions, command.Arguments...)
	for _, definition := range definitions {
		if definition == nil {
			continue
		}
		fieldValue, ok := defaultValues.Fields.Get(definition.Name)
		if !ok {
			continue
		}
		ret[definition.Name] = fieldValue.Value
	}
	return ret
}
