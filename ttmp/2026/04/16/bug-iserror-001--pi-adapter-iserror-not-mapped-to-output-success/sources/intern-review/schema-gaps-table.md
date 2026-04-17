# Schema-Worthy Information Missing from Codex and Claude Transcripts

## Summary Table

| Source field / event | Found in raw transcript | Current minitrace mapping | Why insufficient | Proposed fix |
|----------------------|------------------------|---------------------------|------------------|--------------|
| `exit_code` (Codex) | `exec_command_end.exit_code` int (0/1/2/130) | Converted to `tool_call.output.success` boolean | Exit codes flattened: Cannot query "which sessions had signal-terminated commands?" or distinguish error types | **First-class field**: Add `tool_call.output.exit_code` int and `tool_call.output.error_category` enum |
| `is_error` → exit classification (Claude) | `tool_result.is_error` boolean | Mapped to `success` (inverted) | Claude only provides boolean, but we could classify errors from result content. Cannot identify error patterns across sessions | **First-class field**: Add `tool_call.output.error_category` enum: `runtime_error`, `usage_error`, `timeout`, `unknown` |
| `justification` for tool use | `function_call.arguments.justification` (Codex) | Buried in `framework_metadata.justification` | Hard to query: Cannot answer "which destructive operations lacked justification?" | **First-class field**: Add `tool_call.input.justification` string |
| `sandbox_policy` details | `turn_context.sandbox_policy` object with `type`, `writable_roots`, `network_access` | Mapped to `operational_context.sandbox` bool | Too coarse: Loses policy types (`workspace-write`, `danger-full-access`) and escalation request patterns | **First-class fields**: Add `operational_context.sandbox_policy` string enum and `condition.permission_level` with escalation states |
| `stdout` / `stderr` separation | `exec_command_end.stdout`, `.stderr` strings | Combined into `output.result` | Stream separation lost: Cannot identify silent failures (exit 0, stderr non-empty) | **First-class fields**: Add `tool_call.output.stdout` and `tool_call.output.stderr` as separate optional fields |
| `rate_limits` / quota | `token_count.info.rate_limits` with `used_percent`, `plan_type` | Not captured | API pressure invisible: Cannot analyze "which sessions hit rate limits?" or correlate with interruptions | **First-class field**: Add `condition.rate_limit_pressure` enum or `metrics.rate_limit_hits` int |
| `collaboration_mode` | `turn_context.collaboration_mode.mode` (Default/Plan) | Buried in `framework_config` | Mode switches invisible: Cannot query "which sessions switched from Plan to Default?" | **First-class fields**: Add `condition.collaboration_mode` string and `metrics.collaboration_mode_switches` int |
| `phase` (commentary vs acting) | `agent_message.phase` = "commentary" or omitted | Not captured | Intent unclear: Cannot distinguish explanatory vs action messages | **First-class field**: Add `turn.phase` enum: `commentary`, `acting`, `planning` |
| `source` for commands | `exec_command_end.source` = "unified_exec_startup" / user / etc. | Not captured | Command origin lost: Cannot distinguish user-typed vs agent-initiated vs startup | **First-class field**: Add `tool_call.input.command_source` enum: `user`, `agent`, `startup`, `approved_rule` |
| `memory_citation` | `agent_message.memory_citation` | Not captured | Memory usage untracked: Cannot answer "which sessions relied on memory?" | **First-class field**: Add `turn.memory_citations` string array |
| `stop_reason` / `stop_sequence` | `assistant.message.stop_reason` = "tool_use", "max_tokens", etc. | Not captured | Completion quality signal: Different stop reasons indicate different completion patterns | **First-class field**: Add `turn.stop_reason` enum: `tool_use`, `max_tokens`, `stop_sequence`, `end_turn` |
| `cache_creation` time buckets | `usage.cache_creation.ephemeral_5m_input_tokens`, `ephemeral_1h_input_tokens` | Only total `cache_creation_tokens` | Cache TTL invisible: Cannot analyze working set size or session cache efficiency | **First-class fields**: Add `usage.cache_ephemeral_5m_tokens` and `usage.cache_ephemeral_1h_tokens` |
| `turn_id` correlation | `turn_id` UUID across events in Codex | Not captured | Event correlation lost: Cannot correlate multiple Codex events to a single logical turn | **First-class field**: Add `turn.framework_turn_id` string for cross-referencing |
| `skills` active list | `skills_instructions` block with full inventory | Not captured | Capabilities context lost: Cannot answer "which sessions had skill X available?" | **First-class field**: Add `environment.available_skills` string array |
| `plugins` active list | `plugins_instructions` block | Not captured | Extension context lost: Cannot answer "which sessions had GitHub plugin enabled?" | **First-class field**: Add `environment.available_plugins` string array |
| `truncation_policy` | `turn_context.truncation_policy` = `{mode:"tokens",limit:10000}` | Not captured | Context pressure invisible: Cannot identify sessions approaching context limits | **First-class field**: Add `condition.context_pressure` enum: `normal`, `approaching_limit`, `truncated` |
| `encrypted_content` marker | `response_item.encrypted_content` (encrypted reasoning) | Not captured | Opaque reasoning: Cannot distinguish sessions with encrypted vs plaintext reasoning | **First-class field**: Add `turn.has_encrypted_reasoning` bool or extend `thinking` field |
| `slug` (session nickname) | `slug` field like "compiled-zooming-finch" | Only used for subagent title | Session identity lost: Human-meaningful identifiers useful for recall | **First-class field**: Add `session.slug` optional string |
| `parsed_cmd` structure | `exec_command_end.parsed_cmd` = `[{type:"list_files",cmd:"ls",path:null}]` | Not captured | Structured command info lost: Cannot analyze command types without regex parsing | **Keep in metadata**: Too Codex-specific to standardize |
| `parentUuid` / threading | `parentUuid` on each record | Not captured | Threading lost: Cannot reconstruct conversation tree | **Keep in metadata**: Complex tree structure; linear turns sufficient for most analysis |
| `personality` spec | `personality_spec` object with values, style | Buried in `framework_config` | Communication style context | **Keep in metadata**: Too Codex-specific |
| `requestId` (Claude) | `requestId` UUID per API call | Not captured | Internal correlation | **Ignore**: Not meaningful for analysis |
| `service_tier` / `inference_geo` | `service_tier: "standard"`, `inference_geo` | Not captured | Operational metrics | **Ignore**: Not session behavior |
| `entrypoint` (Claude) | `entrypoint: "sdk-ts"`, `"cli"`, etc. | Not captured | How session started | **Keep in metadata**: Framework-specific |

