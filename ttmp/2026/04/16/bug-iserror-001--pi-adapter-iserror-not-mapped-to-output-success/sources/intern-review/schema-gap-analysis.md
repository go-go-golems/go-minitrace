# Minitrace Schema Gap Analysis

**Date**: 2026-04-16  
**Analyst**: Intern Review Assignment  
**Scope**: Claude Code and Codex transcript sources

---

## Executive Summary

After reviewing raw Codex and Claude Code transcripts and comparing them with the resulting minitrace conversions, I identified **5 high-priority schema gaps** that significantly hinder cross-session analysis, **5 items that should remain in framework_metadata**, and several fields that are interesting but not worth standardizing at this time.

The most critical gaps are around **permission/approval tracking**, **command exit code fidelity**, **session lifecycle context**, **tool use justification**, and **model/streaming state changes**.

---

## Full Gap Analysis Table

| Source field / event | Found in raw transcript | Current minitrace mapping | Why insufficient | Proposed fix |
|----------------------|------------------------|---------------------------|------------------|--------------|
| `is_error` on tool_result (Claude) | `tool_result.is_error` boolean | Mapped to `tool_call.output.success` (inverted) | **Exit code ignored**: Codex provides explicit `exit_code` (0/1/2), but Claude only provides `is_error` boolean. The schema loses the distinction between different non-zero exit codes. Claude tool results with `is_error: true` have no structured error classification. | Add `tool_call.output.exit_code` as optional int, and `tool_call.output.error_category` enum for classifying failures |
| `exit_code` from Codex exec | `exec_command_end.exit_code` int | Only used to compute `success` boolean | **Exit codes flattened**: Codex captures rich exit codes (0, 1, 2, 130, etc.) but minitrace only stores success=true/false. Cannot query "which sessions had signal-terminated commands?" | Promote `exit_code` to first-class field in `tool_call.output`. Add `exit_signal` for signal-terminated processes |
| `justification` for tool use | `function_call.arguments.justification` | Buried in `framework_metadata.justification` | **Hard to query**: Tool use rationale is scattered in metadata. Cannot easily answer "which sessions used tools without justification?" or "what reasons were given for destructive operations?" | Add `tool_call.input.justification` as optional string field |
| `sandbox_policy` / `sandbox_permissions` | `turn_context.sandbox_policy` object | Mapped to `operational_context.sandbox` bool | **Too coarse**: Only captures sandbox=true/false, loses policy details (workspace-write, danger-full-access, etc.) and escalation request patterns | Add `operational_context.sandbox_policy` string enum and `condition.permission_level` should capture escalation events |
| `stdout` vs `stderr` separation | `exec_command_end.stdout`, `.stderr` | Combined into `output.result` | **Stream separation lost**: Cannot identify sessions where errors went to stderr but command "succeeded", or where stdout was empty but stderr had warnings | Add `tool_call.output.stdout` and `tool_call.output.stderr` as separate optional fields |
| `rate_limits` / quota | `token_count.info.rate_limits` object | Not captured | **API pressure invisible**: Cannot analyze "which sessions hit rate limits?" or "which had high quota usage?" Useful for understanding interruptions and retries. | Add `condition.rate_limit_pressure` enum or `metrics.rate_limit_hits` int |
| `collaboration_mode` changes | `turn_context.collaboration_mode.mode` | Buried in `framework_config` | **Mode switches invisible**: Cannot query "which sessions switched from Plan to Default mode?" The `metrics.model_switches` exists but no equivalent for collaboration mode | Add `metrics.collaboration_mode_switches` int and `condition.collaboration_mode` string |
| `phase` (commentary vs acting) | `agent_message.phase` | Not captured | **Intent unclear**: Codex distinguishes "commentary" (thinking aloud) from other phases. Useful for understanding when the model is explaining vs doing | Add `turn.phase` enum field with values like `commentary`, `acting`, `planning` |
| `memory_citation` | `agent_message.memory_citation` | Not captured | **Memory usage untracked**: Cannot answer "which sessions relied on memory?" or "which tool calls cited memory?" | Add `turn.memory_citations` string array or `tool_call.context.memory_cited` bool |
| `source` for commands | `exec_command_end.source` | Not captured | **Command origin lost**: Cannot distinguish user-typed commands vs agent-initiated vs startup commands | Add `tool_call.input.command_source` enum: `user`, `agent`, `startup`, `approved_rule` |
| `stop_reason` / `stop_sequence` | `assistant.message.stop_reason` | Not captured | **Completion quality signal**: stop_reason=tool_use vs max_tokens vs stop_sequence indicates different completion patterns | Add `turn.stop_reason` enum: `tool_use`, `max_tokens`, `stop_sequence`, `end_turn` |
| `cache_creation` time buckets | `usage.cache_creation.ephemeral_5m_input_tokens`, `ephemeral_1h_input_tokens` | Only `cache_creation_tokens` total | **Cache TTL invisible**: Claude distinguishes 5-min vs 1-hour ephemeral cache. Useful for understanding working set size and session patterns | Add `usage.cache_ephemeral_5m_tokens` and `usage.cache_ephemeral_1h_tokens` |
| `parentUuid` / conversation threading | `parentUuid` on each record | Not captured | **Threading lost**: Cannot reconstruct conversation tree or identify sidechains | Add `turn.parent_turn_index` int and `turn.is_sidechain` bool |
| `slug` (session nickname) | `slug` field | Only used for subagent title prefixing | **Session identity lost**: Human-meaningful slugs like "compiled-zooming-finch" are useful for recall | Add `session.slug` optional string field |
| `skills` active list | `skills_instructions` block with full skill inventory | Not captured | **Capabilities context lost**: Cannot answer "which sessions had skill X available?" or "which used skills vs raw tools?" | Add `environment.available_skills` string array |
| `personality` spec | `personality_spec` object | Buried in `framework_config` | **Communication style**: Useful for analyzing how personality affects output style, verbosity, token usage | Keep in `framework_config` (too specific to Codex) |
| `truncation_policy` | `turn_context.truncation_policy` | Not captured | **Context pressure invisible**: Cannot identify sessions approaching context limits | Add `condition.context_pressure` enum: `normal`, `approaching_limit`, `truncated` |
| `encrypted_content` (reasoning) | `response_item.encrypted_content` | Not captured | **Opaque reasoning**: Codex encrypts some reasoning content. Having a marker that reasoning was present but encrypted is useful | Add `turn.has_encrypted_reasoning` bool or `thinking` field should indicate encryption status |
| `plugins` active list | `plugins_instructions` block | Not captured | **Extension context**: Cannot answer "which sessions had GitHub plugin enabled?" | Add `environment.available_plugins` string array |
| `turn_id` correlation | `turn_id` across events | Not captured | **Event correlation**: Multiple Codex events share a `turn_id`. Minitrace turns don't expose this correlation ID | Add `turn.framework_turn_id` string field for cross-referencing |

