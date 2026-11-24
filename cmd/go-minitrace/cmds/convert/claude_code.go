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
	"github.com/go-go-golems/go-minitrace/pkg/adapters/claudecode"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ConvertClaudeCodeCommand struct {
	*cmds.CommandDescription
}

type ConvertClaudeCodeSettings struct {
	SourceDir      string   `glazed:"source-dir"`
	SourceSessions []string `glazed:"source-session"`
	SourceList     string   `glazed:"source-list"`
	OutputDir      string   `glazed:"output-dir"`
	DryRun         bool     `glazed:"dry-run"`
	NoSubagents    bool     `glazed:"no-subagents"`
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
  - subagent transcripts (<session>/subagents/**/agent-<id>.jsonl), converted as
    separate sessions linked to their parent

Explicit sources (--source-session, --source-list) also convert the subagents
stored next to each listed transcript unless --no-subagents is set. A listed
agent-<id>.jsonl path converts as a subagent of the session directory that
contains it, and a path listed twice converts once.

Examples:
  go-minitrace convert claude-code --source-dir ~/.claude/projects --output-dir ./output
  go-minitrace convert claude-code --source-dir ~/.claude/projects --dry-run --output yaml
  go-minitrace convert claude-code --source-session ~/.claude/projects/my-project/0123456789abcdef0123456789abcdef.jsonl --output-dir ./output
  go-minitrace convert claude-code --source-list ./sessions.txt --output-dir ./output
  go-minitrace convert claude-code --source-list ./sessions.txt --no-subagents --output-dir ./output
`),
		cmds.WithFlags(
			fields.New("source-dir", fields.TypeString, fields.WithDefault("~/.claude/projects"), fields.WithHelp("Claude Code projects directory")),
			fields.New("source-session", fields.TypeStringList, fields.WithDefault([]string{}), fields.WithHelp("Explicit Claude Code session transcripts to convert instead of scanning --source-dir (repeatable)")),
			fields.New("source-list", fields.TypeString, fields.WithDefault(""), fields.WithHelp("File with newline-separated Claude Code session paths; blank lines and # comments are ignored")),
			fields.New("output-dir", fields.TypeString, fields.WithDefault("./output"), fields.WithHelp("Target minitrace archive directory")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Inspect sources without writing output")),
			fields.New("no-subagents", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Do not convert the subagent transcripts stored next to explicit --source-session/--source-list transcripts")),
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

	explicitPaths, err := collectSourceSessions(settings_.SourceSessions, settings_.SourceList)
	if err != nil {
		return err
	}
	var locators []adapters.SessionLocator
	var subagentLocators []claudecode.SubagentLocator
	if len(explicitPaths) > 0 {
		locators, subagentLocators, err = locateExplicitClaudeSources(explicitPaths, !settings_.NoSubagents)
		if err != nil {
			return err
		}
	} else {
		locators, err = claudecode.Discover(settings_.SourceDir)
		if err != nil {
			return err
		}
		subagentLocators, err = claudecode.DiscoverSubagents(settings_.SourceDir)
		if err != nil {
			return err
		}
	}

	type convertedPrimary struct {
		locator adapters.SessionLocator
		session *minitrace.Session
	}
	type convertedSubagent struct {
		locator claudecode.SubagentLocator
		session *minitrace.Session
	}
	primaries := make([]convertedPrimary, 0, len(locators))
	primaryByID := map[string]*minitrace.Session{}
	for _, locator := range locators {
		session, err := claudecode.ConvertLocator(locator)
		if err != nil {
			return errors.Wrapf(err, "converting Claude Code source %s before publication", locator.SourcePath)
		}
		if err := applySourceFingerprint(session, locator.SourcePath); err != nil {
			return errors.Wrapf(err, "fingerprinting Claude Code source %s", locator.SourcePath)
		}
		primaries = append(primaries, convertedPrimary{locator: locator, session: session})
		primaryByID[session.ID] = session
	}

	subagents := make([]convertedSubagent, 0, len(subagentLocators))
	linksByParent := map[string][]claudecode.SubagentLink{}
	for _, locator := range subagentLocators {
		session, link, err := claudecode.ConvertSubagentLocator(locator)
		if err != nil {
			return errors.Wrapf(err, "converting Claude Code subagent %s before publication", locator.SourcePath)
		}
		if err := applySourceFingerprint(session, locator.SourcePath); err != nil {
			return errors.Wrapf(err, "fingerprinting Claude Code subagent %s", locator.SourcePath)
		}
		subagents = append(subagents, convertedSubagent{locator: locator, session: session})
		linksByParent[locator.ParentSessionID] = append(linksByParent[locator.ParentSessionID], link)
	}
	for parentSessionID, links := range linksByParent {
		if parent := primaryByID[parentSessionID]; parent != nil {
			claudecode.LinkParentSubagents(parent, links)
		}
	}

	indexEntries := make([]*minitrace.SessionIndexEntry, 0, len(primaries)+len(subagents))
	publicationByID := map[string]minitrace.PublicationResult{}
	if !settings_.DryRun {
		sessions := make([]*minitrace.Session, 0, len(primaries)+len(subagents))
		for _, source := range primaries {
			sessions = append(sessions, source.session)
		}
		for _, source := range subagents {
			sessions = append(sessions, source.session)
		}
		publications, err := minitrace.PublishSessionBatch(sessions, settings_.OutputDir, minitrace.CollisionError)
		if err != nil {
			return errors.Wrap(err, "publishing staged Claude Code batch")
		}
		for _, publication := range publications {
			publicationByID[publication.SessionID] = publication
			indexEntries = append(indexEntries, publication.Entry)
		}
	}

	emitRow := func(session *minitrace.Session, kind, parentID, sourceFormat, sourcePath string) error {
		quality := ""
		if session.Quality != nil {
			quality = *session.Quality
		}
		sessionPath := ""
		status := "dry-run"
		if publication, ok := publicationByID[session.ID]; ok {
			sessionPath = publication.Entry.FilePath
			status = string(publication.Status)
		}
		row := types.NewRow(
			types.MRP("framework", "claude-code"),
			types.MRP("session_id", session.ID),
			types.MRP("session_kind", kind),
			types.MRP("parent_session_id", parentID),
			types.MRP("source_format", sourceFormat),
			types.MRP("source_path", sourcePath),
			types.MRP("turn_count", session.Metrics.TurnCount),
			types.MRP("tool_call_count", session.Metrics.ToolCallCount),
			types.MRP("quality", quality),
			types.MRP("classification", session.Classification),
			types.MRP("dry_run", settings_.DryRun),
			types.MRP("session_path", sessionPath),
			types.MRP("status", status),
		)
		return gp.AddRow(ctx, row)
	}
	for _, source := range primaries {
		if err := emitRow(source.session, "primary", "", source.locator.FormatHint, source.locator.SourcePath); err != nil {
			return err
		}
	}
	for _, source := range subagents {
		if err := emitRow(source.session, "subagent", source.locator.ParentSessionID, "jsonl-v2+subagent", source.locator.SourcePath); err != nil {
			return err
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

// locateExplicitClaudeSources resolves explicit paths into primary and
// subagent locators. Listed agent-<id>.jsonl files are subagents; with
// expandSubagents each primary also contributes the subagents stored next to
// it. A transcript reached twice is converted once.
func locateExplicitClaudeSources(paths []string, expandSubagents bool) ([]adapters.SessionLocator, []claudecode.SubagentLocator, error) {
	locators := make([]adapters.SessionLocator, 0, len(paths))
	subagents := []claudecode.SubagentLocator{}
	seenSubagents := map[string]struct{}{}
	addSubagent := func(locator claudecode.SubagentLocator) {
		key := filepath.Clean(locator.SourcePath)
		if _, ok := seenSubagents[key]; ok {
			return
		}
		seenSubagents[key] = struct{}{}
		subagents = append(subagents, locator)
	}
	for _, path := range paths {
		if claudecode.IsSubagentTranscriptPath(path) {
			locator, err := claudecode.LocateSubagent(path)
			if err != nil {
				return nil, nil, err
			}
			addSubagent(locator)
			continue
		}
		locator, err := claudecode.LocateSession(path)
		if err != nil {
			return nil, nil, err
		}
		locators = append(locators, locator)
		if !expandSubagents {
			continue
		}
		children, err := claudecode.SubagentsForSession(locator)
		if err != nil {
			return nil, nil, err
		}
		for _, child := range children {
			addSubagent(child)
		}
	}
	return locators, subagents, nil
}

func NewClaudeCodeCommand() (*cobra.Command, error) {
	cmd, err := NewConvertClaudeCodeGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
