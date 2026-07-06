---
Title: Adapter fidelity real-transcript audit and improvement guide
Ticket: GMT-012-adapter-fidelity-real-transcript-audit
Status: active
Topics:
    - tooling
    - cli
    - diagnostics
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/doc/adapter-reference.md
      Note: Starting point and final destination for measured adapter fidelity claims.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/adapters/pi/convert.go
      Note: Pi adapter under audit.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/adapters/claudecode/convert.go
      Note: Claude Code adapter under audit.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/adapters/codex/convert.go
      Note: Codex adapter under audit.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/adapters/copilot/discover.go
      Note: Copilot support and source-shape entry point under audit.
    - Path: /home/manuel/code/others/agentsview/internal/parser/provider.go
      Note: Reference architecture for provider-owned source identity, fingerprinting, partial parse outcomes, and incremental parsing.
    - Path: /home/manuel/code/others/agentsview/internal/parser/codex.go
      Note: Reference Codex parser with fork/replay handling and rich lifecycle semantics.
    - Path: /home/manuel/code/others/agentsview/internal/parser/pi.go
      Note: Reference Pi parser with header/title-slot and lineage handling.
    - Path: /home/manuel/code/others/agentsview/internal/parser/claude_provider.go
      Note: Reference Claude provider with discovery, parse, upload, and incremental parse surfaces.
ExternalSources: []
Summary: Plan for a real-transcript, evidence-based audit of go-minitrace adapters. The goal is to verify conversion fidelity against actual source stores, compare against agentsview parsing patterns, implement high-confidence fixes, and update adapter-reference.md with measured claims rather than assumptions.
LastUpdated: 2026-07-06T14:10:00-04:00
WhatFor: Guide a systematic adapter-quality pass after the single-query-engine migration.
WhenToUse: Use before changing adapter conversion code, adding support for new source event shapes, or updating adapter fidelity documentation.
---

# Adapter fidelity real-transcript audit and improvement guide

## 1. Executive summary

GMT-012 is an evidence-first adapter quality pass. The single-query-engine migration made querying converted archives easier; now we should use that capability to verify that the archives are as faithful as possible to real Pi, Claude Code, Codex, Copilot, ChatGPT, claude.ai, and turnsdb source transcripts.

The starting point is `pkg/doc/adapter-reference.md`, which already documents a per-adapter fidelity matrix. The risk is that some claims are based on source reading and targeted tests rather than a broad real-transcript corpus. GMT-012 turns those claims into measured evidence:

1. collect representative real transcripts for each source type;
2. summarize source shapes without leaking private transcript content;
3. convert with current adapters;
4. query coverage/null/drop patterns from normalized SQLite;
5. compare against source facts and agentsview parser behavior;
6. implement high-confidence fixes with regression fixtures;
7. update adapter docs with measured fidelity and known losses.

This ticket is deliberately broad in investigation but conservative in implementation: fix only issues that can be reproduced with a real source sample and captured in tests.

## 2. Problem statement

Adapters are where transcript truth enters go-minitrace. If an adapter drops a lifecycle event, misclassifies a tool call, loses a timestamp, misses a subagent relationship, or silently truncates provenance, every downstream query and UI feature inherits that flaw.

GMT-009 improved several known gaps, but we still need a systematic pass because:

- source formats evolve independently;
- real transcript stores contain edge cases absent from synthetic tests;
- multiple frameworks encode similar concepts differently;
- `adapter-reference.md` should say what is measured, not merely intended;
- agentsview has an extensive parser/provider implementation that can reveal cases go-minitrace does not yet model.

## 3. Source systems in scope

Primary go-minitrace adapters:

| Adapter | Source | Primary code | Audit emphasis |
|---|---|---|---|
| Pi | `~/.pi/agent/sessions` JSONL | `pkg/adapters/pi/convert.go` | title slot/header variants, thinking, usage/cost, tool duration, edit diff metadata, model changes |
| Claude Code | `~/.claude/projects` JSONL/subagents | `pkg/adapters/claudecode/convert.go` | `toolUseResult`, subagents, permission mode, attachments, cwd/git branch records, incremental/lifecycle records |
| Codex | `~/.codex/sessions` and exec JSONL | `pkg/adapters/codex/convert.go` | fork/replay history, spawn/wait agent relationships, exec metadata, turn IDs, rate limits, truncation |
| Copilot | GitHub Copilot CLI/session JSONL | `pkg/adapters/copilot` | permission events, deferred tool ordering, cwd/branch/version, opaque/encrypted content caveats |
| ChatGPT | exports/conversations | `pkg/adapters/chatgpt` | tool/search payloads, attachments, system/content role mapping, timestamps |
| claude.ai | exported chats | `pkg/adapters/claudeai` | reasoning blocks, attachments, extended search/subagent-like facts, summaries |
| turnsdb | Geppetto/Pinocchio turns DB | `pkg/adapters/turnsdb` | DB row identity, delta/tool-call mapping, raw payload preservation |

