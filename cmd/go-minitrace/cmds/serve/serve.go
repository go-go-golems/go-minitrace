package serve

//glazedclilint:file-ignore legacy serve command exposes output flags pending row processor migration

import (
	"context"
	stderrors "errors"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/common"
	"github.com/go-go-golems/go-minitrace/pkg/annotate"
	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
	"github.com/go-go-golems/go-minitrace/pkg/minitracejs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ServeCommand struct {
	*cmds.CommandDescription
}

type ServeSettings struct {
	ArchiveGlob     []string `glazed:"archive-glob"`
	PresetDir       []string `glazed:"preset-dir"`
	QueryDir        []string `glazed:"query-dir"`
	QueryRepository []string `glazed:"query-repository"`
	Port            int      `glazed:"port"`
	MaxRows         int      `glazed:"max-rows"`
	QueryTimeoutMS  int      `glazed:"query-timeout-ms"`
	DevMode         bool     `glazed:"dev"`
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
Build (or reuse from cache) the normalized SQLite database for a converted
minitrace archive and expose the Transcript Explorer backend over HTTP. All
SQL surfaces run through the sandboxed read-only query runner against the
normalized schema (sessions, turns, tool_calls, ... plus the sessions_base
compatibility view); the live annotation store is attached as anno.annotations
when present.

This command is authored as a Glazed bare command: settings are defined through
Glazed fields and sections, but execution is long-running and does not emit
rows to a Glazed processor.

Examples:
  go-minitrace serve --archive-glob './output/active/*/*.minitrace.json'
  go-minitrace serve --archive-glob './output/active/*/*.minitrace.json' --archive-glob './archive/*.minitrace.json'
  go-minitrace serve --archive-glob './output/active/*/*.minitrace.json' --port 8090
  go-minitrace serve --archive-glob './output/active/*/*.minitrace.json' --dev
  go-minitrace serve --archive-glob './output/active/*/*.minitrace.json' --max-rows 50000
  go-minitrace serve --archive-glob './output/active/*/*.minitrace.json' \
    --preset-dir ./presets/team --preset-dir ./presets/project \
    --query-dir ./queries/shared --query-dir ./queries/private
  go-minitrace serve --archive-glob './output/active/*/*.minitrace.json' \
    --query-repository ./query-commands/team --query-repository ./query-commands/private
`),
		cmds.WithFlags(
			fields.New("archive-glob", fields.TypeStringList, fields.WithDefault([]string{"./output/active/*/*.minitrace.json"}), fields.WithHelp("Repeatable glob flag for converted minitrace session files to load")),
			fields.New("preset-dir", fields.TypeStringList, fields.WithDefault([]string{}), fields.WithHelp("Repeatable directory flag for additional read-only SQL preset roots")),
			fields.New("query-dir", fields.TypeStringList, fields.WithDefault([]string{"./queries"}), fields.WithHelp("Repeatable directory flag for user-saved SQL query roots; new queries are created in the first root")),
			fields.New("query-repository", fields.TypeStringList, fields.WithDefault([]string{}), fields.WithHelp("Repeatable directory flag for additional structured query-command repository roots")),
			fields.New("port", fields.TypeInteger, fields.WithDefault(8080), fields.WithHelp("HTTP listen port")),
			fields.New("max-rows", fields.TypeInteger, fields.WithDefault(10000), fields.WithHelp("Maximum number of rows returned per SQL query (web Query Editor and session list)")),
			fields.New("query-timeout-ms", fields.TypeInteger, fields.WithDefault(30000), fields.WithHelp("Per-query timeout in milliseconds for the SQL surfaces")),
			fields.New("dev", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Run API-only mode and expect the frontend to be served by Vite")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ServeCommand{CommandDescription: desc}, nil
}

var _ cmds.BareCommand = &ServeCommand{}

// serveQueryOptions derives the lifted sandbox limits for serve's SQL
// surfaces: rows/timeout are configurable, columns and cell size are raised
// above the CLI defaults because the web Query Editor renders whole blobs.
func serveQueryOptions(maxRows int, queryTimeoutMS int) minitracedb.QueryOptions {
	opts := minitracedb.DefaultQueryOptions()
	if maxRows > 0 {
		opts.MaxRows = maxRows
	}
	if queryTimeoutMS > 0 {
		opts.Timeout = time.Duration(queryTimeoutMS) * time.Millisecond
	}
	opts.MaxColumns = 256
	opts.MaxCellChars = 65536
	return opts
}

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

	queryOpts := serveQueryOptions(settings_.MaxRows, settings_.QueryTimeoutMS)

	// Open the annotation store (SQLite) first so annotations.db exists and
	// can be attached live to the SQL surface below.
	outputDir, err := outputDirFromGlobs(settings_.ArchiveGlob)
	if err != nil {
		return err
	}
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

	// Build (or reuse from cache) the normalized SQLite database once.
	queryTarget, annotationsAttached, err := buildServeQueryTarget(signalCtx, settings_.ArchiveGlob, outputDir, queryOpts)
	if err != nil {
		return err
	}
	defer func() { _ = queryTarget.Close() }()

	sessionIndex, err := buildSessionIndex(settings_.ArchiveGlob)
	if err != nil {
		return err
	}

	log.Info().
		Strs("archive_globs", settings_.ArchiveGlob).
		Int("indexed_sessions", len(sessionIndex)).
		Int("max_rows", queryOpts.MaxRows).
		Bool("annotations_attached", annotationsAttached).
		Bool("dev_mode", settings_.DevMode).
		Msg("built normalized minitrace database for serve")

	commandSourceRoots, err := minitracecmd.CommandSourceRoots("go-minitrace", settings_.QueryRepository)
	if err != nil {
		return err
	}

	server := NewServer(queryTarget, settings_, sessionIndex, annoStore, sessionIndex)
	server.commandSourceRoots = commandSourceRoots
	if err := server.ListenAndServe(signalCtx, settings_.Port); err != nil {
		if stderrors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}

// buildServeQueryTarget builds (or reuses from cache) the normalized SQLite
// database for the archive globs and, when a live annotation store exists
// under outputDir, reopens the cached database file with annotations.db
// ATTACHed as `anno` so annotations stay live-queryable from SQL. Returns the
// query target and whether the annotation attach is active.
func buildServeQueryTarget(ctx context.Context, archiveGlobs []string, outputDir string, queryOpts minitracedb.QueryOptions) (minitracedb.QueryTarget, bool, error) {
	handle, err := minitracejs.NewArchiveQueryTarget(ctx, archiveGlobs, queryOpts)
	if err != nil {
		return nil, false, err
	}

	if outputDir == "" {
		return handle, false, nil
	}
	annoPath := filepath.Join(outputDir, "annotations.db")
	if _, err := os.Stat(annoPath); err != nil {
		return handle, false, nil
	}
	cachePath := handle.CacheInfo().Path
	if cachePath == "" {
		log.Warn().Str("annotations_db", annoPath).Msg("normalized database is not disk-cached; serving without live annotation attach")
		return handle, false, nil
	}

	attachedDB, err := minitracedb.OpenSQLiteReadOnlyAttached(ctx, cachePath, map[string]string{
		minitracedb.AnnotationsAttachSchema: annoPath,
	})
	if err != nil {
		log.Warn().Err(err).Str("annotations_db", annoPath).Msg("could not attach annotations database; serving without live annotation SQL")
		return handle, false, nil
	}
	attachedTarget, err := minitracedb.NewDBQueryTarget(attachedDB, minitracedb.AllowedObjectNamesWithLiveAnnotations(), queryOpts)
	if err != nil {
		_ = attachedDB.Close()
		log.Warn().Err(err).Msg("could not build annotation-attached query target; serving without live annotation SQL")
		return handle, false, nil
	}
	// The attached target reads the same cached file directly; release the
	// builder handle's reference.
	_ = handle.Close()
	return attachedTarget, true, nil
}

// outputDirFromGlobs infers the output directory from the first matching file
// of the given archive globs. It assumes the standard layout: outputDir/active/period/*.minitrace.json.
// Returns the output directory (e.g. ./output) or "" if no files match.
func outputDirFromGlobs(globs []string) (string, error) {
	if len(globs) == 0 {
		return "", nil
	}
	files, err := minitracedb.ExpandArchiveGlobs(globs)
	if err != nil || len(files) == 0 {
		return "", nil
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
