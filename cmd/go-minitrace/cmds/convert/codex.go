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
	"github.com/go-go-golems/go-minitrace/pkg/adapters/codex"
	"github.com/spf13/cobra"
)

type ConvertCodexCommand struct {
	*cmds.CommandDescription
}

type ConvertCodexSettings struct {
	SourceDir string `glazed:"source-dir"`
	OutputDir string `glazed:"output-dir"`
	DryRun    bool   `glazed:"dry-run"`
}

func NewConvertCodexGlazeCommand() (*ConvertCodexCommand, error) {
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
		cmds.WithShort("Plan Codex conversion"),
		cmds.WithLong(`
Bootstrap conversion command for Codex.

The current implementation performs source discovery and emits a structured
planning row. Actual session-jsonl and exec-jsonl conversion will be added once
the shared normalization, metrics, and validation packages are in place.

Examples:
  go-minitrace convert codex --source-dir ~/.codex --output-dir ./output
  go-minitrace convert codex --source-dir ~/.codex --dry-run --output json
`),
		cmds.WithFlags(
			fields.New("source-dir", fields.TypeString, fields.WithDefault("~/.codex"), fields.WithHelp("Codex home directory")),
			fields.New("output-dir", fields.TypeString, fields.WithDefault("./output"), fields.WithHelp("Target minitrace archive directory")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Inspect sources without writing output")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ConvertCodexCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &ConvertCodexCommand{}

func (c *ConvertCodexCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &ConvertCodexSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}

	locators, err := codex.Discover(settings_.SourceDir)
	if err != nil {
		return err
	}

	row := types.NewRow(
		types.MRP("framework", "codex"),
		types.MRP("source_dir", settings_.SourceDir),
		types.MRP("output_dir", settings_.OutputDir),
		types.MRP("dry_run", settings_.DryRun),
		types.MRP("discovered_sessions", len(locators)),
		types.MRP("implemented", false),
		types.MRP("status", "conversion engine not implemented yet"),
	)
	return gp.AddRow(ctx, row)
}

func NewCodexCommand() (*cobra.Command, error) {
	cmd, err := NewConvertCodexGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
