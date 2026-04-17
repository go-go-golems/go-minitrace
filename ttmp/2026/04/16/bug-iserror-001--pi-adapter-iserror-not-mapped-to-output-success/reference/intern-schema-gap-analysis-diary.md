---
Title: Intern Brief - Schema Gap Analysis Research Diary
Ticket: bug-iserror-001
Status: active
Topics:
    - schema-review
    - minitrace
    - claude-code
    - codex
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/code/wesen/corporate-headquarters/go-minitrace/README.md
    - /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/doc/minitrace-schema.md
    - /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/doc/adapter-reference.md
    - /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitrace/schema.go
    - /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/claudecode/convert.go
    - /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/codex/convert.go
ExternalSources: []
Summary: "Research diary documenting the analysis of raw Codex and Claude Code transcripts to identify schema-worthy information missing from the minitrace schema."
LastUpdated: 2026-04-16T21:30:00-04:00
WhatFor: "Track research steps, findings, and recommendations for schema improvements."
WhenToUse: "Reference when implementing schema changes or reviewing converter adapters."
---

# Intern Schema Gap Analysis - Research Diary

**Date**: 2026-04-16  
**Researcher**: Intern Brief Assignment  
**Objective**: Review raw Codex and Claude Code transcripts and identify information that is (1) present in raw source data, (2) useful for analysis, and (3) not currently preserved well in the minitrace schema.

---

## Step 1: Read Documentation

### Files Read

1. **README.md** - Overview of go-minitrace capabilities, conversion commands, and query options
2. **pkg/doc/minitrace-schema.md** - Field-by-field reference for the minitrace JSON format
3. **pkg/doc/adapter-reference.md** - How each source format maps to the minitrace schema
4. **pkg/minitrace/schema.go** - Authoritative Go types for the schema
5. **pkg/adapters/claudecode/convert.go** - Claude Code adapter implementation
6. **pkg/adapters/codex/convert.go** - Codex adapter implementation

### Key Insights from Documentation

- Schema version: `minitrace-v0.2.0`
- Core entities: Session, Turns, ToolCalls, with nested metadata structures
- `framework_metadata` and `framework_config` are escape hatches for adapter-specific data
- Current gaps are acknowledged in "What is not preserved" sections but not systematically documented

---

## Step 2: Convert Sample Sessions

### Commands Run

```bash
# Create output directory
mkdir -p /tmp/intern-review

# Convert Claude Code sessions from pretext project
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace
go run ./cmd/go-minitrace convert claude-code \
    --source-dir ~/.claude/projects/-home-manuel-code-others-pretext \
    --output-dir /tmp/intern-review

# Attempted Codex conversion (failed on older sessions)
go run ./cmd/go-minitrace convert codex \
    --source-dir ~/.codex/sessions \
    --output-dir /tmp/intern-review
```

### Conversion Results

- **Claude Code**: 24 sessions converted successfully
- **Codex**: Failed on older sessions with `unsupported Codex format hint: unknown-jsonl`

---

## Step 3: Compare Raw vs Converted

### Claude Code Session: `3e25fe06-537b-40cf-9903-2e7bf1b1cd0d`

**Raw Source**: `~/.claude/projects/-home-manuel-code-others-pretext/3e25fe06-537b-40cf-9903-2e7bf1b1cd0d.jsonl`

**Rich fields in raw data**:
```json
{
  "type": "assistant",
  "parentUuid": "40b9f139-ff52-45ae-900c-8e7ae88bcc94",
  "isSidechain": false,
  "caller": {"type": "direct"},
  "slug": "compiled-zooming-finch",
  "entrypoint": "sdk-ts",
  "message": {
    "model": "claude-opus-4-6",
    "stop_reason": "tool_use",
    "stop_sequence": null,
    "usage": {
      "input_tokens": 2,
      "cache_creation_input_tokens": 723,
      "cache_read_input_tokens": 75735,
      "cache_creation": {
        "ephemeral_5m_input_tokens": 0,
        "ephemeral_1h_input_tokens": 723
      }
    }
  }
}
```

**Converted Output**: `/tmp/intern-review/active/2026-03/3e25fe06-537b-40cf-9903-2e7bf1b1cd0d.minitrace.json`

**Status of rich fields**:
- ✅ `model` → `turn.model` and `session.environment.model`
- ✅ Token counts → `turn.usage.*`
- ❌ `parentUuid` / `isSidechain` → NOT CAPTURED
- ❌ `caller` → NOT CAPTURED
- ❌ `slug` → Only used for title prefix, not preserved as field
- ❌ `entrypoint` → NOT CAPTURED
- ❌ `stop_reason` / `stop_sequence` → NOT CAPTURED
- ❌ `cache_creation` time buckets → Only total preserved

### Codex Session Analysis

**Raw Source**: `~/.codex/sessions/2026/04/14/rollout-2026-04-14T20-17-00-019d8e7f-99a5-7243-b19a-845778ff2b5a.jsonl`

**Rich fields identified in raw data** (could not convert due to format hint issues on this specific file, but analyzed raw structure):

```json
{
  "type": "event_msg",
  "payload": {
    "type": "exec_command_end",
    "exit_code": 2,
    "stdout": "",
    "stderr": "ls: cannot access 'go-go-goja/web': No such file",
    "parsed_cmd": [{"type": "list_files", "cmd": "ls -la", "path": null}],
    "source": "unified_exec_startup"
  }
}
```

