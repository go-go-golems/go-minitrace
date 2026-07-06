---
Title: Missing adapter functionality report
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
      Note: Adapter with signed/empty thinking blocks, rich toolUseResult shapes, attachments, subagents, and lifecycle events.
    - Path: pkg/adapters/codex/convert.go
      Note: Adapter with unsupported old JSONL samples and partial reasoning/token/lifecycle coverage.
    - Path: pkg/adapters/pi/convert.go
      Note: Adapter with high thinking/tool-duration coverage and newly fixed image attachment mapping.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts/scripts/04-profile-source-vs-archive-coverage.py
      Note: |-
        Coverage profiler used to classify findings.
        Source-vs-archive classifier
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts/sources/coverage-profile/01-coverage-profile.md
      Note: |-
        Generated source-vs-archive profile backing this report.
        Generated evidence report
ExternalSources: []
Summary: Evidence-backed report of likely missing or incomplete adapter functionality from the first source-vs-archive coverage pass.
LastUpdated: 2026-07-06T19:00:00-04:00
WhatFor: Prioritize adapter fixes and documentation updates based on measured local transcript evidence.
WhenToUse: Use before implementing GMT-012 adapter changes or updating adapter-reference.md.
---


# Missing adapter functionality report

## Executive summary

This report summarizes the first measured source-vs-archive coverage pass for GMT-012. It is based on local sampled transcript sources and converted minitrace archives. The goal is to identify likely missing adapter functionality, not to prove final product quality.

The most important result is that “missing thinking” is not one problem. It differs by adapter:

- **Claude Code:** source has 983 `thinking` content blocks, but manual structural inspection shows they are empty cleartext blocks with `signature` fields. This is not a direct “adapter dropped thinking text” bug. It is a preservation/classification gap: the archive should probably record that signed thinking blocks existed, but it cannot expose cleartext thinking that is not present.
- **Codex:** source has many reasoning records/events and only some are visible as `turn.thinking`. This needs mapping/granularity investigation.
- **Pi:** source has 1,572 non-empty thinking blocks and archive has 1,523 thinking turns. This is close, but the delta needs explanation.

Highest-priority missing functionality:

1. ~~Codex old JSONL convertibility: 4/12 sampled Codex files failed conversion.~~ Fixed in commit pending after this report update: legacy rollout JSONL now converts.
2. Codex reasoning granularity: source reasoning signals greatly exceed archive `turn.thinking` count.
3. Claude Code signed thinking preservation: record signed/empty thinking block presence even when cleartext is unavailable.
4. ~~Pi image block preservation: source image blocks observed but no archive attachments emitted in the sampled corpus.~~ Fixed: Pi image blocks now become bounded `attachments[]`.
5. Copilot conversion coverage: source sample contains useful events, but the current conversion pass did not produce a Copilot archive.
6. Tool result metadata promotion: many rich tool result facts are preserved only as capped metadata or not query-visible.

## Evidence base

Scripts run:

```bash
scripts/01-inventory-source-shapes.py --max-per-adapter 12 --max-lines 1500
scripts/02-convert-sampled-jsonl.sh
scripts/03-query-converted-fidelity.sh
scripts/04-profile-source-vs-archive-coverage.py --max-lines 20000
```

Generated report:

```text
sources/coverage-profile/01-coverage-profile.md
```

Converted archive totals from the profile:

| Adapter | Converted sessions | Turns | Tool calls | Events | Attachments |
|---|---:|---:|---:|---:|---:|
| Claude Code | 12 | 3,375 | 1,661 | 2,392 | 1,555 |
| Codex | 8 | 649 | 2,416 | 20 | 1 |
| Pi | 12 | 3,854 | 4,179 | 144 | 0 |
| Copilot | 0 | 0 | 0 | 0 | 0 |

## Finding 1: Codex old JSONL files did not convert — fixed

**Severity:** resolved high  
**Classification:** formerly unsupported-source-shape  
**Evidence before fix:** 12 sampled Codex files; 8 converted; 4 failed as `unsupported Codex format hint: unknown-jsonl`.
**Evidence after fix:** all 12 sampled Codex files convert; the four older files produce archives with `provenance.source_format = codex-legacy-rollout-jsonl-v0`.

### Why this matters

A conversion failure is more severe than a missing field. The entire session disappears from downstream query, UI, and report workflows.

### Likely cause

The failing samples are older Codex rollout JSONL shapes. Manual structural inspection showed records like:

- first record with top-level `id`, `timestamp`, `instructions`, `git`;
- follow-up records with `record_type`;
- no modern `session_meta` wrapper.

The adapter was optimized for newer `session_meta` / `event_msg` / `response_item` shapes. It now also detects older top-level rollout records (`message`, `reasoning`, `function_call`, `function_call_output`, and `record_type: state`).

### Implemented fix

The adapter now has a legacy rollout parser that maps:

- top-level `id` to session ID;
- top-level `git.branch`/`git.commit_hash` to normalized operational context;
- `message` records to turns;
- `reasoning.summary[].text` to assistant-turn thinking when cleartext is present;
- legacy `shell` function calls to normalized `exec_command` tool calls;
- `function_call_output` payloads through the existing output parser.

