---
Title: Four-task persona matrix and command registration results
Ticket: GMT-015
Status: active
Topics:
    - analysis
    - minitrace
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/minitracecmd/parse_javascript.go
      Note: Runnable scanner correction under test
    - Path: repo://ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/persona-live/matrix.py
      Note: Crossed task and effort launcher
    - Path: repo://ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/persona-live/matrix_results.py
      Note: Input and setting checks for the published receipts
ExternalSources: []
Summary: Exact SQL replay, command registration, and observed analyst effort across two models and four tasks.
LastUpdated: 2026-11-19T20:15:53.880557-05:00
WhatFor: ""
WhenToUse: ""
---

# Four-task persona matrix and command registration results

## Results

All 16 completed conversations registered their root ledger command on the first save. All 48 submitted ledgers matched the independent SQL row for row. Each conversation received a GPT-6.2 Sol medium judgment. The same nine archives and CLI binary were verified by hash, and the recorded Pi model and effort matched the intended profile in every completed cell.

| Task | Exact checks | Rows at opening / follow-up / narrowed scope | First successful ledger call, GLM low / high | DeepSeek low / high |
| --- | ---: | --- | --- | --- |
| File-change attribution | 12/12 | 34 / 34 / 1 | 11 / 21 | 15 / 19 |
| Recovery | 12/12 | 26 / 26 / 6 | 6 / 6 | 16 / 10 |
| Cost | 12/12 | 9 / 9 / 3 | 6 / 6 | 7 / 8 |
| Outcome coverage | 12/12 | 27 / 27 / 11 | 5 / 7 | 6 / 7 |

Call positions count all analyst tools from the opening. Exploration before submission can be valuable. Higher effort reached the ledger later on file changes for both models and earlier on recovery for DeepSeek. Row correctness was identical across effort levels. This single trial per cell supports no general ranking of the models or thinking levels.

On the matched low-effort cost task, both earlier analysts needed to correct their first ledger save. Both new analysts registered it on their first save. GLM’s first successful ledger invocation moved from call 9 to 6. DeepSeek's stayed at 7. One scratch save elsewhere in the completed matrix still elicited a catalog diagnostic.

All 16 opening ledgers used tagged SQL inside the JS command. Only one contained `.from(...)` calls. Inspection of that file showed two unused table builders and a tagged SQL plan passed to `db.execute`. These results support handing off JS command authoring, archive selection, DB execution and repeated scope changes. A direct test of fluent query composition would ask an analyst to extend an existing builder with a reusable filter or fragment, then check both the returned rows and the submitted implementation.

Eleven original conversations completed. Five high-effort conversations hit a closed console pipe during a server restart. Their five completed checks had passed before interruption. Fresh runs filled those five cells. The 48 checks above count each completed conversation once. The JSON retains the partial attempts separately. [Machine-readable receipts](../sources/four-task-persona-matrix.json) · [Interactive run viewer](http://127.0.0.1:8805/)

## What this experiment measures

Can an analyst write a discoverable JS command on the first attempt, use it to answer a stakeholder, and rerun it when the stakeholder changes the archive selection? Saving a runnable command and calculating the right answer are distinct operations. We measure the former from the catalog receipt returned to Pi. We measure the latter by executing the saved file in isolation and comparing the multiset of projected rows with a separately authored SQL query.

Each analyst receives the same tools and is free to write exploratory JS or SQL. The host checks the opening submission, asks a GLM stakeholder for one adaptive follow-up and checks the next submission, then narrows the scope and checks a third time. The judge receives selected tool signals and host receipts. It assesses developer experience. Its verdict does not independently verify every statement in the analyst’s prose.

| Stakeholder and task | Evidence the ledger must preserve | Final selection |
| --- | --- | --- |
| Playback maintainer investigating file changes | Every eligible write-like target, its ordinal and recorded evidence status | Codex |
| Reliability reviewer investigating recovery | Every failed call and the first qualifying later success for the same tool, including absent matches | Codex |
| Maintainer comparing cost | Turn-level sums and their non-null denominators, including sessions with missing usage | Codex |
| Observability reviewer prioritizing gaps | Per-session, per-tool counts of calls, unknown success, missing exit code and marked truncation | Pi |

GLM-5.3 Flash and DeepSeek 4.1 Flash each received every task once at low and once at high thinking through Pi on Lunaroute. These are four model/effort profiles sharing the analyst role. The stakeholder always used GLM-5.3 Flash at low effort. The independent judge used GPT-6.2 Sol at medium effort. Each task uses the same frozen nine normalized archives, three each from Claude Code, Codex and Pi. Repeated observations within a conversation share history.

## Interpreting the comparison

The historical comparison is limited to the earlier efficiency pair. That pair had the same indexed corpus, task, SQL oracle, analyst models and low effort. The new binary illustrates a complete static verb declaration in the scanner error and both authoring help topics. The analyst prompt also now uses quoted names. Differences reflect the whole authoring environment and stochastic model behavior.

The first two new low-effort batches had a 240-second limit per model turn. The other six batches allowed 360 seconds. Concurrent provider runs, adaptive follow-ups and one observation per cell limit timing comparisons. A Pi session records the effort requested from its provider. That record does not measure how much internal computation the provider performed.

## A reproducible sandbox failure during exploration

The low-effort DeepSeek coverage run hit a disk I/O denial while examining tool payloads. I reduced its query to a grouped count. All six invocations used the same frozen binary, archives, working directory and `--cache-mode memory`. Each row below changes only the aggregate predicate. The two columns compare the saved macOS Seatbelt policy with the ordinary local process.

| Predicate inside SUM | With Seatbelt | Ordinary CLI |
| --- | --- | --- |
| `success IS NULL` | 27 rows, complete | 27 rows, complete |
| `result IS NULL` | Disk I/O denied | 27 rows, complete |
| `COALESCE(result,'')=''` | Disk I/O denied | 27 rows, complete |

The query groups by session ID and tool name. Even the null check over the payload column triggers the failure on this corpus. The harness's historic signal labels any “operation not permitted” message as a raw-source access attempt. Here the SQL reads only a normalized table. SQLite temporary storage is a hypothesis. No filesystem trace identifies the failed operation yet. The script prints row counts and diagnostics so a colleague can investigate without sharing payloads. [Local reproduction script](../scripts/persona-live/recheck_grouped_io.py)

The [fluent filter and SQLite follow-up](21-fluent-filter-composition-and-in-memory-sqlite-sandbox-results.md) tests the composition gap and verifies the sandbox fix against the original controls.
