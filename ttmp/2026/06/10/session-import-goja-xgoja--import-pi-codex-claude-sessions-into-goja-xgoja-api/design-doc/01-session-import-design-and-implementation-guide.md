---
Title: Session import design and implementation guide
Ticket: session-import-goja-xgoja
Status: active
Topics:
    - minitrace
    - goja
    - xgoja
    - transcript
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/adapters/claudecode/convert.go
      Note: Claude Code conversion and subagent behavior
    - Path: pkg/adapters/codex/convert.go
      Note: Codex JSONL conversion behavior and latest-format gaps
    - Path: pkg/adapters/pi/convert.go
      Note: Pi JSONL conversion behavior and latest-format gaps
    - Path: pkg/doc/js-api-reference.md
      Note: Public JavaScript API documentation updated for Preview
    - Path: pkg/minitracedb/convert.go
      Note: Auto-detection and adapter routing evidence for session imports
    - Path: pkg/minitracejs/import_builder.go
      Note: Goja importer and new Preview API implementation
    - Path: pkg/minitracejs/typescript.go
      Note: TypeScript declaration for xgoja minitrace module
ExternalSources: []
Summary: ""
LastUpdated: 2026-06-10T14:30:52.478395442-04:00
WhatFor: Intern-oriented design and implementation guide for session import preview and latest-format support.
WhenToUse: Use when extending Pi, Codex, or Claude Code imports through the goja/xgoja minitrace API.
---



# Session import design and implementation guide

## Executive summary

`go-minitrace` already has the core pieces needed to load Pi, Codex, and Claude Code sessions into a normalized minitrace representation and expose them through the Goja / xgoja module API. The main import path is `mt.importer()` in `pkg/minitracejs/import_builder.go`, which calls `pkg/minitracedb.LoadSessionFileAuto` / `LoadSessionContentAuto`; that auto loader detects native minitrace JSON first and then routes JSONL records to the Pi, Codex, or Claude Code adapters.

This ticket adds a concrete preview surface to that import flow: `mt.importer().File(path).AutoDetect().Convert().Preview()`. The preview returns role counts, tool counts, sample turns, sample tool calls, subagent signals, system-prompt/thinking/image indicators, and conversion diagnostics. That gives an operator or xgoja application a low-cost way to verify that system prompts, agent/user turns, tool calls/results, subagents, and image/blob-like signals survived parsing before saving or rendering the session.

The latest local sessions show that the adapters are close but not complete for the newest formats. Pi latest sessions include `custom`, `session_info`, and `compaction` records in addition to `session`, `model_change`, `thinking_level_change`, and `message`. Codex latest sessions include newer tool names and payload shapes such as `spawn_agent`, `wait_agent`, `apply_patch`, `write_stdin`, `custom_tool_call`, and `view_image`. Claude Code latest sessions include `attachment`, `mode`, `permission-mode`, `ai-title`, `file-history-snapshot`, subagent files under `subagents/`, and assistant `diagnostics` / `stop_details` fields. The implementation plan below treats these as compatibility work, not a rewrite.

## Implementation status update

The follow-up implementation completed the highest-priority latest-format support described in this guide:

- Codex `spawn_agent` and `wait_agent` now classify as `DELEGATE` and populate spawned-agent metadata where possible.
- Codex `view_image` now classifies as `READ`, preserves the image path, marks `content_origin=image`, and trips preview image-signal detection.
- Codex `custom_tool_call` / `custom_tool_call_output` records, including `apply_patch`, are normalized into tool calls with status/custom metadata and parsed outputs.
- Codex `write_stdin` now classifies as `EXECUTE`.
- Claude Code `mode`, `permission-mode`, and `ai-title` records are preserved in `OperationalContext.FrameworkConfig`, and `ai-title` becomes the session title when present.
- Claude Code `attachment` records now become session-scoped annotations with bounded detail text and image tags when media/image signals are present.
- Claude Code top-level `agentId`, `sessionId`, `parentUuid`, `isSidechain`, `userType`, and `attributionAgent` are preserved in framework metadata/config where relevant.
- The CLI now exposes `go-minitrace preview session`, with one-file mode, framework directory/latest-N mode, `--sample-limit`, and `--privacy structural|snippets|full`.

