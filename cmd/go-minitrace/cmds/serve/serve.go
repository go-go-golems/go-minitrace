package serve

import (
	"context"
	stderrors "errors"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/common"
	"github.com/go-go-golems/go-minitrace/pkg/annotate"
	queryengine "github.com/go-go-golems/go-minitrace/pkg/query"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type ServeCommand struct {
	*cmds.CommandDescription
}

type ServeSettings struct {
	ArchiveGlob []string `glazed:"archive-glob"`
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
  go-minitrace serve --archive-glob './output/active/*/*.minitrace.json' --archive-glob './archive/*.minitrace.json'
  go-minitrace serve --archive-glob './output/active/*/*.minitrace.json' --port 8090
  go-minitrace serve --archive-glob './output/active/*/*.minitrace.json' --dev
  go-minitrace serve --archive-glob './output/active/*/*.minitrace.json' \
    --preset-dir ./presets/team --preset-dir ./presets/project \
    --query-dir ./queries/shared --query-dir ./queries/private
`),
		cmds.WithFlags(
			fields.New("archive-glob", fields.TypeStringList, fields.WithDefault([]string{"./output/active/*/*.minitrace.json"}), fields.WithHelp("Repeatable glob flag for converted minitrace session files to load")),
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
	if len(settings_.ArchiveGlob) == 0 {
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

	// Attach annotations SQLite DB to DuckDB via sqlite_scanner.
	// This makes annotations directly queryable from DuckDB SQL.
	outputDir, err := outputDirFromGlobs(settings_.ArchiveGlob)
	if err == nil && outputDir != "" {
		if err := annotate.AttachAnnotationsToDuckDB(signalCtx, conn, outputDir); err != nil {
			log.Warn().Err(err).Str("output_dir", outputDir).Msg("could not attach annotations database")
		}
	}

	if err := queryengine.LoadArchive(signalCtx, conn, queryengine.LoadOptions{
		ArchiveGlobs: settings_.ArchiveGlob,
		TableName:    settings_.TableName,
	}); err != nil {
		return err
	}

	sessionIndex, err := buildSessionIndex(settings_.ArchiveGlob)
	if err != nil {
		return err
	}

	// Open the annotation store (SQLite).
	var annoStore *annotate.Store
	if outputDir != "" {
		annoStore, err = annotate.Open(signalCtx, outputDir)
		if err != nil {
			log.Warn().Err(err).Str("output_dir", outputDir).Msg("could not open annotation store")
			annoStore = nil
		}
	}
	if annoStore != nil {
		defer func() { _ = annoStore.Close() }()
	}

	log.Info().
		Strs("archive_globs", settings_.ArchiveGlob).
		Str("db_path", settings_.DBPath).
		Str("table_name", settings_.TableName).
		Int("indexed_sessions", len(sessionIndex)).
		Bool("dev_mode", settings_.DevMode).
		Msg("loaded minitrace archive for serve")

	server := NewServer(conn, settings_, sessionIndex, annoStore, sessionIndex)
	if err := server.ListenAndServe(signalCtx, settings_.Port); err != nil {
		if stderrors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}

// outputDirFromGlobs infers the output directory from the first matching file
// of the given archive globs. It assumes the standard layout: outputDir/active/period/*.minitrace.json.
// Returns the output directory (e.g. ./output) or "" if no files match.
func outputDirFromGlobs(globs []string) (string, error) {
	if len(globs) == 0 {
		return "", nil
	}
	files, err := queryengine.ExpandArchiveGlobs(globs)
	if err != nil || len(files) == 0 {
		return "", err
	}
	// outputDir is three levels up from the session file: outputDir/active/YYYY-MM/file.minitrace.json
	absDir := filepath.Dir(filepath.Dir(filepath.Dir(files[0])))
	rel, err := filepath.Rel(".", absDir)
	if err != nil {
		return absDir, nil
	}
	return rel, nil
}

func NewCommand() (*cobra.Command, error) {
	cmd, err := NewGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
