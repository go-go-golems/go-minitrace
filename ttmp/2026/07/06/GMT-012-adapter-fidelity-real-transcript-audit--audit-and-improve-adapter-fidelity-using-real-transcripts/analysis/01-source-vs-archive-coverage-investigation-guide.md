---
Title: Source-vs-archive coverage investigation guide
Ticket: GMT-012-adapter-fidelity-real-transcript-audit
Status: active
Topics:
    - tooling
    - cli
    - diagnostics
    - documentation
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/adapters/claudecode/convert.go
      Note: |-
        Claude Code adapter, especially assistant content block and toolUseResult handling.
        Claude Code adapter behavior under investigation
    - Path: pkg/adapters/codex/convert.go
      Note: |-
        Codex adapter, especially old JSONL detection, reasoning, token-count, and lifecycle events.
        Codex adapter behavior under investigation
    - Path: pkg/adapters/pi/convert.go
      Note: |-
        Pi adapter, especially thinking block, image block, usage, and tool output handling.
        Pi adapter behavior under investigation
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts/scripts/04-profile-source-vs-archive-coverage.py
      Note: |-
        Implements the structural source-vs-archive coverage profiling workflow.
        Coverage profiling implementation
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts/sources/coverage-profile/01-coverage-profile.md
      Note: Generated aggregate coverage profile used by this guide.
ExternalSources: []
Summary: Detailed guide for investigating adapter coverage by comparing structural facts in native transcript sources with facts preserved in converted minitrace archives.
LastUpdated: 2026-07-06T18:50:00-04:00
WhatFor: Use this guide before changing adapter code so every claimed missing feature is backed by source-side evidence and archive-side measurements.
WhenToUse: Use when investigating missing thinking, usage, attachments, lineage, tool metadata, lifecycle events, or unsupported source formats.
---


# Source-vs-archive coverage investigation guide

## Goal

The central question is not “does the archive have field X?” It is:

> When the native transcript source contains useful fact X, does the adapter preserve that fact in the minitrace archive in a queryable, semantically appropriate place?

This guide defines the cross-adapter method for answering that question efficiently and safely. It covers thinking/reasoning blocks, usage/token fields, attachments/images, tool result metadata, subagent/lineage fields, lifecycle events, environment metadata, and unsupported source formats.

The workflow is designed to avoid two common mistakes:

1. **False bugs:** reporting a missing archive field when the source never contained that fact.
2. **False confidence:** seeing a field somewhere in `framework_metadata` and assuming it is usable, even though the normalized query layer or web UI cannot find it.

## Coverage model

For every fact family, measure each layer separately:

```text
native source JSONL/export/db
    ↓ structural source profiler
source fact counts
    ↓ adapter conversion
.minitrace.json archive
    ↓ archive profiler / SQLite query layer
query-visible fact counts
    ↓ human classification
covered / absent-in-source / dropped / not-query-visible / unsupported
```

The classification vocabulary is:

| Classification | Meaning | Example |
|---|---|---|
| `covered` | Source has fact and archive/query layer exposes it. | Pi non-empty thinking blocks become `turn.thinking`. |
| `absent-in-source` | Output is empty because source had no fact. | Copilot sample has no thinking records. |
| `encrypted-or-signature-only` | Source has structural blocks but no cleartext payload. | Claude Code `thinking` blocks have signatures but empty cleartext thinking. |
| `dropped` | Source has clear fact and archive loses it. | Image blocks exist in source but no archive attachment/event is emitted. |
| `partial` | Some facts are preserved, but count or granularity is lower. | Codex reasoning events exceed archive turns with thinking. |
| `not-query-visible` | Archive contains fact only in opaque metadata or raw JSON. | ToolUseResult object preserved but not exposed in normalized columns. |
| `unsupported-source-shape` | Converter rejects the source before preservation. | Old Codex `unknown-jsonl` sampled files. |
| `needs-human-review` | Source has a useful shape, but the correct destination is unclear. | Claude Code reminders: attachment, event, or metadata? |

## Script: `04-profile-source-vs-archive-coverage.py`

The script is located at:

