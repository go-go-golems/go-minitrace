---
Title: Turnsdb Tool Call Conversion Fix Design
Ticket: GMT-008
Status: active
Topics:
    - minitrace
    - turnsdb
    - conversion
    - tool-calls
    - coinvault
    - web-ui
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/go-minitrace/cmds/serve/blocks.go
      Note: |-
        Server block builder renders tools via Turn.ToolCallsInTurn.
        Server block builder uses Turn.ToolCallsInTurn to render transcript tool rows.
    - Path: pkg/adapters/turnsdb/convert.go
      Note: |-
        Converter source for block delta extraction, tool-call creation, turn linking, and text normalization.
        Converter source for block delta extraction
    - Path: pkg/adapters/turnsdb/convert_test.go
      Note: Regression tests for the turnsdb converter fixes.
ExternalSources:
    - ../2026-03-16--gec-rag/ttmp/2026/05/12/GMINI-0002--debug-missing-tool-calls-in-coinvault-minitrace-transcript-ui/analysis/01-tool-call-rendering-root-cause-analysis.md
Summary: Design and implementation plan for fixing turnsdb tool-call conversion so Coinvault transcripts render tool calls and blank text correctly.
LastUpdated: 2026-05-12T00:00:00Z
WhatFor: Guide GMT-008 implementation in go-minitrace after the GMINI-0002 root-cause investigation.
WhenToUse: Read before modifying turnsdb conversion, converter tests, or transcript block rendering for Coinvault archives.
---


> Ported from the GMINI-0002 root-cause analysis workspace. This GMT-008 copy is the implementation design local to the go-minitrace repository.

# Turnsdb Tool Call Conversion Fix Design

## Executive summary

The missing tool calls are very likely caused by the go-minitrace `turnsdb` converter, not by Coinvault failing to store the tool calls and not primarily by the React UI. The `turns.db` source contains `tool_call` and `tool_use` blocks. The generated minitrace JSON contains top-level `tool_calls` with valid `emitting_turn_index` values. However, the converter does not populate `turns[].tool_calls_in_turn`. The go-minitrace server and UI render transcript tool rows from `turn.ToolCallsInTurn`, so the tools disappear in the transcript blocks even though they exist elsewhere in the session JSON.

The raw `{"text":"\n"}` and `{"text":""}` assistant rounds are also likely a converter issue. The source DB stores `llm_text` blocks as JSON payloads with a `text` field. The converter returns the `text` field only when `strings.TrimSpace(text) != ""`. For whitespace-only text, it falls back to serializing the whole JSON payload, which turns a blank assistant segment into a visible JSON object string.

A patch/probe against generated JSON confirms both hypotheses. After linking tool calls into `turns[].tool_calls_in_turn`, the served `/api/v2/sessions/{id}/blocks` endpoint reports 6 tool calls in block 2 and 6 in block 3 for the representative session `8730...`. After unwrapping JSON-looking text payloads, the raw `{"text":...}` artifacts disappear from the archive.

## Symptoms

Observed in the UI at:

```text
http://127.0.0.1:8090/sessions/8730fef8-2f37-40bb-96e3-73687c55f6ab
```

Symptoms:

- Tool calls are not shown in transcript blocks.
- Assistant turns repeatedly show raw JSON strings such as `{"text":"\n"}` and `{"text":""}`.
- The session summary still reports non-zero tool counts, so tools are present somewhere in the data path.

## Evidence from turns.db

Representative conversation:

```text
8730fef8-2f37-40bb-96e3-73687c55f6ab
```

Comparison script:

```text
scripts/01_compare_session_turns_minitrace.py
```

Output:

```text
turns.db canonical snapshots
snapshots=2
- turn=73e6bc55-64b9-4c4e-9421-049ea7c28f12 phase=final blocks=28 kinds={'system': 1, 'user': 1, 'reasoning': 7, 'llm_text': 7, 'tool_call': 6, 'tool_use': 6}
- turn=b1675625-a9e8-40cd-9bef-0195e4789674 phase=final blocks=31 kinds={'system': 1, 'user': 2, 'reasoning': 8, 'llm_text': 8, 'tool_call': 6, 'tool_use': 6}
total tool_call blocks across canonical snapshots=12
total tool_use blocks across canonical snapshots=12
whitespace-only llm_text blocks across canonical snapshots=12
```

Interpretation:

- Tool calls are stored in `turns.db`.
- Tool results are stored in `turns.db`.
- The blank text artifacts originate from `llm_text` blocks in the source, but the visible raw JSON string is a conversion artifact.

## Evidence from timeline.db

For the same representative conversation, `timeline.db` sessionstream tables contain independent tool-call events:

```text
ChatToolCallStarted: 6
ChatToolCallRequested: 6
ChatToolExecutionStarted: 6
ChatToolResultReady: 6
ChatToolCallFinished: 6
ChatToolCallArgumentsDelta: 22
```

Interpretation:

- The runtime produced tool-call lifecycle events.
- The sessionstream representation agrees with the turns representation that tools happened.
- We do not need to blame Coinvault or Pinocchio storage first for the missing UI rows.

## Evidence from minitrace JSON

For the same representative conversation:

```text
turns=18
tool_calls=12
tool_call ids linked from turns=0
json-looking blank assistant turns=12
tool_calls by emitting_turn_index={8: 6, 17: 6}
```

Interpretation:

- The converter successfully creates top-level `tool_calls`.
- It preserves which transcript turn emitted the tools through `emitting_turn_index`.
- It fails to populate the field the UI uses: `turns[].tool_calls_in_turn`.

## Evidence from served API

Original archive served at port 8090:

```text
api blocks=3
api block toolCalls values=[None, None, None]
```

Patched archive served at port 8091:

```text
block 1 toolCalls=None turn tool lens [0]
block 2 toolCalls=6 turn tool lens [0, 0, 0, 0, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0]
block 3 toolCalls=6 turn tool lens [0, 6]
```

Interpretation:

- The server/UI contract works when `ToolCallsInTurn` is populated.
- The UI disappearance is downstream of conversion, but not necessarily a React rendering bug.

## Source-level root cause

### Tool link bug

The current converter code in go-minitrace:

```text
/home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/turnsdb/convert.go
```

Relevant behavior:

- It computes `emittingTurnIndex`.
- It calls `buildToolCallsFromBlocks(deltaBlocks, timestampPtr, emittingTurnIndex)`.
- `buildToolCallsFromBlocks` creates `minitrace.ToolCall` values with `EmittingTurnIndex`.
- It does not append tool call IDs to `turns[emittingTurnIndex].ToolCallsInTurn`.

The serve block builder uses:

```text
/home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/blocks.go
```

Important behavior:

- `buildRawSessionBlocks` increments block tool counts using `len(turn.ToolCallsInTurn)`.
- `normalizeTurn` resolves visible tool rows by iterating `turn.ToolCallsInTurn` and looking up each ID in `tcByID`.

Therefore a top-level tool call with only `emitting_turn_index` is not enough for the transcript UI.

### Blank JSON text bug

The converter's `stringifyBlockPayload` behaves conceptually like:

```go
if text := stringValue(payload["text"]); strings.TrimSpace(text) != "" {
    return text
}
return stringifyAny(payload)
```

For payloads like:

```json
{"text":"\n"}
{"text":""}
```

this returns the JSON object string, not the intended blank text. The quick patch script proved that unwrapping the object removes those raw JSON artifacts.

## Prototype patch result

Ticket script:

```text
scripts/02_patch_minitrace_archive_for_ui_probe.py
```

Patch actions:

1. Link each top-level tool call into its emitting turn.
2. Unwrap one-field JSON text objects.

Result across all 31 converted archives:

```json
{
  "files": 31,
  "linked_tools": 149,
  "unwrapped_text": 63
}
```