The remaining notable gap is Pi preservation of `custom`, `session_info`, and `compaction` records; this was outside the requested Claude/Codex continuation and remains a future adapter-hardening task.

## Problem statement and scope

The goal is to make session import usable from the new Goja / xgoja API for three local agent ecosystems:

- **Pi** sessions from `~/.pi/agent/sessions`.
- **Codex** sessions from `~/.codex/sessions`.
- **Claude Code** sessions from `~/.claude/projects`, including subagents.

The desired operator workflow is:

1. Select a session file or uploaded JSONL body.
2. Detect its source format.
3. Convert it into a minitrace session.
4. Print a compact preview showing whether important semantic elements were parsed.
5. Save or query/render the normalized minitrace session if the preview looks right.

The scope of this ticket is the import and preview architecture. It does not require a full transcript renderer, but it should make a renderer straightforward by ensuring normalized session data exposes enough summary and samples to catch bad parsing early.

## System map

```text
Pi JSONL          Codex JSONL           Claude Code JSONL
   │                  │                         │
   └──────────────┬───┴──────────────┬──────────┘
                  │                  │
                  ▼                  ▼
       pkg/minitracedb/convert.go: auto detect + route
                  │
        ┌─────────┼─────────┐
        ▼         ▼         ▼
  adapters/pi  adapters/codex  adapters/claudecode
        │         │         │
        └─────────┴─────────┘
                  ▼
          pkg/minitrace.Session
                  ▼
        pkg/minitracejs ImportBuilder
                  ▼
       Goja/xgoja require("minitrace")
                  ▼
   Detect / Converted / Preview / Save / DB query views
```

Important files:

- `pkg/minitracedb/convert.go`: native-vs-JSONL loader and JSONL format detection.
- `pkg/adapters/pi/convert.go`: Pi JSONL-to-minitrace mapping.
- `pkg/adapters/codex/convert.go`: Codex JSONL-to-minitrace mapping.
- `pkg/adapters/claudecode/convert.go`: Claude Code JSONL and subagent conversion.
- `pkg/minitracejs/import_builder.go`: Goja-facing import builder and preview API.
- `pkg/minitracejs/typescript.go`: xgoja TypeScript declaration source for the minitrace module.
- `pkg/doc/js-api-reference.md`: user-facing API documentation.

## Current-state architecture with evidence

### Auto-loading and format detection

`pkg/minitracedb/convert.go` is the convergence point for Goja import, DB source loading, and tests. `LoadSessionContentAuto` first tries native minitrace JSON, then requires `AutoConvert`, parses JSONL, detects the format, and calls the adapter for `pi-jsonl`, `codex-jsonl`, or `claude-code-jsonl` (`pkg/minitracedb/convert.go:54-91`). Detection currently looks for top-level event types: Codex is identified by `session_meta`, `turn_context`, `response_item`, or `event_msg`; Pi by `session`, `model_change`, `thinking_level_change`, `message`, or role-only records; Claude Code by `system`, `user`, or `assistant` records with a `message` object (`pkg/minitracedb/convert.go:94-112`).

This is a simple and useful boundary. New source formats should usually be handled by expanding detection and adapter support rather than by adding separate import paths in the Goja layer.

### Goja / xgoja module surface

`pkg/minitracejs/module.go` registers the `minitrace` module and exposes factories such as `importer`, `db`, `sources`, `query`, `view`, and `session`. The xgoja provider wraps the same module under package ID `go-minitrace`; `pkg/minitracejs/provider/provider.go` registers both the module and the repository-backed query command provider.

The import builder previously exposed detect/convert/save operations. This ticket adds preview types and the `Preview()` method in `pkg/minitracejs/import_builder.go:54-102` and wires it into the Goja object at `pkg/minitracejs/import_builder.go:164-171`. The API now supports this pattern:

```js
const mt = require("minitrace");

const importer = mt.importer()
  .File(sessionPath)
  .AutoDetect()
  .Convert();

const preview = importer.Preview();
console.log(JSON.stringify(preview, null, 2));

// If preview looks correct:
const saved = importer.Into(sessionsDir).Save();
```

The xgoja TypeScript declaration source was updated so `ImportBuilder` advertises `Preview()` alongside `Converted()` and `Diagnostics()`.

### Pi adapter behavior