---

## Top 5 Schema Gaps (High Priority)

### 1. **Tool Exit Codes as First-Class Field**

**Gap**: Both adapters flatten exit codes to boolean `success`. Codex provides rich exit codes (0, 1, 2, 130, etc.) and Claude provides `is_error` boolean.

**Why it matters**: 
- "Which commands failed with exit code 2 (usage error) vs 1 (runtime error)?"
- "Which sessions had signal-terminated processes (exit code 130+)?"
- "Do retry patterns correlate with specific exit codes?"

**Raw example**: 
- Source: `~/.codex/sessions/2026/04/14/rollout-*.jsonl`
- Raw: `{"exit_code": 2, "status": "failed", ...}`
- Converted: `minitrace.json` has only `"success": false`

**Proposed fix**: 
- Add `tool_call.output.exit_code` as optional int
- Add `tool_call.output.exit_signal` for signal-terminated processes
- Add `tool_call.output.error_category` enum: `runtime_error`, `usage_error`, `signal_terminated`, `timeout`, `unknown`

**Schema impact**: New optional fields in `ToolCallOutput` struct.

---

### 2. **Tool Use Justification**

**Gap**: Codex captures `justification` in `function_call.arguments.justification`, currently buried in `framework_metadata.justification`.

**Why it matters**:
- "Which destructive operations (rm, git reset) had no justification?"
- "What patterns of justification lead to successful outcomes?"
- "Do sessions with more justifications have better tool success rates?"