This is not the final implementation; it is a hypothesis test. The real fix belongs in `pkg/adapters/turnsdb/convert.go`.

## Proposed code fix

### Fix 1: attach tool calls to emitting turns

After building `deltaToolCalls`, attach them to the corresponding turn:

```go
deltaToolCalls, deltaAnnotations := buildToolCallsFromBlocks(deltaBlocks, timestampPtr, emittingTurnIndex)
if emittingTurnIndex >= 0 && emittingTurnIndex < len(turns) {
    for _, tc := range deltaToolCalls {
        turns[emittingTurnIndex].ToolCallsInTurn = append(turns[emittingTurnIndex].ToolCallsInTurn, tc.ID)
    }
}
toolCalls = append(toolCalls, deltaToolCalls...)
annotations = append(annotations, deltaAnnotations...)
```

Caveat: if a snapshot has tool calls but no assistant text turn was emitted in the same delta, `emittingTurnIndex` may point to an earlier turn or be `-1`. In that case, consider creating a synthetic assistant/tool turn or attaching to the nearest previous assistant turn. The existing converter already computes an emitting turn index, so start with using it and add tests.

### Fix 2: unwrap text payload even when blank

Replace the whitespace-sensitive text extraction with key-presence-sensitive extraction:

```go
func stringifyBlockPayload(payload map[string]any) string {
    if payload == nil {
        return "{}"
    }
    if value, ok := payload["text"]; ok {
        return stringValue(value)
    }
    return stringifyAny(payload)
}
```

Then decide whether whitespace-only `llm_text` turns should be emitted at all. It may be better to skip empty/whitespace assistant turns unless they carry meaningful `thinking` that the UI should show.

### Optional Fix 3: fold thinking-only assistant rounds

The representative session contains multiple assistant turns whose visible content is only newline/empty string, but whose `thinking` field is meaningful. That may still clutter the UI. A later improvement could fold repeated thinking-only streaming segments into the next non-empty assistant turn.

Pseudocode:

```text
pending_thinking = []
for each delta block:
  if block.kind == reasoning:
    pending_thinking append text
  if block.kind == llm_text:
    text = payload.text
    if blank(text) and pending_thinking not empty:
      keep pending thinking for later
      continue
    emit assistant turn with thinking = join(pending_thinking)
    clear pending_thinking
```

This requires care because the current UI may intentionally show reasoning segments separately.

## What not to fix first

Do not start by rewriting Coinvault persistence. Current evidence shows:

- `turns.db` has `tool_call` and `tool_use` blocks.
- `timeline.db` has `ChatTool*` events.
- converted minitrace JSON has top-level `tool_calls`.

The loss happens between top-level `tool_calls` and per-turn `tool_calls_in_turn`, which is converter normalization.

Do not start by rewriting the React UI. The API returns zero block tool counts for the original archive and non-zero block tool counts for the patched archive. The UI can only render what the API gives it.

## Reproduction commands

Original server:

```bash
/home/manuel/code/wesen/corporate-headquarters/go-minitrace/go-minitrace serve \
  --archive-glob 'k3s-recovery/minitrace-output/active/*/*.minitrace.json' \
  --port 8090
```

Patched server:

```bash
/home/manuel/code/wesen/corporate-headquarters/go-minitrace/go-minitrace serve \
  --archive-glob 'k3s-recovery/minitrace-output-patched/active/*/*.minitrace.json' \
  --port 8091
```

Compare one session:

```bash
./ttmp/2026/05/12/GMINI-0002--debug-missing-tool-calls-in-coinvault-minitrace-transcript-ui/scripts/01_compare_session_turns_minitrace.py \
  --conv-id 8730fef8-2f37-40bb-96e3-73687c55f6ab \
  --turns-db k3s-recovery/clean/coinvault-turns.sqlite \
  --timeline-db k3s-recovery/clean/coinvault-timeline.sqlite \
  --minitrace-json k3s-recovery/minitrace-output/active/2026-05/8730fef8-2f37-40bb-96e3-73687c55f6ab.minitrace.json \
  --api http://127.0.0.1:8090
```

