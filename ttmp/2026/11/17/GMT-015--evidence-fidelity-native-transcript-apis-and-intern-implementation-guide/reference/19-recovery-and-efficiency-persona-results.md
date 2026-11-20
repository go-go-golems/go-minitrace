---
Title: Recovery and efficiency persona results
Ticket: GMT-015
Status: active
Topics:
    - analysis
    - minitrace
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-11-19T20:04:08.405425-05:00
WhatFor: ""
WhenToUse: ""
---

# Recovery and efficiency persona results

## Results

Both briefs used the same nine archives, DeepSeek and GLM at low thinking, and the Codex judge at medium effort. Every three-turn conversation and judge completed. Recovery checked exact failed tool calls and the first later successful call with the same tool name. Efficiency checked per-session reported turn totals, observed token denominators and all turn counts, including missing usage.

| Case | Analyst | Live exact replays | First successful JS call | Schema lookups | Saves returning paths |
| --- | --- | ---: | ---: | ---: | ---: |
| Recovery | DeepSeek | 3/3 | 11 | 3 | 1 |
| Recovery | GLM | 0/3 | 9 | 3 | 2 |
| Efficiency | DeepSeek | 3/3 | 7 | 4 | 9 |
| Efficiency | GLM | 3/3 | 9 | 5 | 3 |

The corpus produced 26 recovery candidates overall and six in Codex. The efficiency command returned nine sessions overall and three in Codex. These are artifact checks. A later success alone does not establish causal recovery. Recorded token totals only cover turns carrying those fields.

## What the runs exposed

The analysts successfully looked up schema 15 times and got registered command paths on 15 saves. None attempted `sqlite_master`. Five saves returned catalog diagnostics. One unresolved invocation returned command guidance. The lookup was used under both workloads. The file-ledger baseline used a different task, so the counts here should not be treated as a paired estimate of the discovery change.

GLM returned its recovery rows under `{count, rows}`. The normal command renderer emitted that object as a one-element JSON array. The live verifier interpreted the wrapper as a single data row and failed with `'session_id'`. I preserved those three failures, added support for the bounded envelope shape, and replayed the original artifacts with no model calls. All three exact checks matched. The verifier still rejects incomplete envelopes and count mismatches.

The next API authoring question is how to show the static metadata form where the model already writes its code. Returning paths made successful registration visible. Returning diagnostics let agents correct malformed definitions before running. The traces still show repeated guesses at inline handler metadata even with the current examples.

[Compact receipts](../sources/recovery-efficiency-persona-results.json) · [Recovery viewer](http://127.0.0.1:8801/) · [Efficiency viewer](http://127.0.0.1:8802/)
