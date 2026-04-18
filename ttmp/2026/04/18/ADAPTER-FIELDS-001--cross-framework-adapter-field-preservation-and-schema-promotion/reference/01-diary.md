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
    - Path: pkg/adapters/codex/convert.go
      Note: Diary Step 2 records the Codex population logic for exit_code and justification
    - Path: pkg/adapters/codex/convert_test.go
      Note: Diary Step 2 references the regression coverage for the promoted fields
    - Path: pkg/minitrace/builders.go
      Note: Diary Step 2 explains why the shared builder initializes the new fields to nil
    - Path: pkg/minitrace/schema.go
      Note: Diary Step 2 records the addition of first-class exit_code and justification fields
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

**Commit (code):** `1da7a17065d19c283e88f4154fa6db1fac5bdb1f` — `Add adapter field analysis ticket`

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
- My first frontmatter validation command used a docs-root-relative path incorrectly:
  - `docmgr validate frontmatter --doc ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/index.md --suggest-fixes`
  - Error: `open /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/index.md: no such file or directory`
- Re-running with docs-root-relative paths fixed it.

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

## Step 2: Promote `exit_code` and `justification` into the schema and Codex adapter

Once the field matrix narrowed the first implementation slice, I added the two first-class schema fields instead of doing a broader metadata-preservation pass. The goal here was to make the most query-worthy Codex fields directly available to DuckDB users without forcing them to dig through framework-specific metadata blobs.

I kept the implementation narrow: update the schema and shared builder, populate the fields in the Codex adapter where the source format already exposes them, then lock the behavior in with regression tests and doc updates. I intentionally did not roll any of the metadata-only fields into this same change.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the first two schema promotions now that the cross-framework field research is written down.

**Inferred user intent:** Turn the strongest validated findings into immediately usable schema fields while preserving a clean continuation path for later metadata work.

**Commit (code):** `ffbd6a1d62512a01250d9a780b5dcc6be2b73f3a` — `Promote Codex exit code and justification`

### What I did
- Added `ToolCallInput.Justification` to `pkg/minitrace/schema.go`.
- Added `ToolCallOutput.ExitCode` to `pkg/minitrace/schema.go`.
- Updated `pkg/minitrace/builders.go` so shared tool-call construction initializes the new fields consistently.
- Updated `pkg/adapters/codex/convert.go` so:
  - session-jsonl-v1 `function_call` records promote `arguments.justification` into `tool_call.input.justification`
  - session-jsonl-v1 `function_call_output` records promote parsed exit codes into `tool_call.output.exit_code`
  - exec-jsonl-v1 `command_execution` records promote `item.exit_code` into `tool_call.output.exit_code`
- Updated `pkg/adapters/codex/convert_test.go` to assert both promoted fields.
- Updated docs:
  - `pkg/doc/minitrace-schema.md`
  - `pkg/doc/adapter-reference.md`
- Ran:
  - `gofmt -w pkg/minitrace/schema.go pkg/minitrace/builders.go pkg/adapters/codex/convert.go pkg/adapters/codex/convert_test.go`
  - `go test ./pkg/adapters/codex ./pkg/minitrace -count=1`
  - `go test ./... -count=1`

### Why
- `exit_code` is the most concrete missing execution field and dramatically improves queryability over a boolean success flag.
- `justification` belongs alongside the tool input because it describes why a tool was called, not an adapter-specific curiosity.
- Keeping the change focused reduces the risk of mixing low-confidence metadata decisions into a high-confidence schema improvement.

### What worked
- The schema change compiled cleanly across the repo.
- Codex adapter tests passed after adding direct assertions for the new fields.
- Full test coverage (`go test ./... -count=1`) passed, including the pre-commit hook during `git commit`.
- The existing adapter structure made it easy to populate the new fields without changing non-Codex adapters yet.

### What didn't work
- The pre-commit hook reran broader validation than my manual local test slice, which was slower than the focused package tests. It succeeded, but it is worth remembering that even small adapter commits in this repo trigger a full lint/test pass.
- Real sampled Codex raw sessions still do not give a live `justification` example, so the implementation remains based on source-format expectations and adapter/test coverage rather than a local production sample.

### What I learned
- `exit_code` fits naturally as a core output field without any schema ambiguity.
- `justification` is also straightforward structurally, but the evidence story is different: it is a valid field promotion with weaker raw-session evidence in the current local corpus.
- Extending the shared builder was enough to keep the schema change tidy without forcing all adapters to know about the new fields immediately.

### What was tricky to build
- The tricky part was deciding how much builder API churn to take on. I avoided widening the `BuildToolCall(...)` parameter list and instead let the shared builder initialize the new fields to `nil`, with the Codex adapter filling them in where appropriate. That kept the call sites stable and limited the implementation surface area.

### What warrants a second pair of eyes
- Whether `justification` should remain duplicated in Codex `framework_metadata` as well as the new first-class input field.
- Whether later metadata-preservation work should use `framework_config`, `turn.framework_metadata`, or `tool_call.framework_metadata` for each of the remaining Codex/Claude/Pi fields.

### What should be done in the future
- Add the next metadata-preservation slice for Codex, Claude Code, and Pi.
- Decide later whether `stdout/stderr`, `stop_reason`, or a normalized sandbox field deserve promotion.

### Code review instructions
- Start with `pkg/minitrace/schema.go` and `pkg/minitrace/builders.go`.
- Then review `pkg/adapters/codex/convert.go` for the two promotion points.
- Confirm the tests in `pkg/adapters/codex/convert_test.go`.
- Validate with:
  - `go test ./pkg/adapters/codex ./pkg/minitrace -count=1`
  - `go test ./... -count=1`

### Technical details
- Code commit: `ffbd6a1d62512a01250d9a780b5dcc6be2b73f3a`
- Manual validation commands:
  - `gofmt -w pkg/minitrace/schema.go pkg/minitrace/builders.go pkg/adapters/codex/convert.go pkg/adapters/codex/convert_test.go`
  - `go test ./pkg/adapters/codex ./pkg/minitrace -count=1`
  - `go test ./... -count=1`
