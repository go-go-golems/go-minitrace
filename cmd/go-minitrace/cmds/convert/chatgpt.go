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

type ConvertChatGPTCommand struct {
	*cmds.CommandDescription
}

type ConvertChatGPTSettings struct {
	Source    string `glazed:"source"`
	OutputDir string `glazed:"output-dir"`
	IDFilter  string `glazed:"id-filter"`
	DryRun    bool   `glazed:"dry-run"`
}

func NewConvertChatGPTGlazeCommand() (*ConvertChatGPTCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"chatgpt",
		cmds.WithShort("Convert ChatGPT export ZIPs into minitrace"),
		cmds.WithLong(`
Convert a ChatGPT data export ZIP into minitrace JSON files.

Expected input is the export ZIP from:
  Settings > Data controls > Export data

Examples:
  go-minitrace convert chatgpt --source ~/Downloads/chatgpt-export.zip --output-dir ./output
  go-minitrace convert chatgpt --source ./data-export.zip --id-filter 670c0928,abc123 --dry-run --output json
`),
		cmds.WithFlags(
			fields.New("source", fields.TypeString, fields.WithHelp("Path to ChatGPT data export ZIP")),
			fields.New("output-dir", fields.TypeString, fields.WithDefault("./output"), fields.WithHelp("Target minitrace archive directory")),
			fields.New("id-filter", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Comma-separated conversation ID prefixes to include")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Inspect sources without writing output")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ConvertChatGPTCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &ConvertChatGPTCommand{}

func (c *ConvertChatGPTCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &ConvertChatGPTSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}
	if strings.TrimSpace(settings_.Source) == "" {
		return errors.New("source is required")
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
	err := chatgpt.StreamConversations(settings_.Source, idFilter, func(conv map[string]any) error {
		if chatgpt.CountRealMessages(conv) < 2 {
			skippedTrivial++
			return nil
		}

		session, err := chatgpt.ConvertConversation(conv, settings_.Source)
		if err != nil {
			return errors.Wrapf(err, "converting ChatGPT conversation %s", conv["conversation_id"])
		}
		converted++

		var sessionPath string
		if !settings_.DryRun {
			entry, err := minitrace.WriteSession(session, settings_.OutputDir)
			if err != nil {
				return errors.Wrapf(err, "writing ChatGPT session %s", session.ID)
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
			types.MRP("source_format", chatgpt.SourceFormat),
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
		types.MRP("framework", "chatgpt-web"),
		types.MRP("converted", converted),
		types.MRP("skipped_trivial", skippedTrivial),
		types.MRP("dry_run", settings_.DryRun),
	)
	if err := gp.AddRow(ctx, summaryRow); err != nil {
		return err
	}

	if !settings_.DryRun {
		if err := minitrace.WriteManifests(indexEntries, settings_.OutputDir); err != nil {
			return errors.Wrap(err, "writing ChatGPT manifests")
		}
		manifestRow := types.NewRow(
			types.MRP("framework", "chatgpt-web"),
			types.MRP("manifest_path", filepath.Join(settings_.OutputDir, "manifest.json")),
			types.MRP("session_count", len(indexEntries)),
			types.MRP("dry_run", false),
		)
		return gp.AddRow(ctx, manifestRow)
	}

	return nil
}

func NewChatGPTCommand() (*cobra.Command, error) {
	cmd, err := NewConvertChatGPTGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
