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
	"github.com/go-go-golems/go-minitrace/pkg/adapters/codex"
	"github.com/spf13/cobra"
)

type CodexCommand struct {
	*cmds.CommandDescription
}

type CodexSettings struct {
	SourceDir   string `glazed:"source-dir"`
	CwdContains string `glazed:"cwd-contains"`
	Since       string `glazed:"since"`
	ActiveSince string `glazed:"active-since"`
}

func NewCodexGlazeCommand() (*CodexCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"codex",
		cmds.WithShort("Discover Codex sessions"),
		cmds.WithLong(`
Discover Codex session JSONL files.

The discovery logic prefers ~/.codex/sessions/... when present and falls back
to scanning the source tree. It also attempts to distinguish richer persisted
session JSONL from exec JSONL streams.

Examples:
  go-minitrace discover codex --source-dir ~/.codex
  go-minitrace discover codex --source-dir /tmp/codex --output yaml
`),
		cmds.WithFlags(
			append(append([]*fields.Definition{
				fields.New(
					"source-dir",
					fields.TypeString,
					fields.WithDefault("~/.codex"),
					fields.WithHelp("Codex home directory"),
				),
			}, filterFlags()...), activityFilterFlags()...)...,
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &CodexCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &CodexCommand{}

func (c *CodexCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &CodexSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}

	since, err := parseSince(settings_.Since)
	if err != nil {
		return err
	}
	activeSince, err := parseSince(settings_.ActiveSince)
	if err != nil {
		return err
	}

	locators, err := codex.Discover(settings_.SourceDir)
	if err != nil {
		return err
	}
	for _, locator := range locators {
		if !keepLocator(locator, settings_.CwdContains, since, nil) {
			continue
		}
		if activeSince != nil {
			if err := populateLastActivity(&locator, codex.LastActivityAt); err != nil {
				return err
			}
		}
		if !keepLocator(locator, settings_.CwdContains, since, activeSince) {
			continue
		}
		row := types.NewRow(
			types.MRP("id", locator.ID),
			types.MRP("format_hint", locator.FormatHint),
			types.MRP("source_path", locator.SourcePath),
			types.MRP("cwd", locator.Cwd),
			types.MRP("started_at", locator.StartedAt),
			types.MRP("last_activity_at", locator.LastActivityAt),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

func NewCodexCommand() (*cobra.Command, error) {
	cmd, err := NewCodexGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
