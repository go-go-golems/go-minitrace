---
Title: Analyze persona Pi sessions with go-minitrace itself
Ticket: GMT-015
Status: active
DocType: reference
Topics: [analysis, minitrace]
Summary: Independent conversion and SQL accounting of finished persona runs, with explicit native/event/normalized count comparisons.
---

# Method and reproducible command

The persona runner emits event counts and usage in its receipts. This separate check analyzes the persisted Pi session files through the updated go-minitrace conversion and normalized SQLite query path. It does not infer success from the persona's answer, invoke models, execute historical commands, or modify the runner/matrix/native stores.

From the implementation worktree root:

```sh
python3 ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/persona-testing/analyze_agent_sessions.py
```

Inputs default to `/private/tmp/gmt015-persona/matrix`. A receipt is finished only when `elapsed_seconds` exists; timeout and nonzero-exit runs remain eligible because failed runs are evidence too. The command copies only finished persisted sessions into `/private/tmp/gmt015-persona/agent-analysis/raw/<run>/`, verifies original bytes remained unchanged during copying, converts the explicit Pi source list with `go run ./cmd/go-minitrace convert pi`, validates the archive, and runs saved `session-analysis.sql`. Native files are never changed. Existing identical snapshots are reusable; a previously finished source that changes bytes causes an explicit failure rather than silent replacement.

Raw snapshots, normalized archives, source lists and full command output stay private under `agent-analysis`. The compact committed receipt is `sources/persona-session-analysis.json`; it contains hashes, paths, metadata counts, SQL results and comparison deltas, not transcript bodies. The final run covers all 32 finished sessions: all 32 normalize and validate, with zero tool-count mismatches, zero compared failure/usage mismatches and no warnings. Earlier checkpoints below are historical snapshots.

# Three distinct evidence units

- Runner receipt `tool_calls` counts streamed tool-start events.
- Native persisted sessions count `assistant.content[type=toolCall]` records. Tool-result errors are counted separately from `toolResult.isError`.
- Normalized SQL counts `tool_calls` rows; failures require `success=0`, and `success IS NULL` remains unknown.

The receipt compares these units but does not assert they must always be equal. Missing persistence, interrupted calls, adapter behavior or duplicate event handling could produce meaningful differences. A run with no persisted sessions is warned about rather than silently omitted.

Usage is compared field by field: input, output, cacheRead, cacheWrite and reasoning. Native sums come only from assistant message usage; SQL sums recorded normalized turn fields; runner sums come from its event receipt. The analysis does not sum `totalTokens` together with those fields, derive marginal cost, or treat reasoning as automatically disjoint from output. Matching sums demonstrate consistency of these representations, not independent verification of provider billing. Missing SQL usage remains null with an explicit reporting-session count.

# Initial execution results

The first snapshot found **four finished deepseek-4.1-flash/high/r1 runs**, one for each task A–D. All four converted and passed archive validation. There were no source-ID warnings or tool-count discrepancies:

| Task | Native assistant calls | Normalized tool rows | Runner tool-start events | Native errors / normalized failures / runner errors |
| --- | ---: | ---: | ---: | --- |
| A | 54 | 54 | 54 | 1 / 1 / 1 |
| B | 50 | 50 | 50 | 1 / 1 / 1 |
| C | 80 | 80 | 80 | 0 / 0 / 0 |
| D | 53 | 53 | 53 | 0 / 0 / 0 |

All four had zero unknown normalized outcomes. For each run, all five compared usage fields matched across native assistant records, normalized SQL and runner receipts. The four runs were predominantly bash-driven; A also wrote and edited, B wrote, C wrote and edited, and D used only bash. These are workflow observations, not evidence that their answers were correct or that every CLI call behaved as intended. The matrix's separately graded answers and forensic traces own those conclusions.

This table preserves the initial audit checkpoint. The generated JSON contains the latest rerun counts and should be consulted for final coverage. No source-code fix was necessary to obtain these initial matching counts.
