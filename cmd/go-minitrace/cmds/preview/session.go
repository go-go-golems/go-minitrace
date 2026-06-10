package preview

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
	"github.com/go-go-golems/go-minitrace/pkg/minitracejs"
)

type SessionCommand struct {
	*cmds.CommandDescription
}

type SessionSettings struct {
	SourceSession string `glazed:"source-session"`
}

func NewSessionCommand() (*SessionCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}
	desc := cmds.NewCommandDescription(
		"session",
		cmds.WithShort("Preview a Pi, Codex, Claude Code, or native minitrace session"),
		cmds.WithLong(`Preview one source session without writing an archive.

The command uses the same auto-detection path as mt.importer().Preview(). It
loads native minitrace JSON directly and auto-converts supported Pi, Codex, and
Claude Code JSONL files before emitting a compact structural preview.

Examples:
  go-minitrace preview session --source-session ~/.pi/agent/sessions/project/session.jsonl --output yaml
  go-minitrace preview session --source-session ~/.codex/sessions/2026/06/10/rollout.jsonl --output json
  go-minitrace preview session --source-session ~/.claude/projects/project/session.jsonl
`),
		cmds.WithFlags(
			fields.New("source-session", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Session file to preview (native minitrace JSON or supported JSONL)")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)
	return &SessionCommand{CommandDescription: desc}, nil
}

var _ cmds.GlazeCommand = &SessionCommand{}

func (c *SessionCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings_ := &SessionSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}
	if settings_.SourceSession == "" {
		return fmt.Errorf("source-session is required")
	}
	loaded, err := minitracedb.LoadSessionFileAuto(settings_.SourceSession, minitracedb.LoadOptions{
		SourcePath:  settings_.SourceSession,
		SourceName:  filepath.Base(settings_.SourceSession),
		AutoConvert: true,
	})
	if err != nil {
		return err
	}
	preview := minitracejs.PreviewLoadedSession(loaded)
	return gp.AddRow(ctx, types.NewRow(
		types.MRP("session_id", preview.SessionID),
		types.MRP("format", preview.Format),
		types.MRP("adapter", preview.Adapter),
		types.MRP("title", preview.Title),
		types.MRP("agent_framework", preview.AgentFramework),
		types.MRP("model", preview.Model),
		types.MRP("working_directory", preview.WorkingDir),
		types.MRP("has_system_prompt", preview.HasSystemPrompt),
		types.MRP("has_thinking", preview.HasThinking),
		types.MRP("has_image_signals", preview.HasImageSignals),
		types.MRP("turn_count", preview.TurnCount),
		types.MRP("tool_call_count", preview.ToolCallCount),
		types.MRP("subagent_count", preview.SubagentCount),
		types.MRP("role_counts", preview.RoleCounts),
		types.MRP("tool_counts", preview.ToolCounts),
		types.MRP("sample_turns", preview.SampleTurns),
		types.MRP("sample_tools", preview.SampleTools),
		types.MRP("diagnostics", preview.Diagnostics),
	))
}