**Raw example**:
- Source: `~/.codex/sessions/2026/04/14/rollout-*.jsonl`
- Raw: `{"type":"function_call","arguments":"{\"cmd\":\"rm -rf node_modules\",\"justification\":\"clean install per user request\"}"}`
- Converted: Buried in `tool_calls[N].framework_metadata.justification`

**Proposed fix**:
- Add `tool_call.input.justification` as optional string field
- Add `tool_call.input.requires_approval` bool (inferred from tool type + sandbox policy)

**Schema impact**: New optional field in `ToolCallInput` struct.

---

### 3. **Sandbox Policy and Escalation Events**

**Gap**: `operational_context.sandbox` is boolean, but Codex has rich policy: `workspace-write`, `danger-full-access`, with escalation requests.

**Why it matters**:
- "Which sessions ran in restricted vs full-access sandbox?"
- "Which commands required escalation requests?"
- "What approval rules were created during sessions?"

**Raw example**:
- Source: `~/.codex/sessions/2026/04/14/rollout-*.jsonl`
- Raw: `{"sandbox_policy":{"type":"workspace-write","writable_roots":[...],"network_access":false}}`
- Converted: `"sandbox": true` (boolean)

**Proposed fix**:
- Add `operational_context.sandbox_policy` string enum: `read-only`, `workspace-write`, `danger-full-access`, `custom`
- Add `condition.permission_level` string enum: `unrestricted`, `sandboxed`, `escalation-requested`, `escalation-approved`, `escalation-denied`
- Capture escalation requests and approvals as events/annotations

**Schema impact**: New fields in `OperationalContext` and `Condition` structs.

---

### 4. **stdout/stderr Separation**

**Gap**: Codex captures separate `stdout` and `stderr` in `exec_command_end`, but minitrace merges them into `output.result`.

**Why it matters**:
- "Which 'successful' commands produced stderr warnings?"
- "Which sessions had silent failures (exit 0 but stderr not empty)?"
- "What's the ratio of stdout to stderr volume per session?"

**Raw example**:
- Source: `~/.codex/sessions/2026/04/14/rollout-*.jsonl`
- Raw: `{"stdout":"","stderr":"warning: unused variable\n","exit_code":0}`
- Converted: Merged into result string

**Proposed fix**:
- Add `tool_call.output.stdout` optional string
- Add `tool_call.output.stderr` optional string
- Keep `tool_call.output.result` as the combined/aggregated view

**Schema impact**: New optional fields in `ToolCallOutput` struct.

---

### 5. **Collaboration Mode and Streaming State**

**Gap**: Codex tracks `collaboration_mode` (Default vs Plan) and `realtime_active` (streaming state). These are buried in `framework_config` or not captured.

**Why it matters**:
- "Which sessions switched from Plan to Default mode?"
- "Does streaming (realtime_active) correlate with longer sessions?"
- "Do Plan mode sessions have different tool-use patterns?"

**Raw example**:
- Source: `~/.codex/sessions/2026/04/14/rollout-*.jsonl`
- Raw: `{"collaboration_mode":{"mode":"default","settings":{"reasoning_effort":"high"}},"realtime_active":false}`
- Converted: Buried in `operational_context.framework_config.collaboration_mode`

**Proposed fix**:
- Add `condition.collaboration_mode` string: `default`, `plan`, `custom`
- Add `metrics.collaboration_mode_switches` int
- Add `turn.streaming.was_streamed` already exists but verify `realtime_active` maps to it correctly
- Add `turn.streaming.stream_phase` enum: `thinking`, `acting`, `commentary`

**Schema impact**: New fields in `Condition`, `Metrics`, and `Streaming` structs.

---

## Top 5 "Keep in framework_metadata" Items

These fields are too source-specific to standardize, but valuable for debugging:

| Field | Location | Reason for keeping in metadata |
|-------|----------|-------------------------------|
| `codex_function` | `tool_call.framework_metadata.codex_function` | Codex-specific function names (exec_command, read_file) that map to generic operation types |
| `personality` spec | `session.operational_context.framework_config.personality` | Codex-specific communication style configuration; other frameworks don't have equivalent |
| `caller` (Claude) | `turn.framework_metadata.caller` | Claude-specific detail about who initiated tool use (`direct`, `auto`); too specific to generalize |
| `entrypoint` (Claude) | `session.framework_config.entrypoint` | How Claude session was started (`sdk-ts`, `cli`, `api`); framework-specific |
| `sourceToolAssistantUUID` | `turn.framework_metadata.source_tool_assistant_uuid` | Claude internal correlation ID; not useful for cross-session analysis |