Agentsview reference material:

- `internal/parser/provider.go` — provider-owned source identity, fingerprints, partial parse outcomes, incremental parsing.
- `internal/parser/codex.go` — fork/replay gate, task lifecycle, agent spawn/wait handling.
- `internal/parser/pi.go` — title slot/header variants, branch lineage, visible ancestor mapping.
- `internal/parser/claude_provider.go` — discovery + upload parse + incremental parse design.
- `internal/parser/copilot_provider.go` — source discovery/fingerprint structure for Copilot.

Agentsview is not a drop-in replacement. Use it as an idea/reference oracle: what edge cases does it know about, and should go-minitrace preserve equivalent facts in minitrace schema fields/events/attachments/framework metadata?

## 4. Evidence workflow

### 4.1 Create ticket scripts, not ad hoc shell history

All scripts created for this audit should live under:

```text
ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts/scripts/
```

Recommended scripts:

```text
scripts/01-inventory-source-shapes.py
scripts/02-sample-real-transcripts.sh
scripts/03-convert-corpus.sh
scripts/04-query-fidelity-coverage.sql
scripts/05-compare-source-vs-archive.py
scripts/06-agentsview-pattern-notes.sh
```

### 4.2 Real transcript sampling rules

Sample from local stores when present:

```bash
# Pi
find ~/.pi/agent/sessions -name '*.jsonl' | shuf | head -50

# Claude Code
find ~/.claude/projects -name '*.jsonl' | shuf | head -50

# Codex
find ~/.codex/sessions -name '*.jsonl' | shuf | head -50
```

For each adapter, try to include:

- small/simple session;
- long session;
- tool-heavy session;
- error/failed-tool session;
- session with attachments/images, if supported;
- session with subagents/forks/branching, if supported;
- recent session and older historical session.

Do not commit raw private transcripts. Commit only redaction-safe summaries, synthetic minimized fixtures, or copied fixture fragments that are confirmed safe.

### 4.3 Source-shape summary format

For each sampled source file, produce a JSON summary like:

```json
{
  "adapter": "codex",
  "source_path_hash": "sha256:...",
  "line_count": 1234,
  "event_types": {"session_meta": 1, "response_item": 450},
  "tool_names": {"shell": 120, "apply_patch": 8},
  "has_errors": true,
  "has_subagents": true,
  "has_attachments": false,
  "timestamp_fields": ["timestamp", "created_at"],
  "interesting_keys": ["parent_thread_id", "agent_role", "duration_seconds"]
}
```

These summaries are safe to commit and give reviewers evidence without exposing content.

### 4.4 Conversion run

Convert into a ticket-local scratch output directory, not a user output directory:

```bash
OUT=ttmp/.../sources/converted-corpus
mkdir -p "$OUT"

go run ./cmd/go-minitrace convert pi \
  --source-list ttmp/.../sources/pi-sample-list.txt \
  --output-dir "$OUT/pi"

go run ./cmd/go-minitrace convert claude-code \
  --source-list ttmp/.../sources/claude-code-sample-list.txt \
  --output-dir "$OUT/claude-code"

go run ./cmd/go-minitrace convert codex \
  --source-list ttmp/.../sources/codex-sample-list.txt \
  --output-dir "$OUT/codex"
```

Keep command output logs in `scripts/logs/` and summarize failures in the diary.

### 4.5 Fidelity queries

Use normalized SQLite through `query run`, not bespoke JSON grep, for the converted output:

```sql
SELECT agent_framework, COUNT(*) AS sessions
FROM sessions
GROUP BY agent_framework;

SELECT agent_framework,
       COUNT(*) AS tool_calls,
       SUM(duration_ms IS NULL) AS missing_duration,
       SUM(exit_code IS NULL) AS missing_exit_code,
       SUM(error IS NOT NULL AND error <> '') AS errors
FROM tool_calls
GROUP BY agent_framework;

SELECT agent_framework,
       SUM(thinking IS NOT NULL AND thinking <> '') AS turns_with_thinking,
       COUNT(*) AS turns
FROM turns
GROUP BY agent_framework;

SELECT agent_framework, kind, COUNT(*) AS events
FROM events
GROUP BY agent_framework, kind
ORDER BY agent_framework, events DESC;
```