```text
scripts/04-profile-source-vs-archive-coverage.py
```

It reads sample lists produced by `01-inventory-source-shapes.py`, profiles the raw JSONL structurally, profiles converted `.minitrace.json` archives, and writes aggregate reports under:

```text
sources/coverage-profile/
```

Generated files:

| File | Purpose |
|---|---|
| `source-aggregate.json` | Per-adapter source-side structural counts. |
| `source-per-file.json` | Per-sampled-file structural counts, path-hashed and content-free. |
| `archive-aggregate.json` | Per-adapter archive-side fact counts. |
| `coverage-findings.json` | Machine-readable classified findings. |
| `01-coverage-profile.md` | Human-readable generated report with docmgr-compatible frontmatter. |

Run it with:

```bash
scripts/04-profile-source-vs-archive-coverage.py --max-lines 20000 \
  | tee scripts/logs/04-profile-source-vs-archive-coverage.log
```

## Redaction and safety policy

The script must not copy raw prompts, assistant prose, tool output, absolute source paths, or raw transcript files into the ticket. It records only:

- record types;
- content block types;
- key names, with dynamic path-like keys collapsed to `<dynamic-key>`;
- structural payload shapes;
- counts of field families;
- string length buckets, not string values;
- archive-side counts.

If a suspected bug requires inspecting raw JSONL, inspect only a few local lines manually and then translate the finding into a minimized synthetic fixture. Do not commit the original transcript.

## Fact families to investigate

### 1. Thinking and reasoning

Source-side signals:

- content block types: `thinking`, `reasoning`, `redacted_thinking`;
- Codex records: `reasoning`, `agent_reasoning`, payload type `reasoning`;
- non-empty payload fields: `thinking`, `text`, `summary`, `content`;
- encrypted/signature-only fields: `signature`, `encrypted_content`.

Archive-side signals:

- `turn.thinking` count;
- `turn.framework_metadata` keys preserving summary or encrypted-only status;
- `events` if reasoning is lifecycle-like rather than a turn attribute.

Important nuance: a source `thinking` block is not automatically a dropped cleartext thought. Claude Code samples contained 983 `thinking` blocks, but manual inspection found empty `thinking` strings with `signature` fields. That should be classified as `encrypted-or-signature-only`, not a direct adapter bug.

### 2. Usage and token accounting

Source-side signals:

- `message.usage.*` in Pi and Claude Code;
- Codex `token_count` events;
- fields such as `input_tokens`, `output_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens`, `reasoning_output_tokens`, `total_tokens`;
- Claude Code nested `usage.iterations[]` and `server_tool_use`.

Archive-side signals:

- `turn.usage` presence;
- `metrics` aggregate token totals;
- `framework_metadata` for fields not represented in `minitrace.Usage`.

The current schema has first-class fields for input, output, cache read, cache creation, reasoning, and tool tokens. If the source contains extra fields such as service tier, speed, server tool use, or per-iteration usage, preserve them in framework metadata or add an explicit event/metric if they become query targets.

### 3. Attachments and images

Source-side signals:

- content block type `image`;
- keys containing `attachment`, `image`, `mediaType`, `filePath`, `url`;
- Codex `view_image` function/tool events;
- Claude Code `attachment` records.

Archive-side signals:

- `attachments[]` count and kind;
- `events.kind = attachment` or `image_view`;
- tool-call `content_origin` and `attachment_id` links.

Correct destination:

- first-class `Attachment` for durable artifact references;
- `Event` for timeline facts like “image viewed”;
- `ToolCall.Output.ContentOrigin` when the image is a tool result modality.

### 4. Tool result richness

Source-side signals:

- Claude Code `toolUseResult` shapes;
- stdout/stderr/interrupted/duration/exit fields;
- file edit patches (`structuredPatch`, `oldString`, `newString`, `originalFile`);
- task/subagent results (`totalDurationMs`, `totalTokens`, `toolStats`, `usage`);
- Codex `function_call_output`, exec metadata, custom tool outputs.

Archive-side signals:

- `tool_calls.output.result`, `error`, `duration_ms`, `exit_code`, `truncated`;
- `tool_calls.framework_metadata.tool_use_result`;
- `spawned_agent` fields;
- annotations for orphan tool calls.

