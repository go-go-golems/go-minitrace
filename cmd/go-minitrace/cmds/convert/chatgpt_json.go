package convert

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/common"
	"github.com/go-go-golems/go-minitrace/pkg/adapters/chatgpt"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ConvertChatGPTJSONCommand struct {
	*cmds.CommandDescription
}

type ConvertChatGPTJSONSettings struct {
	SourceDir string `glazed:"source-dir"`
	OutputDir string `glazed:"output-dir"`
	IDFilter  string `glazed:"id-filter"`
	DryRun    bool   `glazed:"dry-run"`
}

func NewConvertChatGPTJSONGlazeCommand() (*ConvertChatGPTJSONCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"chatgpt-json",
		cmds.WithShort("Convert alternate ChatGPT per-conversation JSON exports into minitrace"),
		cmds.WithLong(`
Convert a directory of per-conversation ChatGPT JSON transcript exports into minitrace JSON files.

This command is for the richer one-file-per-conversation export format, not the standard account export ZIP.

Examples:
  go-minitrace convert chatgpt-json --source-dir /tmp/chatgpt-exports --output-dir ./output
  go-minitrace convert chatgpt-json --source-dir /tmp/chatgpt-exports --id-filter 69c7,69c8 --dry-run --output json
`),
		cmds.WithFlags(
			fields.New("source-dir", fields.TypeString, fields.WithHelp("Directory containing one ChatGPT conversation JSON per file")),
			fields.New("output-dir", fields.TypeString, fields.WithDefault("./output"), fields.WithHelp("Target minitrace archive directory")),
			fields.New("id-filter", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Comma-separated conversation ID prefixes to include")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Inspect sources without writing output")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ConvertChatGPTJSONCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &ConvertChatGPTJSONCommand{}

func (c *ConvertChatGPTJSONCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &ConvertChatGPTJSONSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}
	if strings.TrimSpace(settings_.SourceDir) == "" {
		return errors.New("source-dir is required")
	}

	idFilter := []string{}
	if strings.TrimSpace(settings_.IDFilter) != "" {
		for _, part := range strings.Split(settings_.IDFilter, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				idFilter = append(idFilter, part)
			}
		}
	}

	indexEntries := []*minitrace.SessionIndexEntry{}
	converted := 0
	skippedTrivial := 0
	err := chatgpt.StreamTranscriptFiles(settings_.SourceDir, idFilter, func(conv map[string]any, path string) error {
		session, err := chatgpt.ConvertTranscriptConversation(conv, path)
		if err != nil {
			return errors.Wrapf(err, "converting ChatGPT transcript %s", path)
		}
		if len(session.Turns) < 2 {
			skippedTrivial++
			return nil
		}
		converted++

		var sessionPath string
		if !settings_.DryRun {
			entry, err := minitrace.WriteSession(session, settings_.OutputDir)
			if err != nil {
				return errors.Wrapf(err, "writing ChatGPT transcript session %s", session.ID)
			}
			indexEntries = append(indexEntries, entry)
			sessionPath = entry.FilePath
		}

		quality := ""
		if session.Quality != nil {
			quality = *session.Quality
		}
		row := types.NewRow(
			types.MRP("framework", "chatgpt-web"),
			types.MRP("session_id", session.ID),
			types.MRP("source_format", chatgpt.TranscriptSourceFormat),
			types.MRP("source_path", path),
			types.MRP("turn_count", session.Metrics.TurnCount),
			types.MRP("tool_call_count", session.Metrics.ToolCallCount),
			types.MRP("quality", quality),
			types.MRP("classification", session.Classification),
			types.MRP("dry_run", settings_.DryRun),
			types.MRP("session_path", sessionPath),
		)
		return gp.AddRow(ctx, row)
	})
	if err != nil {
		return err
	}

	summaryRow := types.NewRow(
		types.MRP("framework", "chatgpt-web"),
		types.MRP("source_format", chatgpt.TranscriptSourceFormat),
		types.MRP("converted", converted),
		types.MRP("skipped_trivial", skippedTrivial),
		types.MRP("dry_run", settings_.DryRun),
	)
	if err := gp.AddRow(ctx, summaryRow); err != nil {
		return err
	}

	if !settings_.DryRun {
		if err := minitrace.WriteManifests(indexEntries, settings_.OutputDir); err != nil {
			return errors.Wrap(err, "writing ChatGPT transcript manifests")
		}
		manifestRow := types.NewRow(
			types.MRP("framework", "chatgpt-web"),
			types.MRP("source_format", chatgpt.TranscriptSourceFormat),
			types.MRP("manifest_path", filepath.Join(settings_.OutputDir, "manifest.json")),
			types.MRP("session_count", len(indexEntries)),
			types.MRP("dry_run", false),
		)
		return gp.AddRow(ctx, manifestRow)
	}

	return nil
}

func NewChatGPTJSONCommand() (*cobra.Command, error) {
	cmd, err := NewConvertChatGPTJSONGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