The Pi adapter maps message content blocks into minitrace turns and tool calls. It recognizes `text`, `thinking`, `toolCall` / `tool_use`, and `toolResult` / `tool_result` blocks (`pkg/adapters/pi/convert.go:114-166`). It also tracks model and provider from `model_change` and assistant message metadata, records token usage, and annotates orphan tool calls.

The latest local Pi survey found these record types in recent files:

- `message`
- `custom`
- `thinking_level_change`
- `model_change`
- `session_info`
- `compaction`
- `session`

The current adapter handles the core `message` and model/thinking/session events, but `custom`, `session_info`, and `compaction` are not promoted into first-class events or annotations. The preview still shows whether the normalized core survived because it reports roles, tools, thinking, and sample turns/tools.

### Codex adapter behavior

The Codex adapter handles session JSONL records by looking at `event_msg` and `response_item` payloads. For `response_item` records, it maps `reasoning` into turn thinking, `function_call` into minitrace tool calls, and `function_call_output` into tool outputs (`pkg/adapters/codex/convert.go:320-390`). Command-specific handling currently extracts `exec_command` command text into the normalized `input.command` field (`pkg/adapters/codex/convert.go:339-352`).

The latest local Codex survey found new or important tool names and payload shapes:

- `exec_command`
- `spawn_agent`
- `wait_agent`
- `apply_patch`
- `write_stdin`
- `view_image`
- `custom_tool_call`

The existing generic `function_call` path will preserve many non-`exec_command` calls as tool calls with argument metadata, but some semantics are still under-modeled: `spawn_agent` / `wait_agent` should map to `ToolCall.SpawnedAgent` or an equivalent subagent relation, `view_image` should set an image/blob signal, and `custom_tool_call` should be represented consistently with standard function calls.

A compatibility issue also surfaced while running `go run ./cmd/go-minitrace convert codex --source-dir ~/.codex --dry-run --output json`: the command scans older files and fails on the first unsupported historical format with `unsupported Codex format hint: unknown-jsonl`. That is separate from latest-format support, but it matters for operator workflows over a whole home directory.

### Claude Code adapter behavior

The Claude Code adapter maps `system`, `user`, and `assistant` records. User records containing `tool_result` blocks attach outputs to pending tool calls (`pkg/adapters/claudecode/convert.go:126-168`). Assistant records inspect `text`, `thinking`, and `tool_use` blocks and create minitrace tool calls, including `SpawnedAgent` for `Agent` tool uses (`pkg/adapters/claudecode/convert.go:226-270`).

The latest local Claude survey found top-level record types beyond the currently promoted message set:

- `assistant`
- `user`
- `file-history-snapshot`
- `mode`
- `permission-mode`
- `ai-title`
- `attachment`
- `last-prompt`

Subagents were present under `subagents/agent-*.jsonl` and included `agentId`, `sessionId`, `parentUuid`, `isSidechain`, and `attachment` records. The adapter already has subagent discovery and conversion functions, but auto-loading one arbitrary subagent file through `LoadSessionFileAuto` only treats it as a Claude Code JSONL session. A full directory import must still use `claudecode.DiscoverSubagents` / `ConvertSubagentLocator` through the convert command path to link parent and child sessions.

## Latest format survey

The ticket script `scripts/01-survey-agent-session-shapes.py` inspected recent local sessions without copying prompt or message bodies. It wrote the structural report to `sources/01-agent-session-shape-survey.md`.

Observed highlights:

- **Pi latest sessions**: large recent files contain hundreds of tool calls and tool results, with `bash`, `read`, `edit`, `write`, and Playwright tools prominent. Message key shapes include `stopReason`, `errorMessage`, and `usage`.
- **Codex latest sessions**: recent files use `response_item` / `event_msg`; one sample contains `spawn_agent`, `wait_agent`, `apply_patch`, `write_stdin`, and `view_image` in addition to `exec_command`.
- **Claude latest sessions**: recent parent files include `attachment`, `mode`, `permission-mode`, and `ai-title`; recent subagent files are very small and identified by path plus `agentId` metadata.
- **Image/blob signals**: the structural survey did not see base64 image blocks in the latest sampled Pi or Claude sessions, but Codex had a `view_image` tool name. The preview API therefore uses conservative `HasImageSignals` detection rather than claiming full binary preservation.

