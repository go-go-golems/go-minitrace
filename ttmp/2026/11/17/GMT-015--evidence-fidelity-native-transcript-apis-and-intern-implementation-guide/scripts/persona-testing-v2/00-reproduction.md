---
Title: Persona harness v2 reproduction
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
LastUpdated: 2026-11-17T18:18:09.462631-05:00
WhatFor: ""
WhenToUse: ""
---

# Persona harness v2

This directory is a new experiment. The original persona-testing scripts and results are immutable. Preparation builds one binary and records its SHA256, Git revision, source diff hash and copied input hashes. Each worker starts in a fresh scratch directory and receives explicit synthetic/real and archive/chronological answer contracts.

## Reproduce

Run from the repository root on macOS with Python 3, Go, Pi, the Lunaroute extension and existing provider authentication. Never print credentials. Set SCRIPT to this directory and REPO to the worktree.

```bash
python3 "$SCRIPT/prepare.py" --repo "$REPO" --bundle /private/tmp/gmt015-persona-v2
python3 "$SCRIPT/run_cohort.py" --bundle /private/tmp/gmt015-persona-v2 --cohort pilot --mode focused --execute
python3 "$SCRIPT/evaluate.py" --bundle /private/tmp/gmt015-persona-v2 --cohort pilot
python3 "$SCRIPT/run_cohort.py" --bundle /private/tmp/gmt015-persona-v2 --cohort matrix --mode matrix --execute
python3 "$SCRIPT/evaluate.py" --bundle /private/tmp/gmt015-persona-v2 --cohort matrix
```

Preparation defaults to the original private nine-source corpus and fixtures at /private/tmp/gmt015-persona. Override --corpus and --fixtures if restoring the private evidence archive. It copies only four fixture inputs; oracle files remain in a separate private directory. Bundles and cohort names must be new. Without --execute, the runner performs sandbox preflight and writes prompts but makes no model calls. Focused mode runs four tasks with one model/high thinking. Matrix mode runs four tasks x two models x low/high requested thinking x two repeats. Defaults: four concurrent workers, 480 seconds/run, 90 seconds for the first partial checkpoint. Requested thinking is not proof of backend reasoning behavior.

## Enforcement and limits

macOS sandbox-exec denies writes to the repository, shared git directory, source/copy corpus, fixtures, pinned binary, public contract and control data. It denies oracle/ticket reads. Before each worker it tests direct writes, child-shell writes, symlink writes, oracle reads and an allowed scratch write. Unsupported OS or failed enforcement stops execution; there is no prompt-only fallback. This is protected-path isolation, not a complete hostile-code sandbox. Network and provider state outside those paths remain available. Worker reports are untrusted evidence; inspect transcripts and independent replays before drawing conclusions.

Each minitrace invocation is requested through record_command.py, which saves argv, cwd, true exit code, timeout, stdout/stderr hashes and stream files. A worker could bypass the wrapper; inspect its Pi transcript. Every worker must write answer.partial.json early; mandatory answers precede optional exploration. The host checks pinned input hashes before and after each run. D commands are replayed with fixed host argv under the same policy, never by executing an answer-provided shell command.

## Grading and evidence

A checks all nine session identities and actual turn/tool counts. B isolates synthetic clipping/order/manifest answers and requires real coverage across three frameworks. C independently checks whole-source hashes, bytes, lines and api-request counts for all raw sources. D replays generated commands against three real archives and verifies actual counts, help and sibling diagnostics. Automated scores do not establish quality of optional prose or full SQL/JS implementation parity; review saved scripts and traces.

Private evidence stays outside Git. Publish compact receipts, grades and review notes; never commit raw transcripts or provider state. Run `python3 -m unittest discover -s "$SCRIPT" -p test_harness.py -v` for no-model enforcement and grading tests.


## Recorded matrix and post-cohort hardening

The recorded matrix used revision 7a7c0c6, bundle `/private/tmp/gmt015-persona-v2-matrix`, cohort `matrix`, and `--workers 8`. Its binary SHA256 is 14363bf439063edaf4006e5cf5ea434f10db5cd1c0a5674dd65350e2a579d01c. The pilot used the separate `/private/tmp/gmt015-persona-v2` bundle. Preserve those policy files and prompts when interpreting results.

After the matrix, an audit found that a worker could list host-analysis and grading filenames. Current preparation additionally denies reads of `control/` and `agent-analysis/`; startup probes now test a real control file, and unit tests prove files and directory listings created after policy construction are denied. This stronger policy was tested locally, not retroactively applied to the completed matrix. Neither version claims complete filesystem confidentiality.

Use `summarize.py` for compact metrics, `analyze_agent_sessions.py --binary PINNED --matrix RUNS --out PRIVATE_OUTPUT` to cross-check persisted Pi transcripts with normalized SQL, `zero_tool_probe.py --help` for separate unscored generated-command edge checks, `collect_artifacts.py` for runnable source collection, and `preserve_evidence.py` for a private archive and public integrity receipt. The summary distinguishes executed checks from planned/unassessed checks when answers are absent.
