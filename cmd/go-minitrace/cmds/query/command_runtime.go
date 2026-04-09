package query

import (
	"context"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/layout"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	glazedvalues "github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
	queryengine "github.com/go-go-golems/go-minitrace/pkg/query"
	"github.com/pkg/errors"
)

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

	flags := []*fields.Definition{
		fields.New("archive-glob", fields.TypeStringList, fields.WithDefault([]string{"./output/active/*/*.minitrace.json"}), fields.WithHelp("Repeatable glob flag for minitrace session JSON files to load")),
		fields.New("db-path", fields.TypeString, fields.WithDefault(":memory:"), fields.WithHelp("DuckDB database path to use; :memory: keeps the query session ephemeral")),
		fields.New("table-name", fields.TypeString, fields.WithDefault("sessions_base"), fields.WithHelp("Table name to create from the loaded archive")),
		fields.New("persist-loaded", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Request a persistent loaded table rather than a temp table")),
	}
	flags = append(flags, command.Flags...)

	options := []cmds.CommandDescriptionOption{
		cmds.WithShort(command.Short),
		cmds.WithLong(command.Long),
		cmds.WithFlags(flags...),
		cmds.WithArguments(command.Arguments...),
		cmds.WithTags(command.Tags...),
		cmds.WithMetadata(command.Metadata),
		cmds.WithSections(glazedSection, commandSettingsSection),
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
	if err := vals.DecodeSectionInto(schema.DefaultSlug, runtimeSettings); err != nil {
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

	sqlText, err := minitracecmd.RenderCommand(resolvedCommand, minitracecmd.RenderContext{
		TableName: runtimeSettings.TableName,
		Values:    resolvedValues,
	})
	if err != nil {
		return err
	}
	if err := queryengine.ValidateReadOnlyQuery(sqlText); err != nil {
		return err
	}

	db, conn, err := queryengine.OpenConnection(ctx, runtimeSettings.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	defer func() { _ = db.Close() }()

	if err := queryengine.LoadArchive(ctx, conn, queryengine.LoadOptions{
		ArchiveGlobs:  runtimeSettings.ArchiveGlob,
		TableName:     runtimeSettings.TableName,
		PersistLoaded: runtimeSettings.PersistLoaded,
	}); err != nil {
		return err
	}

	return queryengine.RunIntoProcessor(ctx, conn, sqlText, gp)
}

func collectCommandValues(vals *glazedvalues.Values, command *minitracecmd.MinitraceCommand) map[string]any {
	ret := map[string]any{}
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
