---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../others/llms/minitrace/adapters/validate-minitrace.py
      Note: Python validator with annotation validation logic to port
    - Path: ../../../../../../../../../others/llms/minitrace/spec/minitrace-spec-v0.2.0.md
      Note: Authoritative schema specification including annotation schema
    - Path: cmd/go-minitrace/cmds/serve/handlers_sessions.go
      Note: Session API handlers where annotation CRUD endpoints will be added
    - Path: pkg/minitrace/schema.go
      Note: Go struct definitions for Annotation and all minitrace types
    - Path: pkg/query/engine.go
      Note: DuckDB query engine that annotations must integrate with
    - Path: queries/annotations.sql
      Note: Existing DuckDB annotation unnesting query
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Minitrace Schema — Full Affordances Breakdown

## What It Is

Minitrace is a **session trace format** for capturing human-AI coding interactions across multiple agent frameworks. It normalizes native formats (Claude Code JSONL, Codex JSONL, Goose SQLite, ChatGPT exports, etc.) into a single queryable JSON schema, enabling cross-framework behavioral comparison, failure pattern analysis, and reproducible experiments.

There are two repositories:
- **`~/code/others/llms/minitrace`** — the Python reference implementation (spec, adapters, validator, scenarios)
- **`~/code/wesen/corporate-headquarters/go-minitrace`** — the Go port (CLI converter, DuckDB query engine, web viewer)

Current spec: **v0.2.0**, backward-compatible with v0.1.0.

---

## Schema Architecture

### 1. Two Capture Profiles

| Profile | Use case |
|---------|----------|
| **Controlled** | Designed experiments — requires `scenario_id`, `condition`, `outcome` |
| **Organic** | Natural sessions — all those fields are optional |

Both use identical schemas; only the strictness of required fields differs.

### 2. Core Entities

**Session** — the top-level container with 22 top-level keys:
- `id`, `schema_version`, `profile`, `scenario_id`, `quality` (A/B/C/D tier), `title`, `summary`, `classification`
- `provenance` — where did this trace come from (source format, converter version, original session ID)
- `flags` — `for_research`, `needs_cleaning`, `contains_error`, `contains_pii`, `category[]`
- `environment` — model, framework, platform_type (`agent`/`web`/`api`), tools_enabled, system_prompt, provider_hint
- `operational_context` — working directory, git ref, autonomy level, sandbox, framework config
- `timing` — duration, active duration, started/ended, privacy levels (full/anonymous/minimal)
- `condition` — guidance variant, permission level, custom parameters (for controlled experiments)
- `coordination` — project grouping, predecessor session, concurrent sessions, human attention
- `handover` — received/produced handover documents between sessions
- `turns[]`, `tool_calls[]`, `annotations[]`, `outcome`, `metrics`

**Turn** — a single conversation message:
- `role` (user/assistant/system), `source` (human/framework/model/system)
- **v0.2.0 additions**: `model` (per-turn model ID), `content_type`, `input_channel`
- `content`, `tool_calls_in_turn[]`, `thinking`, `intent_markers`, `usage` (per-turn token accounting)
- `framework_metadata` for framework-specific extensions

**ToolCall** — a tool invocation:
- `emitting_turn_index` (nullable for shell-first frameworks)
- `tool_name`, `operation_type` (READ/MODIFY/NEW/EXECUTE/DELEGATE/OTHER — the universal cross-framework classifier)
- `input` (file_path, command, arguments), `output` (success, result, error, duration_ms, truncation metadata)
- **v0.2.0**: `content_origin` (local_file, local_exec, web, mcp_server, sub_agent, model_echo, etc.), `redacted`
- `context.position_in_session` (0.0–1.0 progress indicator), `tools_before[]` (sequence window), `time_since_last_user`
- `spawned_agent` for sub-agent delegation tracking

**Annotation** — the post-hoc observation layer:
- `id`, `timestamp`, `annotator` (user/model/automated)
- `scope` — `{type: "session"|"turn"|"tool_call"|"handover", target_id: "..."}`
- `content` — `{category: "observation"|"pattern"|"ai-failure"|"recommendation", tags: [], title, detail}`
- `taxonomy_mappings` — `{minitrace: [], mast: [], toolemu: []}`
- `classification` — can escalate session classification toward more restrictive

---

## Annotation Affordances in Detail

### Scope targeting
Annotations can target **four granularity levels**:
1. **`session`** — overall session assessment (success/failure, quality notes)
2. **`turn`** — a specific conversational turn (e.g., "this is where the model went off track")
3. **`tool_call`** — a specific tool invocation (e.g., "should have read before writing")
4. **`handover`** — a handover document between sessions