These queries become the measured evidence behind the updated matrix.

## 5. Agentsview comparison checklist

Use agentsview as a reference for edge cases and parser architecture.

### Provider/source identity

Agentsview `Provider` separates:

- discovery (`Discover`),
- source identity (`SourceRef`),
- fingerprinting (`Fingerprint`),
- parsing (`Parse`),
- partial parse outcomes and source errors.

Compare go-minitrace's `discover` and `convert --source-session/--source-list` flows against this model. Watch for:

- unstable session IDs derived from paths;
- inability to find one logical source from a displayed ID;
- batch failure semantics vs per-session source errors;
- missing fingerprint freshness facts.

### Codex

Agentsview's Codex parser has a fork/replay gate. Audit whether go-minitrace handles equivalent Codex forked-session replay history or double-counts messages/tool usage.

Questions:

- Does source JSON contain `forked_from_id`?
- Are turn IDs UUIDv7 or otherwise timestamp-anchored?
- Does go-minitrace suppress replayed parent history or record it as provenance?
- Do spawned agents and wait-agent outcomes map correctly?

### Pi

Agentsview handles Pi title-slot lines, v1/v2 headers, `branchedFrom`, OMP `parentSession`, and visible ancestor mapping for metadata rows.

Questions:

- Does go-minitrace skip title slot lines before session header?
- Does it handle old Pi sessions without explicit IDs?
- Does it preserve branch/parent lineage when present?
- Does metadata-row parentage affect turn/tool attachment?

### Claude Code

Agentsview has provider surfaces for discovered sessions, uploaded transcript files, and incremental parse. go-minitrace may not need incremental parsing, but the source-shape handling is worth comparing.

Questions:

- Are subagent sessions discovered and linked consistently?
- Are attachment events preserved as `attachments[]` and `events[]`?
- Are permission/mode/title lifecycle records preserved?
- Does `toolUseResult` mapping cover all observed forms?

### Copilot

Agentsview's Copilot provider emphasizes discovery/fingerprint/source identity. Compare source layouts and event ordering.

Questions:

- Does go-minitrace discover both bare and directory layouts if they exist?
- Does it deduplicate duplicate representations of one session?
- Are permission/deferred events attached to the right turn/tool call?

## 6. Implementation policy

Implement fixes only when all of the following are true:

1. A real transcript or minimized fixture demonstrates the issue.
2. The source fact has a natural minitrace destination: schema field, event, attachment, annotation, or framework metadata.
3. The behavior can be covered by a unit test or fixture test.
4. The docs can state the resulting fidelity honestly.

Avoid broad rewrites that only copy agentsview architecture. go-minitrace's archive schema and CLI workflows are different. Borrow edge-case knowledge and parsing patterns, not entire subsystems.

## 7. Expected deliverables

1. `analysis` or `design-doc` report with adapter-by-adapter findings.
2. Redaction-safe source-shape summaries under `sources/`.
3. Conversion logs and query outputs under `scripts/logs/`.
4. Adapter fixes with tests, if findings warrant code changes.
5. Updated `pkg/doc/adapter-reference.md` with measured matrix notes.
6. Diary entries after each task group.

## 8. Validation checklist

Before closing the ticket:

```bash
GOWORK=off go test ./pkg/adapters/... ./pkg/minitrace ./pkg/minitracedb ./cmd/go-minitrace/cmds/convert ./cmd/go-minitrace/cmds/discover
GOWORK=off go test ./...
make glazed-lint
# plus corpus conversion scripts from this ticket
```

Run docmgr hygiene:

```bash
docmgr --root ./ttmp doctor --ticket GMT-012-adapter-fidelity-real-transcript-audit --stale-after 30
```

## 9. Open questions

- Should go-minitrace adopt an explicit provider/source identity layer similar to agentsview, or is that too large for this ticket?
- Which real transcript summaries are safe to commit?
- Should adapter quality become a recurring regression suite, with sanitized source fixtures for every discovered edge case?
- Which source facts belong in first-class schema fields vs `framework_metadata`?