---

## Interesting But Not Worth Standardizing

| Field | Why not worth standardizing |
|-------|----------------------------|
| `requestId` (Claude) | Internal API correlation ID, not meaningful for analysis |
| `service_tier` / `inference_geo` | Operational metrics, not session behavior |
| `cache_creation` time buckets (5m/1h) | Very specific to Anthropic's cache implementation |
| `encrypted_content` marker | Only indicates content was encrypted, not useful for analysis |
| `skills_instructions` full text | Too large, too specific; `available_skills` list is sufficient |
| `truncation_policy` details | Framework-specific token counting; session-level `context_pressure` is sufficient |
| `slug` (session nickname) | Cute but not analytically valuable |
| `parentUuid` / conversation tree | Complex to model; linear `turns` array is sufficient for most analysis |
| `plugins` full configuration | `available_plugins` list is sufficient |
| `approval_policy` on-request vs never | Covered by `sandbox` boolean and `autonomy_level` |

---

## Concrete Examples from Conversion

### Claude Code Session: `3e25fe06-537b-40cf-9903-2e7bf1b1cd0d`

**Raw source**: `~/.claude/projects/-home-manuel-code-others-pretext/3e25fe06-537b-40cf-9903-2e7bf1b1cd0d.jsonl`

**Converted**: `/tmp/intern-review/active/2026-03/3e25fe06-537b-40cf-9903-2e7bf1b1cd0d.minitrace.json`

**Lost information**:
- `caller: {type: "direct"}` on tool_use → not captured
- `service_tier: "standard"`, `inference_geo: "not_available"` → not captured
- `stop_reason: "tool_use"` on assistant messages → not captured
- `slug: "compiled-zooming-finch"` → only used for title, not preserved
- `sourceToolAssistantUUID` linking tool results to assistant messages → implicit via ordering only

**Preserved well**:
- Token usage (input, output, cache_read, cache_creation)
- Tool calls and results
- Timestamps
- Model (`claude-opus-4-6`)
- Working directory, git branch

### Codex Session: `rollout-2026-04-14T20-17-00-019d8e7f-99a5-7243-b19a-845778ff2b5a`

**Raw source**: `~/.codex/sessions/2026/04/14/rollout-2026-04-14T20-17-00-019d8e7f-99a5-7243-b19a-845778ff2b5a.jsonl`

**Not converted** (failed with `unsupported Codex format hint: unknown-jsonl` on older sessions)

**Observed rich data**:
- `exit_code: 0/1/2` → converted only to `success: true/false`
- `justification: "inspect file"` → buried in `framework_metadata`
- `parsed_cmd: [{"type":"list_files","cmd":"ls -la","path":null}]` → not captured
- `source: "unified_exec_startup"` → not captured
- `rate_limits: {primary:{used_percent:10.0}, secondary:{used_percent:36.0}}` → not captured
- `sandbox_policy: {type:"workspace-write",writable_roots:[...]}` → flattened to `sandbox: true`
- `stdout: ""`, `stderr: ""` → merged into aggregated output
- `phase: "commentary"` on agent_message → not captured
- `memory_citation: null` → not captured
- `turn_id: "019d8e81-d681-7521-bd7d-83847355e132"` → not captured (for event correlation)

---

## Summary

The minitrace schema captures the essential structure of agent sessions well, but flattens or loses several dimensions critical for rich analysis:

1. **Command outcomes** are reduced to booleans, losing exit codes and stderr patterns
2. **Approval and sandbox context** is simplified, losing escalation patterns and policy details
3. **Tool use intent** (justification) is buried in metadata
4. **Session mode and streaming state** changes are not tracked as first-class events
5. **API pressure signals** (rate limits, context window pressure) are dropped

These gaps make it difficult to answer important cross-session questions about reliability, safety, performance, and user experience patterns.

---

*End of analysis*
