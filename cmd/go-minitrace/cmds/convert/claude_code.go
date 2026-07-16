package convert

import (
	"context"
	"path/filepath"
	"sort"

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
  go-minitrace convert claude-code --source-session ~/.claude/projects/my-project/0123456789abcdef0123456789abcdef.jsonl --output-dir ./output
  go-minitrace convert claude-code --source-list ./sessions.txt --output-dir ./output
`),
		cmds.WithFlags(
			fields.New("source-dir", fields.TypeString, fields.WithDefault("~/.claude/projects"), fields.WithHelp("Claude Code projects directory")),
			fields.New("source-session", fields.TypeStringList, fields.WithDefault([]string{}), fields.WithHelp("Explicit Claude Code session transcripts to convert instead of scanning --source-dir (repeatable)")),
			fields.New("source-list", fields.TypeString, fields.WithDefault(""), fields.WithHelp("File with newline-separated Claude Code session paths; blank lines and # comments are ignored")),
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

	explicitPaths, err := collectSourceSessions(settings_.SourceSessions, settings_.SourceList)
	if err != nil {
		return err
	}
	var locators []adapters.SessionLocator
	var subagentLocators []claudecode.SubagentLocator
	if len(explicitPaths) > 0 {
		locators = make([]adapters.SessionLocator, 0, len(explicitPaths))
		for _, path := range explicitPaths {
			locator, err := claudecode.LocateSession(path)
			if err != nil {
				return err
			}
			locators = append(locators, locator)
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
		primaries = append(primaries, convertedPrimary{locator: locator, session: session})
		primaryByID[session.ID] = session
	}

	subagents := make([]convertedSubagent, 0, len(subagentLocators))
	subagentIDsByParent := map[string][]string{}
	for _, locator := range subagentLocators {
		session, resolvedAgentID, err := claudecode.ConvertSubagentLocator(locator)
		if err != nil {
			return errors.Wrapf(err, "converting Claude Code subagent %s before publication", locator.SourcePath)
		}
		subagents = append(subagents, convertedSubagent{locator: locator, session: session})
		subagentIDsByParent[locator.ParentSessionID] = append(subagentIDsByParent[locator.ParentSessionID], resolvedAgentID)
	}
	for parentSessionID, subagentIDs := range subagentIDsByParent {
		if parent := primaryByID[parentSessionID]; parent != nil {
			sort.Strings(subagentIDs)
			claudecode.LinkParentSubagents(parent, subagentIDs)
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

func NewClaudeCodeCommand() (*cobra.Command, error) {
	cmd, err := NewConvertClaudeCodeGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
