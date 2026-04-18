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
    - Path: pkg/adapters/claudecode/convert.go
      Note: Diary Step 4 records Claude session/turn/tool metadata preservation
    - Path: pkg/adapters/claudecode/convert_test.go
      Note: Diary Step 4 references regression coverage for Claude metadata preservation
    - Path: pkg/adapters/codex/convert.go
      Note: Diary Step 2 records the Codex population logic for exit_code and justification
    - Path: pkg/adapters/codex/convert_test.go
      Note: Diary Step 2 references the regression coverage for the promoted fields
    - Path: pkg/adapters/pi/convert.go
      Note: Diary Step 5 records Pi assistant/tool-result metadata preservation
    - Path: pkg/adapters/pi/convert_test.go
      Note: Diary Step 5 references regression coverage for Pi metadata preservation
    - Path: pkg/doc/framework-metadata-mappings.md
      Note: Diary Step 6 references the new public mapping doc for preserved framework metadata
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

## Step 3: Preserve Codex session, turn, and tool metadata

With the first-class schema additions done, I moved to the larger metadata-preservation slice for Codex. The aim here was to keep the richer raw Codex execution/runtime context available for analysis without exploding the shared schema further.

I treated this as a separate step and commit because it touches a different class of data: framework-specific raw detail rather than cross-framework first-class fields. That separation keeps review simpler and makes it easier to revert or refine the storage conventions later.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Preserve the remaining validated Codex metadata in the adapter now that the initial schema promotions are done.

**Inferred user intent:** Keep the raw Codex richness accessible in minitrace so future analysis does not depend on re-reading native JSONL files.

**Commit (code):** `585db79bab918ce61131144c78918b8591b71c65` — `Preserve Codex adapter metadata`

### What I did
- Extended `pkg/adapters/codex/convert.go` to preserve richer session/runtime metadata in `operational_context.framework_config`:
  - `approval_policy`
  - detailed `sandbox_policy`
  - `collaboration_mode_detail`
  - `truncation_policy`
  - `rate_limits`
  - `session_source`
- Preserved turn-level metadata in `turns[].framework_metadata`:
  - `turn_id`
  - `phase`
  - `memory_citation`
- Preserved tool-call metadata in `tool_calls[].framework_metadata`:
  - `source`
  - `parsed_cmd`
  - `stdout`
  - `stderr`
  - `status`
  - `turn_id`
- Added/extended Codex regression tests to cover both session-jsonl-v1 and exec-jsonl-v1 metadata preservation.
- Ran:
  - `gofmt -w pkg/adapters/codex/convert.go pkg/adapters/codex/convert_test.go`
  - `go test ./pkg/adapters/codex -count=1`
  - pre-commit validation via `git commit`

### Why
- Codex’s raw format exposes far more execution/runtime detail than the normalized schema can reasonably absorb all at once.
- Preserving those fields in `framework_config` and `framework_metadata` keeps them queryable without forcing a premature schema redesign.
- The split between session/turn/tool metadata lines up well with the underlying raw Codex event structure.

### What worked
- The metadata landed cleanly in the intended storage locations.
- Session-jsonl-v1 and exec-jsonl-v1 both stayed green after the preservation changes.
- The metadata keys are now stable enough to document externally.

### What didn't work
- I initially wired `codexTurnMetadata(...)` to return `any`, which caused a compile error when merging tool metadata:
  - `pkg/adapters/codex/convert.go:366:79: cannot use codexTurnMetadata(currentTurnID, nil, nil) (value of interface type any) as map[string]any value in argument to mergeMetadataMap: need type assertion`
- I fixed that by making `codexTurnMetadata(...)` return `map[string]any` directly.

### What I learned
- The raw Codex event model maps naturally onto minitrace’s three escape hatches: session config, turn metadata, and tool metadata.
- `exec_command_end` is especially valuable because it carries structured execution detail that is richer than the text-only tool output path.

### What was tricky to build
- The tricky part was preserving detailed metadata without breaking the existing meaning of keys that already existed in converted sessions. I kept the existing string `collaboration_mode` and added `collaboration_mode_detail` for the full object instead of silently changing the type of the older key.

### What warrants a second pair of eyes
- Whether `rate_limits` belongs permanently in `framework_config` or should eventually move into a separate session/turn analytics model.
- Whether we should later synthesize a separate query-oriented projection for parsed command metadata.

### What should be done in the future
- Keep the stored key names stable now that the metadata is preserved and documented.
- Consider follow-up presets/queries that use `phase`, `rate_limits`, or `parsed_cmd`.

### Code review instructions
- Start with `pkg/adapters/codex/convert.go`.
- Then review the assertions in `pkg/adapters/codex/convert_test.go`.
- Validate with:
  - `go test ./pkg/adapters/codex -count=1`
  - `go test ./... -count=1`

