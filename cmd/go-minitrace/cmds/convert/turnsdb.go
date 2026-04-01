package convert

import (
	"context"
	"path/filepath"
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
	"github.com/go-go-golems/go-minitrace/pkg/adapters/turnsdb"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ConvertTurnsDBCommand struct {
	*cmds.CommandDescription
}

type ConvertTurnsDBSettings struct {
	Source    string `glazed:"source"`
	OutputDir string `glazed:"output-dir"`
	ConvID    string `glazed:"conv-id"`
	DryRun    bool   `glazed:"dry-run"`
}

func NewConvertTurnsDBGlazeCommand() (*ConvertTurnsDBCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"turnsdb",
		cmds.WithShort("Convert Geppetto/Pinocchio turns.db snapshots into minitrace"),
		cmds.WithLong(`
Convert a snapshot-oriented turns.db SQLite database into minitrace JSON files.

This converter reconstructs logical conversation deltas from canonical turn snapshots rather than treating each stored row as a transcript turn.

Examples:
  go-minitrace convert turnsdb --source /tmp/turns.db --output-dir ./output
  go-minitrace convert turnsdb --source /tmp/turns.db --conv-id 5cf06c5f-0460-485e-a7c5-92d56af826f9 --dry-run --output json
`),
		cmds.WithFlags(
			fields.New("source", fields.TypeString, fields.WithHelp("Path to turns.db SQLite database")),
			fields.New("output-dir", fields.TypeString, fields.WithDefault("./output"), fields.WithHelp("Target minitrace archive directory")),
			fields.New("conv-id", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Only convert the selected conversation ID")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Inspect sources without writing output")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ConvertTurnsDBCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &ConvertTurnsDBCommand{}

func (c *ConvertTurnsDBCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &ConvertTurnsDBSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}
	if strings.TrimSpace(settings_.Source) == "" {
		return errors.New("source is required")
	}

	sessions, err := turnsdb.ConvertDB(settings_.Source, strings.TrimSpace(settings_.ConvID))
	if err != nil {
		return err
	}

	indexEntries := []*minitrace.SessionIndexEntry{}
	for _, session := range sessions {
		var sessionPath string
		if !settings_.DryRun {
			entry, err := minitrace.WriteSession(session, settings_.OutputDir)
			if err != nil {
				return errors.Wrapf(err, "writing turns.db session %s", session.ID)
			}
			indexEntries = append(indexEntries, entry)
			sessionPath = entry.FilePath
		}

		quality := ""
		if session.Quality != nil {
			quality = *session.Quality
		}
		row := types.NewRow(
			types.MRP("framework", "pinocchio"),
			types.MRP("session_id", session.ID),
			types.MRP("source_format", turnsdb.SourceFormat),
			types.MRP("source_path", settings_.Source),
			types.MRP("turn_count", session.Metrics.TurnCount),
			types.MRP("tool_call_count", session.Metrics.ToolCallCount),
			types.MRP("quality", quality),
			types.MRP("classification", session.Classification),
			types.MRP("dry_run", settings_.DryRun),
			types.MRP("session_path", sessionPath),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}

	summaryRow := types.NewRow(
		types.MRP("framework", "pinocchio"),
		types.MRP("source_format", turnsdb.SourceFormat),
		types.MRP("converted", len(sessions)),
		types.MRP("dry_run", settings_.DryRun),
	)
	if err := gp.AddRow(ctx, summaryRow); err != nil {
		return err
	}

	if !settings_.DryRun {
		if err := minitrace.WriteManifests(indexEntries, settings_.OutputDir); err != nil {
			return errors.Wrap(err, "writing turns.db manifests")
		}
		manifestRow := types.NewRow(
			types.MRP("framework", "pinocchio"),
			types.MRP("source_format", turnsdb.SourceFormat),
			types.MRP("manifest_path", filepath.Join(settings_.OutputDir, "manifest.json")),
			types.MRP("session_count", len(indexEntries)),
			types.MRP("dry_run", false),
		)
		return gp.AddRow(ctx, manifestRow)
	}

	return nil
}

func NewTurnsDBCommand() (*cobra.Command, error) {
	cmd, err := NewConvertTurnsDBGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