## Gap analysis

### What now works

- Import builder can auto-detect and convert native minitrace JSON, Pi JSONL, Codex JSONL, and Claude Code JSONL through one Goja-facing API.
- `Preview()` gives a compact validation object that is safe to print in CLIs, xgoja apps, and web upload flows.
- Preview captures the most important operator questions:
  - Did detection choose the expected adapter?
  - Did user/assistant/system turns appear?
  - Did thinking text appear?
  - Did tool calls and tool results appear?
  - Did subagent fields appear on normalized tool calls?
  - Are there image/blob-like signals?
- Tests cover Pi preview behavior with user/assistant/tool result flow.

### Gaps to close for latest formats

1. **Codex whole-directory conversion should be resilient.** The converter currently aborts on older unsupported files when scanning `~/.codex`. Add skip/error aggregation modes so latest valid sessions can still be converted.
2. **Codex subagents need semantic promotion.** `spawn_agent` / `wait_agent` should populate `ToolCall.SpawnedAgent` and link to sub-session metadata when available.
3. **Codex image handling needs explicit mapping.** `view_image` should set a content-origin or framework metadata marker that `Preview()` and renderers can detect.
4. **Claude attachment records need representation.** Current conversion skips most non-message records. Attachments should become annotations/events or turn metadata, especially if they represent images or file blobs.
5. **Claude mode and permission records should be preserved.** `mode`, `permission-mode`, and `ai-title` should update `OperationalContext.FrameworkConfig`, annotations, or events rather than disappearing.
6. **Pi custom/session_info/compaction should be preserved.** These records are important for reconstructing context compression and session metadata.
7. **Preview should be exposed as a command.** The Goja API now has `Preview()`, but there should be a first-class CLI command or query verb that prints a preview for files/directories without writing an archive.

## Proposed architecture

### Keep import detection centralized

Do not add Pi/Codex/Claude branching in xgoja application code. Keep source detection and adapter routing inside `pkg/minitracedb` so every consumer gets the same behavior:

```go
loaded, err := minitracedb.LoadSessionFileAuto(path, minitracedb.LoadOptions{
    SourcePath:  path,
    SourceName:  filepath.Base(path),
    AutoConvert: true,
})
```

The Goja import builder should remain a small orchestration layer around this loader:

```go
func (b *ImportBuilder) load() (*minitracedb.LoadedSession, error) {
    if b.path != "" {
        return minitracedb.LoadSessionFileAuto(b.path, options)
    }
    return minitracedb.LoadSessionContentAuto([]byte(b.content), options)
}
```

### Treat preview as a stable validation contract

`Preview()` should be treated as a stable, JSON-serializable validation contract. It is not a transcript renderer; it is a structured checklist for import correctness.

Recommended fields:

- identity: `sessionId`, `format`, `adapter`, `title`, `agentFramework`, `model`, `workingDirectory`
- booleans: `hasSystemPrompt`, `hasThinking`, `hasImageSignals`
- counts: `turnCount`, `toolCallCount`, `subagentCount`, `roleCounts`, `toolCounts`
- samples: `sampleTurns[]`, `sampleTools[]`
- diagnostics: conversion warnings/errors/info

This ticket implemented that shape in `pkg/minitracejs/import_builder.go`.

### Add a CLI and xgoja command wrapper next

The code-level API is enough for xgoja applications, but operators need a command. Add a command such as:

```bash
go-minitrace preview session --source /path/to/session.jsonl --output yaml
```

or a JS query verb under the command repository:

```js
// @verb preview-session
// @arg source string
function previewSession({ source }) {
  const mt = require("minitrace");
  return mt.importer().File(source).AutoDetect().Convert().Preview();
}
```

The command should support:

- `--source-session` for one file.
- `--source-dir` plus `--framework pi|codex|claude-code` for directory scans.
- `--latest N` for sampling recent sessions.
- `--include-samples` and `--sample-limit` for controlling preview verbosity.
- `--privacy structural|snippets|full` with `structural` as the default.

### Normalize non-message records as events or annotations

The minitrace schema currently has turns, tool calls, annotations, environment, and operational context. There is also an `events` table in the materialized database. New source records should be categorized as follows:

