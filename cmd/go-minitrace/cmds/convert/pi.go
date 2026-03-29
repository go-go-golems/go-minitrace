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
	"github.com/go-go-golems/go-minitrace/pkg/adapters/pi"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ConvertPiCommand struct {
	*cmds.CommandDescription
}

type ConvertPiSettings struct {
	SourceDir     string `glazed:"source-dir"`
	SourceSession string `glazed:"source-session"`
	OutputDir     string `glazed:"output-dir"`
	DryRun        bool   `glazed:"dry-run"`
}

func NewConvertPiGlazeCommand() (*ConvertPiCommand, error) {
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
		cmds.WithShort("Convert Pi sessions into minitrace"),
		cmds.WithLong(`
Convert Pi local JSONL sessions into minitrace JSON files.

Examples:
  go-minitrace convert pi --source-dir ~/.pi/agent/sessions --output-dir ./output
  go-minitrace convert pi --source-session ~/.pi/agent/sessions/project/example.jsonl --dry-run --output json
`),
		cmds.WithFlags(
			fields.New("source-dir", fields.TypeString, fields.WithDefault("~/.pi/agent/sessions"), fields.WithHelp("Pi sessions directory")),
			fields.New("source-session", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Single Pi session JSONL file to convert")),
			fields.New("output-dir", fields.TypeString, fields.WithDefault("./output"), fields.WithHelp("Target minitrace archive directory")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Inspect sources without writing output")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ConvertPiCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &ConvertPiCommand{}

func (c *ConvertPiCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &ConvertPiSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}

	indexEntries := []*minitrace.SessionIndexEntry{}
	if settings_.SourceSession != "" {
		session, err := pi.ConvertFile(settings_.SourceSession)
		if err != nil {
			return errors.Wrap(err, "converting Pi source session")
		}
		entry, err := emitConvertedSession(ctx, gp, session, settings_.SourceSession, "jsonl-v3", settings_.OutputDir, settings_.DryRun)
		if err != nil {
			return err
		}
		if entry != nil {
			indexEntries = append(indexEntries, entry)
		}
	} else {
		locators, err := pi.Discover(settings_.SourceDir)
		if err != nil {
			return err
		}
		for _, locator := range locators {
			session, err := pi.ConvertLocator(locator)
			if err != nil {
				return errors.Wrapf(err, "converting Pi session %s", locator.ID)
			}
			entry, err := emitConvertedSession(ctx, gp, session, locator.SourcePath, locator.FormatHint, settings_.OutputDir, settings_.DryRun)
			if err != nil {
				return err
			}
			if entry != nil {
				indexEntries = append(indexEntries, entry)
			}
		}
	}

	if !settings_.DryRun {
		if err := minitrace.WriteManifests(indexEntries, settings_.OutputDir); err != nil {
			return errors.Wrap(err, "writing Pi manifests")
		}
		manifestRow := types.NewRow(
			types.MRP("framework", "pi"),
			types.MRP("manifest_path", filepath.Join(settings_.OutputDir, "manifest.json")),
			types.MRP("session_count", len(indexEntries)),
			types.MRP("dry_run", false),
		)
		return gp.AddRow(ctx, manifestRow)
	}

	return nil
}

func emitConvertedSession(ctx context.Context, gp middlewares.Processor, session *minitrace.Session, sourcePath, sourceFormat, outputDir string, dryRun bool) (*minitrace.SessionIndexEntry, error) {
	var entry *minitrace.SessionIndexEntry
	var err error
	sessionPath := ""
	if !dryRun {
		entry, err = minitrace.WriteSession(session, outputDir)
		if err != nil {
			return nil, errors.Wrapf(err, "writing Pi session %s", session.ID)
		}
		sessionPath = entry.FilePath
	}

	quality := ""
	if session.Quality != nil {
		quality = *session.Quality
	}
	row := types.NewRow(
		types.MRP("framework", "pi"),
		types.MRP("session_id", session.ID),
		types.MRP("source_format", sourceFormat),
		types.MRP("source_path", sourcePath),
		types.MRP("turn_count", session.Metrics.TurnCount),
		types.MRP("tool_call_count", session.Metrics.ToolCallCount),
		types.MRP("quality", quality),
		types.MRP("classification", session.Classification),
		types.MRP("dry_run", dryRun),
		types.MRP("session_path", sessionPath),
	)
	if err := gp.AddRow(ctx, row); err != nil {
		return nil, err
	}

	return entry, nil
}

func NewPiCommand() (*cobra.Command, error) {
	cmd, err := NewConvertPiGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
