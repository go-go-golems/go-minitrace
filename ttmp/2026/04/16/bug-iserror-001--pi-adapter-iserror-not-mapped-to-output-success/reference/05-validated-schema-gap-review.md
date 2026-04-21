# Validated Schema Gap Review

## Scope

This note is a source-backed follow-up to the intern schema-gap write-up. I re-reviewed the claims against:

- raw Claude Code transcript:
  - `~/.claude/projects/-home-manuel-code-others-pretext/3e25fe06-537b-40cf-9903-2e7bf1b1cd0d.jsonl`
- converted Claude Code output from the intern review archive:
  - `ttmp/2026/04/16/bug-iserror-001--pi-adapter-iserror-not-mapped-to-output-success/sources/intern-review/active/2026-03/3e25fe06-537b-40cf-9903-2e7bf1b1cd0d.minitrace.json`
- raw Codex transcripts:
  - `~/.codex/sessions/2026/01/12/rollout-2026-01-12T15-46-57-019bb3f6-3c71-7013-b585-4f16d9bdceb6.jsonl`
  - `~/.codex/sessions/2026/04/14/rollout-2026-04-14T20-17-00-019d8e7f-99a5-7243-b19a-845778ff2b5a.jsonl`
- current adapters:
  - `pkg/adapters/codex/convert.go`
  - `pkg/adapters/claudecode/convert.go`
  - `pkg/minitrace/schema.go`

I also locally converted the two Codex sessions above with the current adapter to confirm what survives into minitrace today.

## Important correction

The intern review was directionally useful, but it overstated the evidence quality on the Codex side. The checked-in `sources/intern-review/` archive contains Claude Code conversions only, not Codex conversions. So the strongest Codex claims should be treated as validated by raw transcript + adapter behavior, not by the intern archive itself.

## Findings

### 1. High-confidence schema promotions we should do first

#### A. Codex `exit_code` should become a first-class `ToolCallOutput` field

This is the clearest missing field.

Evidence:
- Codex raw session data contains explicit command exit codes.
- `pkg/adapters/codex/convert.go` already reads exit codes in both parser paths.
- Today the schema keeps only `output.success` and sometimes buries the raw exit code in `framework_metadata`, which makes cross-session analysis awkward and inconsistent.

Recommendation:
- Add `tool_calls[].output.exit_code` as `*int` in `pkg/minitrace/schema.go`.
- Populate it for both Codex session-jsonl-v1 and exec-jsonl-v1 conversion paths.

#### B. Codex `justification` should become a first-class `ToolCallInput` field

This is also high-confidence and already partially extracted.

Evidence:
- Codex raw `function_call.arguments` includes `justification` strings for command/tool use.
- `pkg/adapters/codex/convert.go` already extracts that value into `framework_metadata.justification`.
- Because it is buried in metadata, it is hard to query across sessions.

Recommendation:
- Add `tool_calls[].input.justification` as `*string` in `pkg/minitrace/schema.go`.
- Populate it in the Codex adapter while keeping the raw metadata if useful.

### 2. High-confidence fields we should preserve first, but keep in metadata/config for now

These fields are real and valuable, but I do not think they should all become first-class schema fields immediately.

#### Codex: preserve richer session/runtime context

Validated in raw Codex transcripts:
- `approval_policy`
- `sandbox_policy` object
- `collaboration_mode`
- `truncation_policy`
- `rate_limits`
- `turn_id`
- `phase`
- `memory_citation`
- `exec_command_end.source`
- `exec_command_end.parsed_cmd`
- `exec_command_end.stdout`
- `exec_command_end.stderr`

Current state:
- some are flattened too aggressively (`sandbox` bool, `autonomy_level` heuristic)
- some are partly preserved in `framework_config`
- many are dropped entirely

Recommendation:
- first preserve these in `framework_config`, `turn.framework_metadata`, or `tool_call.framework_metadata`
- only promote to first-class schema fields after we see repeated analytical need across frameworks

#### Claude Code: preserve richer message/tool metadata

Validated in raw Claude transcripts:
- `caller` on tool use blocks
- `entrypoint`
- `stop_reason`
- `stop_sequence`
- `slug`
- `parentUuid`
- `isSidechain`
- `usage.cache_creation.ephemeral_5m_input_tokens`
- `usage.cache_creation.ephemeral_1h_input_tokens`

Current state:
- tool calls and usage totals survive well
- the fields above are mostly dropped
- `framework_metadata` is often `null` in converted output even when the raw transcript carries richer structure

Recommendation:
- preserve these first in metadata/config
- defer schema promotion except maybe `stop_reason`, which is the strongest later candidate

### 3. Things to defer until after the metadata-preservation pass

These may be useful, but I do not think they should be first-wave schema work:

- `stdout` / `stderr` as first-class fields
- `stop_reason` as a first-class turn field
- normalized sandbox policy enum
- `session.slug`
- thread/tree modeling from `parentUuid` / `isSidechain`
- `available_skills` / `available_plugins`
- collaboration/stream-phase counters
- heuristic `error_category`

These should be reconsidered after we preserve the raw information and prove concrete query use cases.

## Recommended implementation order

1. Promote `exit_code` into `ToolCallOutput`.
2. Promote `justification` into `ToolCallInput`.
3. Do a Codex metadata-preservation pass for policy/mode/rate-limit/turn/tool context.
4. Do a Claude Code metadata-preservation pass for caller/entrypoint/stop-reason/slug/thread/cache-bucket detail.
5. Add regression tests for both adapters.
6. Update schema/help docs after field names settle.

## Bottom line

The best next move is not a large schema redesign. It is:

- two small high-confidence schema additions now: `exit_code`, `justification`
- followed by a metadata-preservation pass in Codex and Claude Code adapters
- then a second decision on which preserved fields deserve first-class schema status
