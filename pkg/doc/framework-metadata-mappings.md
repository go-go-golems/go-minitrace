---
Title: Framework Metadata Mappings
Slug: framework-metadata-mappings
Short: Where adapter-specific raw fields are preserved in minitrace for Codex, Claude Code, and Pi
Topics:
- minitrace
- schema
- adapters
- codex
- claude-code
- pi
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This page documents the adapter-specific metadata that minitrace preserves **without** promoting it into shared first-class schema fields.

Use it together with:
- `go-minitrace help minitrace-schema` for the shared schema
- `go-minitrace help adapter-reference` for adapter overviews

## Storage conventions

Framework-specific raw fields are preserved in three places:

| Location | Use for |
|---|---|
| `operational_context.framework_config` | Session/runtime configuration that applies broadly across the session |
| `turns[].framework_metadata` | Raw turn/message metadata that does not fit the shared turn schema |
| `tool_calls[].framework_metadata` | Tool-call-specific metadata that is useful but framework-specific |

## Codex mappings

### `operational_context.framework_config`

| Stored key | Raw source |
|---|---|
| `personality` | `turn_context.payload.personality` |
| `collaboration_mode` | `turn_context.payload.collaboration_mode.mode` |
| `collaboration_mode_detail` | `turn_context.payload.collaboration_mode` |
| `reasoning_effort` | `turn_context.payload.effort` or collaboration settings |
| `originator` | `session_meta.payload.originator` |
| `session_source` | `session_meta.payload.source` |
| `approval_policy` | `turn_context.payload.approval_policy` |
| `sandbox_policy` | `turn_context.payload.sandbox_policy` |
| `truncation_policy` | `turn_context.payload.truncation_policy` |
| `rate_limits` | latest `event_msg.payload.rate_limits` from token-count events |
| `model_context_window` | `task_started` / token-count info |
| `timezone` | `turn_context.payload.timezone` |

### `turns[].framework_metadata`

| Stored key | Raw source |
|---|---|
| `turn_id` | `turn_context.payload.turn_id`, `task_started.turn_id`, or exec item `turn_id` |
| `phase` | assistant event/item `phase` |
| `memory_citation` | assistant event `memory_citation` |

### `tool_calls[].framework_metadata`

| Stored key | Raw source |
|---|---|
| `codex_function` | tool/function name used by the adapter |
| `justification` | `function_call.arguments.justification` when present |
| `source` | `exec_command_end.source` or exec item `source` |
| `parsed_cmd` | `exec_command_end.parsed_cmd` or exec item `parsed_cmd` |
| `stdout` | `exec_command_end.stdout` or exec item `stdout` |
| `stderr` | `exec_command_end.stderr` or exec item `stderr` |
| `status` | exec event/item status |
| `turn_id` | exec event/item `turn_id` |
| `exit_code` | retained in metadata for compatibility, even though a first-class `output.exit_code` now exists |

## Claude Code mappings

### `operational_context.framework_config`

| Stored key | Raw source |
|---|---|
| `entrypoint` | top-level Claude record `entrypoint` |

### `turns[].framework_metadata`

| Stored key | Raw source |
|---|---|
| `entrypoint` | top-level Claude record `entrypoint` |
| `slug` | top-level Claude record `slug` |
| `parent_uuid` | top-level Claude record `parentUuid` |
| `is_sidechain` | top-level Claude record `isSidechain` |
| `stop_reason` | `message.stop_reason` |
| `stop_sequence` | `message.stop_sequence` |
| `cache_creation` | `message.usage.cache_creation` |

### `tool_calls[].framework_metadata`

| Stored key | Raw source |
|---|---|
| `caller` | `assistant.message.content[].caller` on `tool_use` blocks |
| `entrypoint` | preserved from the emitting assistant/tool-result record |
| `slug` | preserved from the emitting assistant/tool-result record |
| `parent_uuid` | preserved from the emitting assistant/tool-result record |
| `is_sidechain` | preserved from the emitting assistant/tool-result record |

## Pi mappings

### `operational_context.framework_config`

Pi currently preserves a small session config blob rather than the larger framework-specific metadata set used by Codex and Claude Code.

| Stored key | Raw source |
|---|---|
| `thinking_level` | `thinking_level_change.thinkingLevel` |
| `api` | `model_change.api` or message-level `api` |

### `turns[].framework_metadata`

| Stored key | Raw source |
|---|---|
| `stop_reason` | `message.stopReason` |
| `error_message` | `message.errorMessage` |

### `tool_calls[].framework_metadata`

| Stored key | Raw source |
|---|---|
| `diff` | `toolResult.details.diff` |
| `first_changed_line` | `toolResult.details.firstChangedLine` |

## Notes

- These metadata keys are intentionally framework-specific. They are preserved for analysis and debugging, but they are not guaranteed to exist across adapters.
- First-class fields should be reserved for values with stable cross-framework meaning. For example, `output.exit_code` and `input.justification` were promoted because they are directly queryable and have clear semantics.
- If a field graduates from metadata into the shared schema later, this page should be updated to document both the first-class field and any metadata retained for backward compatibility.
