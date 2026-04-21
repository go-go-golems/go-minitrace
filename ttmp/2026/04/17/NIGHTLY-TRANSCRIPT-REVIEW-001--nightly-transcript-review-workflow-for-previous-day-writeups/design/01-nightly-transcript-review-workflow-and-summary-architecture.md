---
Title: Nightly transcript review workflow and summary architecture
Ticket: NIGHTLY-TRANSCRIPT-REVIEW-001
Status: active
Topics:
    - transcript-analysis
    - minitrace
    - documentation
    - go-minitrace
    - codex
    - analysis
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: >
  Design for a resumable nightly review workflow that turns the previous day's
  Pi and Codex transcript stores into a structured markdown writeup, using
  discovery, conversion, reusable query commands, and annotations.
LastUpdated: 2026-04-17T12:39:47-04:00
WhatFor: >
  Define a durable, multi-window process for reviewing prior-day agent work and
  producing a readable daily summary without relying on one giant SQL query.
WhenToUse: >
  Use when implementing or revising the nightly transcript review workflow or
  when deciding how to split a large daily review into resumable steps.
---

# Nightly transcript review workflow and summary architecture

## Executive summary

The nightly review problem is not just “run one SQL query.” The goal is to turn a large, mixed transcript corpus into a readable summary of the previous day’s work, while preserving enough structure that the analysis can be resumed in another context window if needed.

The design in this ticket treats the review as a small pipeline:
1. discover the relevant Pi and Codex sessions for the target day,
2. convert them into a temporary minitrace archive,
3. run a sequence of reusable structured queries,
4. render the results into a markdown report,
5. optionally add annotations for anything that needs follow-up.

That approach gives us a durable analysis process instead of a brittle one-off query. It also matches how the review actually feels in practice: one pass to inventory the day, one pass to identify the story by workspace, and one pass to pull out follow-up candidates.

## Problem statement

A single transcript review query is usually too flat for a daily writeup. The previous day’s work tends to be spread across several workspaces and several kinds of tasks:

- long infrastructure or debugging sessions,
- short review or documentation passes,
- late-day follow-ups,
- sometimes no Codex sessions at all.

The data is also split across stores. Pi sessions live under `~/.pi/agent/sessions`, while Codex sessions live under `~/.codex/sessions/YYYY/MM/DD/...`. The storage shape is different, and discovery is different, so the workflow needs to normalize both sources before analysis.

The current sample day makes the problem obvious. For 2026-04-16, the review found 10 Pi sessions and 0 Codex sessions. The day clustered into a few major workspaces:

- `~/code/wesen/crib-k3s`
- `~/code/wesen/corporate-headquarters/go-minitrace`
- `~/code/wesen/obsidian-vault/...`
- `~/workspaces/2026-04-10/pinocchiorc`

That mix cannot be reduced to a single query result without losing the story.

## Proposed solution

Keep the workflow split into reusable artifacts, but move the canonical SQL into the embedded `minitracecmd` catalog so the nightly review can be run as a real subverb:

- `scripts/00-nightly-review.sh` — the orchestrator
- `pkg/minitracecmd/core/nightly/*.sql` — embedded reusable sqleton-style commands
- `scripts/render-nightly-report.py` — markdown report renderer
- `scripts/query-catalog/*.sql` — historical ticket-local query bundle kept for reference during the migration

The shell orchestrator should do the following:

1. Discover candidate sessions from the native Pi and Codex stores using `go-minitrace discover`.
2. Convert the matching session files into a temporary minitrace output directory.
3. Run several named structured queries against the converted archive.
4. Save raw JSON results for future windows.
5. Render a human-readable markdown report.

The structured queries should be small and purpose-built:

- `go-minitrace query commands nightly session-inventory` — the detailed session list
- `go-minitrace query commands nightly workspace-summary` — group work by working directory
- `go-minitrace query commands nightly tool-breakdown` — summarize operation mix
- `go-minitrace query commands nightly followup-candidates` — identify sessions worth revisiting
- `go-minitrace query commands nightly annotation-summary` — bridge to the annotation workflow

## Design decisions

### 1. Use discovery + conversion rather than raw session parsing

`go-minitrace discover` gives us the source paths we need for a given day without forcing us to know the internal file layout in advance. Conversion then normalizes both Pi and Codex into the same analysis shape.

That keeps the pipeline aligned with the way go-minitrace itself works and avoids inventing a separate transcript reader for the nightly review.

### 2. Use structured commands for reusable analysis

The query files in this ticket are intentionally sqleton-style structured commands, not anonymous SQL snippets. The reason is simple: the same queries will be reused every day.

Using structured commands gives us:

- stable command names,
- typed parameters,
- a place to evolve the SQL without rewriting the shell script,
- compatibility with future `query commands` usage outside this ticket.

### 3. Keep the report resumable

The output is not just a markdown file. The workflow also stores:

- the discovered source lists,
- the converted minitrace archive,
- raw JSON outputs for each query,
- a log file for conversion and execution.

That matters because the review may need to continue in another context window. A future pass can open the workspace summary, inspect the follow-up candidates, and continue from there without rerunning everything.

### 4. Make annotations part of the design, not an afterthought

The annotation summary query exists because the nightly review should eventually be able to mark sessions as “reviewed,” “follow up,” or “interesting.”

Even if the first pass does not add new annotations, the workflow should already be ready to read them back and use them in the next review window.

## Alternatives considered

### One giant SQL query

This was the obvious first idea, but it is too brittle for a real daily review. It makes the analysis hard to resume, hard to debug, and hard to extend when a day has multiple major workspaces.

### Manual notes only

Manual notes are useful, but they do not scale when the transcript set is large. They are also harder to compare day-over-day.

### Raw `.sql` files without command metadata

Raw SQL files are simple, but they do not describe intent as well as structured commands. For a workflow that is meant to be reused daily, the metadata layer is worth it.

## Implementation plan

1. Keep the ticket-local scripts in `scripts/`.
2. Make the nightly review script discover, convert, and query a chosen day.
3. Save the JSON outputs so later windows can continue from those artifacts.
4. Render a markdown report with:
   - source inventory,
   - workspace summary,
   - detailed session table,
   - follow-up list,
   - tool breakdown,
   - annotation summary.
5. Add or sync annotations when the review identifies something that should be revisited.
6. Re-run the same pipeline for the next day instead of reinventing the workflow.

## Notes from the first analysis pass

The current sample review of 2026-04-16 showed that the nightly summary needs to be story-driven, not just data-driven. A few highlights already stand out:

- the largest session was the proxmox/jellyfin/k3s workspace,
- `go-minitrace` itself had three sessions on the day,
- the day ended with Obsidian vault research and cleanup,
- Codex had no sessions for this day, so the report is Pi-only.

Those are the kinds of facts that should flow into the final daily writeup.