### Acceptance criteria

- The 4 sampled old Codex files now convert.
- `adapter-reference.md` documents `codex-legacy-rollout-jsonl-v0`.

## Finding 2: Codex reasoning coverage is partial

**Severity:** medium  
**Classification:** partial / needs-human-review  
**Evidence:** profiler counted 3,110 Codex reasoning-related source signals versus 198 archive turns with `thinking`.

### Why this matters

Reasoning summaries are valuable for review, debugging, and agent behavior analysis. If multiple source reasoning events are merged per assistant turn, lower archive counts can be correct. If they are dropped, the adapter loses important context.

### What to inspect next

Codex source shapes include:

- `record_types.reasoning = 66`
- `payload_types.reasoning = 1432`
- `payload_types.agent_reasoning = 806`
- `token_count` events with `reasoning_output_tokens`

Archive output includes:

- `turn.thinking = 198`
- `turn.usage = 567`
- only 20 events total across 8 converted sessions.

### Recommended fix

First determine intended granularity:

- If many reasoning records belong to one assistant turn, preserve merged summaries in `turn.thinking` and record source counts in metadata.
- If reasoning records are timeline events independent of turns, emit `events.kind = reasoning` or `agent_reasoning`.
- If encrypted-only reasoning is present, preserve that fact as metadata without raw encrypted payload unless useful and safe.

### Acceptance criteria

- A regression fixture with multiple Codex reasoning payloads proves whether they merge, append, or emit events.
- Coverage report classifies Codex reasoning as covered/partial with a written explanation.

## Finding 3: Claude Code thinking blocks are signature-only in sampled data

**Severity:** info-to-medium  
**Classification:** encrypted-or-signature-only / not-query-visible  
**Evidence:** 983 Claude Code `thinking` blocks; 0 archive `turn.thinking`; manual structural inspection showed empty `thinking` strings and present `signature` fields.

### Why this matters

The initial output-only metric looked alarming: Claude Code had zero turns with thinking. Source-side inspection corrected the interpretation. The adapter cannot populate cleartext `turn.thinking` when the source does not contain cleartext.

However, the source does contain useful metadata: signed thinking blocks occurred. This may matter for completeness, UI explanation, and documentation.

### Recommended fix

Do not synthesize thinking text. Instead, preserve signed-thinking presence.

Possible representation:

- `turn.framework_metadata.signed_thinking_blocks = N`
- `turn.framework_metadata.thinking_signature_present = true`
- optionally `events.kind = signed_thinking` if the UI needs timeline visibility

Avoid storing raw signatures unless there is a known replay/verification use case.

### Acceptance criteria

- A minimized Claude Code fixture with a `thinking` block containing `signature` but empty `thinking` converts to a turn that records signature-only thinking presence.
- `adapter-reference.md` says: cleartext Claude Code thinking not observed; signed/empty thinking blocks preserved as metadata.

## Finding 4: Claude Code toolUseResult has rich fields that are not all first-class/query-visible

**Severity:** medium  
**Classification:** not-query-visible / partial  
**Evidence:** 1,663 `toolUseResult` records across 12 Claude Code samples with many shapes.

Common shapes include:

- Bash-like: `interrupted,isImage,noOutputExpected,stderr,stdout`
- Edit/write-like: `filePath,newString,oldString,originalFile,replaceAll,structuredPatch,userModified`
- Subagent/task-like: `agentId,agentType,content,prompt,resolvedModel,status,toolStats,totalDurationMs,totalTokens,totalToolUseCount,usage`
- Search/fetch-like: `durationSeconds,query,results,searchCount`

The adapter currently preserves capped `tool_use_result` metadata and maps some exit/duration/error facts. That is good, but not all useful fields are query-visible.

### Recommended fix

Promote repeatedly useful fields:

| Source field family | Suggested destination |
|---|---|
| `totalDurationMs`, `durationSeconds` | `tool_calls.output.duration_ms` if semantically the tool duration |
| `totalTokens`, `usage` | tool metadata now; future tool usage schema if needed |
| `structuredPatch`, `oldString`, `newString` | capped framework metadata; maybe file-edit event summary |
| `isImage` | attachment/content-origin if paired with image path/content |
| `status`, `statusChange` | event or tool metadata depending on tool type |
| `interrupted` | `output.success=false`, `output.error`, metadata flag |

### Acceptance criteria

- Add query examples showing how to find interrupted Claude tools, edit tools, and task tools.
- Consider adding normalized JSON metadata helper functions if SQL/JS users need to search these frequently.

## Finding 5: Pi thinking coverage is close but not exact

**Severity:** medium  
**Classification:** partial / needs-human-review  
**Evidence:** source profiler counted 1,572 non-empty Pi thinking blocks; archive profiler counted 1,523 turns with thinking.

### Why this matters

Pi thinking coverage appears mostly working. The delta could be expected if multiple thinking blocks are attached to a single turn or if some blocks are empty/invalid. It could also hide a small adapter bug.

### Recommended fix

Add a per-file comparator for Pi:

- source non-empty thinking block count per file;
- archive turns with thinking per converted session;
- number of assistant turns with multiple thinking blocks;
- number of thinking-only messages, if any.

If multiple source blocks are joined into one turn, document this as intended. If blocks are missing, add a fixture.

### Acceptance criteria

- Explain the 1,572 vs 1,523 difference.
- Add a test for multiple thinking blocks in one Pi assistant message.

## Finding 6: Pi image blocks were not represented as attachments — fixed

**Severity:** resolved medium  
**Classification:** formerly dropped/not-yet-mapped  
**Evidence before fix:** Pi source profiler found 6 `image` content blocks and attachment/image key signals; converted Pi archives had `attachments = 0`.
**Evidence after fix:** rerunning the sampled Pi conversion and coverage profile removes the Pi attachment/image finding; Pi image blocks become first-class bounded attachments.

### Why this matters

Images are high-value context for multimodal agent sessions. Dropping them makes replay and audit incomplete.

### Implemented fix

The Pi adapter now maps image content blocks to `Attachment` records:

- `Attachment.Kind = image`
- `Attachment.MediaType` from `mimeType` / `media_type`
- inline image data is not embedded in `RawJSON`; instead a sha256 hash, size, and `content_ref = inline:image` are recorded
- normal message images link to `turn_index`
- tool-result images link to `tool_call_id`
- tool-result textual output uses an image placeholder such as `[image image/png]` instead of serializing base64 data

### Acceptance criteria

- Minimized Pi fixtures cover assistant-turn image blocks and tool-result image blocks.
- The sampled coverage profile no longer reports Pi attachment/image loss.

## Finding 7: Copilot is inventoried but not converted in the first pass

**Severity:** high for coverage completeness  
**Classification:** missing audit coverage  
**Evidence:** one local Copilot JSONL candidate was profiled; no converted Copilot archive exists in the sampled corpus.

Source sample contains:

- `assistant.turn_start`, `assistant.message`, `assistant.turn_end`
- `tool.execution_start`, `tool.execution_complete`
- `permission.requested`, `permission.completed`
- `session.model_change`, `session.info`, `session.shutdown`
- usage/token and attachment/image key signals

### Recommended fix

Add a Copilot conversion script:

```text
scripts/05-convert-sampled-copilot.sh
```

It should use the real adapter CLI path rather than forcing the JSONL `--source-list` shape if Copilot discovery expects session-state directories.

### Acceptance criteria

- The one sampled Copilot session either converts or fails with a documented source-layout reason.
- Coverage profiler includes archive facts for Copilot.

## Finding 8: Usage exists but exact field coverage needs a dedicated comparator

**Severity:** medium  
**Classification:** partial / not-yet-verified  
**Evidence:** all major JSONL adapters show source usage/token signals and archive `turn.usage`, but counts are not directly comparable because key occurrence counts are much larger than turn counts.

### Why this matters

Token accounting is critical for cost and performance analysis. The current profiler proves presence, not exact preservation.

### Recommended fix

Create a usage-specific comparator:

```text
scripts/05-compare-usage-fields.py
```

It should compute per-session totals for fields available in the source and compare them to archive `metrics` and per-turn `usage`.

Field families:

- input tokens;
- output tokens;
- cache read tokens;
- cache creation tokens;
- reasoning tokens;
- total tokens;
- service tier / speed / server tool use as metadata.

### Acceptance criteria

- Per-adapter usage totals match within known semantic boundaries.
- Any intentionally unmodeled fields are listed in adapter docs.

## Implementation priority table

| Priority | Adapter | Work item | Reason |
|---:|---|---|---|
| 1 | Codex | Legacy/old JSONL convertibility | Fixed: all 12 sampled Codex files now convert. |
| 2 | Copilot | Add conversion coverage script | Current report cannot judge adapter output. |
| 3 | Claude Code | Preserve signature-only thinking metadata | Clarifies “missing thinking” without inventing text. |
| 4 | Pi | Map image blocks to attachments | Fixed: image blocks become bounded attachments. |
| 5 | Codex | Reasoning granularity audit/fix | Valuable reasoning context likely partially hidden. |
| 6 | All | Usage exact comparator | Important for cost/performance, but not yet proven broken. |
| 7 | Claude Code | Promote useful toolUseResult fields | Improves queryability after core coverage gaps. |

## Documentation updates required after fixes

Update `pkg/doc/adapter-reference.md` with measured statements, not aspirational claims. Suggested wording style:

- “Claude Code cleartext thinking: not observed in sampled local corpus; signed thinking block presence preserved as metadata.”
- “Codex legacy rollout JSONL: supported/unsupported as of commit X.”
- “Pi image blocks: converted to first-class attachments.”
- “Usage fields: input/output/cache fields mapped; service tier and per-iteration details preserved in framework metadata.”

## Related documentation issue

A separate GitHub issue was created for broader docs cleanup:

- https://github.com/go-go-golems/go-minitrace/issues/23

That issue should not block GMT-012 adapter work, but GMT-012 findings should feed into the eventual docs rewrite. In particular, examples should prefer the JS query API for user-facing analysis and use raw SQL as advanced/reference material.