- **Affects session state**: put in `Environment` or `OperationalContext.FrameworkConfig`.
- **Affects a turn/tool**: put in `FrameworkMetadata` on that turn/tool.
- **Explains a session transition**: add an annotation or event.
- **Contains binary/image/file reference**: preserve metadata and mark `HasImageSignals`; do not inline large blobs into normalized turn text by default.

Examples:

```go
switch recordType {
case "compaction":
    annotations = append(annotations, BuildAnnotation(..., tags: []string{"compaction"}))
case "session_info":
    frameworkConfig["session_info"] = sanitizedRecord
case "attachment":
    annotations = append(annotations, attachmentAnnotation(record))
case "mode", "permission-mode":
    frameworkConfig[recordType] = recordSpecificValue(record)
}
```

## Decision records

### Decision: Centralize import through `minitracedb.LoadSession*Auto`

- **Context:** Goja apps, CLI conversion commands, query DB loading, and upload flows all need the same format-detection behavior.
- **Options considered:** Duplicate adapter selection in the Goja import builder; keep adapter selection centralized in `minitracedb`; add separate per-framework Goja APIs.
- **Decision:** Keep detection and routing centralized in `pkg/minitracedb`.
- **Rationale:** `LoadSessionContentAuto` already routes Pi/Codex/Claude Code JSONL based on record shapes. Adding logic elsewhere would create divergent behavior.
- **Consequences:** Format support work must include minitracedb tests; xgoja consumers get new support automatically.
- **Status:** accepted.

### Decision: Preview normalized sessions, not raw JSONL

- **Context:** The user wants to verify system prompts, roles, tools, subagents, image blobs, and parsing quality.
- **Options considered:** Print raw JSONL records; print normalized minitrace previews; print both.
- **Decision:** Preview the normalized minitrace session, with diagnostics and conservative image/blob signals.
- **Rationale:** The main failure to detect is not “what did the raw file contain?” but “what survived conversion into minitrace?” Raw structural survey remains useful as a research script.
- **Consequences:** Preview may miss a raw source feature that was dropped before normalization; this is why the survey script remains part of the ticket and why adapters should preserve raw metadata as annotations/events.
- **Status:** accepted.

### Decision: Keep preview privacy-safe by default

- **Context:** Local sessions may contain private prompts, paths, secrets, and tool outputs.
- **Options considered:** Full transcript preview; structural preview only; structural preview plus short content snippets.
- **Decision:** Include short turn content previews and tool command snippets, but not full prompts or outputs.
- **Rationale:** A preview needs enough human context to spot role/order bugs, but should not dump multi-megabyte prompts or tool outputs.
- **Consequences:** CLI wrappers should add an explicit `--privacy full` or `--include-full-text` if full content is required.
- **Status:** proposed.

### Decision: Preserve latest-format extras as metadata/events first

- **Context:** Latest Pi/Codex/Claude formats include extra records that do not map cleanly to turn or tool rows.
- **Options considered:** Drop unknown records; add many new schema fields immediately; preserve extras as annotations/events/framework metadata first.
- **Decision:** Preserve extra records as annotations/events or framework metadata before promoting to stable schema fields.
- **Rationale:** This avoids schema churn while preventing data loss.
- **Consequences:** Query/render layers need conventions for rendering annotations/events; later schema promotion can be based on observed query needs.
- **Status:** proposed.

## Implementation guide for a new intern

### Phase 1: Understand the conversion path

Start by reading these files in order:

1. `pkg/minitrace/schema.go` — the normalized data model.
2. `pkg/minitracedb/convert.go` — auto detection and routing.
3. `pkg/adapters/pi/convert.go` — Pi mapping.
4. `pkg/adapters/codex/convert.go` — Codex mapping.
5. `pkg/adapters/claudecode/convert.go` — Claude Code mapping and subagent behavior.
6. `pkg/minitracejs/import_builder.go` — Goja-facing import API.
7. `pkg/doc/js-api-reference.md` — public API docs.

Then run:

```bash
go test ./pkg/adapters/... ./pkg/minitracedb ./pkg/minitracejs/... ./cmd/go-minitrace/cmds/query -count=1
```

### Phase 2: Re-run the structural survey

Use the ticket script:

