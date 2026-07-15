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
	"github.com/go-go-golems/go-minitrace/pkg/adapters/codex"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ConvertCodexCommand struct {
	*cmds.CommandDescription
}

type ConvertCodexSettings struct {
	SourceDir      string   `glazed:"source-dir"`
	SourceSessions []string `glazed:"source-session"`
	SourceList     string   `glazed:"source-list"`
	OutputDir      string   `glazed:"output-dir"`
	Collision      string   `glazed:"collision"`
	DryRun         bool     `glazed:"dry-run"`
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
		cmds.WithShort("Convert Codex sessions into minitrace"),
		cmds.WithLong(`
Convert Codex sessions into minitrace JSON files.

The current implementation supports:
  - session JSONL persisted under ~/.codex/sessions/
  - exec JSONL produced by codex exec --json

Examples:
  go-minitrace convert codex --source-dir ~/.codex --output-dir ./output
  go-minitrace convert codex --source-dir ~/.codex --dry-run --output json
  go-minitrace convert codex --source-session ~/.codex/sessions/2026/07/05/rollout-abc.jsonl --output-dir ./output
  go-minitrace convert codex --source-list ./sessions.txt --output-dir ./output
`),
		cmds.WithFlags(
			fields.New("source-dir", fields.TypeString, fields.WithDefault("~/.codex"), fields.WithHelp("Codex home directory")),
			fields.New("source-session", fields.TypeStringList, fields.WithDefault([]string{}), fields.WithHelp("Explicit Codex session JSONL files to convert instead of scanning --source-dir (repeatable)")),
			fields.New("source-list", fields.TypeString, fields.WithDefault(""), fields.WithHelp("File with newline-separated Codex session paths; blank lines and # comments are ignored")),
			fields.New("output-dir", fields.TypeString, fields.WithDefault("./output"), fields.WithHelp("Target minitrace archive directory")),
			fields.New("collision", fields.TypeString, fields.WithDefault(string(minitrace.CollisionError)), fields.WithHelp("Archive ID collision policy: error (default) or replace")),
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

	explicitPaths, err := collectSourceSessions(settings_.SourceSessions, settings_.SourceList)
	if err != nil {
		return err
	}
	var locators []adapters.SessionLocator
	if len(explicitPaths) > 0 {
		locators = make([]adapters.SessionLocator, 0, len(explicitPaths))
		for _, path := range explicitPaths {
			locator, err := codex.LocateSession(path)
			if err != nil {
				return err
			}
			locators = append(locators, locator)
		}
	} else {
		locators, err = codex.Discover(settings_.SourceDir)
		if err != nil {
			return err
		}
	}

	collisionPolicy := minitrace.CollisionPolicy(settings_.Collision)
	if collisionPolicy != minitrace.CollisionError && collisionPolicy != minitrace.CollisionReplace {
		return errors.Errorf("unsupported collision policy %q", settings_.Collision)
	}

	indexEntries := make([]*minitrace.SessionIndexEntry, 0, len(locators))
	convertedCount := 0
	failedCount := 0
	for _, locator := range locators {
		session, err := codex.ConvertLocator(locator)
		if err != nil {
			failedCount++
			if rowErr := emitFailedSessionRow(ctx, gp, "codex", locator.ID, "", locator.FormatHint, locator.SourcePath, settings_.DryRun, err); rowErr != nil {
				return rowErr
			}
			continue
		}

		var sessionPath string
		if !settings_.DryRun {
			entry, err := minitrace.WriteSessionWithCollisionPolicy(session, settings_.OutputDir, collisionPolicy)
			if err != nil {
				return errors.Wrapf(err, "writing minitrace session %s", locator.ID)
			}
			indexEntries = append(indexEntries, entry)
			sessionPath = entry.FilePath
		}

		quality := ""
		if session.Quality != nil {
			quality = *session.Quality
		}
		row := types.NewRow(
			types.MRP("framework", "codex"),
			types.MRP("session_id", session.ID),
			types.MRP("source_format", locator.FormatHint),
			types.MRP("source_path", locator.SourcePath),
			types.MRP("turn_count", session.Metrics.TurnCount),
			types.MRP("tool_call_count", session.Metrics.ToolCallCount),
			types.MRP("quality", quality),
			types.MRP("classification", session.Classification),
			types.MRP("dry_run", settings_.DryRun),
			types.MRP("session_path", sessionPath),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
		convertedCount++
	}
	if failedCount > 0 && convertedCount == 0 {
		return errors.Errorf("all %d Codex sessions failed to convert", failedCount)
	}

	if !settings_.DryRun {
		if err := minitrace.WriteManifests(indexEntries, settings_.OutputDir); err != nil {
			return errors.Wrap(err, "writing Codex manifests")
		}
		manifestRow := types.NewRow(
			types.MRP("framework", "codex"),
			types.MRP("manifest_path", filepath.Join(settings_.OutputDir, "manifest.json")),
			types.MRP("session_count", len(indexEntries)),
			types.MRP("dry_run", false),
		)
		return gp.AddRow(ctx, manifestRow)
	}

	return nil
}

func NewCodexCommand() (*cobra.Command, error) {
	cmd, err := NewConvertCodexGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
