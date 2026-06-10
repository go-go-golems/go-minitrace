package preview

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-minitrace/pkg/adapters"
	"github.com/go-go-golems/go-minitrace/pkg/adapters/claudecode"
	"github.com/go-go-golems/go-minitrace/pkg/adapters/codex"
	"github.com/go-go-golems/go-minitrace/pkg/adapters/pi"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
	"github.com/go-go-golems/go-minitrace/pkg/minitracejs"
)

type SessionCommand struct {
	*cmds.CommandDescription
}

type SessionSettings struct {
	SourceSession string `glazed:"source-session"`
	SourceDir     string `glazed:"source-dir"`
	Framework     string `glazed:"framework"`
	Latest        int    `glazed:"latest"`
	SampleLimit   int    `glazed:"sample-limit"`
	Privacy       string `glazed:"privacy"`
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
		cmds.WithShort("Preview Pi, Codex, Claude Code, or native minitrace sessions"),
		cmds.WithLong(`Preview source sessions without writing an archive.

The command uses the same auto-detection path as mt.importer().Preview(). It
loads native minitrace JSON directly and auto-converts supported Pi, Codex, and
Claude Code JSONL files before emitting compact structural previews.

Use --source-session for one file. Use --source-dir with --framework to preview
the latest discovered files for a framework. Directory mode emits one row per
session and records per-file conversion errors as error rows instead of aborting
the whole run.

Examples:
  go-minitrace preview session --source-session ~/.pi/agent/sessions/project/session.jsonl --output yaml
  go-minitrace preview session --source-session ~/.codex/sessions/2026/06/10/rollout.jsonl --output json
  go-minitrace preview session --source-dir ~/.claude/projects --framework claude-code --latest 3 --output json
  go-minitrace preview session --framework codex --latest 5 --privacy structural
`),
		cmds.WithFlags(
			fields.New("source-session", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Single session file to preview (native minitrace JSON or supported JSONL)")),
			fields.New("source-dir", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Session store directory for directory/latest-N preview mode")),
			fields.New("framework", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Framework for directory mode: pi, codex, claude-code, or claude")),
			fields.New("latest", fields.TypeInteger, fields.WithDefault(1), fields.WithHelp("Number of latest discovered sessions to preview in directory mode")),
			fields.New("sample-limit", fields.TypeInteger, fields.WithDefault(12), fields.WithHelp("Maximum sample turns/tools to include per preview")),
			fields.New("privacy", fields.TypeString, fields.WithDefault("snippets"), fields.WithHelp("Preview privacy: structural, snippets, or full")),
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
	settings_.Privacy = normalizePrivacy(settings_.Privacy)
	if settings_.SampleLimit <= 0 {
		settings_.SampleLimit = 12
	}

	if strings.TrimSpace(settings_.SourceSession) != "" {
		path, err := expandHome(settings_.SourceSession)
		if err != nil {
			return err
		}
		preview, err := previewSessionPath(path, previewOptions(settings_))
		if err != nil {
			return err
		}
		return emitPreviewRow(ctx, gp, path, preview, "")
	}

	paths, err := discoverPreviewPaths(settings_.Framework, settings_.SourceDir, settings_.Latest)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no sessions discovered for framework %q", settings_.Framework)
	}
	for _, path := range paths {
		preview, err := previewSessionPath(path, previewOptions(settings_))
		if err != nil {
			if rowErr := emitPreviewRow(ctx, gp, path, minitracejs.SessionPreview{}, err.Error()); rowErr != nil {
				return rowErr
			}
			continue
		}
		if err := emitPreviewRow(ctx, gp, path, preview, ""); err != nil {
			return err
		}
	}
	return nil
}

func previewOptions(settings_ *SessionSettings) minitracejs.PreviewOptions {
	return minitracejs.PreviewOptions{SampleLimit: settings_.SampleLimit, Privacy: settings_.Privacy}
}

func previewSessionPath(path string, options minitracejs.PreviewOptions) (minitracejs.SessionPreview, error) {
	loaded, err := minitracedb.LoadSessionFileAuto(path, minitracedb.LoadOptions{
		SourcePath:  path,
		SourceName:  filepath.Base(path),
		AutoConvert: true,
	})
	if err != nil {
		return minitracejs.SessionPreview{}, err
	}
	return minitracejs.PreviewLoadedSessionWithOptions(loaded, options), nil
}

func discoverPreviewPaths(framework, sourceDir string, latest int) ([]string, error) {
	if latest <= 0 {
		latest = 1
	}
	framework = normalizeFramework(framework)
	if framework == "" {
		return nil, fmt.Errorf("framework is required when source-session is not set")
	}
	if sourceDir == "" {
		sourceDir = defaultSourceDir(framework)
	}
	locators, err := discoverLocators(framework, sourceDir)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(locators))
	for _, locator := range locators {
		paths = append(paths, locator.SourcePath)
	}
	sort.SliceStable(paths, func(i, j int) bool {
		left, leftOK := modTime(paths[i])
		right, rightOK := modTime(paths[j])
		if leftOK && rightOK && !left.Equal(right) {
			return left.After(right)
		}
		return paths[i] > paths[j]
	})
	if latest < len(paths) {
		paths = paths[:latest]
	}
	return paths, nil
}

func discoverLocators(framework, sourceDir string) ([]adapters.SessionLocator, error) {
	switch framework {
	case "pi":
		return pi.Discover(sourceDir)
	case "codex":
		return codex.Discover(sourceDir)
	case "claude-code":
		return claudecode.Discover(sourceDir)
	default:
		return nil, fmt.Errorf("unsupported framework %q", framework)
	}
}

func defaultSourceDir(framework string) string {
	switch framework {
	case "pi":
		return "~/.pi/agent/sessions"
	case "codex":
		return "~/.codex"
	case "claude-code":
		return "~/.claude/projects"
	default:
		return ""
	}
}

func normalizeFramework(framework string) string {
	switch strings.TrimSpace(strings.ToLower(framework)) {
	case "claude", "claude_code", "claudecode":
		return "claude-code"
	default:
		return strings.TrimSpace(strings.ToLower(framework))
	}
}

func normalizePrivacy(privacy string) string {
	switch strings.TrimSpace(strings.ToLower(privacy)) {
	case "", "snippet", "snippets":
		return "snippets"
	case "structural", "full":
		return strings.TrimSpace(strings.ToLower(privacy))
	default:
		return "snippets"
	}
}

func modTime(path string) (timeValue comparableTime, ok bool) {
	info, err := os.Stat(path)
	if err != nil {
		return comparableTime{}, false
	}
	return comparableTime{unixNano: info.ModTime().UnixNano()}, true
}

type comparableTime struct{ unixNano int64 }

func (t comparableTime) After(other comparableTime) bool { return t.unixNano > other.unixNano }
func (t comparableTime) Equal(other comparableTime) bool { return t.unixNano == other.unixNano }

func emitPreviewRow(ctx context.Context, gp middlewares.Processor, sourcePath string, preview minitracejs.SessionPreview, errText string) error {
	return gp.AddRow(ctx, types.NewRow(
		types.MRP("source_path", sourcePath),
		types.MRP("error", errText),
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
		types.MRP("event_count", preview.EventCount),
		types.MRP("attachment_count", preview.AttachmentCount),
		types.MRP("subagent_count", preview.SubagentCount),
		types.MRP("role_counts", preview.RoleCounts),
		types.MRP("tool_counts", preview.ToolCounts),
		types.MRP("event_counts", preview.EventCounts),
		types.MRP("attachment_counts", preview.AttachmentCounts),
		types.MRP("sample_turns", preview.SampleTurns),
		types.MRP("sample_tools", preview.SampleTools),
		types.MRP("sample_events", preview.SampleEvents),
		types.MRP("sample_attachments", preview.SampleAttachments),
		types.MRP("diagnostics", preview.Diagnostics),
	))
}

func expandHome(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return filepath.Clean(path), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
}
