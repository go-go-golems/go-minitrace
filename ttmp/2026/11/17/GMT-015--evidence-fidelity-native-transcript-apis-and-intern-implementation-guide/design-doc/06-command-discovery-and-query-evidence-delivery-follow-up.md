---
Title: Command discovery and query evidence delivery follow-up
Ticket: GMT-015
Status: active
Topics:
    - analysis
    - minitrace
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/go-minitrace/cmds/query/commands.go
      Note: Rendered root and grouped JS examples
    - Path: repo://pkg/minitracecmd/parse_javascript.go
      Note: Command scanner corrective guidance
    - Path: repo://ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/persona-live/analyst.ts
      Note: Capture and model preview boundary
ExternalSources: []
Summary: ""
LastUpdated: 2026-11-18T17:38:40.293236-05:00
WhatFor: ""
WhenToUse: ""
---

# Command discovery and query evidence delivery follow-up

## Review

Both analysts ultimately produced the exact ledger, including the Codex-only scope change. They repeated command-registration and routing failures, tried SQLite catalog queries and sometimes received clipped tool output. The fluent query grammar handled the task. The [API friction review](../reference/17-live-persona-api-friction-and-asset-studio-lessons.md) records the baseline and the Asset Studio precedent.

## Tasks

- [x] Show executable root and grouped JS command examples in catalog help. Give a rejected registration its filename, existing scanner reason, expected declaration shape, and help topic.
- [x] Point the SQL policy error and first SQL example to schema help. Show `db.schema()` at the first JS entry point.
- [x] Separate query completeness, tool preview truncation, and process completion. Save full captured output locally with an explicit capture limit and file references. Let the API judge see a bounded, path-free projection of any ledger rows the analyst explicitly cites by tool call ID.
- [x] Run the two historical personas with the updated binary and harness together. Compare observed first successful JS execution, failure categories, and exact ledger replay after the scope change. Use the default Codex judge.

## Design

Keep static `__verb__` metadata and the current query envelope. Catalog help should paste into a scratch repository directly. On the tool boundary, report process failure using a separate field and capture completeness using a boolean. Inspect SQL envelope completeness before reducing the output to its preview. A JS result can have any output shape, so query completeness is unknown unless an envelope explicitly provides it. Preserve captured UTF-8 output in each private run directory. Host verification stays independent of model previews.

The rerun tests the combined change. It uses the same historical inputs and model settings and records each analyst's exact replay. One run per model can show concrete outcomes and remaining friction. It cannot isolate individual edits or establish a reliable improvement rate.

## Persona rerun results

Both final-build personas passed all three independent checks for 34, 34 and one target. Both Codex judges completed at medium effort. Nine archive hashes matched the baseline and both completed final runs used the same CLI binary.

| Persona | First successful JS call, baseline → final | First successful ledger call, baseline → final | Misleading flag messages, baseline → final | Actionable unresolved-command messages, final |
| --- | ---: | ---: | ---: | ---: |
| DeepSeek 4.1 Flash low | 13 → 7 | 34 → 21 | 4 → 0 | 2 |
| GLM-5.3 Flash low | 28 → 16 | 28 → 24 | 15 → 0 | 6 |

Ordinals count all Pi tool calls from the start of the run. The revised wording eliminated the misleading flag message in both completed final runs. Command discovery still caused retries. Each analyst still attempted `sqlite_master` once. Eleven shortened previews had complete local captures across those two runs.

The initial combined rerun passed both ledgers, then a terminal smoke exposed the unfenced help formatting. The final cohort used code fences verified in the actual rendered help. GLM's first final-build attempt failed when its upstream stream interrupted the opening turn. The table uses its fresh retry on the same pinned binary. All attempts remain in the receipt. The changes ran together, so the observed ordinal differences cannot attribute effect to one edit.

[Comparison receipts](../sources/live-persona-discovery-follow-up.json) · [DeepSeek viewer](http://127.0.0.1:8799/) · [GLM retry viewer](http://127.0.0.1:8800/)
