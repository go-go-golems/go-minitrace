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
	"github.com/go-go-golems/go-minitrace/pkg/adapters/claudeai"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ConvertClaudeAICommand struct {
	*cmds.CommandDescription
}

type ConvertClaudeAISettings struct {
	Source     string `glazed:"source"`
	OutputDir  string `glazed:"output-dir"`
	UUIDFilter string `glazed:"uuid-filter"`
	DryRun     bool   `glazed:"dry-run"`
}

func NewConvertClaudeAIGlazeCommand() (*ConvertClaudeAICommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"claude-ai",
		cmds.WithShort("Convert claude.ai export ZIPs into minitrace"),
		cmds.WithLong(`
Convert a claude.ai privacy export ZIP into minitrace JSON files.

Expected input is the data export ZIP from:
  Settings > Privacy > Export data

Examples:
  go-minitrace convert claude-ai --source ~/Downloads/data-2026-03-29-11-53-11-batch-0000.zip --output-dir ./output
  go-minitrace convert claude-ai --source ./data-export.zip --uuid-filter 7756135a,f7a8838f --dry-run --output json
`),
		cmds.WithFlags(
			fields.New("source", fields.TypeString, fields.WithHelp("Path to claude.ai data export ZIP")),
			fields.New("output-dir", fields.TypeString, fields.WithDefault("./output"), fields.WithHelp("Target minitrace archive directory")),
			fields.New("uuid-filter", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Comma-separated conversation UUID prefixes to include")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Inspect sources without writing output")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ConvertClaudeAICommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &ConvertClaudeAICommand{}

func (c *ConvertClaudeAICommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &ConvertClaudeAISettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}
	if strings.TrimSpace(settings_.Source) == "" {
		return errors.New("source is required")
	}

	uuidFilter := []string{}
	if strings.TrimSpace(settings_.UUIDFilter) != "" {
		for _, part := range strings.Split(settings_.UUIDFilter, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				uuidFilter = append(uuidFilter, part)
			}
		}
	}

	indexEntries := []*minitrace.SessionIndexEntry{}
	converted := 0
	skippedTrivial := 0
	err := claudeai.StreamConversations(settings_.Source, uuidFilter, func(conv map[string]any) error {
		messages, _ := conv["chat_messages"].([]any)
		if len(messages) < 2 {
			skippedTrivial++
			return nil
		}

		session, err := claudeai.ConvertConversation(conv, settings_.Source)
		if err != nil {
			return errors.Wrapf(err, "converting claude.ai conversation %s", conv["uuid"])
		}
		converted++

		var sessionPath string
		if !settings_.DryRun {
			entry, err := minitrace.WriteSession(session, settings_.OutputDir)
			if err != nil {
				return errors.Wrapf(err, "writing claude.ai session %s", session.ID)
			}
			indexEntries = append(indexEntries, entry)
			sessionPath = entry.FilePath
		}

		quality := ""
		if session.Quality != nil {
			quality = *session.Quality
		}
		row := types.NewRow(
			types.MRP("framework", "claude.ai"),
			types.MRP("session_id", session.ID),
			types.MRP("source_format", claudeai.SourceFormat),
			types.MRP("source_path", settings_.Source),
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
		types.MRP("framework", "claude.ai"),
		types.MRP("converted", converted),
		types.MRP("skipped_trivial", skippedTrivial),
		types.MRP("dry_run", settings_.DryRun),
	)
	if err := gp.AddRow(ctx, summaryRow); err != nil {
		return err
	}

	if !settings_.DryRun {
		if err := minitrace.WriteManifests(indexEntries, settings_.OutputDir); err != nil {
			return errors.Wrap(err, "writing claude.ai manifests")
		}
		manifestRow := types.NewRow(
			types.MRP("framework", "claude.ai"),
			types.MRP("manifest_path", filepath.Join(settings_.OutputDir, "manifest.json")),
			types.MRP("session_count", len(indexEntries)),
			types.MRP("dry_run", false),
		)
		return gp.AddRow(ctx, manifestRow)
	}

	return nil
}

func NewClaudeAICommand() (*cobra.Command, error) {
	cmd, err := NewConvertClaudeAIGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