```json
{
  "type": "turn_context",
  "payload": {
    "sandbox_policy": {"type": "workspace-write", "writable_roots": [...]},
    "approval_policy": "on-request",
    "collaboration_mode": {"mode": "default", "settings": {"reasoning_effort": "high"}},
    "model": "gpt-5.3-codex",
    "effort": "high"
  }
}
```

```json
{
  "type": "response_item",
  "payload": {
    "type": "function_call",
    "arguments": "{\"cmd\":\"ls -la\",\"justification\":\"inspect file\"}",
    "call_id": "call_..."
  }
}
```

**Status of rich fields**:
- ✅ `exit_code` → Used to compute `success` boolean, BUT exit code value LOST
- ❌ `stdout` / `stderr` → Combined, separation LOST
- ❌ `parsed_cmd` → NOT CAPTURED
- ❌ `source` → NOT CAPTURED
- ❌ `justification` → Buried in `framework_metadata`
- ❌ `sandbox_policy.type` → Flattened to `sandbox: true` boolean
- ❌ `collaboration_mode` → Buried in `framework_config`
- ❌ `approval_policy` → NOT CAPTURED (replaced by `autonomy_level` heuristic)

---

## Step 4: Identify Systematic Gaps

### Gap Pattern 1: Exit Codes → Boolean

**Issue**: Both adapters flatten exit codes to `success` boolean.

**Impact**: 
- Cannot distinguish error types (usage error vs runtime error vs signal)
- Cannot identify signal-terminated processes
- Cannot analyze retry patterns by error type

**Evidence**:
- Codex adapter: `parseFunctionOutput()` extracts exit_code, uses to set `success`, discards code
- Claude adapter: `is_error` boolean inverted to `success`

### Gap Pattern 2: Rich Metadata → framework_metadata Escape Hatch

**Issue**: Valuable fields buried in `framework_metadata` or `framework_config` make querying difficult.

**Impact**:
- Cannot easily query "which tool calls had no justification?"
- Cannot correlate collaboration mode with tool usage patterns
- Sandbox policy nuances invisible to queries

**Evidence**:
- Codex `justification` → `tool_call.framework_metadata.justification`
- Codex `collaboration_mode` → `session.operational_context.framework_config`

### Gap Pattern 3: Stream Separation → Merged Output

**Issue**: Codex captures stdout/stderr separately, minitrace merges them.

**Impact**:
- Cannot identify "silent failures" (exit 0, stderr non-empty)
- Cannot analyze warning patterns
- Cannot distinguish output streams for parsing

**Evidence**:
- Codex raw has `stdout` and `stderr` fields
- Minitrace only has `result` (combined)

### Gap Pattern 4: Session/State Changes → Static Values

**Issue**: Changes during session (mode switches, rate limits) not captured as events.

**Impact**:
- Cannot track collaboration mode switches
- Cannot identify rate limit pressure
- Cannot analyze context window pressure

**Evidence**:
- `turn_context` events in Codex can change mode mid-session
- Only final values captured, not change history

### Gap Pattern 5: Tool Intent → Opaque

**Issue**: Why tools were called (justification, phase, commentary) not captured.

**Impact**:
- Cannot distinguish exploratory vs action tool calls
- Cannot analyze commentary vs acting phases
- Cannot evaluate tool use rationale quality

**Evidence**:
- Codex `phase: "commentary"` on agent_message → NOT CAPTURED
- Codex `justification` → Buried in metadata

---

## Step 5: Synthesize Recommendations

### Top 5 Schema Gaps (Priority Order)

1. **Tool Exit Codes** - Add `exit_code` int and `error_category` enum
2. **Tool Use Justification** - Promote from metadata to `tool_call.input.justification`
3. **Sandbox Policy & Escalation** - Replace boolean with policy enum and permission level tracking
4. **stdout/stderr Separation** - Add separate fields, keep `result` as combined view
5. **Collaboration Mode & Phase** - Add mode enum, phase enum, and switch counters

### Top 5 "Keep in framework_metadata"

1. `codex_function` - Codex-specific function names
2. `personality` spec - Codex communication style
3. `caller` (Claude) - Who initiated tool use
4. `entrypoint` (Claude) - How session started
5. `sourceToolAssistantUUID` - Internal correlation ID

### Fields to Ignore

- `requestId`, `service_tier`, `inference_geo` - Operational metrics
- `cache_creation` time buckets - Implementation-specific
- `slug` - Human-friendly but not analytically valuable
- `parentUuid` / conversation tree - Complex; linear sufficient

---

## Deliverables Produced

1. **Full Analysis Memo**: `/tmp/intern-review/schema-gap-analysis.md`
2. **Summary Table**: `/tmp/intern-review/schema-gaps-table.md`
3. **This Research Diary**: Ticket-local documentation

---

## Key Files Referenced

| File | Purpose |
|------|---------|
| `~/.claude/projects/-home-manuel-code-others-pretext/*.jsonl` | Claude Code raw transcripts |
| `~/.codex/sessions/2026/04/14/*.jsonl` | Codex raw transcripts |
| `/tmp/intern-review/active/*/*.minitrace.json` | Converted output |
| `pkg/adapters/claudecode/convert.go` | Claude adapter (lines 50-350) |
| `pkg/adapters/codex/convert.go` | Codex adapter (lines 50-450) |
| `pkg/minitrace/schema.go` | Schema definition |

---

## Research Complete

All findings documented. Recommendations ready for review.

*End of diary*
