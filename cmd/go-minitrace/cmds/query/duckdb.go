package query

import (
	"context"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/common"
	queryengine "github.com/go-go-golems/go-minitrace/pkg/query"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type DuckDBQueryCommand struct {
	*cmds.CommandDescription
}

type DuckDBQuerySettings struct {
	ArchiveGlob   []string `glazed:"archive-glob"`
	DBPath        string   `glazed:"db-path"`
	TableName     string   `glazed:"table-name"`
	Preset        string   `glazed:"preset"`
	SQL           string   `glazed:"sql"`
	SQLFile       string   `glazed:"sql-file"`
	LoadOnly      bool     `glazed:"load-only"`
	PersistLoaded bool     `glazed:"persist-loaded"`
}

func NewDuckDBQueryGlazeCommand() (*DuckDBQueryCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"duckdb",
		cmds.WithShort("Query converted minitrace archives with DuckDB"),
		cmds.WithLong(`
Load a converted minitrace archive into DuckDB and run either a named preset
or ad hoc SQL.

This command is authored as a normal Glazed command. All settings are defined
through Glazed fields and decoded from the default Glazed section.

Examples:
  go-minitrace query duckdb --archive-glob './output/active/*/*.minitrace.json' --preset session-list
  go-minitrace query duckdb --archive-glob './output/active/*/*.minitrace.json' --archive-glob './archive/*.minitrace.json' --preset session-list
  go-minitrace query duckdb --archive-glob './output/active/*/*.minitrace.json' --preset framework-summary --output json
  go-minitrace query duckdb --archive-glob './output/active/*/*.minitrace.json' --sql 'SELECT COUNT(*) AS sessions FROM sessions_base'
  go-minitrace query duckdb --archive-glob './output/active/*/*.minitrace.json' --sql-file ./custom.sql
`),
		cmds.WithFlags(
			fields.New("archive-glob", fields.TypeStringList, fields.WithDefault([]string{"./output/active/*/*.minitrace.json"}), fields.WithHelp("Repeatable glob flag for minitrace session JSON files to load")),
			fields.New("db-path", fields.TypeString, fields.WithDefault(":memory:"), fields.WithHelp("DuckDB database path to use; :memory: keeps the query session ephemeral")),
			fields.New("table-name", fields.TypeString, fields.WithDefault("sessions_base"), fields.WithHelp("Table name to create from the loaded archive")),
			fields.New("preset", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Named built-in query preset to execute")),
			fields.New("sql", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Inline SQL query to execute after loading the archive")),
			fields.New("sql-file", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Path to a SQL file to execute after loading the archive")),
			fields.New("load-only", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Only load the archive and emit a summary row without executing a query")),
			fields.New("persist-loaded", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Request a persistent loaded table rather than a temp table")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &DuckDBQueryCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &DuckDBQueryCommand{}

func (c *DuckDBQueryCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &DuckDBQuerySettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}
	if len(settings_.ArchiveGlob) == 0 {
		return errors.New("archive-glob is required")
	}

	modeCount := 0
	if strings.TrimSpace(settings_.Preset) != "" {
		modeCount++
	}
	if strings.TrimSpace(settings_.SQL) != "" {
		modeCount++
	}
	if strings.TrimSpace(settings_.SQLFile) != "" {
		modeCount++
	}
	if settings_.LoadOnly {
		modeCount++
	}
	if modeCount == 0 {
		return errors.New("one of preset, sql, sql-file, or load-only must be specified")
	}
	if settings_.LoadOnly && modeCount > 1 {
		return errors.New("load-only cannot be combined with preset, sql, or sql-file")
	}
	if !settings_.LoadOnly && modeCount > 1 {
		return errors.New("preset, sql, and sql-file are mutually exclusive")
	}

	db, conn, err := queryengine.OpenConnection(ctx, settings_.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	defer func() { _ = db.Close() }()

	if err := queryengine.LoadArchive(ctx, conn, queryengine.LoadOptions{
		ArchiveGlobs:  settings_.ArchiveGlob,
		TableName:     settings_.TableName,
		PersistLoaded: settings_.PersistLoaded,
	}); err != nil {
		return err
	}

	if settings_.LoadOnly {
		row := types.NewRow(
			types.MRP("status", "loaded"),
			types.MRP("backend", "duckdb"),
			types.MRP("archive_globs", settings_.ArchiveGlob),
			types.MRP("db_path", settings_.DBPath),
			types.MRP("table_name", settings_.TableName),
			types.MRP("persist_loaded", settings_.PersistLoaded),
		)
		return gp.AddRow(ctx, row)
	}

	sqlText, err := queryengine.ResolveSQL(settings_.Preset, settings_.SQL, settings_.SQLFile, settings_.TableName)
	if err != nil {
		return err
	}
	return queryengine.RunIntoProcessor(ctx, conn, sqlText, gp)
}

func NewDuckDBCommand() (*cobra.Command, error) {
	cmd, err := NewDuckDBQueryGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
