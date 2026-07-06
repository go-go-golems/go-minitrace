package query

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/common"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
	"github.com/go-go-golems/go-minitrace/pkg/minitracejs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type RunQueryCommand struct {
	*cmds.CommandDescription
}

type RunQuerySettings struct {
	ArchiveGlob  []string `glazed:"archive-glob"`
	Preset       string   `glazed:"preset"`
	SQL          string   `glazed:"sql"`
	SQLFile      string   `glazed:"sql-file"`
	MaxRows      int      `glazed:"max-rows"`
	MaxCellChars int      `glazed:"max-cell-chars"`
	TimeoutMS    int      `glazed:"timeout-ms"`
}

func NewRunQueryGlazeCommand() (*RunQueryCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"run",
		cmds.WithShort("Query converted minitrace archives with the normalized SQLite engine"),
		cmds.WithLong(`
Build (or reuse from cache) the normalized SQLite database for the given
archives and run either a named preset or ad hoc SQL through the sandboxed
read-only query runner.

Queries target the normalized schema (sessions, turns, tool_calls,
annotations, metrics, files, events, attachments, handovers) plus the
sessions_base compatibility view.

Examples:
  go-minitrace query run --archive-glob './output/active/*/*.minitrace.json' --preset session-list
  go-minitrace query run --archive-glob './output/active/*/*.minitrace.json' --preset framework-summary --output json
  go-minitrace query run --archive-glob './output/active/*/*.minitrace.json' --sql 'SELECT COUNT(*) AS sessions FROM sessions'
  go-minitrace query run --archive-glob './output/active/*/*.minitrace.json' --sql-file ./custom.sql --max-rows 5000
`),
		cmds.WithFlags(
			fields.New("archive-glob", fields.TypeStringList, fields.WithDefault([]string{"./output/active/*/*.minitrace.json"}), fields.WithHelp("Repeatable glob flag for minitrace session JSON files to load")),
			fields.New("preset", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Named built-in query preset to execute")),
			fields.New("sql", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Inline SQL query to execute against the normalized database")),
			fields.New("sql-file", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Path to a SQL file to execute against the normalized database")),
			fields.New("max-rows", fields.TypeInteger, fields.WithDefault(1000), fields.WithHelp("Maximum number of rows to return")),
			fields.New("max-cell-chars", fields.TypeInteger, fields.WithDefault(4000), fields.WithHelp("Maximum number of characters per result cell")),
			fields.New("timeout-ms", fields.TypeInteger, fields.WithDefault(5000), fields.WithHelp("Query timeout in milliseconds")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &RunQueryCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &RunQueryCommand{}

func (c *RunQueryCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &RunQuerySettings{}
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
	if modeCount == 0 {
		return errors.New("one of preset, sql, or sql-file must be specified")
	}
	if modeCount > 1 {
		return errors.New("preset, sql, and sql-file are mutually exclusive")
	}

	sqlText, err := resolveRunSQL(settings_)
	if err != nil {
		return err
	}

	target, err := minitracejs.NewArchiveQueryTarget(ctx, settings_.ArchiveGlob, minitracedb.QueryOptions{
		MaxRows:      settings_.MaxRows,
		MaxCellChars: settings_.MaxCellChars,
		Timeout:      time.Duration(settings_.TimeoutMS) * time.Millisecond,
	})
	if err != nil {
		return err
	}
	defer func() { _ = target.Close() }()

	return RunQueryTargetIntoProcessor(ctx, target, sqlText, gp)
}

func resolveRunSQL(settings_ *RunQuerySettings) (string, error) {
	if preset := strings.TrimSpace(settings_.Preset); preset != "" {
		return minitracedb.ResolvePresetSQL(preset)
	}
	if strings.TrimSpace(settings_.SQL) != "" {
		return settings_.SQL, nil
	}
	sqlFile := strings.TrimSpace(settings_.SQLFile)
	payload, err := os.ReadFile(sqlFile)
	if err != nil {
		return "", fmt.Errorf("reading sql file %q: %w", sqlFile, err)
	}
	return string(payload), nil
}

func NewRunCommand() (*cobra.Command, error) {
	cmd, err := NewRunQueryGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