### Technical details
- Code commit: `585db79bab918ce61131144c78918b8591b71c65`
- Exact compile failure encountered before the fix:
  - `pkg/adapters/codex/convert.go:366:79: cannot use codexTurnMetadata(currentTurnID, nil, nil) (value of interface type any) as map[string]any value in argument to mergeMetadataMap: need type assertion`

## Step 4: Preserve Claude Code session, turn, and tool metadata

After Codex, I applied the same preservation strategy to Claude Code, but with a different emphasis: Claude’s interesting metadata is mostly about turn/session/thread context rather than execution runtime. I kept the implementation narrow and only preserved the fields we had already validated from raw transcripts.

This step also needed special handling for tool-result pseudo-turns. Because the adapter intentionally absorbs those records into tool calls instead of emitting separate turns, any record-level metadata on those tool-result messages would otherwise be lost.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Preserve the validated Claude-specific metadata without promoting it into shared schema fields.

**Inferred user intent:** Keep Claude’s thread/session context and cache metadata available for analysis, especially where minitrace intentionally collapses raw records.

**Commit (code):** `01e5ddccb19e5165dc5248a4afbfd067957d5a63` — `Preserve Claude adapter metadata`

### What I did
- Preserved Claude session-level `entrypoint` in `operational_context.framework_config`.
- Preserved turn metadata in `turns[].framework_metadata`:
  - `entrypoint`
  - `slug`
  - `parent_uuid`
  - `is_sidechain`
  - `stop_reason`
  - `stop_sequence`
  - `cache_creation`
- Preserved tool metadata in `tool_calls[].framework_metadata`:
  - `caller`
  - record context from skipped tool-result messages so `entrypoint`, `slug`, `parent_uuid`, and `is_sidechain` are not lost when those pseudo-turns are absorbed into tool calls
- Added regression coverage for the preserved Claude metadata.
- Relied on the pre-commit hook plus `go test ./... -count=1` validation already in the working loop.

### Why
- Claude’s raw transcript has important metadata that is orthogonal to the normalized schema and different from Codex’s execution-heavy details.
- Preserving it in metadata is the least invasive way to keep it available.
- The tool-result absorption path would otherwise silently discard some of the most useful thread/session context.

### What worked
- The adapter preserved the validated Claude metadata without changing the existing turn/tool model.
- The new test clearly demonstrates both the assistant-turn metadata and the tool-result metadata merge.
- The session-level `entrypoint` preservation is simple and stable.

### What didn't work
- There was no major implementation failure in this slice after the Codex helper issue was resolved, but the pre-commit hook again expanded each focused code change into a full-repo validation run.

### What I learned
- Claude’s metadata is much more about context and provenance than command execution mechanics.
- Tool-result record merging is the key preservation seam in the Claude adapter because that is where raw-record detail can disappear even when tool outputs remain intact.

### What was tricky to build
- The tricky part was choosing where to keep record-level metadata from skipped tool-result pseudo-turns. I merged it into the corresponding tool call’s `framework_metadata`, which preserves the information without reintroducing the fake turns the adapter intentionally avoids.

### What warrants a second pair of eyes
- Whether `entrypoint` should remain both session-level and turn/tool-level when present, or whether one of those layers is redundant.
- Whether `cache_creation` should eventually get a more structured first-class representation if multiple adapters start exposing comparable bucket detail.

### What should be done in the future
- Revisit whether Claude thread metadata (`parent_uuid`, `is_sidechain`, `slug`) warrants a higher-level session/thread model.

### Code review instructions
- Start with `pkg/adapters/claudecode/convert.go`.
- Then review `pkg/adapters/claudecode/convert_test.go`.
- Validate with:
  - `go test ./pkg/adapters/claudecode -count=1`
  - `go test ./... -count=1`

### Technical details
- Code commit: `01e5ddccb19e5165dc5248a4afbfd067957d5a63`

## Step 5: Preserve Pi assistant and tool-result metadata

The Pi slice was smaller than Codex and Claude, but it closed two of the concrete gaps that originally came up during the Jellyfin debugging work: assistant stop/error detail and edit diffs on tool results. These are exactly the kinds of fields that are highly useful during forensic analysis yet not obviously cross-framework enough for immediate schema promotion.

I implemented this by preserving assistant message details on turns and tool-result details on tool calls. That keeps the normalized schema clean while making the missing raw context available again.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Preserve the validated Pi-only metadata that was left out of the first bug-fix ticket.

**Inferred user intent:** Avoid losing Pi’s useful debugging context now that the Go-side `toolResult.isError` fix is already complete.

**Commit (code):** `332a9d7c648a45fa2c2a33a366508366f4aeeb49` — `Preserve Pi adapter metadata`

