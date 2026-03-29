package validate

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
	validatepkg "github.com/go-go-golems/go-minitrace/pkg/validate"
	"github.com/spf13/cobra"
)

type ValidateCommand struct {
	*cmds.CommandDescription
}

type ValidateSettings struct {
	Path      string `glazed:"path"`
	Recursive bool   `glazed:"recursive"`
}

func NewGlazeCommand() (*ValidateCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"validate",
		cmds.WithShort("Validate JSON files targeted for minitrace processing"),
		cmds.WithLong(`
Validate JSON files targeted for minitrace processing.

This bootstrap implementation performs JSON syntax validation and directory
walking. Full schema and semantic validation from the Python validator will be
ported into pkg/validate next.

Examples:
  go-minitrace validate --path ./examples/v0.2.0 --recursive
  go-minitrace validate --path ./session.minitrace.json --output json
`),
		cmds.WithFlags(
			fields.New("path", fields.TypeString, fields.WithRequired(true), fields.WithHelp("File or directory to validate")),
			fields.New("recursive", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Recursively scan directories for JSON files")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ValidateCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &ValidateCommand{}

func (c *ValidateCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &ValidateSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}

	results, err := validatepkg.ValidatePath(settings_.Path, settings_.Recursive)
	if err != nil {
		return err
	}

	for _, result := range results {
		row := types.NewRow(
			types.MRP("path", result.Path),
			types.MRP("valid_json", result.Valid),
			types.MRP("error", result.Error),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

func NewCommand() (*cobra.Command, error) {
	cmd, err := NewGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
