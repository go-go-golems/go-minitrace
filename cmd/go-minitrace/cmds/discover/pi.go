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
	"github.com/go-go-golems/go-minitrace/pkg/adapters/pi"
	"github.com/spf13/cobra"
)

type PiCommand struct {
	*cmds.CommandDescription
}

type PiSettings struct {
	SourceDir   string `glazed:"source-dir"`
	CwdContains string `glazed:"cwd-contains"`
	Since       string `glazed:"since"`
}

func NewPiGlazeCommand() (*PiCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"pi",
		cmds.WithShort("Discover Pi session JSONL files"),
		cmds.WithLong(`
Discover Pi session JSONL files under the local Pi session directory.

Examples:
  go-minitrace discover pi --source-dir ~/.pi/agent/sessions
  go-minitrace discover pi --source-dir /tmp/pi-sessions --output json
`),
		cmds.WithFlags(
			append([]*fields.Definition{
				fields.New(
					"source-dir",
					fields.TypeString,
					fields.WithDefault("~/.pi/agent/sessions"),
					fields.WithHelp("Pi sessions directory"),
				),
			}, filterFlags()...)...,
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &PiCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &PiCommand{}

func (c *PiCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &PiSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}

	since, err := parseSince(settings_.Since)
	if err != nil {
		return err
	}

	locators, err := pi.Discover(settings_.SourceDir)
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

func NewPiCommand() (*cobra.Command, error) {
	cmd, err := NewPiGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
