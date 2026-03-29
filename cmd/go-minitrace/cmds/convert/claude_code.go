package convert

import (
	"context"
	"encoding/json"
	"os"
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
	"github.com/go-go-golems/go-minitrace/pkg/adapters/claudecode"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ConvertClaudeCodeCommand struct {
	*cmds.CommandDescription
}

type ConvertClaudeCodeSettings struct {
	SourceDir string `glazed:"source-dir"`
	OutputDir string `glazed:"output-dir"`
	DryRun    bool   `glazed:"dry-run"`
}

func NewConvertClaudeCodeGlazeCommand() (*ConvertClaudeCodeCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"claude-code",
		cmds.WithShort("Convert Claude Code sessions into minitrace"),
		cmds.WithLong(`
Convert Claude Code sessions into minitrace JSON files.

The current implementation supports:
  - Claude Code JSONL v2 transcripts
  - Claude Code dir-v1 tool-results sessions

Examples:
  go-minitrace convert claude-code --source-dir ~/.claude/projects --output-dir ./output
  go-minitrace convert claude-code --source-dir ~/.claude/projects --dry-run --output yaml
`),
		cmds.WithFlags(
			fields.New("source-dir", fields.TypeString, fields.WithDefault("~/.claude/projects"), fields.WithHelp("Claude Code projects directory")),
			fields.New("output-dir", fields.TypeString, fields.WithDefault("./output"), fields.WithHelp("Target minitrace archive directory")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Inspect sources without writing output")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ConvertClaudeCodeCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &ConvertClaudeCodeCommand{}

func (c *ConvertClaudeCodeCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &ConvertClaudeCodeSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}

	locators, err := claudecode.Discover(settings_.SourceDir)
	if err != nil {
		return err
	}
	subagentLocators, err := claudecode.DiscoverSubagents(settings_.SourceDir)
	if err != nil {
		return err
	}

	indexEntries := make([]*minitrace.SessionIndexEntry, 0, len(locators))
	indexEntriesByID := map[string]*minitrace.SessionIndexEntry{}
	parentSessionPaths := map[string]string{}
	for _, locator := range locators {
		session, err := claudecode.ConvertLocator(locator)
		if err != nil {
			return errors.Wrapf(err, "converting Claude Code session %s", locator.ID)
		}

		var sessionPath string
		if !settings_.DryRun {
			entry, err := minitrace.WriteSession(session, settings_.OutputDir)
			if err != nil {
				return errors.Wrapf(err, "writing minitrace session %s", locator.ID)
			}
			indexEntries = append(indexEntries, entry)
			indexEntriesByID[session.ID] = entry
			parentSessionPaths[session.ID] = entry.FilePath
			sessionPath = entry.FilePath
		}

		quality := ""
		if session.Quality != nil {
			quality = *session.Quality
		}
		row := types.NewRow(
			types.MRP("framework", "claude-code"),
			types.MRP("session_id", session.ID),
			types.MRP("session_kind", "primary"),
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
	}

	subagentIDsByParent := map[string][]string{}
	for _, locator := range subagentLocators {
		session, resolvedAgentID, err := claudecode.ConvertSubagentLocator(locator)
		if err != nil {
			return errors.Wrapf(err, "converting Claude Code subagent %s", locator.AgentID)
		}
		subagentIDsByParent[locator.ParentSessionID] = append(subagentIDsByParent[locator.ParentSessionID], resolvedAgentID)

		var sessionPath string
		if !settings_.DryRun {
			entry, err := minitrace.WriteSession(session, settings_.OutputDir)
			if err != nil {
				return errors.Wrapf(err, "writing Claude Code subagent %s", resolvedAgentID)
			}
			indexEntries = append(indexEntries, entry)
			indexEntriesByID[session.ID] = entry
			sessionPath = entry.FilePath
		}

		quality := ""
		if session.Quality != nil {
			quality = *session.Quality
		}
		row := types.NewRow(
			types.MRP("framework", "claude-code"),
			types.MRP("session_id", session.ID),
			types.MRP("session_kind", "subagent"),
			types.MRP("parent_session_id", locator.ParentSessionID),
			types.MRP("source_format", "jsonl-v2+subagent"),
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
	}

	if !settings_.DryRun {
		for parentSessionID, subagentIDs := range subagentIDsByParent {
			parentPath, ok := parentSessionPaths[parentSessionID]
			if !ok || parentPath == "" {
				continue
			}
			data, err := os.ReadFile(parentPath)
			if err != nil {
				return errors.Wrapf(err, "reading parent Claude Code session %s for backlink", parentSessionID)
			}
			var parentSession minitrace.Session
			if err := json.Unmarshal(data, &parentSession); err != nil {
				return errors.Wrapf(err, "decoding parent Claude Code session %s for backlink", parentSessionID)
			}
			claudecode.LinkParentSubagents(&parentSession, subagentIDs)

			payload, err := json.MarshalIndent(&parentSession, "", "  ")
			if err != nil {
				return errors.Wrapf(err, "encoding parent Claude Code session %s after backlink", parentSessionID)
			}
			payload = append(payload, '\n')
			if err := os.WriteFile(parentPath, payload, 0o644); err != nil {
				return errors.Wrapf(err, "rewriting parent Claude Code session %s after backlink", parentSessionID)
			}
			if entry, ok := indexEntriesByID[parentSessionID]; ok {
				info, err := os.Stat(parentPath)
				if err == nil {
					entry.FileSizeBytes = info.Size()
				}
			}
		}
	}

	if !settings_.DryRun {
		if err := minitrace.WriteManifests(indexEntries, settings_.OutputDir); err != nil {
			return errors.Wrap(err, "writing Claude Code manifests")
		}
		manifestRow := types.NewRow(
			types.MRP("framework", "claude-code"),
			types.MRP("manifest_path", filepath.Join(settings_.OutputDir, "manifest.json")),
			types.MRP("session_count", len(indexEntries)),
			types.MRP("dry_run", false),
		)
		return gp.AddRow(ctx, manifestRow)
	}

	return nil
}

func NewClaudeCodeCommand() (*cobra.Command, error) {
	cmd, err := NewConvertClaudeCodeGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