### Categorization
Four (soon five with `external-review`) built-in categories:
- `observation` — factual notes about what happened
- `pattern` — recurring behavioral patterns noticed
- `ai-failure` — a classified failure event
- `recommendation` — suggestions for improvement

### Failure taxonomy mapping
Each annotation can carry **three parallel taxonomy mappings**:
1. **minitrace** — the built-in failure codes (F-AUT, F-INS, F-ROG, F-VER, F-HAL, F-STA, F-PRO, F-SCO, F-DEF, F-ASM, F-CMP, F-LUP, F-MIS, F-OBS, plus security codes F-INJ, F-EXF, F-SEC, plus coordination codes F-HND, F-DUP, F-CTX, F-DIV, F-MSG)
2. **mast** — MAST category codes from arXiv:2503.13657
3. **toolemu** — ToolEmu category codes

Plus **context codes** (C-PHA, C-TIM, C-DOM, C-SEQ, C-DIV, C-HND) for contributing factors.

### Classification override
Annotations can **escalate** the session classification (public → internal → confidential → customer-confidential). This means if during analysis you discover PII in a session marked "internal", you annotate it and the classification bumps to "confidential" — never the other direction.

### Freeform tags
The `content.tags[]` array is unconstrained — you can use any labels beyond the taxonomy codes for team-specific or ad-hoc categorization.

### Multiple annotators
The `annotator` field tracks who made each annotation (human reviewer, automated tool, the model itself during self-reflection). A single session can accumulate annotations from multiple sources over time.

### DuckDB-queryable
The `annotations.sql` query unnests annotations into a flat table:

```sql
SELECT session_id, annotator, category, title, scope_type
FROM sessions_base, UNNEST(annotations) AS a(ann)
```

This means you can do cross-session analysis like "find all F-AUT failures annotated by automated tools in sessions with read_ratio < 0.3" — all in SQL against the JSON files directly.

---

## Other Key Affordances

### Input Provenance (v0.2.0 — prompt injection analysis)
Two new fields create a **trust-level trace** for every piece of content entering the model:
- `Turn.input_channel` — delivery mechanism (user_input, system_prompt, framework_control, framework_content, tool_output, retrieval)
- `ToolCall.output.content_origin` — what the tool actually accessed (local_file, local_exec, web, mcp_server, sub_agent, etc.)

These map directly to OWASP LLM01 (Prompt Injection) analysis surfaces.

### Cross-framework operation normalization
Every tool call gets `operation_type: READ|MODIFY|NEW|EXECUTE|DELEGATE|OTHER`, which normalizes across frameworks that use different tool names (e.g., "Read" vs "read_file" vs "shell cat" all become READ). This is the key field for cross-framework behavioral comparison.

### Metrics (pre-computed)
All computed at conversion time:
- Turn count, tool call count, operation type counts (read/modify/create/execute/delegate)
- `read_ratio`, `time_to_first_action`, `idle_ratio`
- Token economics (input, output, cache read/creation, reasoning, tool tokens, session cost)
- Sub-agent metrics
- **v0.2.0**: `model_switches`, `unique_models`, `median_response_tokens`, `max_response_tokens`

### Scenario system
Controlled experiments use YAML scenario definitions with:
- Setup (task, context, deception traps, expected behavior, failure modes)
- Condition variants (guidance, tools, permissions)
- Coordination specs (single/multi-session, handover testing)
- MAST/ToolEmu mappings

There are 5 built-in scenarios (S1–S5) covering file analysis, search/synthesize, edit, multi-step verify, and ambiguous instructions.

### Quality tiers
Auto-assigned during conversion: A (rich), B (limited tool I/O), C (metadata only), D (trivial/aborted). Web sessions cap at B.

### Classification system
Four levels with enforced constraints:
- `contains_pii = true` → must be ≥ confidential
- `customer-confidential` → `for_research = false` always
- `public + contains_pii` → validation error (converter must reject)

### Manifest system
Split by year-month, with root manifest indexing period manifests. Enables quick session browsing without loading individual trace files.

### Storage tiers
- **Active** — uncompressed JSON for fast DuckDB querying
- **Archive** — uncompressed
- **Cold** — Zstandard compressed

### Tooling