Rule: preserve source-native tool result objects under capped framework metadata first; promote repeatedly queried fields later.

### 5. Subagents and lineage

Source-side signals:

- Claude Code: `isSidechain`, `parentUuid`, `agentId`, `Task`/`Agent` tool results;
- Codex: `spawn_agent`, `wait_agent`, `parent_thread_id`, fork/replay fields;
- Pi: `parentId`, `branchedFrom`, OMP `parentSession` if present.

Archive-side signals:

- `tool_calls.spawned_agent`;
- session `coordination.predecessor_session` or framework config backlinks;
- `events.kind = subagent_spawn/subagent_wait`;
- handover records if applicable.

### 6. Lifecycle and environment metadata

Source-side signals:

- mode changes;
- permission/approval changes;
- title changes;
- model changes;
- cwd/git branch/version/session IDs;
- rate-limit and compaction events.

Archive-side signals:

- session environment and operational context;
- lifecycle events;
- framework config;
- metrics.

Rule: do not bury important time-varying lifecycle facts only in final session config. If a value changes during the transcript, emit an event too.

## Efficient manual inspection before scripting

Before adding a new broad aggregation rule, inspect just enough raw structure to avoid misclassifying facts. Good patterns:

```bash
# Show structural keys for first two records of sampled files, without printing full contents.
python3 - <<'PY'
import json
from pathlib import Path
sample = Path('sources/source-shape-inventory/claude-code-sample-list.txt')
for raw in sample.read_text().splitlines()[:1]:
    for i, line in enumerate(open(raw, encoding='utf-8', errors='replace')):
        if i >= 2: break
        obj = json.loads(line)
        print(obj.get('type'), sorted(obj.keys()))
PY
```

```bash
# Count cleartext vs signature-only thinking blocks.
python3 - <<'PY'
import json
from pathlib import Path
for adapter in ['claude-code','pi']:
    total = nonempty = signature = 0
    sample = Path(f'sources/source-shape-inventory/{adapter}-sample-list.txt')
    for raw in sample.read_text().splitlines():
        for line in open(raw, encoding='utf-8', errors='replace'):
            obj = json.loads(line)
            msg = obj.get('message') or obj
            content = msg.get('content') if isinstance(msg, dict) else None
            if isinstance(content, list):
                for block in content:
                    if isinstance(block, dict) and block.get('type') == 'thinking':
                        total += 1
                        nonempty += bool(block.get('thinking'))
                        signature += bool(block.get('signature'))
    print(adapter, total, nonempty, signature)
PY
```

## Implementation rule for adapter fixes

A fix should follow this sequence:

1. Add or extend a source profiler rule so the missing fact is measurable.
2. Confirm the fact exists in at least one sampled source.
3. Inspect a few raw records locally to understand shape and semantics.
4. Create a minimized synthetic fixture with only the relevant structure.
5. Add a failing unit test against that fixture.
6. Update the adapter.
7. Re-run conversion and coverage scripts.
8. Update `pkg/doc/adapter-reference.md` with measured claims.

## Priorities from the first coverage run

1. **Codex convertibility:** 4/12 sampled Codex files did not convert; this is the highest-priority gap because whole sessions are missing.
2. **Claude Code thinking classification:** source has 983 thinking blocks, but they are signature-only/empty in the sample. The adapter may be correct to leave `turn.thinking` empty, but it should preserve the fact that signed thinking blocks existed if that is useful.
3. **Codex reasoning granularity:** source reasoning signals greatly exceed `turn.thinking` counts; inspect whether multiple reasoning events are intentionally merged per assistant turn or dropped.
4. **Pi thinking granularity:** source non-empty thinking blocks (1,572) are close to archive thinking turns (1,523); investigate whether the difference is expected due to multiple blocks per turn or a small drop.
5. **Attachments/images:** Pi and Copilot show source attachment/image signals without archive attachments in the current converted corpus.
6. **Copilot coverage:** the sample has usage/token and attachment/image signals, but no converted archive exists yet because the first conversion script did not cover Copilot.
