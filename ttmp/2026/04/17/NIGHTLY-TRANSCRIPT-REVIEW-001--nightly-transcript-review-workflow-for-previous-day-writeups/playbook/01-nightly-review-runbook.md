---
Title: Nightly review runbook
Ticket: NIGHTLY-TRANSCRIPT-REVIEW-001
Status: active
Topics:
    - transcript-analysis
    - minitrace
    - documentation
    - go-minitrace
    - codex
    - analysis
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/minitracecmd/render.go
      Note: Renderer path that reuses clay sqlDate/sqlDateTime helpers
    - Path: pkg/minitracecmd/render_helpers.go
      Note: Local safe escaping overrides for SQL string helpers
    - Path: pkg/minitracecmd/core/nightly/session-inventory.sql
      Note: Embedded nightly session inventory command
    - Path: pkg/minitracecmd/core/nightly/workspace-summary.sql
      Note: Embedded nightly workspace summary command
    - Path: ttmp/2026/04/17/NIGHTLY-TRANSCRIPT-REVIEW-001--nightly-transcript-review-workflow-for-previous-day-writeups/scripts/00-nightly-review.sh
      Note: Nightly orchestration entrypoint
ExternalSources: []
Summary: |
    Repeatable operational steps for generating a previous-day transcript review from Pi and Codex session stores.
LastUpdated: 2026-04-17T12:39:47-04:00
WhatFor: |
    Provide a command sequence for producing the nightly review bundle and report, with clear failure modes and exit criteria.
WhenToUse: Use this when running the nightly transcript review manually or when debugging the ticket-local workflow scripts.
---


# Nightly review runbook

## Purpose

Generate a readable writeup of the previous day’s transcript work by converting the relevant Pi and Codex sessions into minitrace, then running a small bundle of reusable structured queries.

## Environment assumptions

- `go-minitrace` is built and available at the repository root, or available via `GO_MINITRACE_BIN`.
- If you need a fresh rebuild, be aware that the current repo still has a DuckDB linker conflict between static libraries; package-level renderer tests can still validate the query-command templating changes.
- `jq` is installed.
- `python3` is installed.
- The machine can read `~/.pi/agent/sessions` and `~/.codex`.
- The working directory is writable so the review bundle can be staged under `/tmp` or another scratch path.

## Commands

Run the nightly review for a specific day:

```bash
NIGHTLY_TRANSCRIPT_REVIEW_WORKDIR=/tmp/nightly-review-run \
  ttmp/2026/04/17/NIGHTLY-TRANSCRIPT-REVIEW-001--nightly-transcript-review-workflow-for-previous-day-writeups/scripts/00-nightly-review.sh \
  2026-04-16
```

The script will produce a day-scoped bundle like:

- `nightly-review.md`
- `report/session-inventory.json`
- `report/workspace-summary.json`
- `report/tool-breakdown.json`
- `report/followup-candidates.json`
- `report/annotation-summary.json`
- `pi-sources.txt`
- `codex-sources.txt`
- `nightly-review.log`

## Expected workflow shape

1. Discover Pi and Codex sources for the target day.
2. Convert the matching JSONL files into a temporary minitrace archive.
3. Run the structured commands from the embedded `nightly` subverb.
4. Render the markdown report.
5. Review the follow-up candidates.
6. Add annotations if anything needs to be revisited in a later window.

## Exit criteria

The run is successful when:

- the markdown report exists,
- the workspace summary and session inventory are both populated,
- follow-up candidates are listed for any obviously complex sessions,
- any annotations that already exist are visible in the annotation summary,
- the report clearly states when there are no Codex sessions for the day.

## Common failure modes

### `go-minitrace` cannot be found

Set `GO_MINITRACE_BIN` to the built binary or run from a checkout that contains `go-minitrace` at the repo root.

### The report fails on a date parameter

The structured commands in this ticket use a date-typed `day` parameter and the renderer now relies on `sqlDate`. If the query command errors about `sqlDate` or the date value looks wrong, check that the query catalog still declares `day: date` and that the renderer is using the clay-backed `pkg/minitracecmd/render.go` path rather than an older binary.

### Codex discovery returns nothing

That can be correct. The workflow should still produce a valid Pi-only report.

### The report looks too flat

Use the workspace summary first, then drill into the follow-up candidates. If needed, annotate the session or tool call and rerun the annotation summary.

## Review order for a human

1. `nightly-review.md`
2. `workspace-summary.json`
3. `followup-candidates.json`
4. `session-inventory.json`
5. `annotation-summary.json`

That order lets you understand the story first, then inspect the raw data that supports it.
