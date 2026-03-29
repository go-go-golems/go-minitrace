package convert

import (
	"context"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/common"
	"github.com/go-go-golems/go-minitrace/pkg/adapters/claudecode"
	"github.com/spf13/cobra"
)

type ConvertClaudeCodeCommand struct {
	*cmds.CommandDescription
}

type ConvertClaudeCodeSettings struct {
	SourceDir string `glazed:"source-dir"`
	OutputDir string `glazed:"output-dir"`
	DryRun    bool   `glazed:"dry-run"`
}

func NewConvertClaudeCodeGlazeCommand() (*ConvertClaudeCodeCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"claude-code",
		cmds.WithShort("Plan Claude Code conversion"),
		cmds.WithLong(`
Bootstrap conversion command for Claude Code.

The current implementation performs source discovery and emits a structured
summary row. Actual JSONL-to-minitrace conversion will be implemented next on
top of the shared Go normalization and validator packages.

Examples:
  go-minitrace convert claude-code --source-dir ~/.claude/projects --output-dir ./output
  go-minitrace convert claude-code --source-dir ~/.claude/projects --dry-run --output yaml
`),
		cmds.WithFlags(
			fields.New("source-dir", fields.TypeString, fields.WithDefault("~/.claude/projects"), fields.WithHelp("Claude Code projects directory")),
			fields.New("output-dir", fields.TypeString, fields.WithDefault("./output"), fields.WithHelp("Target minitrace archive directory")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Inspect sources without writing output")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ConvertClaudeCodeCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &ConvertClaudeCodeCommand{}

func (c *ConvertClaudeCodeCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &ConvertClaudeCodeSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}

	locators, err := claudecode.Discover(settings_.SourceDir)
	if err != nil {
		return err
	}

	row := types.NewRow(
		types.MRP("framework", "claude-code"),
		types.MRP("source_dir", settings_.SourceDir),
		types.MRP("output_dir", settings_.OutputDir),
		types.MRP("dry_run", settings_.DryRun),
		types.MRP("discovered_sessions", len(locators)),
		types.MRP("implemented", false),
		types.MRP("status", "conversion engine not implemented yet"),
	)
	return gp.AddRow(ctx, row)
}

func NewClaudeCodeCommand() (*cobra.Command, error) {
	cmd, err := NewConvertClaudeCodeGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
