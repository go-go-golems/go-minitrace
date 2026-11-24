---
Title: Claude Code normalization design
Ticket: CLAUDE-CODE-NORMALIZE-2026-11-24
Status: active
Topics:
    - claude-code
    - adapters
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/go-minitrace/cmds/discover/claude_code.go
      Note: --include-subagents
    - Path: repo://pkg/adapters/claudecode/signals.go
      Note: context loads, hook events, outcome reasons, system markers
    - Path: repo://pkg/adapters/claudecode/subagents.go
      Note: subagent discovery and native parent linking
    - Path: repo://pkg/minitracedb/convert.go
      Note: autoConvert subagent and stub detection
    - Path: repo://pkg/minitracedb/schema.go
      Note: context_loads table, outcome_reason, metrics columns
ExternalSources: []
Summary: Normalize Claude Code subagents for explicit sources, guidance loads (context_loads table), tool outcome reasons, hook events, compaction boundaries and turn durations.
LastUpdated: 2026-11-24T14:30:42.533552-05:00
WhatFor: Design record for the Claude Code normalization PR
WhenToUse: Before changing the Claude Code adapter, context_loads, outcome_reason or Claude Code event kinds
---

# Claude Code normalization design

## Executive Summary

go-minitrace leaves several Claude Code signals raw, so analysis verbs (for example the playback
repository's PB-MINITRACE-001 verbs) fall back to heuristics. This change makes six of those signals
first-class:

1. **Subagents for explicit sources.** `convert claude-code --source-session/--source-list` pulls in
   `<stem>/subagents/**/agent-*.jsonl` by default (`--no-subagents` opts out), and
   `discover claude-code --include-subagents` lists them with their parent. Parent `Agent` tool calls
   link to subagent sessions through native ids instead of list order.
2. **Guidance loads** become a `context_loads` table: one row per CLAUDE.md, `.claude/rules/*.md`,
   session-start instruction file, or re-injected skill.
3. **Tool outcome reasons**: a new `tool_calls.outcome_reason` column (`hook_blocked`,
   `permission_denied`, `classifier_denied`, `classifier_no_verdict`, `user_rejected`, `interrupted`,
   `tool_use_error`, `harness_blocked`). A hook-blocked call is no longer `success=1`.
4. **Hook events**: hook attachments and `stop_hook_summary` records become `events.kind = 'hook'`,
   linked to the tool call they concern.
5. **Compaction boundaries**: `system/compact_boundary` becomes `events.kind = 'compaction'`, and
   `history context-window` prefers it over its heuristics.
6. **Turn durations**: `system/turn_duration` becomes `events.kind = 'turn_duration'`; sessions get
   `metrics.compaction_count` and `metrics.total_turn_duration_ms`.

A metadata-only transcript stub (left by `/resume`) converts to a nearly empty session instead of
aborting `autoConvert` with "JSONL format is unsupported".

## Problem Statement

### Current behavior with evidence

The evidence below comes from the 192 native transcripts (16 parents with subagent directories, 177
subagent files) under `~/.claude/projects/-Users-manuel-odendahl-code-tulip-playback/` and the
consumer's converted archive (160 sessions converted from an explicit `--source-list`).

**Subagents.** `convert claude-code` only expands subagents in `--source-dir` mode
(`DiscoverSubagents`). With an explicit list that names subagent files, every file becomes a primary:

```text
select source_format, predecessor_session is not null as has_parent, count(*) from sessions group by 1,2
claude-code-jsonl-v2 | 0 | 160
select count(*), sum(spawned_agent_sub_session_id is not null) from tool_calls where spawned_agent_type is not null
149 | 0
```

Even in `--source-dir` mode, `LinkParentSubagents` assigns subagent ids to `Agent` tool calls by
position after sorting the ids lexically. The ids are random hex, so the pairing is effectively
random. Native data carries the exact link twice:

- the parent's `Agent` tool result has `toolUseResult.agentId`;
- `subagents/agent-<id>.meta.json` has `toolUseId` (the parent's tool_use id), `agentType` and
  `description`.

`DiscoverSubagents` also accepts any `.jsonl` under `subagents/`, including
`subagents/workflows/wf_*/journal.jsonl` (a workflow journal, not a transcript).

`discover claude-code` skips subagents entirely, so a consumer cannot build a source list that
includes them without its own directory walk.

**Guidance loads.** Claude Code records context injections as `attachment` records: 1911
`nested_memory` (path-scoped CLAUDE.md / rules), 205 `instructions` (session start and post-compaction;
1356 files of type `Project` or `AutoMem`) and 47 `invoked_skills`. They exist only in
`attachments.raw_json`. The query sandbox denies `json_each` (a virtual table), so a consumer must pull
whole raw JSON cells into JS to list loaded paths.

**Tool outcomes.** Prefix counts over tool results (is_error in parentheses):

| Result text prefix | Count |
|---|---|
| `The command did not run. A PreToolUse hook blocked it before execution` | 1 (is_error=false) |
| `Permission for this action was denied by the Claude Code auto mode classifier` | 13 (true) |
| `Permission to use <Tool> with command ...` | 2 (true) |
| `The user doesn't want to proceed with this tool use` / `... take this action right now` | 8 (true) |
| `The server-side auto mode classifier gave no verdict` | 15 (true) |
| `<tool_use_error>...` | 30 (true) |
| `This agent is isolated in the worktree ...` | 20 (true) |

The hook-blocked call is stored as `success=1`, and the others are indistinguishable from a non-zero
exit. `duration_ms` is the emit-to-result interval, so it includes permission prompts and classifier
waits: one 1-second grep appears as 61 minutes.

**Hooks.** `hook_success` attachments (and upstream-documented `hook_blocking_error`,
`hook_additional_context`, `hook_non_blocking_error`, `system/stop_hook_summary`) only appear as generic
`attachment` events or as system turns.

**System markers.** 655 `turn_duration`, 58 `compact_boundary`, 108 `away_summary`, 9 `informational`
and 6 `local_command` system records all become `role='system'` turns (646 system turns in the
archive), mostly with empty content. `history context-window` infers compaction from continuation text
and cache-read collapse even though `compact_boundary` carries `compactMetadata.{trigger, preTokens,
postTokens}`.

**Stub transcripts.** `/resume` can leave a seven-line file with only `system/local_command`,
`cost-state`, `mode`, `permission-mode` and `last-prompt` records. `DetectJSONLFormat` recognizes
Claude Code only from a `system|user|assistant` record carrying `message`, so autoConvert fails with
`JSONL format is unsupported` and aborts strict JS runs over a project glob.

### Upstream comparison (fukami/minitrace v0.3.0)

The upstream adapter (`adapters/claude-code/claude-code.dsl.yaml`, `preprocess_claude_code.py`) is
used only for semantics; its trace format is not adopted.

| Signal | Upstream | This change |
|---|---|---|
| Subagents | "not yet handled" in the DSL; preprocess maps spawn prompt → subagent trace id | native `agentId` / meta `toolUseId`, order fallback |
| nested_memory / instructions / invoked_skills | not mapped | `context_loads` table |
| hook_success / hook_additional_context | observation annotations | `events.kind='hook'` |
| hook_blocking_error | annotation scoped to `toolUseID` | `events.kind='hook'` + `outcome_reason='hook_blocked'` for PreToolUse |
| stop_hook_summary | policy_trigger per `hookInfos[]` entry | one `hook` event with the commands in metadata |
| turn_duration | `metrics.total_turn_duration_ms` | per-turn event plus the same metric |
| compact_boundary | not mapped | `events.kind='compaction'`, `metrics.compaction_count` |
| tool-result denial texts | not mapped | `outcome_reason` |

## Proposed Solution

### 1. Subagents

`pkg/adapters/claudecode`:

- `SubagentsForSession(locator)` returns the subagent locators below the session's directory
  (`<dir>/<stem>/subagents/**/agent-*.jsonl` for a JSONL transcript, `<dir>/subagents/...` for a
  dir-v1 session). `DiscoverSubagents` shares the same `agent-*.jsonl` filter.
- `SubagentLocator` gains `ToolUseID`, `AgentType`, `Description` from `agent-<id>.meta.json` when
  present, plus `Cwd` / `StartedAt` from the transcript header.
- `LocateSession` recognizes `.../subagents/.../agent-*.jsonl` paths. Explicitly listed subagent files
  become subagent locators with the parent taken from the directory.
- `ConvertRecords` sets `spawned_agent.sub_session_id` from `toolUseResult.agentId`.
  `LinkParentSubagents` then fills unlinked spawns by meta `toolUseId`, and finally by start time for
  any remaining spawn without a native link.

`convert claude-code`: explicit sources expand subagents unless `--no-subagents`; paths are
deduplicated so a list that names both parent and subagent files converts each once.

`discover claude-code --include-subagents`: after each kept parent, emit its subagents. All rows then
carry `session_kind` (`primary` | `subagent`), `parent_session_id` and `agent_type`. A subagent
follows its parent's selection: `--cwd-contains`, `--since` and `--active-since` apply to the parent
only (a subagent of a kept parent is always listed; a subagent of a dropped parent never is). Without
the flag the output is unchanged.

`autoConvert` (`minitracedb.LoadSessionContentAuto`): a Claude Code transcript whose conversation
records are sidechain records with an `agentId` converts as a subagent (`id = agentId`,
`predecessor_session = sessionId`, `source_format = claude-code-jsonl-v2+subagent`). Parent linking
needs no batch context because the parent's tool result names the `agentId`.

### 2. `context_loads`

New archive field `context_loads[]` (adapter-agnostic shape, populated by the Claude Code adapter) and
normalized table:

| Column | Meaning |
|---|---|
| `session_id`, `load_id` (PK) | `load_id` = `<attachment uuid>:<n>` |
| `ordinal` | position within the session |
| `timestamp` | attachment timestamp |
| `turn_index` | first turn emitted after the load (the first turn whose context includes it); NULL when none follows |
| `tool_call_id` | for `nested_memory`: the tool call whose result triggered the load (resolved through the `parentUuid` chain); otherwise NULL |
| `attachment_id` | the `attachments` row that carried it |
| `kind` | `nested_memory` \| `instructions` \| `invoked_skill` |
| `path` | home-normalized path (`~/...`); for skills the native identifier, for example `userSettings:diary` |
| `display_path` | native `displayPath` (nested_memory) |
| `name` | skill name (invoked_skill), else file basename |
| `file_type` | native type: `Project`, `User`, `Local`, `AutoMem`, ... (NULL for skills) |
| `content_chars`, `content_bytes` | rune and byte counts of the injected content |
| `content_sha256` | hex SHA-256 of the injected content |
| `differs_from_disk` | nested_memory `contentDiffersFromDisk` (0/1/NULL) |

### 3. `outcome_reason`

`ToolCallOutput.Reason` (`output.reason` in the archive) is materialized as
`tool_calls.outcome_reason`. Classification uses the start of the tool-result text, so tool output
that merely contains the phrase (for example grep output) is not misclassified:

| Value | Prefix / signal | success |
|---|---|---|
| `hook_blocked` | `The command did not run. A PreToolUse hook blocked it`, or a PreToolUse `hook_blocking_error` attachment for the call | forced 0 |
| `classifier_denied` | `Permission for this action was denied by the Claude Code auto mode classifier` | 0 |
| `permission_denied` | `Permission to use ` (permission rules / user deny) | 0 |
| `classifier_no_verdict` | `The server-side auto mode classifier gave no verdict` | 0 |
| `user_rejected` | `The user doesn't want to proceed with this tool use`, `The user doesn't want to take this action` | 0 |
| `interrupted` | `toolUseResult.interrupted` | 0 |
| `tool_use_error` | `<tool_use_error>` (harness validation: file not read, string not found, blocked sleep) | 0 |
| `harness_blocked` | `This agent is isolated in the worktree` | 0 |

`success` / `outcome_status` keep their meaning (succeeded/failed); the only value change is that a
hook-blocked call is now `success=0` (`failed`) because the command did not run. For every reason
except `interrupted` the tool did not execute, so `duration_ms` measures waiting, not work.

### 4. Hook events

Each `hook_success`, `hook_non_blocking_error`, `hook_blocking_error`, `hook_additional_context`,
`hook_cancelled` attachment emits its event with `kind='hook'` instead of `kind='attachment'` (the
`attachments` row is unchanged and still points at the event). `system/stop_hook_summary` emits a
`hook` event too. Hook events set `tool_call_id` when `toolUseID` names a tool call, `title` to the hook
name, `severity` to `error` for blocking errors and `warning` for non-blocking errors, and
`framework_metadata_json` to `{hook_type, hook_event, hook_name, command, exit_code, duration_ms,
permission_decision, prevented_continuation, ...}` as available.

### 5–6. Compaction and turn durations

`system` records with subtype `compact_boundary`, `turn_duration` or `stop_hook_summary` no longer
become turns; they become events (`compaction`, `turn_duration`, `hook`). Other system subtypes
(`away_summary`, `informational`, `local_command`) remain system turns.

- `compaction`: `turn_index` = first turn after the boundary; metadata `trigger`, `pre_tokens`,
  `post_tokens`, `duration_ms`.
- `turn_duration`: `turn_index` = the last turn before the record (the turn it closes); metadata
  `duration_ms`, `message_count`.
- `metrics.compaction_count`, `metrics.total_turn_duration_ms` (NULL when the source has none).

`history context-window` uses the latest `compaction` event at or before the target turn when present
(`boundary_source = "compact-boundary"`), and keeps the heuristics for transcripts without markers. A
new `--boundary-method marker-only` exposes the marker alone.

### Stub transcripts

`DetectJSONLFormat` also recognizes Claude Code from a record that has `sessionId` and a Claude Code
record type (`permission-mode`, `last-prompt`, `cost-state`, `mode`, `ai-title`,
`file-history-snapshot`, `attachment`, `queue-operation`, or `system` with a `subtype`). Such a file
converts to a session with no user/assistant turns, and the load diagnostic has severity `warning`
("no user or assistant records"). `discover` already lists these files; `convert` already converts
them.

## Design Decisions

- **Archive field, not materialize-time JSON parsing**, for `context_loads`: keeps Claude-specific
  parsing in the adapter and lets other adapters (Codex AGENTS.md, Pi) populate the same table later.
  Existing archives need reconversion.
- **Events, not new tables**, for hooks, compaction and turn durations: the `events` table already
  carries `kind`, `turn_index`, `tool_call_id`, `severity` and JSON metadata, and the web timeline
  renders events.
- **Add `outcome_reason`, keep `success`/`outcome_status`**: existing presets keep working; one
  documented value change (hook-blocked → failed).
- **Removing marker system turns changes `turn_count` and turn indexes** for Claude Code sessions.
  The repository does not keep compatibility layers; the change is documented in the schema help.
- **Native link ids over list order** for subagents; the ordering fallback only covers transcripts
  without `agentId` / meta files.

## Alternatives Considered

- Allow `json_each` in the sandbox: broad change to the security boundary and still forces every
  consumer to know Claude Code's attachment shapes.
- Port upstream v0.3.0 annotations/policy triggers: rejected; go-minitrace diverged deliberately.
- Separate `hooks` / `compactions` tables: more surface for little gain over typed events.

## Implementation Plan

1. Subagent discovery/locate/linking + convert/discover flags + autoConvert subagent detection.
2. Stub transcript detection.
3. `context_loads` archive field, adapter extraction, table, schema bump to `normalized-sqlite-v7`.
4. `outcome_reason` + hook events.
5. compaction / turn_duration events and metrics; context-window update.
6. Docs (`pkg/doc`), skills prose, fixtures and tests, real-corpus validation.

## Test Plan

- Unit tests in `pkg/adapters/claudecode` over synthetic records for each signal.
- A synthetic fixture tree `testdata/claude-code/projects/` (parent, subagent with meta, workflow
  journal, stub) used by discover/convert/autoConvert tests.
- `pkg/minitracedb` tests for `context_loads`, `outcome_reason` and new metrics columns.
- Real-corpus check (not committed): convert the playback project into a scratch directory and
  compare counts against the evidence above.

## Open Questions

- `hook_blocking_error` and `stop_hook_summary` field names come from upstream's preprocessor; no
  local transcript contains them, so they are tested with synthetic records only.
- Permission wait time cannot be separated from execution time for calls that eventually ran; the
  transcript records no approval timestamp.

## References

- Upstream: `fukami/minitrace` v0.3.0 `adapters/claude-code/claude-code.dsl.yaml`,
  `preprocess_claude_code.py`.
- Consumer ticket: playback PB-MINITRACE-001.