```bash
cd go-minitrace
./ttmp/2026/06/10/session-import-goja-xgoja--import-pi-codex-claude-sessions-into-goja-xgoja-api/scripts/01-survey-agent-session-shapes.py \
  --max-files 5 \
  --format markdown \
  > ttmp/2026/06/10/session-import-goja-xgoja--import-pi-codex-claude-sessions-into-goja-xgoja-api/sources/01-agent-session-shape-survey.md
```

If you need special cases, ask for specific files containing:

- screenshots or image upload blocks,
- Codex subagent / spawned-agent workflows,
- Claude Code attachment-heavy sessions,
- Pi compaction-heavy sessions.

### Phase 3: Add format tests from minimized fixtures

Do not copy private full transcripts into `testdata`. Create minimized JSONL fixtures with only the fields required to reproduce a format shape.

Example Codex fixture for `spawn_agent`:

```jsonl
{"type":"session_meta","payload":{"id":"codex-spawn","cwd":"/tmp/project"}}
{"type":"turn_context","payload":{"model":"gpt-5.4-mini"}}
{"type":"response_item","payload":{"type":"function_call","call_id":"call-1","name":"spawn_agent","arguments":"{\"task\":\"audit frontend\"}"}}
{"type":"response_item","payload":{"type":"function_call_output","call_id":"call-1","output":"{\"status\":\"ok\"}"}}
```

Expected assertion:

```go
loaded, err := minitracedb.LoadSessionContentAuto(payload, minitracedb.LoadOptions{
    SourceName: "codex-spawn.jsonl",
    AutoConvert: true,
})
require.NoError(t, err)
require.Equal(t, "codex", loaded.Adapter)
require.Equal(t, "spawn_agent", loaded.Session.ToolCalls[0].ToolName)
require.NotNil(t, loaded.Session.ToolCalls[0].SpawnedAgent)
```

### Phase 4: Implement adapter preservation for latest records

Recommended order:

1. Codex `spawn_agent`, `wait_agent`, `custom_tool_call`, `view_image`.
2. Claude `attachment`, `mode`, `permission-mode`, `ai-title`.
3. Pi `custom`, `session_info`, `compaction`.
4. Directory conversion resilience for Codex older unsupported sessions.

Pseudocode for Codex tool promotion:

```go
func promoteCodexTool(toolCall *minitrace.ToolCall, funcName string, args map[string]any) {
    switch funcName {
    case "spawn_agent":
        toolCall.OperationType = "DELEGATE"
        toolCall.SpawnedAgent = &minitrace.SpawnedAgent{
            AgentType: "codex",
            TaskScope: firstNonEmpty(stringValue(args["task"]), stringValue(args["prompt"])),
        }
    case "view_image":
        origin := "image"
        toolCall.Output.ContentOrigin = &origin
        toolCall.FrameworkMetadata = mergeMetadataMap(toolCall.FrameworkMetadata, map[string]any{"has_image_signal": true})
    case "apply_patch":
        toolCall.OperationType = "MODIFY"
    }
}
```

Pseudocode for Claude attachment preservation:

```go
case "attachment":
    annotations = append(annotations, minitrace.BuildAnnotation(
        "ann-attachment-" + truncateID(stringValue(record["uuid"])),
        "adapter",
        "session",
        session.ID,
        "observation",
        "Claude Code attachment",
        summarizeAttachment(record["attachment"]),
        []string{"attachment", maybeMediaTag(record)},
        nil,
    ))
```

Pseudocode for Pi compaction preservation:

```go
case "compaction":
    annotations = append(annotations, minitrace.BuildAnnotation(
        "ann-compaction-" + truncateID(stringValue(record["id"])),
        "adapter",
        "session",
        fallbackID,
        "observation",
        "Pi context compaction",
        summarizeCompaction(record),
        []string{"compaction", "context-management"},
        nil,
    ))
```

### Phase 5: Add a preview command

A first-class command can be implemented as either a Go command under `cmd/go-minitrace/cmds` or a JS verb in a query repository. Prefer a Go command if it must scan framework directories and handle subagent linking. Prefer a JS verb if it only previews one file.

Command contract:

```bash
go-minitrace preview session \
  --source-session ~/.pi/agent/sessions/.../session.jsonl \
  --output yaml
```

Output contract:

```yaml
sessionId: ...
adapter: pi
format: pi-jsonl
turnCount: 42
toolCallCount: 23
hasSystemPrompt: false
hasThinking: true
hasImageSignals: false
roleCounts:
  user: 7
  assistant: 7
toolCounts:
  bash: 10
  read: 8
sampleTurns:
  - index: 0
    role: user
    hasContent: true
sampleTools:
  - id: call-1
    toolName: bash
    operationType: EXECUTE
    success: true
```

### Phase 6: Validate with real latest sessions

Use both unit fixtures and local-session smoke tests:

```bash
# Unit tests
go test ./pkg/adapters/... ./pkg/minitracedb ./pkg/minitracejs/... -count=1

# Structural survey
./ttmp/2026/06/10/session-import-goja-xgoja--import-pi-codex-claude-sessions-into-goja-xgoja-api/scripts/01-survey-agent-session-shapes.py --max-files 3

# Single Pi dry-run
go run ./cmd/go-minitrace convert pi --source-session "$PI_SESSION" --dry-run --output json

# Claude directory dry-run
go run ./cmd/go-minitrace convert claude-code --source-dir ~/.claude/projects --dry-run --output json
```

For Codex, prefer single-file preview until directory resilience is implemented, because full-directory conversion currently fails on older unsupported sessions.

## Testing and validation strategy

### Unit tests

- `pkg/minitracejs/import_builder_test.go`: verify `Preview()` reports adapter/format, role counts, tool counts, thinking, sample turns, and sample tools.
- `pkg/minitracedb/convert_test.go`: add detection tests for new JSONL shapes.
- `pkg/adapters/codex/convert_test.go`: add minimized fixtures for `spawn_agent`, `wait_agent`, `apply_patch`, `view_image`, and `custom_tool_call`.
- `pkg/adapters/claudecode/convert_test.go`: add minimized fixtures for `attachment`, `mode`, `permission-mode`, and subagent attachment records.
- `pkg/adapters/pi/convert_test.go`: add minimized fixtures for `custom`, `session_info`, and `compaction`.

### Smoke tests

- Run the survey script against local latest sessions.
- Run preview/import on the latest Pi file.
- Run preview/import on the latest Codex file.
- Run preview/import on a Claude parent file and at least one subagent file.
- Check that previews show nonzero turns/tools and expected adapter names.

### Regression assertions

Every new format fixture should assert:

- adapter and format are detected correctly,
- turn count and tool count match expectations,
- system prompt handling is intentional,
- thinking is retained when present,
- tool result success/error is retained,
- subagent metadata is retained when present,
- image/blob signals are retained as metadata or annotations.

## Risks and open questions

- **Privacy:** Previews should not dump full transcripts by default. Keep output bounded and add explicit flags for full text.
- **Schema churn:** Latest formats contain many framework-specific fields. Preserve them as metadata/events first; promote only after repeated query/render needs.
- **Codex historical compatibility:** Whole-directory conversion currently fails on older unknown JSONL. Decide whether the command should skip unsupported files by default or require `--skip-errors`.
- **Image blobs:** No base64 image blobs were observed in the latest sampled Pi/Claude files. Request explicit image-heavy sessions before implementing blob storage semantics.
- **Subagent matching:** Claude Code has explicit subagent files. Codex has `spawn_agent` / `wait_agent` semantics, but the exact child-session storage/linking contract needs more samples.

## References

- `pkg/minitracedb/convert.go:54-112` — auto load/detect/route logic.
- `pkg/minitracejs/import_builder.go:54-102` — preview structs.
- `pkg/minitracejs/import_builder.go:164-171` — Goja `Preview()` method registration.
- `pkg/adapters/pi/convert.go:114-166` — Pi content block and tool result mapping.
- `pkg/adapters/codex/convert.go:320-390` — Codex response item mapping.
- `pkg/adapters/claudecode/convert.go:126-168` — Claude tool-result mapping.
- `pkg/adapters/claudecode/convert.go:226-270` — Claude tool-use and spawned-agent mapping.
- `pkg/doc/js-api-reference.md` — public JavaScript API documentation.
- `ttmp/2026/06/10/session-import-goja-xgoja--import-pi-codex-claude-sessions-into-goja-xgoja-api/sources/01-agent-session-shape-survey.md` — latest local structural survey.
