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

## Source events and attachments

Minitrace has two first-class primitives for source data that is not a conversational message or normalized tool call:

- `events[]` stores source-observed lifecycle/timeline facts such as compactions, permission-mode changes, title changes, subagent lifecycle signals, or rate-limit snapshots.
- `attachments[]` stores artifact references such as images, uploaded files, downloaded files, or future generated outputs. Attachments are references and bounded previews, not blob storage.

Adapters should use annotations for human or derived review notes. They should use events/attachments for facts that came from the source transcript itself.

## Per-adapter fidelity matrix

Not every source records the same facts, and not every fact an adapter emits is native. The matrix below distinguishes:

- **native** — copied from an explicit field in the source
- **derived** — computed by the adapter (e.g. subtracting timestamps)
- **scraped** — parsed out of free-form strings in the source
- **–** — never populated for this adapter

| Field | pi | claude-code | codex | copilot | chatgpt | claude.ai | turnsdb |
|---|---|---|---|---|---|---|---|
| tool `duration_ms` | derived (emit→result timestamps) | derived (emit→result timestamps) | scraped (output metadata / "Wall time:") | – | n/a (no tools) | derived (block timestamps) | – |
| tool `exit_code` | – | scraped (`"Error: Exit code N"` result string) | native (exec events) + scraped (output text) | – | n/a | – | – |
| tool stderr | – | native (`toolUseResult.stderr` on error) | native (kept in framework metadata) | – | n/a | – | – |
| full tool result object | – | `framework_metadata.tool_use_result` (size-capped) | split into output + framework metadata | telemetry in framework metadata | n/a | result text only | `framework_metadata.payload` |
| turn `thinking` | native | native | native (reasoning events/summaries) | – (opaque/encrypted) | content-type only | native | native (reasoning blocks) |
| input/output tokens | native | native | native | partial (per-turn output only; session totals) | – | – | – |
| cache-read tokens | native | native | native | session totals only | – | – | – |
| cache-creation tokens | native | native | – | – | – | – | – |
| reasoning tokens | native | – | native | – | – | – | – |
| session `git_branch` | – | native | native for legacy rollout JSONL, otherwise – | native | – | – | – |
| `working_directory` | native | native | native | native | – | – | – |
| `agent_version` | – | native | native | native | – | – | – |
| session cost | native (`usage.cost.total`) | – | – | – | – | – | – |
| session `summary` | – | – | – | – | – | native | – |
| spawned agents | – | native (Agent/Task tools) | native (spawn_agent/wait_agent) | – | – | native (extended search) | – |
| `coordination.predecessor_session` | native (`parentSession` ID) | native (subagent parent session) | native (`parent_thread_id`) | – | – | – | – |

Truncation is uniform across adapters: tool results longer than 10 KiB are cut at the limit with a `[truncated]` marker, and `output.full_bytes`/`output.full_hash` always describe the **full pre-truncation** payload (its byte length and sha256), so you can detect and deduplicate truncated outputs honestly.

All converters (`pi`, `codex`, `claude-code`) skip sessions that fail to convert and report them as `status: failed` rows in the output instead of aborting; the command exits 0 as long as at least one session converted.

## Claude Code adapter

**Source**: JSONL v2 transcripts in `~/.claude/projects/`
**Source format**: `claude-code-jsonl-v2` (main), `claude-code-jsonl-v2+subagent` (subagent)
**Converter version**: `go-minitrace-claude-adapter-dev`

### What it reads

Claude Code stores one JSONL file per session. Each line is a JSON event with a type field. The adapter processes:

- **Conversation messages** → turns (user, assistant, system)
- **Tool use blocks** → tool calls with operation type mapping
- **Tool result blocks** → tool call outputs with success/error status and derived `duration_ms`
- **`toolUseResult` payloads** → structured output mapping (see below)
- **Usage metadata** → per-turn token counts (input, output, cache_read, cache_creation)
- **Subagent directories** → separate minitrace sessions with parent backlinking
- **Lifecycle records** → source events where they do not fit turns or tool calls
- **Attachment records** → attachment references plus optional timeline events