### What I did
- Preserved assistant turn metadata in `turns[].framework_metadata`:
  - `stop_reason`
  - `error_message`
- Extended `applyToolResult(...)` to preserve Pi tool-result metadata in `tool_calls[].framework_metadata`:
  - `diff`
  - `first_changed_line`
- Updated the Pi tests so they assert:
  - assistant `stop_reason` / `error_message` preservation
  - message-level tool-result diff preservation
- Ran formatting/tests and relied on the pre-commit full validation during commit.

### Why
- These fields were validated directly from the raw Pi session and were among the highest-value Pi-specific losses.
- They are clearly useful for debugging and review, especially edit diffs.
- Preserving them in metadata avoids locking in a Pi-specific schema design prematurely.

### What worked
- The changes were small and localized.
- Existing Pi behavior remained intact while the missing metadata started surviving conversion.
- The message-level tool-result test now covers both success/failure semantics and diff preservation.

### What didn't work
- There was no major failure in this slice after the earlier Codex compile fix, but the repo’s full pre-commit checks still dominate the runtime for even focused adapter commits.

### What I learned
- Pi’s most valuable extra metadata is not session/runtime policy or thread structure; it is concrete execution/debugging detail attached to assistant messages and tool results.
- `applyToolResult(...)` is the right seam for preserving Pi tool-result metadata because both content-block and message-level result paths already flow through it.

### What was tricky to build
- The tricky part was keeping the preservation logic shared across the content-block and message-level tool-result paths. Extending `applyToolResult(...)` to accept the raw result map was the cleanest way to do that without duplicating extraction logic.

### What warrants a second pair of eyes
- Whether additional Pi tool-result `details.*` fields beyond `diff` and `firstChangedLine` should also be preserved in a later pass.

### What should be done in the future
- Evaluate whether Pi `diff` metadata is common enough and important enough to justify first-class schema treatment later.

### Code review instructions
- Start with `pkg/adapters/pi/convert.go`.
- Then review `pkg/adapters/pi/convert_test.go`.
- Validate with:
  - `go test ./pkg/adapters/pi -count=1`
  - `go test ./... -count=1`

### Technical details
- Code commit: `332a9d7c648a45fa2c2a33a366508366f4aeeb49`

## Step 6: Document the framework-specific metadata contract

Once the three adapter slices were in place, I wrote the docs that explain where these fields live and how to interpret them. The core problem was that the schema page and adapter reference explained the shared schema reasonably well, but they did not explicitly document the now-expanded metadata contract.

I fixed that by adding a dedicated help page for framework metadata mappings, updating the schema doc to explain the three metadata escape hatches, and expanding the adapter reference to list what each adapter now preserves.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish the implementation with documentation that makes the preserved metadata discoverable and reviewable.

**Inferred user intent:** Ensure the new metadata behavior is not hidden in code and tests only.

**Commit (code):** latest docs/bookkeeping commit for this ticket — `Document framework metadata mappings`

### What I did
- Added `pkg/doc/framework-metadata-mappings.md`.
- Updated `pkg/doc/minitrace-schema.md` with framework metadata storage conventions.
- Updated `pkg/doc/adapter-reference.md` with per-adapter preserved metadata summaries and a link to the new mapping doc.
- Updated the ticket tasks/changelog/diary to reflect the completed slices.

### Why
- Metadata preservation without documentation quickly becomes invisible to users and future maintainers.
- The new doc gives one stable place to answer: “where is this raw field stored after conversion?”

### What worked
- The documentation now mirrors the storage design used in code.
- The new help page provides a more durable answer than burying the mappings inside a ticket note only.

### What didn't work
- N/A so far.

### What I learned
- A small shared schema plus explicit metadata contracts is easier to evolve than trying to force every useful raw field into first-class schema immediately.

### What was tricky to build
- The tricky part was balancing the adapter-specific doc detail with the schema doc’s more general role. I kept the schema page focused on conventions and put the per-adapter mapping tables in a separate help page.

### What warrants a second pair of eyes
- The exact metadata key naming (`parent_uuid` vs raw `parentUuid`, `stop_reason` vs raw `stopReason`) should stay consistent across future adapter work.

### What should be done in the future
- Keep the new mapping doc updated whenever metadata is added, renamed, promoted, or removed.

### Code review instructions
- Start with `pkg/doc/framework-metadata-mappings.md`.
- Then review the related updates in:
  - `pkg/doc/minitrace-schema.md`
  - `pkg/doc/adapter-reference.md`

### Technical details
- Code/docs commit: see the latest `Document framework metadata mappings` commit for this ticket
- Key docs added/updated:
  - `pkg/doc/framework-metadata-mappings.md`
  - `pkg/doc/minitrace-schema.md`
  - `pkg/doc/adapter-reference.md`
