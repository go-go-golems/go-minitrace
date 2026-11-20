---
Title: Fluent filter composition and in-memory SQLite sandbox results
Ticket: GMT-015
Status: active
Topics:
    - analysis
    - minitrace
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/minitracedb/open.go
      Note: Memory connection setup
    - Path: repo://ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/persona-live/check_composition.py
      Note: Controlled helper substitution and replay
    - Path: repo://ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/persona-live/composition.sql
      Note: Independent oracle
ExternalSources: []
Summary: Behavioral helper substitution and exact row checks on real transcripts, plus paired SQLite scratch-storage controls.
LastUpdated: 2026-11-19T20:33:17.377196-05:00
WhatFor: ""
WhenToUse: ""
---

# Fluent filter composition and in-memory SQLite sandbox results

## Results

Both GLM-5.3 Flash and DeepSeek 4.1 Flash completed the three-turn task at low and high thinking. All four registered the ledger on the first save. Each run returned nine correct session rows on the first two turns and three correct Claude Code rows after the scope change. GPT-6.2 Sol at medium effort delivered a verdict on all four runs.

| Check on submitted commands | Passed | Evidence |
| --- | ---: | --- |
| Exact agreement with independent SQL | 12/12 | Every required field on all three turns of all four runs |
| Local change to the shared filter swaps both output counts | 12/12 | Eight of nine sessions distinguish the two values at full scope, and all three distinguish them at narrowed scope |
| Fluent table sources through execution | 12/12 | All submitted turns use the fluent builder and execute a plan. No submitted turn uses a tagged SQL template. |

The ordinary function was enough to express this operation. One submission extended it to handle a null argument for an additional unknown-outcome count. Other analysts filtered the tool table before or after correlating it with the session. They obtained the same required results. This provides direct evidence for local reusable filters and branch composition. Fragment registration, parameter widgets and other operators would require separate exercises. This was a directed task with a starter. The earlier free-choice runs all selected tagged SQL.

There was exploratory friction outside the ledger checks. Both GLM runs submitted SQL the single-statement interface rejected. High-effort DeepSeek attempted an unresolved exploratory command. These events did not affect the delivered ledger. [Per-run receipts](../sources/fluent-helper-persona-results.json) · [Run viewer](http://127.0.0.1:8806/)

## What the test asks

The preceding sixteen conversations established that analysts could deliver working JS commands using tagged SQL. This follow-up hands each analyst a short fluent query that counts all tool calls per session. The stakeholder wants to compare known successes with known failures, while keeping sessions with zero calls and accounting for unknown outcomes.

The analyst must derive two filtered branches using one named helper and pass the resulting plan to `db.execute`. An ordinary JS helper is sufficient. A typical implementation is:

```js
function withSuccess(query, value) {
  return query.where(tool => tool.success.eq(value));
}

const scoped = tools.where(tool => tool.session_id.eq(session.session_id));
return {
  session_id: session.session_id,
  calls: scoped.count(),
  successful: withSuccess(scoped, 1).count(),
  failed: withSuccess(scoped, 0).count(),
};
```

This excerpt belongs inside the session projection. The outer query starts from sessions. The inner count refers to the current session. Each branch keeps its own predicate and the original total includes missing outcomes. SQL remains available to the analyst for exploratory questions.

## How we distinguish composition from a decorative helper

1. Execute the original submitted ledger and compare its four required columns with the independent SQL oracle as an unordered multiset. Do this after the opening request, an adaptive follow-up and the fixed change to Claude Code archives.
2. Copy that turn's exact `ledger.js` into a fresh command directory. Insert an exchange of the helper’s numeric 0 and 1 arguments before its body. Preserve all other arguments, including null.
3. Execute the copy against the same scope under the same Seatbelt rules. Require exact agreement with the oracle after exchanging only its success and failure columns. Require at least one asymmetric session. A helper that never affects the executed plan would fail this check.
4. Inspect the source to distinguish table-backed fluent queries from embedded SQL. The mutation tests causal use of this one helper. It does not check the full JS grammar or the analyst's prose.

## SQLite scratch storage

The prior reproduction used a grouped read of the normalized `tool_calls` table. Merely asking whether its `result` column is null failed under macOS Seatbelt with `disk I/O error: operation not permitted`. Opening the main database in memory did not set SQLite's separate temporary-storage policy. The in-memory opener now uses a driver connection hook to set `PRAGMA temp_store=MEMORY` for each physical connection. Foreign-key enforcement is set in the DSN. File-backed openers retain their configured storage behavior.

| Aggregate predicate | Original under Seatbelt | Updated under Seatbelt | Ordinary CLI, both versions |
| --- | --- | --- | --- |
| `success IS NULL` | 27 groups | 27 groups | 27 groups |
| `result IS NULL` | Disk I/O denied | 27 groups | 27 groups |
| `COALESCE(result,'')=''` | Disk I/O denied | 27 groups | 27 groups |

All six candidate outputs were complete. Each matched the original ordinary-CLI output for that predicate by hash over canonicalized rows. This fixes the recorded failure without granting file writes. A large in-memory grouping or sort may consume more RAM. The controls did not measure peak memory or trace a specific denied file. [Paired receipts](../sources/grouped-memory-sandbox-controls.json)