Additionally, dir-v1 tool-results sessions (an older format that stores tool results in a directory structure) are detected and converted.

### Field mapping

| Minitrace field | Source |
|----------------|--------|
| `environment.model` | Extracted from assistant message metadata |
| `environment.agent_framework` | `claude-code` |
| `environment.platform_type` | `agent` |
| `environment.provider_hint` | `anthropic` |
| `environment.agent_version` | `version` field on records |
| `environment.tools_enabled` | Unique tool names from tool use blocks |
| `operational_context.working_directory` | `cwd` field on records |
| `operational_context.git_branch` | `gitBranch` field on records |
| `operational_context.autonomy_level` | Mapped from `permission-mode` records (bypassPermissions → full-auto, plan → suggest) |
| `timing.*` | Computed from message timestamps |
| `title` | `ai-title` record when present, else first human message |
| `turns[].thinking` | Extracted from thinking blocks if present |
| `turns[].usage` | Per-message token counts |
| `tool_calls[].output.duration_ms` | Derived: tool-result timestamp minus tool-use timestamp |
| `events[]` | Source lifecycle records such as mode, permission-mode, title, and attachment events |
| `attachments[]` | Attachment references with media/name/path-like metadata when available |

### toolUseResult mapping

Claude Code attaches a `toolUseResult` payload to tool-result records. The adapter maps it as follows:

- **String form**: on error, the string `"Error: Exit code N"` is parsed and `N` becomes `output.exit_code`.
- **Object form**: `stderr` becomes `output.error` on failing calls; `interrupted: true` marks the call failed with error `interrupted by user`; explicit `exitCode`/`exit_code` keys map to `output.exit_code` when present.
- In both forms the full payload is preserved under `tool_calls[].framework_metadata.tool_use_result`, size-capped (16 KiB total, 2 KiB per field) so archives stay bounded.

### Tool operation mapping

| Tool name | Operation type |
|-----------|---------------|
| `Read`, `Glob`, `Grep`, `TaskGet`, `TaskList`, `TaskOutput`, `WebFetch`, `WebSearch`, `ToolSearch` | `READ` |
| `Edit`, `NotebookEdit` | `MODIFY` |
| `Write` | `NEW` |
| `Bash`, `TaskStop` | `EXECUTE` |
| `Agent`, `Task`, `TaskCreate`, `TaskUpdate` | `DELEGATE` |
| Everything else | `OTHER` |

Content origin is classified separately: `mcp__`-prefixed tools → `mcp_server`, file tools → `local_file`, `Bash` → `local_exec`, web tools → `web`, agent/task tools → `sub_agent`.

### Subagent handling

When a session directory contains a `subagents/` subdirectory, each subagent JSONL file becomes its own minitrace session. The parent session's `Agent`/`Task` tool call receives a `spawned_agent` field with the subagent's `sub_session_id`, and `metrics.subagent_count` reflects the real number of spawned agents. The subagent session's title is prefixed with `[subagent]`; its `framework_config.parent_session` points back at the parent, and `coordination.predecessor_session` is set to the parent session ID so parent-child traversal works through the normalized schema.

### Preserved framework-specific metadata

- `operational_context.framework_config`: `entrypoint`, `agent_id`, `session_id`, `parent_uuid`, `is_sidechain`, `user_type`, `attribution_agent`, `mode`, `permission_mode`, `ai_title`
- `turns[].framework_metadata`: `entrypoint`, `slug`, `agent_id`, `session_id`, `parent_uuid`, `is_sidechain`, `attribution_agent`, `stop_reason`, `stop_sequence`, `cache_creation`
- `tool_calls[].framework_metadata`: `caller`, tool-result record context, `tool_use_result`, `interrupted`

### What is not preserved

- System prompt content (set to null for privacy)
- Streaming event-level detail (only the final content is kept)
- Image and binary content (references only)
- Reasoning token counts (not present in the source usage records)

## Codex adapter

