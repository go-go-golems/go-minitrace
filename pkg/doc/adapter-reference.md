---
Title: Adapter Reference
Slug: adapter-reference
Short: How each source format maps to the minitrace schema during conversion
Topics:
- minitrace
- convert
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Each conversion subcommand uses an adapter that reads a native session format and maps it to the minitrace schema. This page documents what each adapter reads, what gets preserved, what gets synthesized, and what is lost.

## Claude Code adapter

**Source**: JSONL v2 transcripts in `~/.claude/projects/`
**Source format**: `claude-code-jsonl-v2` (main), `claude-code-jsonl-v2+subagent` (subagent)
**Converter version**: `go-minitrace-claude-adapter-dev`

### What it reads

Claude Code stores one JSONL file per session. Each line is a JSON event with a type field. The adapter processes:

- **Conversation messages** → turns (user, assistant, system)
- **Tool use blocks** → tool calls with operation type mapping
- **Tool result blocks** → tool call outputs with success/error status
- **Usage metadata** → per-turn token counts (input, output, cache_read, cache_creation)
- **Subagent directories** → separate minitrace sessions with parent backlinking

Additionally, dir-v1 tool-results sessions (an older format that stores tool results in a directory structure) are detected and converted.

### Field mapping

| Minitrace field | Source |
|----------------|--------|
| `environment.model` | Extracted from assistant message metadata |
| `environment.agent_framework` | `claude-code` |
| `environment.platform_type` | `agent` |
| `environment.provider_hint` | `anthropic` |
| `environment.tools_enabled` | Unique tool names from tool use blocks |
| `timing.*` | Computed from message timestamps |
| `title` | First 80 characters of the first human message |
| `turns[].thinking` | Extracted from thinking blocks if present |
| `turns[].usage` | Per-message token counts |

### Tool operation mapping

| Tool name | Operation type |
|-----------|---------------|
| `Read`, `Grep`, `Glob` | `READ` |
| `Edit` | `MODIFY` |
| `Write` | `NEW` |
| `Bash` | `EXECUTE` |
| `Agent` | `DELEGATE` |
| Everything else | `OTHER` |

### Subagent handling

When a session directory contains a `subagents/` subdirectory, each subagent JSONL file becomes its own minitrace session. The parent session's `Agent` tool call receives a `spawned_agent` field with the subagent's `sub_session_id`. The subagent session's title is prefixed with `[subagent]`.

### Preserved framework-specific metadata

Claude Code now keeps additional raw metadata in minitrace without promoting it into shared schema fields:

- `operational_context.framework_config.entrypoint`
- `turns[].framework_metadata`:
  - `entrypoint`
  - `slug`
  - `parent_uuid`
  - `is_sidechain`
  - `stop_reason`
  - `stop_sequence`
  - `cache_creation`
- `tool_calls[].framework_metadata`:
  - `caller`
  - tool-result record context such as `entrypoint`, `slug`, `parent_uuid`, `is_sidechain`

### What is not preserved

- System prompt content (set to null for privacy)
- Streaming event-level detail (only the final content is kept)
- Image and binary content (references only)

## Codex adapter

**Source**: Session and exec JSONL files in `~/.codex/`
**Source format**: `codex-session-jsonl-v1` or similar
**Converter version**: `go-minitrace-codex-adapter-dev`

### What it reads

Codex stores sessions as JSONL files under `~/.codex/sessions/` and optionally logs exec operations as JSONL. The adapter processes:

- **Session JSONL** → conversation turns and tool invocations
- **Exec JSONL** → tool calls from `codex exec --json` output
- **Command metadata** → exit codes and execution metadata when present
- **Tool-call arguments** → command strings plus optional justification text

### Known limitations

- Older sessions with unrecognized formats produce `unsupported Codex format hint: unknown-jsonl` errors
- The adapter does not skip these automatically; it fails on the first unrecognized session

### Preserved framework-specific metadata

Codex keeps richer raw metadata in the adapter-specific metadata/config fields:

- `operational_context.framework_config`:
  - `approval_policy`
  - detailed `sandbox_policy`
  - `collaboration_mode_detail`
  - `truncation_policy`
  - `rate_limits`
  - `session_source`
  - other session/runtime config such as `originator`, `personality`, `reasoning_effort`, `timezone`
- `turns[].framework_metadata`:
  - `turn_id`
  - `phase`
  - `memory_citation`
- `tool_calls[].framework_metadata`:
  - `codex_function`
  - `justification` (also promoted to `input.justification`)
  - `source`
  - `parsed_cmd`
  - `stdout`
  - `stderr`
  - `status`
  - `turn_id`
  - `exit_code` (also promoted to `output.exit_code`)

### What is not preserved

- Some Codex source-specific records still collapse into the nearest session/turn/tool representation rather than being modeled as standalone event types
- Binary exec output is truncated

## Pi adapter

**Source**: JSONL v3 session files in `~/.pi/agent/sessions/`
**Source format**: `pi-agent-jsonl-v3`
**Converter version**: `go-minitrace-pi-adapter-dev`

### What it reads

