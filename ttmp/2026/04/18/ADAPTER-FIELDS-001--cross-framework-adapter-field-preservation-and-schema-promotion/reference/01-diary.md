---
Title: Diary
Ticket: ADAPTER-FIELDS-001
Status: active
Topics:
    - minitrace
    - adapters
    - codex
    - claude-code
    - pi
    - schema
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/analysis/01-cross-framework-field-matrix.md
      Note: Diary references the validated matrix and its implementation plan
    - Path: ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/scripts/01-scan-field-representations.py
      Note: Diary records the ticket-local raw-field scanner used during research
    - Path: ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/sources/01-field-scan.txt
      Note: Diary points to the captured scan output that supports the matrix
ExternalSources: []
Summary: Implementation diary for the cross-framework adapter field ticket, including the field-matrix research and the exit_code/justification implementation steps.
LastUpdated: 2026-04-18T00:00:00Z
WhatFor: Capture what changed, what evidence supported it, what failed, and how to review the work.
WhenToUse: Use when continuing or reviewing ADAPTER-FIELDS-001.
---


# Diary

## Goal

Track the source-backed field-gap analysis for Pi, Codex, and Claude Code, then record the implementation of the first two schema promotions: `exit_code` and `justification`.

## Step 1: Create the follow-up ticket and build the field matrix

I split the broader schema-gap discussion out of the Pi bug ticket into its own workspace so the cross-framework work can move independently. The first task in this new ticket was to replace the earlier mixed-confidence gap list with a source-backed field matrix that says, for each candidate field, whether Pi, Codex, and Claude actually expose it and whether minitrace already preserves it.

To make the analysis reproducible, I also added a ticket-local scanning script that walks representative raw transcripts and records the JSON paths where each candidate field appears. That gives the ticket a durable artifact to re-run later instead of leaving the evidence only in prose.

### Prompt Context

**User prompt (verbatim):** "turn all the findings for fields into a new ticket, and research how each of pi/codex/claude represent the potential information (or if it is missing). Store the analysis in that new ticket. THen weork on exit_code and justification, keeping a diary and committing at appropriate intervals"

**Assistant interpretation:** Create a new ticket for the cross-framework field work, record a structured Pi/Codex/Claude comparison inside it, then implement the first two validated schema promotions with diary updates and focused commits.

**Inferred user intent:** Separate the exploratory field-gap work from the Pi bug ticket, make the next implementation steps evidence-driven, and preserve enough process detail that the work is easy to audit and continue.

**Commit (code):** N/A

### What I did
- Created ticket `ADAPTER-FIELDS-001` with `docmgr ticket create-ticket`.
- Added the new ticket docs:
  - `analysis/01-cross-framework-field-matrix.md`
  - `reference/01-diary.md`
- Added ticket script `scripts/01-scan-field-representations.py`.
- Ran the script and stored its output in `sources/01-field-scan.txt`.
- Compared the scan results with:
  - `pkg/adapters/pi/convert.go`
  - `pkg/adapters/codex/convert.go`
  - `pkg/adapters/claudecode/convert.go`
  - `pkg/minitrace/schema.go`
- Wrote the field matrix and initial task breakdown for the new ticket.

### Why
- The earlier schema-gap list mixed immediate schema work, metadata-only preservation, and speculative design ideas.
- A dedicated ticket keeps the Pi bug ticket focused and makes the follow-up work easier to review in isolation.
- A scanner script reduces the chance that future follow-up is based on stale memory or paraphrase.

### What worked
- `docmgr` created the new ticket cleanly.
- The raw transcript scan confirmed the strongest real fields quickly:
  - Codex: `exit_code`, policy/runtime fields, command metadata, `stdout/stderr`
  - Claude: `caller`, `entrypoint`, `stop_reason`, thread metadata, cache bucket detail
  - Pi: `details.diff`, `stopReason`, `errorMessage`
- The matrix made it straightforward to narrow the first implementation slice to `exit_code` and `justification`.

### What didn't work
- Searching the sampled Codex raw sessions for `"justification"` returned no hits, including:
  - `rg -n '"justification"' ~/.codex/sessions | head -n 20`
- That means `justification` is not proved by the sampled real sessions in the same way `exit_code` is. The field still appears in adapter/test expectations, so the implementation can proceed, but the evidence is weaker and should be documented as such.

### What I learned
- The most important distinction is not “field exists” versus “field missing”; it is “field should be first-class schema now” versus “field should be preserved in metadata first.”
- Codex provides the richest execution/runtime metadata of the three frameworks.
- Claude provides the richest thread/entrypoint/cache metadata.
- Pi provides useful edit and termination/error details that do not map cleanly onto the Codex/Claude concepts.

### What was tricky to build
- The tricky part was not finding fields; it was deciding which findings were comparable across frameworks and which were framework-local. Some names look related but are not actually equivalent, such as Codex `phase`, Claude `stop_reason`, and Pi `stopReason`. I handled that by recording the raw representations separately and only recommending schema promotion where the cross-framework meaning was clear enough.

### What warrants a second pair of eyes
- The `justification` recommendation is weaker than the `exit_code` recommendation because it is not backed by a real local raw-session hit.
- The boundary between `framework_config` and `turn/tool framework_metadata` for future preservation work still needs a consistency pass.

### What should be done in the future
- Implement `exit_code` and `justification` first.
- After that, do a metadata-preservation pass for the remaining Codex, Claude, and Pi fields.

### Code review instructions
- Start with `analysis/01-cross-framework-field-matrix.md`.
- Verify the raw evidence by re-running:
  - `python ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/scripts/01-scan-field-representations.py > ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/sources/01-field-scan.txt`
- Then compare the matrix against the current adapter/schema files listed above.

### Technical details
- Ticket path: `ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/`
- Key evidence artifacts:
  - `sources/01-field-scan.txt`
  - `scripts/01-scan-field-representations.py`
  - `analysis/01-cross-framework-field-matrix.md`