**Source**: Session and exec JSONL files in `~/.codex/`
**Source format**: `codex-session-jsonl-v1` (sessions), `codex-exec-jsonl-v1` (exec logs), `codex-legacy-rollout-jsonl-v0` (older rollout JSONL)
**Converter version**: `go-minitrace-codex-adapter-dev`

### What it reads

Codex stores sessions as JSONL files under `~/.codex/sessions/` and optionally logs exec operations as JSONL. The adapter processes:

- **Session JSONL** → conversation turns and tool invocations
- **Exec JSONL** → tool calls from `codex exec --json` output
- **Legacy rollout JSONL** → older top-level message/reasoning/function-call records, including `shell` calls normalized to `exec_command`
- **exec_command events** → native exit codes; structured output metadata also yields scraped `exit_code` and `duration_ms` (from `metadata.duration_seconds` or `"Wall time:"` lines)
- **token_count events** → input/output/cached/reasoning token usage
- **Tool-call arguments** → command strings plus optional justification text
- **spawn_agent / wait_agent calls** → spawned-agent records with sub-session IDs and outcome summaries
- **Lifecycle/source signals** → source events when they describe timeline facts outside turns
- **Image-view signals** → attachment references when a `view_image` tool call points at an image

Sessions in unrecognized formats are skipped and reported as failed rows; the rest of the batch converts normally. Older rollout JSONL files that start with top-level session metadata plus `message`, `reasoning`, `function_call`, `function_call_output`, and `record_type: state` records are recognized as legacy rollout JSONL and converted.

### Multi-agent metadata

Codex multi-agent sessions carry native coordination fields, captured into `operational_context.framework_config`:

- `parent_thread_id` — the spawning session's thread; also promoted to `coordination.predecessor_session`
- `agent_nickname` and `agent_role` — how the agent was addressed and what role it played
- `metrics.subagent_count` counts actual `spawn_agent` calls

### Preserved framework-specific metadata

- `operational_context.framework_config`: `approval_policy`, detailed `sandbox_policy`, `collaboration_mode` (+detail), `truncation_policy`, `rate_limits`, `session_source`, `originator`, `personality`, `reasoning_effort`, `timezone`, `model_context_window`, plus the multi-agent fields above
- `turns[].framework_metadata`: `turn_id`, `phase`, `memory_citation`
- `tool_calls[].framework_metadata`: `codex_function`, `justification` (also promoted to `input.justification`), `source`, `parsed_cmd`, `stdout`, `stderr`, `status`, `turn_id`, `exit_code` (also promoted to `output.exit_code`), `targets`, `timed_out`

### What is not preserved

- git branch for modern session JSONL when not recorded in the source
- cache-creation token counts (Codex only reports cached input)
- Token usage for the exec-JSONL format (not present in that format)
- Binary exec output is truncated

## Pi adapter

**Source**: JSONL v3 session files in `~/.pi/agent/sessions/`
**Source format**: `pi-agent-jsonl-v3`
**Converter version**: `go-minitrace-pi-adapter-dev`

### What it reads

Pi stores one JSONL file per session in workspace-named directories (e.g., `--home-manuel-code-foo--/`). Each line is a structured event. The adapter processes:

- **User messages** → user turns
- **Assistant messages** → assistant turns with model metadata
- **Thinking blocks** → `turns[].thinking` plus reasoning token counts
- **Tool calls** → tool calls with mapped operation types
- **Tool results** → tool call outputs with derived `duration_ms` (result timestamp minus emit timestamp)
- **Usage records** → per-turn token counters and session cost (`usage.cost.total`)
- **Lifecycle records** → source events for session info, compactions, model changes, thinking-level changes, and custom records
- **Fork lineage** → `session.parentSession` paths are normalized in framework metadata and the extracted parent session ID is promoted to `coordination.predecessor_session`

### Field mapping

