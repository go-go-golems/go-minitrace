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
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	validatepkg "github.com/go-go-golems/go-minitrace/pkg/validate"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ValidateCommand struct {
	*cmds.CommandDescription
}

type ValidateSettings struct {
	Path        string   `glazed:"path"`
	Recursive   bool     `glazed:"recursive"`
	Archive     bool     `glazed:"archive"`
	Checks      []string `glazed:"check"`
	Rebuild     bool     `glazed:"rebuild-manifests"`
	FailOnError bool     `glazed:"fail-on-error"`
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

By default this command validates JSON syntax and session structure. Use
--archive to validate a complete minitrace archive, including filename/payload
identity, period placement, manifests, orphan entries, source fingerprints,
and conversion receipts.

Examples:
  go-minitrace validate --path ./examples/v0.2.0 --recursive
  go-minitrace validate --path ./session.minitrace.json --output json
  go-minitrace validate --path ./output --archive --output json
`),
		cmds.WithFlags(
			fields.New("path", fields.TypeString, fields.WithRequired(true), fields.WithHelp("File, directory, archive root, or path below an archive root")),
			fields.New("recursive", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Recursively scan directories for JSON files")),
			fields.New("archive", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Run native archive, manifest, source, and receipt integrity checks")),
			fields.New("check", fields.TypeStringList, fields.WithDefault([]string{}), fields.WithHelp("Archive checks to run: archive, manifest, source, receipt (repeatable; default all)")),
			fields.New("rebuild-manifests", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Atomically rebuild root and period manifests from archive files before validation")),
			fields.New("fail-on-error", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Return non-zero when archive validation emits error findings")),
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

	if settings_.Archive {
		if settings_.Rebuild {
			root, err := validatepkg.DetectArchiveRoot(settings_.Path)
			if err != nil {
				return err
			}
			if err := minitrace.WriteManifests(nil, root); err != nil {
				return errors.Wrap(err, "rebuilding archive manifests")
			}
		}
		findings, err := validatepkg.ValidateArchive(settings_.Path, settings_.Checks)
		if err != nil {
			return err
		}
		hasErrors := false
		for _, finding := range findings {
			if finding.Severity == validatepkg.SeverityError {
				hasErrors = true
			}
			row := types.NewRow(
				types.MRP("code", finding.Code),
				types.MRP("severity", finding.Severity),
				types.MRP("path", finding.Path),
				types.MRP("session_id", finding.SessionID),
				types.MRP("detail", finding.Detail),
			)
			if err := gp.AddRow(ctx, row); err != nil {
				return err
			}
		}
		if hasErrors && settings_.FailOnError {
			return errors.New("archive validation found error findings")
		}
		return nil
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