Run patch probe:

```bash
./ttmp/2026/05/12/GMINI-0002--debug-missing-tool-calls-in-coinvault-minitrace-transcript-ui/scripts/02_patch_minitrace_archive_for_ui_probe.py \
  --src k3s-recovery/minitrace-output \
  --dst k3s-recovery/minitrace-output-patched
```

## Next steps

1. Implement the converter fixes in go-minitrace.
2. Add regression tests to `pkg/adapters/turnsdb`.
3. Regenerate archives from `coinvault-turns.sqlite`.
4. Serve the regenerated archives and verify `/api/v2/sessions/{id}/blocks` has non-zero tool calls.
5. Recheck the UI for thinking-only/blank assistant clutter.
6. Decide whether thinking-only streaming segments should be collapsed or displayed in a separate UI affordance.

## Follow-up: why the naive patched UI still shows empty and badly ordered tools

The patch/probe was deliberately minimal: it linked existing top-level tool calls into turns. That proves the API/UI contract depends on `ToolCallsInTurn`, but it does **not** fully fix the conversion.

A second problem appears in the generated minitrace JSON: duplicate top-level tool calls with the same IDs. For session `8730...`, the first six top-level tool calls are successful and have results. The next six have the same IDs but are marked `no tool result received`. The server builds a `map[id]ToolCall`, so the later duplicate pending calls overwrite the earlier successful calls. That is why the patched UI can still show empty/error tool rows.

The likely cause is the converter's LCS delta fingerprint. It includes `block_metadata_json`. In the source DB, repeated `tool_call` blocks across snapshots have identical payloads but different metadata hashes. Repeated `tool_use` blocks have stable metadata. The delta therefore re-emits old `tool_call` blocks without their already-matched `tool_use` results, creating duplicate pending calls.

The ordering problem is separate. The source DB block order is interleaved: assistant reasoning/text, tool call, tool result, assistant reasoning/text, tool call, tool result, and so on. The converter currently creates assistant turns from text-like blocks and then builds tool calls afterward. That means the converter loses the original interleaving. A complete fix should convert delta blocks in a single ordered pass, attaching each tool call to the nearest appropriate assistant turn or introducing a transcript event representation that can preserve tool rows between assistant segments.

Recommended implementation refinement:

1. Make block identity for LCS stable across snapshot metadata churn. For tool calls/results, prefer semantic identity such as `kind + block_id + payload id/name/args/result/error`, not full metadata.
2. Deduplicate top-level tool calls by ID, merging later information carefully rather than overwriting success with pending.
3. Populate `ToolCallsInTurn` while processing blocks in order, not as an afterthought.
4. Consider representing tool calls as ordered transcript events if minitrace's current `ToolCallsInTurn` model cannot express interleaving cleanly.

## GMT-008 implementation validation notes

After implementing the first converter fix set in commit `ce2d48f0e120c79034f6b324362ef34678fe2f1b`, the targeted package test passes:

```bash
go test ./pkg/adapters/turnsdb
```

A real-session smoke check against `/home/manuel/code/gec/2026-03-16--gec-rag/k3s-recovery/clean/coinvault-turns.sqlite` for conversation `8730fef8-2f37-40bb-96e3-73687c55f6ab` produced:

```text
turns 18
tool_calls 6
linked 6
ids 6
json_blank_artifacts 0
tool_outputs_success 6
pending_no_result 0
```

This confirms the implemented fixes address the primary GMT-008 data-shape failures: duplicate pending tool calls, missing `ToolCallsInTurn` links, and JSON-looking blank assistant text.

The remaining limitation is exact interleaving. The representative generated session links all six tool calls to assistant turn 8. That is materially better than missing or pending tool rows, but it does not reconstruct a first-class ordered text/tool/text event stream. Treat exact interleaving as a follow-up model/design question if the UI needs tool rows placed between individual assistant text fragments.
