package discover

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
	"github.com/go-go-golems/go-minitrace/pkg/adapters/claudecode"
	"github.com/spf13/cobra"
)

type ClaudeCodeCommand struct {
	*cmds.CommandDescription
}

type ClaudeCodeSettings struct {
	SourceDir   string `glazed:"source-dir"`
	CwdContains string `glazed:"cwd-contains"`
	Since       string `glazed:"since"`
	ActiveSince string `glazed:"active-since"`
	// IncludeSubagents lists each kept session's subagent transcripts after it.
	IncludeSubagents bool `glazed:"include-subagents"`
}

func NewClaudeCodeGlazeCommand() (*ClaudeCodeCommand, error) {
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
		cmds.WithShort("Discover Claude Code sessions"),
		cmds.WithLong(`
Discover Claude Code sessions under a projects directory.

Discovery rules:
- JSONL v2 transcripts are preferred
- older dir-v1 sessions with tool-results/ are included when no JSONL transcript exists
- subagent transcripts (<session>/subagents/**/agent-<id>.jsonl) are listed only with
  --include-subagents, right after their parent session

With --include-subagents every row carries session_kind (primary or subagent),
parent_session_id and agent_type. A subagent follows its parent's selection:
--cwd-contains, --since and --active-since filter parents, and every subagent of
a kept parent is listed. The source_path column of subagent rows can go into a
--source-list for convert claude-code, which also expands subagents by default.

Examples:
  go-minitrace discover claude-code --source-dir ~/.claude/projects
  go-minitrace discover claude-code --source-dir /tmp/claude-projects --output json
  go-minitrace discover claude-code --cwd-contains my-repo --include-subagents --fields session_kind,id,parent_session_id,source_path
`),
		cmds.WithFlags(
			append(append([]*fields.Definition{
				fields.New(
					"source-dir",
					fields.TypeString,
					fields.WithDefault("~/.claude/projects"),
					fields.WithHelp("Claude Code projects directory"),
				),
				fields.New(
					"include-subagents",
					fields.TypeBool,
					fields.WithDefault(false),
					fields.WithHelp("Also list each kept session's subagent transcripts, with session_kind, parent_session_id and agent_type columns"),
				),
			}, filterFlags()...), activityFilterFlags()...)...,
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ClaudeCodeCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &ClaudeCodeCommand{}

func (c *ClaudeCodeCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &ClaudeCodeSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}

	since, err := parseSince(settings_.Since)
	if err != nil {
		return err
	}
	activeSince, err := parseSince(settings_.ActiveSince)
	if err != nil {
		return err
	}

	locators, err := claudecode.Discover(settings_.SourceDir)
	if err != nil {
		return err
	}
	for _, locator := range locators {
		if !keepLocator(locator, settings_.CwdContains, since, nil) {
			continue
		}
		if activeSince != nil && locator.FormatHint == "jsonl-v2" {
			if err := populateLastActivity(&locator, claudecode.LastActivityAt); err != nil {
				return err
			}
		}
		if !keepLocator(locator, settings_.CwdContains, since, activeSince) {
			continue
		}
		row := types.NewRow(
			types.MRP("id", locator.ID),
			types.MRP("format_hint", locator.FormatHint),
			types.MRP("source_path", locator.SourcePath),
			types.MRP("cwd", locator.Cwd),
			types.MRP("started_at", locator.StartedAt),
			types.MRP("last_activity_at", locator.LastActivityAt),
		)
		if !settings_.IncludeSubagents {
			if err := gp.AddRow(ctx, row); err != nil {
				return err
			}
			continue
		}
		row.Set("session_kind", "primary")
		row.Set("parent_session_id", "")
		row.Set("agent_type", "")
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
		subagents, err := claudecode.SubagentsForSession(locator)
		if err != nil {
			return err
		}
		for _, subagent := range subagents {
			if activeSince != nil {
				lastActivityAt, err := claudecode.LastActivityAt(subagent.SourcePath)
				if err != nil {
					return err
				}
				subagent.LastActivityAt = lastActivityAt
			}
			subRow := types.NewRow(
				types.MRP("id", subagent.AgentID),
				types.MRP("format_hint", "jsonl-v2+subagent"),
				types.MRP("source_path", subagent.SourcePath),
				types.MRP("cwd", subagent.Cwd),
				types.MRP("started_at", subagent.StartedAt),
				types.MRP("last_activity_at", subagent.LastActivityAt),
				types.MRP("session_kind", "subagent"),
				types.MRP("parent_session_id", locator.ID),
				types.MRP("agent_type", subagent.AgentType),
			)
			if err := gp.AddRow(ctx, subRow); err != nil {
				return err
			}
		}
	}
	return nil
}

func NewClaudeCodeCommand() (*cobra.Command, error) {
	cmd, err := NewClaudeCodeGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
