package convert

import (
	"context"
	"path/filepath"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/common"
	"github.com/go-go-golems/go-minitrace/pkg/adapters/copilot"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ConvertCopilotCommand struct {
	*cmds.CommandDescription
}

type ConvertCopilotSettings struct {
	SourceDir string `glazed:"source-dir"`
	OutputDir string `glazed:"output-dir"`
	DryRun    bool   `glazed:"dry-run"`
}

func NewConvertCopilotGlazeCommand() (*ConvertCopilotCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"copilot",
		cmds.WithShort("Convert GitHub Copilot CLI sessions into minitrace"),
		cmds.WithLong(`
Convert GitHub Copilot CLI session-state directories into minitrace JSON files.

The converter reads events.jsonl as the ordered transcript source and uses
workspace.yaml for repository, branch, and working-directory metadata when
available.

Examples:
  go-minitrace convert copilot --source-dir ~/.copilot --output-dir ./output
  go-minitrace convert copilot --source-dir ~/.copilot/session-state/<id> --dry-run --output json
`),
		cmds.WithFlags(
			fields.New("source-dir", fields.TypeString, fields.WithDefault("~/.copilot"), fields.WithHelp("Copilot CLI home, session-state directory, or one session directory")),
			fields.New("output-dir", fields.TypeString, fields.WithDefault("./output"), fields.WithHelp("Target minitrace archive directory")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Inspect sources without writing output")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ConvertCopilotCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &ConvertCopilotCommand{}

func (c *ConvertCopilotCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &ConvertCopilotSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}

	locators, err := copilot.Discover(settings_.SourceDir)
	if err != nil {
		return err
	}

	indexEntries := make([]*minitrace.SessionIndexEntry, 0, len(locators))
	for _, locator := range locators {
		session, err := copilot.ConvertLocator(locator)
		if err != nil {
			return errors.Wrapf(err, "converting Copilot session %s", locator.ID)
		}

		var sessionPath string
		if !settings_.DryRun {
			entry, err := minitrace.WriteSession(session, settings_.OutputDir)
			if err != nil {
				return errors.Wrapf(err, "writing minitrace session %s", locator.ID)
			}
			indexEntries = append(indexEntries, entry)
			sessionPath = entry.FilePath
		}

		quality := ""
		if session.Quality != nil {
			quality = *session.Quality
		}
		row := types.NewRow(
			types.MRP("framework", "copilot"),
			types.MRP("session_id", session.ID),
			types.MRP("source_format", locator.FormatHint),
			types.MRP("source_path", locator.SourcePath),
			types.MRP("turn_count", session.Metrics.TurnCount),
			types.MRP("tool_call_count", session.Metrics.ToolCallCount),
			types.MRP("annotation_count", len(session.Annotations)),
			types.MRP("quality", quality),
			types.MRP("classification", session.Classification),
			types.MRP("dry_run", settings_.DryRun),
			types.MRP("session_path", sessionPath),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}

	if !settings_.DryRun {
		if err := minitrace.WriteManifests(indexEntries, settings_.OutputDir); err != nil {
			return errors.Wrap(err, "writing Copilot manifests")
		}
		manifestRow := types.NewRow(
			types.MRP("framework", "copilot"),
			types.MRP("manifest_path", filepath.Join(settings_.OutputDir, "manifest.json")),
			types.MRP("session_count", len(indexEntries)),
			types.MRP("dry_run", false),
		)
		return gp.AddRow(ctx, manifestRow)
	}

	return nil
}

func NewCopilotCommand() (*cobra.Command, error) {
	cmd, err := NewConvertCopilotGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
