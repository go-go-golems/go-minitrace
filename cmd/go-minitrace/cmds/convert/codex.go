package convert

import (
	"context"
	"path/filepath"
	"time"

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
	RunRecord      string   `glazed:"run-record"`
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
			fields.New("run-record", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Write an atomic JSON conversion run record")),
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
	locators, err = preflightCodexLocators(locators)
	if err != nil {
		return err
	}

	collisionPolicy := minitrace.CollisionPolicy(settings_.Collision)
	if collisionPolicy != minitrace.CollisionError && collisionPolicy != minitrace.CollisionReplace {
		return errors.Errorf("unsupported collision policy %q", settings_.Collision)
	}

	runRecord := newConversionRunRecord("codex", settings_.OutputDir, collisionPolicy, locators)
	type convertedSource struct {
		locator adapters.SessionLocator
		session *minitrace.Session
	}
	converted := make([]convertedSource, 0, len(locators))
	for _, locator := range locators {
		session, err := codex.ConvertLocator(locator)
		if err != nil {
			return errors.Wrapf(err, "converting Codex source %s before publication", locator.SourcePath)
		}
		converted = append(converted, convertedSource{locator: locator, session: session})
	}

	indexEntries := make([]*minitrace.SessionIndexEntry, 0, len(converted))
	for _, source := range converted {
		locator := source.locator
		session := source.session
		var sessionPath string
		if !settings_.DryRun {
			entry, err := minitrace.WriteSessionWithCollisionPolicy(session, settings_.OutputDir, collisionPolicy)
			if err != nil {
				return errors.Wrapf(err, "writing minitrace session %s", locator.ID)
			}
			indexEntries = append(indexEntries, entry)
			runRecord.Outputs = append(runRecord.Outputs, conversionRunOutput{SessionID: session.ID, Path: entry.FilePath})
			sessionPath = entry.FilePath
		}

		quality := ""
		if session.Quality != nil {
			quality = *session.Quality
		}
		identity := locator.Identity
		fingerprint := ""
		identityBasis := ""
		parentSessionID := ""
		warningCount := 0
		if identity != nil {
			fingerprint = identity.SHA256
			identityBasis = identity.IdentityBasis
			parentSessionID = identity.ParentSessionID
			warningCount = len(identity.Warnings)
		}
		row := types.NewRow(
			types.MRP("framework", "codex"),
			types.MRP("session_id", session.ID),
			types.MRP("source_format", locator.FormatHint),
			types.MRP("source_fingerprint", fingerprint),
			types.MRP("identity_basis", identityBasis),
			types.MRP("parent_session_id", parentSessionID),
			types.MRP("warning_count", warningCount),
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
		if err := minitrace.WriteManifests(indexEntries, settings_.OutputDir); err != nil {
			return errors.Wrap(err, "writing Codex manifests")
		}
		runRecord.FinishedAt = minitrace.FormatTimestamp(time.Now().UTC())
		runRecord.Complete = true
		if err := writeConversionRunRecord(settings_.RunRecord, runRecord); err != nil {
			return err
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

// preflightCodexLocators inspects every requested source before the command
// writes an archive. It attaches source evidence to locators, rejects one
// native ID backed by differing bytes, and collapses byte-identical duplicate
// sources so a batch has one deterministic publication candidate per ID.
func preflightCodexLocators(locators []adapters.SessionLocator) ([]adapters.SessionLocator, error) {
	ret := make([]adapters.SessionLocator, 0, len(locators))
	seen := map[string]string{}
	for _, locator := range locators {
		identity, err := codex.InspectSource(locator.SourcePath)
		if err != nil {
			return nil, errors.Wrapf(err, "preflighting Codex source %s", locator.SourcePath)
		}
		locator.Identity = &identity
		if identity.NativeSessionID == "" {
			return nil, errors.Errorf("preflighting Codex source %s: missing native session ID", locator.SourcePath)
		}
		if fingerprint, ok := seen[identity.NativeSessionID]; ok {
			if fingerprint != identity.SHA256 {
				return nil, errors.Errorf("preflighting Codex source %s: native session ID %q conflicts with another source fingerprint", locator.SourcePath, identity.NativeSessionID)
			}
			continue
		}
		seen[identity.NativeSessionID] = identity.SHA256
		ret = append(ret, locator)
	}
	return ret, nil
}

func NewCodexCommand() (*cobra.Command, error) {
	cmd, err := NewConvertCodexGlazeCommand()
	if err != nil {
		return nil, err
	}
	return common.BuildCobraCommand(cmd)
}
