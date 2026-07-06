package discover

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

type ClaudeCodeCommand struct {
	*cmds.CommandDescription
}

type ClaudeCodeSettings struct {
	SourceDir   string `glazed:"source-dir"`
	CwdContains string `glazed:"cwd-contains"`
	Since       string `glazed:"since"`
}

func NewClaudeCodeGlazeCommand() (*ClaudeCodeCommand, error) {
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
		cmds.WithShort("Discover Claude Code sessions"),
		cmds.WithLong(`
Discover Claude Code sessions under a projects directory.

This mirrors the Python adapter's session discovery rules:
- JSONL v2 transcripts are preferred
- subagent transcripts are ignored at this layer
- older dir-v1 sessions with tool-results/ are included when no JSONL transcript exists

Examples:
  go-minitrace discover claude-code --source-dir ~/.claude/projects
  go-minitrace discover claude-code --source-dir /tmp/claude-projects --output json
`),
		cmds.WithFlags(
			append([]*fields.Definition{
				fields.New(
					"source-dir",
					fields.TypeString,
					fields.WithDefault("~/.claude/projects"),
					fields.WithHelp("Claude Code projects directory"),
				),
			}, filterFlags()...)...,
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ClaudeCodeCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &ClaudeCodeCommand{}

func (c *ClaudeCodeCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &ClaudeCodeSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}

	since, err := parseSince(settings_.Since)
	if err != nil {
		return err
	}

	locators, err := claudecode.Discover(settings_.SourceDir)
	if err != nil {
		return err
	}
	for _, locator := range locators {
		if !keepLocator(locator, settings_.CwdContains, since) {
			continue
		}
		row := types.NewRow(
			types.MRP("id", locator.ID),
			types.MRP("format_hint", locator.FormatHint),
			types.MRP("source_path", locator.SourcePath),
			types.MRP("cwd", locator.Cwd),
			types.MRP("started_at", locator.StartedAt),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

func NewClaudeCodeCommand() (*cobra.Command, error) {
	cmd, err := NewClaudeCodeGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
