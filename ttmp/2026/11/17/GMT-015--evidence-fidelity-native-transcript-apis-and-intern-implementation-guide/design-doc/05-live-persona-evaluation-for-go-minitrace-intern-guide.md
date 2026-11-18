---
Title: Live Persona Evaluation for Go Minitrace Intern Guide
Ticket: GMT-015
Status: active
Topics:
    - analysis
    - minitrace
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/persona-live/analyst.ts
      Note: Defines the controlled analyst tools
    - Path: repo://ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/persona-live/run.py
      Note: Coordinates the stakeholder, analyst, verifier and judge
    - Path: repo://ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/persona-live/serve.py
      Note: Streams the local event timeline
ExternalSources: []
Summary: Intern guide to the stakeholder, analyst, deterministic replay, judge and local viewer.
LastUpdated: 2026-11-18T16:43:01.625327-05:00
WhatFor: ""
WhenToUse: ""
---

# Live Persona Evaluation for Go Minitrace Intern Guide

## 1. What this system tests

go-minitrace turns recorded Claude Code, Codex, Pi and other coding-agent sessions into a common archive format. Its query engine materializes the archives into SQLite. An analyst can use SQL directly or write a reusable JavaScript command that reads the same tables. The evaluation measures whether another coding agent can use those capabilities to answer a real question, support its answer with records, and deliver a script a colleague can rerun.

The first scenario is a file-change handoff. A maintainer asks which sessions recorded write-like file targets and how strongly those records support changes. An analyst writes a ledger and cites examples. A stakeholder model asks one adaptive follow-up. A third turn narrows the archive selection to Codex. The host tests the submitted command against an independent SQL baseline on every turn. A judge reviews the conversation and the test results after the three turns.

This guide uses the Playback Asset Studio separation between an adaptive persona, the agent being evaluated, executable checks and a qualitative judge. Our scenario uses a CLI and a set of archives. The evidence a stakeholder sees is an answer with a small ledger preview and replay error when present.

## 2. Know the three evidence levels

A transcript is a record of an agent session. `tool_calls` can tell us that a tool was invoked and may tell us what outcome was recorded. It takes additional evidence to attribute a file effect to the invocation. The `files` table contains individual targets and their evidence status. One tool call may affect multiple files, and two targets may mention the same path. Keep the target ordinal.

| Record or field | Use it for |
| --- | --- |
| `sessions.session_id`, `sessions.agent_framework` | Join a target to a session and describe the selected corpus. |
| `tool_calls.tool_call_id`, `tool_calls.emitting_turn_index` | Locate the producing tool record and its turn when available. |
| `tool_calls.outcome_status` | Describe the recorded operation outcome. |
| `files.session_id`, `files.tool_call_id`, `files.target_ordinal` | Identify a particular target. |
| `files.path`, `files.operation_type` | State the recorded path and operation. |
| `files.evidence_kind`, `files.evidence_status` | Explain the provenance and strength of the target evidence. |
| `files.success` | Read target-specific success where available. A null leaves it unknown. |

An attempted target says what an operation aimed to change. A legacy scalar report records a path supplied by an adapter. A confirmed target has explicit native effect evidence. The ledger can contain all three where the contract allows them, so its row count means recorded targets. Repository state at a later point requires independent evidence.

Go to [file evidence types](../../../../../../pkg/minitrace/file_evidence.go), [normalization schema](../../../../../../pkg/minitracedb/schema.go), and the [file operations query](../../../../../../pkg/minitracedb/presets/files/file-operations.sql) when reviewing the exact field definitions.

## 3. Mental model and module boundaries

```text
 frozen binary and normalized archives
               |
scenario  -->  host  -->  stakeholder Pi (follow-up)
               |                 |
               |                 v
               +----------> analyst Pi session
               |                 |
               |          save JS and SQL
               |                 |
               |          sandboxed CLI query
               |                 |
               +---------> replay candidate ledger
               |          compare with host SQL
               |
               +---------> judge Pi (conversation + replay facts)
               |
               +---------> per-run JSONL timeline --> local viewer
```

The host schedules the conversation and owns dataset selection. It is the only component that reads the expected answers. The analyst sees a scenario contract and a small tool surface. The stakeholder sees the delivered conversation for its follow-up. The judge sees executable check results as well. The viewer reads projected events and has no authority to run scripts.

Keep scenario content in [scenario.json](../scripts/persona-live/scenario.json), the experiment coordinator in [run.py](../scripts/persona-live/run.py), the Pi adapter in [analyst.ts](../scripts/persona-live/analyst.ts), independent SQL in [oracle.sql](../scripts/persona-live/oracle.sql), and the viewer in [serve.py](../scripts/persona-live/serve.py) and [viewer.html](../scripts/persona-live/viewer.html). Each can be read and exercised on its own. Extending to a second task only earns another verifier when it introduces a different evidence contract.

## 4. Follow one run

Each model run gets a separate directory and a Pi session. The binary and normalized inputs are copied into an immutable selection for the batch. The manifest stores archive and binary hashes so a repeat can detect a change in input. The model and thinking setting are experimental inputs. A reviewer model plays both stakeholder and judge in separate, tool-free calls. Analyst turns retain the ongoing Pi session and its scripts.

```text
for scope in [all archives, all archives, Codex archives]:
    emit stakeholder request with scope
    continue analyst session with request
    record submitted command hash
    execute submitted command with the host's archive glob
    independently query expected rows for that glob
    compare target row multisets
    emit counts, differences and bounded observed preview
    pick the next request
        turn 1 -> ask stakeholder for one evidence challenge
        turn 2 -> use the fixed archive-scope change
judge the delivered conversation and host checks
```

