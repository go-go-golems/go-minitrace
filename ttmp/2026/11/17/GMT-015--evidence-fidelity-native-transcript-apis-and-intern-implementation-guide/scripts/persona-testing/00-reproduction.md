---
Title: Persona experiment reproduction
Ticket: GMT-015
Status: active
DocType: reference
Topics: [analysis, minitrace]
---

# Persona experiment reproduction

Run from the `feat-evidence-fidelity` worktree root. Scripts require Python 3, Go, Pi and the configured Lunaroute provider extension. No credentials are stored here. The original implementation under test is `0996822`; later commits in this worktree add testing artifacts only. See `sources/persona-tested-source.json` for source hashes.

## Prepare private inputs

```bash
python3 <ticket>/scripts/persona-testing/select_corpus.py --help
python3 <ticket>/scripts/persona-testing/fixtures.py --help
python3 <ticket>/scripts/persona-testing/run_matrix.py --help
```

The selection script contains the explicit nine-source historical selection for this machine. Inspect its help for roots and rerun behavior; do not silently substitute missing transcripts and call it the same study. Native source hashes and independent normalized array counts are saved in `sources/persona-corpus-manifest.json`.

Fixture generation creates a private oracle and inputs. Copy only `archive.json`, `archive-copy.json`, `archive-conflict.json`, and `native.jsonl` to the model-visible input directory. Do not supply `oracle.json` or the evaluator scripts to tested models. This is prompt scoping plus separate directories, not an OS sandbox; trace auditing checks for violations.

## Run the frozen matrix

The actual command for this study was:

```bash
python3 <ticket>/scripts/persona-testing/run_matrix.py \
  --repo "$PWD" \
  --fixtures /private/tmp/gmt015-persona/inputs \
  --output /private/tmp/gmt015-persona/matrix \
  --workers 4 --timeout 480
```

The default matrix is four paired task/persona groups, two model IDs, two requested thinking levels and two repetitions. The script rejects an already-existing run directory instead of overwriting attempts. It is not an automatic resume engine. For a selected rerun, use a new output root plus `--groups`, `--models`, `--thinking`, and `--repetitions`, and label the result as a separate attempt.

Each run saves the exact prompt, command, revision, timestamps, JSON event stream, stderr, native Pi session, and receipt. Model-written `answer.json`, `report.md`, JS and SQL live beside those receipts. Do not score a run until its receipt has `elapsed_seconds`; receipts exist before model execution as well.

## Grade and audit

```bash
python3 <ticket>/scripts/persona-testing/summarize_results.py \
  --output <ticket>/sources/persona-matrix-results.json
python3 <ticket>/scripts/persona-testing/audit_traces.py \
  --matrix /private/tmp/gmt015-persona/matrix \
  --output /private/tmp/gmt015-persona/trace-audit.json
```

The independent grader checks requested result fields, not broad prose competence. A-grade facts come from verified archive arrays. B-grade facts come from independent synthetic expectations. C-grade facts include exact raw-file hashes, sizes and line counts. D needs separately reviewed and replayed commands; self-reported booleans are excluded from correctness counts.

`tool_error_events` counts Pi tool transport errors. Shell commands that swallow a subprocess error can still yield a successful tool event, so this metric is not a complete error count. Feature-text matches in `audit_traces.py` mean attempts, not verified success. Report timeouts, malformed/missing answers, semantic errors and unsupported conclusions explicitly.

Model cost fields are not billing evidence: the installed Lunaroute extension registers zero cost rates. Usage totals count repeated cached context across calls; they are not unique transcript bytes or peak context length. Low/high are distinct advertised local mappings, not a claim about backend implementation.

## Publication boundary

Keep raw native transcripts, native agent sessions, streamed model events and broad generated outputs private. Commit the selection/provenance metadata, scripts, focused excerpts, independent grades and reviewed source artifacts. No provider API keys or original transcript bodies belong in the ticket.
