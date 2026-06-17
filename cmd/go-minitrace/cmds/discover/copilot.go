package discover

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
	"github.com/spf13/cobra"
)

type CopilotCommand struct {
	*cmds.CommandDescription
}

type CopilotSettings struct {
	SourceDir string `glazed:"source-dir"`
}

func NewCopilotGlazeCommand() (*CopilotCommand, error) {
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
		cmds.WithShort("Discover GitHub Copilot CLI sessions"),
		cmds.WithLong(`
Discover GitHub Copilot CLI session-state directories.

The discovery logic accepts a Copilot home directory (~/.copilot), a direct
session-state directory, or a single session directory containing events.jsonl.
Scaffold-only session directories without events.jsonl are ignored.

Examples:
  go-minitrace discover copilot --source-dir ~/.copilot
  go-minitrace discover copilot --source-dir ~/.copilot/session-state --output yaml
`),
		cmds.WithFlags(
			fields.New(
				"source-dir",
				fields.TypeString,
				fields.WithDefault("~/.copilot"),
				fields.WithHelp("Copilot CLI home, session-state directory, or one session directory"),
			),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &CopilotCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &CopilotCommand{}

func (c *CopilotCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &CopilotSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}

	locators, err := copilot.Discover(settings_.SourceDir)
	if err != nil {
		return err
	}
	for _, locator := range locators {
		sessionDir := filepath.Dir(locator.SourcePath)
		workspacePath := filepath.Join(sessionDir, "workspace.yaml")
		sessionDBPath := filepath.Join(sessionDir, "session.db")
		row := types.NewRow(
			types.MRP("id", locator.ID),
			types.MRP("format_hint", locator.FormatHint),
			types.MRP("source_path", locator.SourcePath),
			types.MRP("session_dir", sessionDir),
			types.MRP("workspace_path", workspacePath),
			types.MRP("session_db_path", sessionDBPath),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

func NewCopilotCommand() (*cobra.Command, error) {
	cmd, err := NewCopilotGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