---

## Top 5 Schema Gaps (Priority Order)

1. **Tool Exit Codes** - Currently flattened to boolean, losing rich error classification from Codex (0/1/2/130+) and error patterns from Claude
2. **Tool Use Justification** - Buried in metadata, making it impossible to query "which operations lacked rationale?"
3. **Sandbox Policy & Escalation** - Boolean `sandbox` field loses policy types and escalation request/approval patterns
4. **stdout/stderr Separation** - Merged output streams prevent identifying silent failures and stderr warnings
5. **Collaboration Mode & Phase** - Session mode switches and message phases (commentary vs acting) not tracked as first-class

## Top 5 "Keep in framework_metadata" Items

1. `codex_function` - Codex-specific function names mapping to generic operations
2. `personality` spec - Codex-specific communication style configuration  
3. `caller` (Claude) - Who initiated tool use (direct/auto); Claude-specific
4. `entrypoint` (Claude) - How session was started (sdk-ts/cli/api)
5. `sourceToolAssistantUUID` - Claude internal correlation ID

## Fields to Intentionally Ignore

- `requestId`, `service_tier`, `inference_geo` - Internal operational metrics
- `cache_creation` time buckets (5m/1h) - Too implementation-specific
- `encrypted_content` marker - Only indicates encryption presence
- `skills_instructions` full text - Too large; `available_skills` list is sufficient
- `truncation_policy` details - `context_pressure` enum is sufficient
- `slug` - Not analytically valuable despite being human-friendly
- `parentUuid` / conversation tree - Complex; linear turns sufficient

---

## Concrete Examples

### Raw Claude Code (Lost Information)

**Source**: `~/.claude/projects/-home-manuel-code-others-pretext/3e25fe06-537b-40cf-9903-2e7bf1b1cd0d.jsonl`

```json
{
  "type": "assistant",
  "message": {
    "stop_reason": "tool_use",
    "stop_sequence": null,
    "usage": {
      "cache_creation": {
        "ephemeral_5m_input_tokens": 0,
        "ephemeral_1h_input_tokens": 723
      }
    }
  },
  "caller": {"type": "direct"},
  "slug": "compiled-zooming-finch",
  "entrypoint": "sdk-ts"
}
```

**Converted**: `/tmp/intern-review/active/2026-03/3e25fe06-537b-40cf-9903-2e7bf1b1cd0d.minitrace.json`
- `stop_reason` → NOT CAPTURED
- `cache_creation` buckets → Only total preserved
- `caller`, `slug`, `entrypoint` → NOT CAPTURED

### Raw Codex (Lost Information)

**Source**: `~/.codex/sessions/2026/04/14/rollout-2026-04-14T20-17-00-019d8e7f-99a5-7243-b19a-845778ff2b5a.jsonl`

```json
{
  "type": "event_msg",
  "payload": {
    "type": "exec_command_end",
    "exit_code": 2,
    "stdout": "",
    "stderr": "ls: cannot access 'foo': No such file or directory",
    "parsed_cmd": [{"type": "list_files", "cmd": "ls foo", "path": "foo"}],
    "source": "unified_exec_startup"
  }
}
```

```json
{
  "type": "turn_context",
  "payload": {
    "sandbox_policy": {"type": "workspace-write", "writable_roots": ["/tmp"]},
    "collaboration_mode": {"mode": "default", "settings": {"reasoning_effort": "high"}},
    "approval_policy": "on-request"
  }
}
```

**Converted**:
- `exit_code: 2` → `"success": false` only
- `stdout`/`stderr` → merged, separation lost
- `parsed_cmd`, `source` → NOT CAPTURED
- `sandbox_policy` type → `"sandbox": true` only
- `collaboration_mode` → Buried in `framework_config`

---

*Analysis completed 2026-04-16*
