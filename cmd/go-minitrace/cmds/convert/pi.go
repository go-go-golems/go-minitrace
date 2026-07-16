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
	"github.com/go-go-golems/go-minitrace/pkg/adapters"
	"github.com/go-go-golems/go-minitrace/pkg/adapters/pi"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ConvertPiCommand struct {
	*cmds.CommandDescription
}

type ConvertPiSettings struct {
	SourceDir      string   `glazed:"source-dir"`
	SourceSessions []string `glazed:"source-session"`
	SourceList     string   `glazed:"source-list"`
	OutputDir      string   `glazed:"output-dir"`
	DryRun         bool     `glazed:"dry-run"`
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
  go-minitrace convert pi --source-list ./sessions.txt --output-dir ./output
`),
		cmds.WithFlags(
			fields.New("source-dir", fields.TypeString, fields.WithDefault("~/.pi/agent/sessions"), fields.WithHelp("Pi sessions directory")),
			fields.New("source-session", fields.TypeStringList, fields.WithDefault([]string{}), fields.WithHelp("Explicit Pi session JSONL files to convert instead of scanning --source-dir (repeatable)")),
			fields.New("source-list", fields.TypeString, fields.WithDefault(""), fields.WithHelp("File with newline-separated Pi session paths; blank lines and # comments are ignored")),
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

	explicitPaths, err := collectSourceSessions(settings_.SourceSessions, settings_.SourceList)
	if err != nil {
		return err
	}
	var locators []adapters.SessionLocator
	if len(explicitPaths) > 0 {
		locators = make([]adapters.SessionLocator, 0, len(explicitPaths))
		for _, path := range explicitPaths {
			locator, err := pi.LocateSession(path)
			if err != nil {
				return err
			}
			locators = append(locators, locator)
		}
	} else {
		locators, err = pi.Discover(settings_.SourceDir)
		if err != nil {
			return err
		}
	}

	type convertedSource struct {
		locator adapters.SessionLocator
		session *minitrace.Session
	}
	converted := make([]convertedSource, 0, len(locators))
	for _, locator := range locators {
		session, err := pi.ConvertLocator(locator)
		if err != nil {
			return errors.Wrapf(err, "converting Pi source %s before publication", locator.SourcePath)
		}
		if err := applySourceFingerprint(session, locator.SourcePath); err != nil {
			return errors.Wrapf(err, "fingerprinting Pi source %s", locator.SourcePath)
		}
		converted = append(converted, convertedSource{locator: locator, session: session})
	}

	indexEntries := make([]*minitrace.SessionIndexEntry, 0, len(converted))
	publicationByID := map[string]minitrace.PublicationResult{}
	if !settings_.DryRun {
		sessions := make([]*minitrace.Session, 0, len(converted))
		for _, source := range converted {
			sessions = append(sessions, source.session)
		}
		publications, err := minitrace.PublishSessionBatch(sessions, settings_.OutputDir, minitrace.CollisionError)
		if err != nil {
			return errors.Wrap(err, "publishing staged Pi batch")
		}
		for _, publication := range publications {
			publicationByID[publication.SessionID] = publication
			indexEntries = append(indexEntries, publication.Entry)
		}
	}
	for _, source := range converted {
		publication := publicationByID[source.session.ID]
		if err := emitConvertedSession(ctx, gp, source.session, source.locator.SourcePath, source.locator.FormatHint, publication, settings_.DryRun); err != nil {
			return err
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

func emitConvertedSession(ctx context.Context, gp middlewares.Processor, session *minitrace.Session, sourcePath, sourceFormat string, publication minitrace.PublicationResult, dryRun bool) error {
	sessionPath := ""
	status := "dry-run"
	if publication.Entry != nil {
		sessionPath = publication.Entry.FilePath
		status = string(publication.Status)
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
		types.MRP("status", status),
	)
	return gp.AddRow(ctx, row)
}

func NewPiCommand() (*cobra.Command, error) {
	cmd, err := NewConvertPiGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