The host's SQL runs on the same normalized database implementation through a different query path. This tests the candidate script's ledger calculation and handling of the scope change. A materialization defect could affect both paths. A fixture with authored expectations tests the comparison further upstream. Use native records or repository evidence for a broader audit.

## 5. Analyst API and reproducible artifact

| Pi tool | Input | Result |
| --- | --- | --- |
| `save_artifact` | `{name, content}` for a flat JS, SQL or Markdown file | Save a workspace artifact. |
| `read_artifact` | `{name}` | Read a saved artifact. |
| `minitrace` | `{kind, text?, path?, scope?}` | Run help, a read-only SQL query or a saved JS verb. Returns exit, stdout, stderr and deliveryTruncated. |

Start with the `minitrace-schema` and `js-api-reference` help topics. Use a command path array for JS, for example `["ledger"]` or `["group", "verb"]`. Use `kind: "help"` and `path: []` to list commands in the same repository the analyst uses. For SQL, inspect the query envelope's `complete` field and diagnostics. The tool's `deliveryTruncated` flag denotes a shortened transport preview. These are independent checks. A complete query can still produce a clipped preview, and an empty result needs a completeness check before it becomes evidence of zero matches.

The analyst saves a file named `ledger.js` which registers a root verb named `ledger`. The root command syntax follows the project convention where identical verb and filename names collapse to one command. The host selects the archives on the command line.

```sh
go-minitrace query commands --query-repository ./artifacts ledger \
  --archive-glob '/path/to/codex-*.minitrace.json' --output json
```

Within the command, obtain the input from `mt.sources().runtimeArchives().build()`. Open the database through `mt.db().sources(selection).build()`, construct a fluent plan or tagged SQL, then call `db.execute(plan)`. Check its result and close the database in `finally`. The analyst chooses what to explore, what evidence to discuss and how to implement the command. The host requires the row identity fields `session_id`, `tool_call_id`, `target_ordinal`, `path` and `evidence_status`.

See the [JavaScript reference](../../../../../../pkg/doc/js-api-reference.md), the [evidence workflow help](../../../../../../pkg/doc/evidence-workflows.md), or the [fixture command](../scripts/persona-live/fixture-ledger.js).

## 6. Check the checker on real archives

The fixture initially used lower-case edit and write verbs. The first historical pilot exposed a contract error. Its expected all-archives result was empty even though a submitted ledger emitted 34 rows. The normalized adapters emit file operation types `NEW`, `MODIFY` and `DELETE`. The next implementation checkpoint must use those values in the scenario, oracle and authored fixture. Recompute the historical baseline before interpreting results. A judge comment in the initial run cannot repair the evaluator's definition after the fact.

For this scenario, keep the operation selection explicit and exclude Codex legacy scalar fallbacks. Preserve legacy reported evidence for other frameworks and explain what it supports in prose. Check the adapter evidence on representative cases before treating the resulting counts as a benchmark. The approved nine historical archives are a small, fixed exercise corpus. They cannot establish framework-wide differences or model rankings.

For each scope, compare the multiset of five-field keys. Equal lengths alone miss substitutions. A multiset also exposes duplicate rows introduced by a mistaken join. Record missing and unexpected counts with limited key samples. A correct script can still accompany a weak conclusion. A qualitative judge assesses grounding, usefulness, uncertainty and efficiency against the transcript evidence it received.

## 7. Read the timeline and diagnose a run

[The runner README](../scripts/persona-live/README.md) has start, repeat and replay commands. With the viewer running, select a model in the Run menu. Each card carries a timestamp, role and event type. Open Details to read a script submission, CLI output, host row differences or usage. Filter to Host to review the replay results and to Judge to read the narrative assessment.

| Symptom | First file or check |
| --- | --- |
| Request never produces a reply | Inspect the phase's Pi JSONL and stderr. Check authentication, provider exit and the turn timeout. |
| CLI prints unknown archive flags | Check whether the JS command is registered and whether the filename and root verb match. |
| SQL results or tool previews are incomplete | Inspect both query completeness and delivery truncation before inferring a finding. |
| Ledger has equal row count and replay fails | Compare key samples and target ordinals. |
| Judge sees too little evidence | Inspect the judge prompt and the delivered preview before revising the agent prompt. |
| Browser looks stalled | Check the local event endpoint and the last timestamp in the per-run log. |

The viewer polls a loopback HTTP endpoint and advances by byte offset. The server only publishes complete newline-terminated events. A half-written line remains available for the next poll. Raw Pi output, expected rows and saved scripts stay in the local batch directory alongside the projected timeline. Generated queries run out of process with Seatbelt restrictions, a pinned binary and staged inputs. Pi makes the provider requests. Run historical archives only where disclosure to the selected provider is authorized.

## 8. Repeat and extend

Start with the scripted smoke test. It runs the actual CLI, host comparison and viewer event flow with a known answer and a failed file attempt. Then test a positive, missing and duplicated ledger key. Verify the changed scope independently. Run both models on matching inputs and preserve the complete trace and scripts.

Use [replay.py](../scripts/persona-live/replay.py) to repeat an opening analyst turn with the original recorded prompt and input hashes, or with one extra piece of guidance. This is a new model response to the same opening task. A complete rerun is still needed to see whether an intervention survives the stakeholder follow-up. Repeat before attributing a change to guidance.

When adding the recovery, efficiency or handoff briefs, follow the [existing brief set](../reference/16-open-ended-persona-briefs-for-real-transcript-analysis.md). Give the analyst room to select its investigative approach and define a small machine-checkable claim. Put recurring script or schema mistakes in help examples or API diagnostics where an agent encounters them. Keep the local transcript viewer and deterministic checks independent of model choice.