| Minitrace field | Source |
|----------------|--------|
| `environment.model` | From assistant message metadata (tracks `model_change` records) |
| `environment.agent_framework` | `pi` |
| `environment.tools_enabled` | Unique tool names used in session |
| `operational_context.working_directory` | `cwd` from session metadata |
| `turns[].usage` | Token counts from usage records (input, output, cache read/write, reasoning) |
| `metrics.session_cost` | Accumulated `usage.cost.total` |
| `tool_calls[].output.duration_ms` | Derived from emit→result timestamps |

### Tool operation mapping

Named tools map directly (`read`/`grep`/`glob` → `READ`, `write` → `NEW`, `edit` → `MODIFY`). `bash` commands are classified by scraping the command string: `cat`/`head`/`ls`/`grep`-style commands → `READ`, redirections and `touch`/`mkdir` → `NEW`, `>>`/`sed -i` → `MODIFY`, everything else → `EXECUTE`. Unknown tools fall back to `OTHER`.

The adapter has no MCP-specific logic; MCP tool names are treated like any other unknown tool.

### Preserved framework-specific metadata

- `turns[].framework_metadata`: `stop_reason`, `error_message`
- `tool_calls[].framework_metadata`: `diff`, `first_changed_line` (for edit results)

### What is not preserved

- git branch and agent version (not recorded in the source)
- exit codes (Pi does not record them)
- stderr as a separate stream

### Targeted conversion

Use `--source-session` (repeatable) or `--source-list` to convert specific JSONL files without scanning the full directory:

```bash
go-minitrace convert pi --source-session /path/to/session.jsonl --output-dir ./output
go-minitrace convert pi --source-list ./sessions.txt --output-dir ./output
```

The same flags exist on `convert codex` and `convert claude-code`.

## Copilot adapter

**Source**: Session-state directories from the GitHub Copilot CLI
**Source format**: `copilot-cli-session-v1`
**Converter version**: `go-minitrace-copilot-adapter-dev`

### What it reads

- **Session events** → turns (user, assistant, system prompt)
- **Tool start/completion events** → tool calls; incomplete tools are marked failed with an explanatory error
- **Workspace metadata** → `working_directory`, `git_branch`, `git_ref` (head/base commits), repository → `project_id`
- **Shutdown token details** → session-level input/output/cache-read totals (per-turn only output tokens)
- **Tool telemetry and permission payloads** → framework metadata

### What is not preserved

- Reasoning content (opaque/encrypted in the source; only presence flags are kept)
- Per-tool durations and exit codes
- Encrypted/redacted payload fields (`encryptedContent`, `reasoningOpaque`, ...) are stripped

## claude.ai adapter

**Source**: Privacy export ZIP from claude.ai
**Source format**: `claude-ai-privacy-export-v1`
**Converter version**: `go-minitrace-claudeai-adapter-dev`

### What it reads

The ZIP contains a JSON file with all conversations. Each conversation has an array of message objects. The adapter maps:

- **Human messages** → user turns (attachments flattened into content and summarized in metadata)
- **Assistant messages** → assistant turns with thinking blocks
- **Tool use/result block pairs** → tool calls with derived `duration_ms` from block timestamps
- **`launch_extended_search_task`** → a spawned-agent record (`extended_search`)
- **Conversation metadata** → title, summary, timestamps

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

The ZIP contains `conversations.json` with all conversations. The adapter processes the nested message tree structure that ChatGPT uses internally and linearizes the active branch. Sessions convert as turns only — no tool calls.

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

This snapshot-diffing approach is fundamentally different from the other adapters and is necessary because the database does not store individual turn events. Tool calls are always classified as `EXECUTE` with content origin `local_exec`; the full block payload is preserved under `tool_calls[].framework_metadata.payload`.

### Single conversation

Use `--conv-id` to convert one specific conversation:

```bash
go-minitrace convert turnsdb --source /tmp/turns.db --conv-id 5cf06c5f-0460-485e-a7c5-92d56af826f9
```

### What is not preserved

- Snapshot-level metadata beyond what maps to minitrace
- Internal Pinocchio orchestration state
- Exact timing between turns (reconstructed from snapshot timestamps)
- Token usage, durations, exit codes

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