Pi stores one JSONL file per session in workspace-named directories (e.g., `--home-manuel-code-foo--/`). Each line is a structured event. The adapter processes:

- **User messages** → user turns
- **Assistant messages** → assistant turns with model metadata
- **Tool calls** → tool calls with mapped operation types
- **Tool results** → tool call outputs
- **Token usage events** → per-turn usage counters

### Field mapping

| Minitrace field | Source |
|----------------|--------|
| `environment.model` | From assistant message metadata |
| `environment.agent_framework` | `pi` |
| `environment.tools_enabled` | Unique tool names used in session |
| `turns[].usage` | Token counts from usage events |

### Tool operation mapping

Pi tools are mapped similarly to Claude Code:

| Tool name | Operation type |
|-----------|---------------|
| `read`, `Read` | `READ` |
| `edit`, `Edit` | `MODIFY` |
| `write`, `Write` | `NEW` |
| `bash`, `Bash` | `EXECUTE` |
| `mcp` tools | `OTHER` (unless name suggests read/write) |

### Preserved framework-specific metadata

Pi now preserves additional raw metadata alongside the normalized schema:

- `turns[].framework_metadata`:
  - `stop_reason`
  - `error_message`
- `tool_calls[].framework_metadata`:
  - `diff`
  - `first_changed_line`

### Single session conversion

Use `--source-session` to convert one specific JSONL file without scanning the full directory:

```bash
go-minitrace convert pi --source-session /path/to/session.jsonl --output-dir ./output
```

## claude.ai adapter

**Source**: Privacy export ZIP from claude.ai
**Source format**: `claude-ai-privacy-export-v1`
**Converter version**: `go-minitrace-claudeai-adapter-dev`

### What it reads

The ZIP contains a JSON file with all conversations. Each conversation has an array of message objects. The adapter maps:

- **Human messages** → user turns
- **Assistant messages** → assistant turns
- **Attachments** → referenced in turn metadata
- **Conversation metadata** → title, model, timestamps

### Filtering

Use `--uuid-filter` to convert only specific conversations by UUID prefix:

```bash
go-minitrace convert claude-ai --source export.zip --uuid-filter abc123,def456
```

### What is not preserved

- File attachments (only references, not content)
- Rendering metadata
- Organization/workspace context
- Token usage (not available in the export format)

## ChatGPT adapter (ZIP)

**Source**: Data export ZIP from ChatGPT
**Source format**: `chatgpt-export-zip-v1`
**Converter version**: `go-minitrace-chatgpt-adapter-dev`

### What it reads

The ZIP contains `conversations.json` with all conversations. The adapter processes the nested message tree structure that ChatGPT uses internally.

### What is not preserved

- Tool call details (the ZIP export has limited tool information)
- Plugin/action metadata
- Image generation details (only text content is preserved)
- Token usage (not available in the export format)

## ChatGPT adapter (JSON)

**Source**: Per-conversation JSON files
**Source format**: `chatgpt-json-transcript-v1`
**Converter version**: `go-minitrace-chatgpt-adapter-dev`

### What it reads

Each JSON file represents one conversation in a richer format than the standard export. The adapter extracts:

- **Messages** → turns with role mapping
- **Tool calls** → extracted from assistant messages
- **Tool results** → mapped to tool call outputs

This format includes tool call details that the standard export ZIP does not provide.

### Filtering

Use `--id-filter` to convert specific conversations:

```bash
go-minitrace convert chatgpt-json --source-dir /tmp/exports --id-filter 69c7,69c8
```

## turnsdb adapter

**Source**: SQLite `turns.db` from Geppetto/Pinocchio
**Source format**: `pinocchio-turns-sqlite-v1`
**Converter version**: `go-minitrace-turnsdb-adapter-dev`

### What it reads

The `turns.db` SQLite database stores conversation snapshots rather than individual turns. Each row is a full snapshot of the conversation at a point in time. The adapter:

1. Groups snapshots by conversation ID
2. Sorts snapshots chronologically
3. Diffs consecutive snapshots to extract the incremental turns
4. Maps the extracted deltas to minitrace turns and tool calls

This snapshot-diffing approach is fundamentally different from the other adapters and is necessary because the database does not store individual turn events.

### Single conversation

Use `--conv-id` to convert one specific conversation:

```bash
go-minitrace convert turnsdb --source /tmp/turns.db --conv-id 5cf06c5f-0460-485e-a7c5-92d56af826f9
```

### What is not preserved

- Snapshot-level metadata beyond what maps to minitrace
- Internal Pinocchio orchestration state
- Exact timing between turns (reconstructed from snapshot timestamps)

## Quality grading across adapters

All adapters use the same quality grading logic after conversion:

| Grade | Criteria |
|-------|----------|
| **A** | Has turns, has tool calls with output, >10 tool calls, >5 turns |
| **B** | Has turns (but doesn't meet A threshold) |
| **C** | No conversation turns |

## See also

- `go-minitrace help convert-commands` — conversion command flags and usage
- `go-minitrace help minitrace-schema` — the target schema these adapters produce
- `go-minitrace help framework-metadata-mappings` — detailed per-adapter metadata preservation tables