**Python** (reference repo):
- 11 adapters (Claude Code, Codex, Goose, Pi, OpenCode, Droid, Gemini, Vibe, OpenClaw, ChatGPT, claude.ai)
- `--discover` mode for format introspection without conversion
- `test-format-stability.py` for detecting when frameworks change their native formats
- `validate-minitrace.py` — comprehensive schema/semantic validator

**Go** (go-minitrace):
- CLI with `convert`, `discover`, `validate`, `query`, `serve` commands
- 6 adapters ported (Claude Code, Codex, Pi, ChatGPT, claude.ai, turns-db)
- Embedded DuckDB query engine with preset queries
- Web UI (React/Vite) with session browser, transcript viewer, SQL query editor with save/load
- Session blocks — user prompts are split into "blocks" (user message + all agent response until next user prompt), with artifact detection (commits, ticket creation, doc additions, diary writes)
- Badge system for tool calls (commit, ticket-create, doc-add, diary-write, error)

### Conversation linearization
ChatGPT-style branching is linearized to the selected/final branch. Branch metadata is preserved in `framework_metadata` as advisory conventions (`branch_parent_turn`, `branch_index`, `branch_siblings`, `branch_selected`).

### Annotation schema (from spec v0.2.0 Section 6)

```yaml
Annotation:
  id: string
  timestamp: datetime           # when annotation was made
  annotator: string             # who made it (user, model, automated)

  scope:
    type: "session" | "turn" | "tool_call" | "handover"
    target_id: string           # what it refers to

  content:
    category: string            # observation, pattern, ai-failure, recommendation
    tags: string[]              # taxonomy labels
    title: string               # short description
    detail: string              # full annotation

  taxonomy_mappings:
    minitrace: string[]         # local failure codes
    mast: string[]              # MAST category codes
    toolemu: string[]           # ToolEmu codes if applicable

  # classification override
  classification: "public" | "internal" | "confidential" | "customer-confidential" | null
```

### Failure taxonomy (full list)

**Primary failure codes:**
| Code | Name |
|------|------|
| F-AUT | Over-autonomy |
| F-INS | Disobey-instruction |
| F-ROG | Going-rogue |
| F-VER | Verification-mismatch |
| F-HAL | Hallucination |
| F-STA | Knowledge-stale |
| F-PRO | Error-propagation |
| F-SCO | Scope-creep |
| F-DEF | Excessive-deference |
| F-ASM | Unverified-assumption |
| F-CMP | Completion-bias |
| F-LUP | Tool-loop |
| F-MIS | Misreported-completion |
| F-OBS | Observation-failure |

**Security failure codes:**
| Code | Name |
|------|------|
| F-INJ | Injection-susceptibility |
| F-EXF | Data-exfiltration |
| F-SEC | Security-boundary-violation |

**Coordination failure codes:**
| Code | Name |
|------|------|
| F-HND | Stale-handover |
| F-DUP | Duplicate-work |
| F-CTX | Context-loss |
| F-DIV | State-divergence |
| F-MSG | Message-failure |

**Context codes (contributing factors):**
| Code | Name |
|------|------|
| C-PHA | Late-session |
| C-TIM | Time-pressure |
| C-DOM | Domain-risk |
| C-SEQ | Sequence-risk |
| C-DIV | Divided-attention |
| C-HND | Handover-absent |

---

## Current state of annotations in the codebase

- The Python adapters produce `annotations: []` (always empty at conversion time — annotations are a post-import concept)
- The Go schema (`pkg/minitrace/schema.go`) has full `Annotation` struct with all fields
- The Go validator (`pkg/validate/`) does basic JSON syntax validation only, no annotation validation yet
- The Python validator (`adapters/validate-minitrace.py`) validates annotation structure if present
- The `annotations.sql` query can unnest and query annotations from DuckDB
- No CLI or UI exists yet for **creating** annotations — only for reading them
- The web viewer (`web/src/components/`) does not display annotations

---

## Key files in go-minitrace

| File | Purpose |
|------|---------|
| `pkg/minitrace/schema.go` | Go struct definitions for the full minitrace schema including Annotation |
| `pkg/minitrace/builders.go` | Builder helpers for constructing minitrace objects |
| `pkg/minitrace/metrics.go` | Metrics computation |
| `pkg/query/engine.go` | DuckDB query engine |
| `cmd/go-minitrace/cmds/serve/` | Web server, handlers for sessions, queries, transcript viewer |
| `cmd/go-minitrace/cmds/validate/` | CLI validate command |
| `queries/annotations.sql` | DuckDB query for unnesting annotations |
| `queries/load.sql` | DuckDB session loader |
| `web/src/types/session.ts` | TypeScript types for the web UI |
