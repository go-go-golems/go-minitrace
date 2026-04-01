package serve

import (
	"context"
	stderrors "errors"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/common"
	queryengine "github.com/go-go-golems/go-minitrace/pkg/query"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type ServeCommand struct {
	*cmds.CommandDescription
}

type ServeSettings struct {
	ArchiveGlob string   `glazed:"archive-glob"`
	PresetDir   []string `glazed:"preset-dir"`
	QueryDir    []string `glazed:"query-dir"`
	Port        int      `glazed:"port"`
	DBPath      string   `glazed:"db-path"`
	TableName   string   `glazed:"table-name"`
	DevMode     bool     `glazed:"dev"`
}

func NewGlazeCommand() (*ServeCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"serve",
		cmds.WithShort("Serve the minitrace transcript explorer API"),
		cmds.WithLong(`
Load a converted minitrace archive into DuckDB, keep it resident in-process,
and expose the Transcript Explorer backend over HTTP.

This command is authored as a Glazed bare command: settings are defined through
Glazed fields and sections, but execution is long-running and does not emit
rows to a Glazed processor.

Examples:
  go-minitrace serve --archive-glob './output/active/*/*.minitrace.json'
  go-minitrace serve --archive-glob './output/active/*/*.minitrace.json' --port 8090
  go-minitrace serve --archive-glob './output/active/*/*.minitrace.json' --dev
  go-minitrace serve --archive-glob './output/active/*/*.minitrace.json' \
    --preset-dir ./presets/team --preset-dir ./presets/project \
    --query-dir ./queries/shared --query-dir ./queries/private
`),
		cmds.WithFlags(
			fields.New("archive-glob", fields.TypeString, fields.WithDefault("./output/active/*/*.minitrace.json"), fields.WithHelp("Glob pattern for converted minitrace session files to load")),
			fields.New("preset-dir", fields.TypeStringList, fields.WithDefault([]string{}), fields.WithHelp("Repeatable directory flag for additional read-only SQL preset roots")),
			fields.New("query-dir", fields.TypeStringList, fields.WithDefault([]string{"./queries"}), fields.WithHelp("Repeatable directory flag for user-saved SQL query roots; new queries are created in the first root")),
			fields.New("port", fields.TypeInteger, fields.WithDefault(8080), fields.WithHelp("HTTP listen port")),
			fields.New("db-path", fields.TypeString, fields.WithDefault(":memory:"), fields.WithHelp("DuckDB database path; :memory: keeps the server ephemeral")),
			fields.New("table-name", fields.TypeString, fields.WithDefault("sessions_base"), fields.WithHelp("DuckDB table name to create from the loaded archive")),
			fields.New("dev", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Run API-only mode and expect the frontend to be served by Vite")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ServeCommand{CommandDescription: desc}, nil
}

var _ cmds.BareCommand = &ServeCommand{}

func (c *ServeCommand) Run(ctx context.Context, vals *values.Values) error {
	settings_ := &ServeSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}
	if strings.TrimSpace(settings_.ArchiveGlob) == "" {
		return errors.New("archive-glob is required")
	}

	signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, conn, err := queryengine.OpenConnection(signalCtx, settings_.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	defer func() { _ = db.Close() }()

	if err := queryengine.LoadArchive(signalCtx, conn, queryengine.LoadOptions{
		ArchiveGlob: settings_.ArchiveGlob,
		TableName:   settings_.TableName,
	}); err != nil {
		return err
	}

	sessionIndex, err := buildSessionIndex(settings_.ArchiveGlob)
	if err != nil {
		return err
	}

	log.Info().
		Str("archive_glob", settings_.ArchiveGlob).
		Str("db_path", settings_.DBPath).
		Str("table_name", settings_.TableName).
		Int("indexed_sessions", len(sessionIndex)).
		Bool("dev_mode", settings_.DevMode).
		Msg("loaded minitrace archive for serve")

	server := NewServer(conn, settings_, sessionIndex)
	if err := server.ListenAndServe(signalCtx, settings_.Port); err != nil {
		if stderrors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}

func NewCommand() (*cobra.Command, error) {
	cmd, err := NewGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
