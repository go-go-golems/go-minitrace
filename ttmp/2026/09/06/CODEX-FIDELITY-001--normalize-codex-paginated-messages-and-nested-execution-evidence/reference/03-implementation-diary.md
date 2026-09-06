---
Title: Implementation diary
Ticket: CODEX-FIDELITY-001
Status: active
Topics:
    - codex
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://Makefile
      Note: Local validation contract
    - Path: repo://cmd/go-minitrace/cmds/serve/file_evidence_test.go
      Note: API protobuf target presence and counter tests
    - Path: repo://pkg/adapters/codex/convert.go
      Note: Parser integration and removal of guessed associations
    - Path: repo://pkg/adapters/codex/execution_identity_test.go
      Note: Adversarial one-to-one identity regressions f0d4811
    - Path: repo://pkg/adapters/codex/executions.go
      Note: Authoritative lifecycle reconciliation in 6e656e7
    - Path: repo://pkg/adapters/codex/fidelity.go
      Note: Bounded visible fidelity diagnostics
    - Path: repo://pkg/adapters/codex/file_changes.go
      Note: Authoritative file effects and lifecycle identity
    - Path: repo://pkg/adapters/codex/files.go
      Note: Structural targets and explicit cwd resolution
    - Path: repo://pkg/adapters/codex/legacy_outcomes.go
      Note: Arrival-order-independent terminal reconciliation in 9658081
    - Path: repo://pkg/adapters/codex/legacy_outcomes_test.go
      Note: Ordering, conflict replay, and nullable exec-stream regression tests
    - Path: repo://pkg/adapters/codex/messages.go
      Note: Message reconciliation and explicit linkage in 22b1f4e
    - Path: repo://pkg/adapters/codex/messages_test.go
      Note: Message identity and linkage regressions
    - Path: repo://pkg/adapters/codex/output_replay_test.go
      Note: Early/repeated/conflicting output coverage 4e61b7b
    - Path: repo://pkg/adapters/codex/outputs.go
      Note: Typed result decoding and independent block outcomes
    - Path: repo://pkg/adapters/codex/shell_targets.go
      Note: Bounded non-evaluating literal redirect grammar
    - Path: repo://pkg/adapters/codex/testdata/paginated-fidelity.jsonl
      Note: Synthetic contract regression source
    - Path: repo://pkg/minitrace/activity_counts.go
      Note: Separates invocations executions and file evidence
    - Path: repo://pkg/minitrace/file_evidence.go
      Note: Typed structural evidence and explicit empty semantics
    - Path: repo://pkg/minitracecmd/core/history/file-history.js
      Note: Codex structural history and null-safe context
    - Path: repo://pkg/minitracedb/file_evidence_test.go
      Note: Multi-target SQL outcomes regression f0a5377
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/design-doc/02-implementation-contracts-and-phased-acceptance.md
      Note: Implementation decisions
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/scripts/04-check-synthetic-baseline.py
      Note: Independent CLI acceptance oracle
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/scripts/05-audit-message-coverage.py
      Note: Independent native-line coverage audit
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/scripts/06-proto-outcomes.go
      Note: Emit five-state synthetic protojson
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/scripts/07-check-proto-outcomes.mjs
      Note: Actual TypeScript decoder roundtrip
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/scripts/08-audit-execution-coverage.py
      Note: Independent native execution acceptance audit
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/scripts/09-audit-file-changes.py
      Note: Independent 472-event 995-target native acceptance
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/scripts/10-check-history-consumers.py
      Note: Repeatable end-to-end history and counter acceptance
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/various/p1/synthetic-before.json
      Note: Eight expected before-state fidelity failures
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/various/p3/outcome-states.png
      Note: Inspected neutral and binary outcome UI
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/various/slips/00-overall-plan.log
      Note: Printed overall phase checklist
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/various/slips/01-p1-start.log
      Note: Printed P1 start confirmation
    - Path: repo://web/src/api/activityMetrics.ts
      Note: Typed API activity counter decoding
ExternalSources: []
Summary: Chronological implementation decisions, commits, validation, and phase printing receipts.
LastUpdated: 2026-09-06T00:00:00Z
WhatFor: Audit the implementation against the fidelity contract.
WhenToUse: Before resuming implementation or reviewing its changes.
---












# Diary

## Goal

Implement all CODEX-FIDELITY-001 acceptance criteria with reproducible evidence, focused commits, and printed phase checkpoints. Earlier investigation and local reproduction are in reference/01 and reference/02.

## Step 1: Print the implementation plan and start P1

At 2026-09-06T20:19Z, printed the overall five-phase plan and the P1 start checklist before starting implementation work. Both remote responses report `printed: true`, `printer_response.ok: true`, and HTTP 200. Read the work-slip and diary skills and the repository's AGENT.md, then began tracing schema and consumers to make explicit implementation decisions rather than patching the parser in isolation.

P1 comprises contracts and regression fixtures; P2 messages and linkage; P3 execution and typed outcomes; P4 file evidence and query consumers; P5 acceptance and documentation. Phases may contain several focused commits and diary steps, but each phase must have a start and completion print receipt.

### Prompt Context

**User prompt (verbatim):** "implement the ticket, budget 1M, commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill). Print out a brutalist work slip with the plan / different phases for the ticket. then before stsarting a phase, plrint a split about the phase, and print one when the phase is done. [REMINDER] Output a <summary>...</summary> block at the VERY END of your response. This is mandatory."

**Assistant interpretation:** Implement the complete ticket with one million tokens available, commit at coherent milestones, keep contemporaneous detailed diary entries, and physically print the plan plus both boundaries of each phase.

**Inferred user intent:** Maintain a trustworthy, inspectable implementation workflow with physical progress tracking and demonstrated local-source fidelity improvements.

### What I did
- Printed via `python3 /home/manuel/.pi/agent/skills/brutalist-work-slip/scripts/work_slip.py plan` using the constrained DSL; retained YAML and full receipts in `various/slips/00-overall-plan.*` and `01-p1-start.*`.
- Read the diary skill and its full reference, docmgr skill, repository AGENT.md, Makefile, Codex parser/tests, core output and metric schema, file-history consumer, and success-field consumers in SQL/API/frontend.
- Created this implementation diary and a separate implementation-contract document through docmgr.
- Preserved the existing uncommitted local-validation work; no production files had user modifications at this checkpoint.

### Why
- Outcome changes cross the archive, SQLite, protobuf API, and UI; an adapter-only default change would silently reclassify unknown as failure elsewhere.
- Print before phase work so paper tracks actual sequencing, not reconstructed milestones.

### What worked
- Remote printing succeeded for both requested initial slips.
- Native source inspection confirmed the precise event keys: response metadata carries native turn IDs; message mirrors can have different user IDs; completed executions retain shell argv, file-URI cwd, stdout/stderr, status, and exit code.

### What didn't work
- No command failure in this checkpoint. Repository AGENT.md requests a `format_file` tool, which is not exposed in this session; use available language formatters and record this limitation rather than claiming that tool ran.

### What I learned
- Current history extraction scans JavaScript wrapper text for redirections and patch headers; preserving false-branch safety requires consumer changes, not just new execution rows.
- `ToolCallOutput.Success` is a Go bool, propagated through protobuf bool and TypeScript boolean; unknown must have a reviewed, end-to-end representation.

### What was tricky to build
- No implementation yet. The core design hazard is conflating model-issued wrapper counts, observed execution counts, and per-file effects; the contract will separate them before code changes.

### What warrants a second pair of eyes
- The forthcoming public outcome/counting contract and deduplication rules need review against all downstream consumers and existing fixtures.

### What should be done in the future
- Finish P1 contract and synthetic fixtures, record baseline regression failures, and print P1 completion before P2 start.

### Code review instructions
- Compare phase lists in `various/slips/00-overall-plan.yaml` with the implementation-contract document and this diary.
- Inspect both `.log` files for real printing confirmation (`dry_run: false`, `printed: true`).

### Technical details
- Overall print receipt: 2026-09-06T20:19:10Z, 384x535.
- P1 start receipt: 2026-09-06T20:19:25Z, 384x483.
- Repository validation entry point is `make all` (lint, test, generate/build); additional generated logging check is `make logcopter-check`.
- Baseline HEAD: `28cd1c2ad215d1f31d54abd45eb439588c8aae12`.
- Committed local baseline, printing receipts, diary start, and explicit contracts as `443cfbd` — `docs(codex): record local fidelity baseline and implementation contracts`.

## Step 2: Freeze a synthetic fidelity oracle

Added a small hand-authored JSONL fixture that models the observed native shapes without copying private transcript content. It includes mirrored messages, repeated genuine user text, a wrapper with an unexecuted false branch, duplicate execution completions, two real identical commands with distinct IDs, failure, missing/unknown/pending/cancelled outcomes, early output, and a quoted history that must remain text.

Ran the unchanged adapter against the fixture and saved an independent JSON assertion report. The identity check passes while all eight fidelity checks fail, matching the real-source baseline. This is an intentional before-state, not a failed implementation fix or a skipped test. Production package tests remain green; implementation phases will add package regressions for each contract and make this oracle pass.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Establish an independently reproducible regression target before changing conversion semantics.

**Inferred user intent:** Prevent private-source-specific fixes and prove the new code repairs genuine loss without inventing events.

### What I did
- Added `pkg/adapters/codex/testdata/paginated-fidelity.jsonl` with synthetic data only.
- Added `scripts/04-check-synthetic-baseline.py`, a before/after archive oracle; it reads data and never evaluates commands.
- Ran `go test ./pkg/adapters/codex -count=1` successfully.
- Ran `go run ./cmd/go-minitrace convert codex --source-session "$PWD/pkg/adapters/codex/testdata/paginated-fidelity.jsonl" --output-dir /tmp/codex-fidelity-001-synthetic-before --run-record <ticket>/various/p1/synthetic-conversion.json`.
- Ran `python3 <ticket>/scripts/04-check-synthetic-baseline.py /tmp/codex-fidelity-001-synthetic-before/active/2026-09/fidelity-synthetic.minitrace.json`; saved its JSON and exit status 1.
- Wrote explicit nullable-outcome, message identity, record-count, provenance, file-evidence, and UI/API verification contracts in design-doc/02.

### Why
- Keeping a content-free fixture and an independent CLI oracle makes the private-source claims testable on other machines.
- Avoid committing tests that only assert current broken behavior; the oracle states the desired end state and intentionally fails before implementation.

### What worked
- Existing Codex tests: `ok github.com/go-go-golems/go-minitrace/pkg/adapters/codex 0.005s`.
- Fixture converts reproducibly with native identity `fidelity-synthetic` and stable fingerprint recorded in the receipt.

### What didn't work
- Expected acceptance failures: `five_deduplicated_messages` (0), `repeated_continue_retained` (0), `no_orphan_associations` (3 orphan tools), `typed_outputs_decoded` (wrapper map string), `one_authoritative_failed_execution` (0), `identical_commands_distinct_ids` (0), `missing_outcome_not_success` (true), and `output_before_call_preserved` (null).
- Before the documentation commit, `git diff --check` reported `changelog.md:20: new blank line at EOF.` The shell sequence did not gate the subsequent commit on this check; correcting the docmgr-generated trailing blank line in this checkpoint and using explicit gating on subsequent commits.

### What I learned
- Output-before-call is silently dropped by the current pending-call map, independent of array decoding.
- A typed wrapper envelope containing a child exit code must not be promoted blindly to wrapper success; the structured child record is authoritative.

### What was tricky to build
- The source mirrors user records with different IDs and assistant records with the same ID. The fixture models both and adds a second real `continue` in another native turn; content-only deduplication cannot pass it correctly.

### What warrants a second pair of eyes
- The five-turn oracle intentionally excludes mirrored records while retaining quoted history as one user message. It is a synthetic ground truth, not a deduplicated count inferred from the large private files.
- Public success nullability and counting contract remain review-critical during P3/P4.

### What should be done in the future
- Finish P1 print/commit bookkeeping; implement P2 message normalization and tests, then the P3/P4 outcomes and evidence that remain red in this oracle.

### Code review instructions
- Read the fixture in source order and compare against `various/p1/synthetic-before.json` and `scripts/04-check-synthetic-baseline.py`.
- Repeat the conversion in a new output directory to avoid collision replacement and rerun the oracle.

### Technical details
- The fixture has six distinct structured execution IDs; one is pending and one is cancelled, with the failed completion deliberately repeated.
- No production parser changes in P1. Full build/lint/UI verification is required after implementation, not claimed by this package-only baseline.
- P1 fixture/oracle commit: `0f67556` — `test(codex): add synthetic paginated fidelity acceptance baseline`.
- P1 completion printed at 20:25:30Z (`02-p1-done.log`); P2 start printed at 20:25:34Z (`03-p2-start.log`). Both receipts confirm real printing.

## Step 3: Reconcile messages and remove speculative linkage

Implemented a message reconciliation pass keyed by native IDs, with the narrowly defined adjacent complementary-representation rule for different-ID mirrors. Message emission still runs in source order within the existing parser so reasoning and usage handling remain in their supported positions. Every emitted message retains all native source-line references; tool linkage is explicit-message-only, otherwise null with native turn context.

The synthetic message checks and Codex package/race tests pass. An independent private-source coverage audit accounts for every supported native message record exactly once and reports no invalid links. Inspection of the first legacy-control result exposed different native block concatenation; corrected comparison to use exact concatenated native text rather than inserting display newlines into the identity comparison. Public display boundaries remain readable, and a new regression rejects whitespace-normalized false mirrors.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement P2 with identity-preserving message recovery and honest tool associations, then validate synthetic and private sources before printing phase completion.

**Inferred user intent:** Restore trustworthy conversational context without fabricating the message that issued a tool.

### What I did
- Added `pkg/adapters/codex/messages.go` and `messages_test.go`; integrated reconciliation into persisted-session parsing.
- Removed legacy and persisted fallback association to index zero or later final answers; removed the now-unused pending-ID sorting helper.
- Added identity/mirror/repetition/missing-ID/conflict/block/legacy/no-message/explicit-link regressions.
- Added `scripts/05-audit-message-coverage.py`, which independently inventories supported native message lines and checks exactly-once provenance and reciprocal non-null links.
- Ran focused and race tests and fresh synthetic/private conversions; saved content-free results in `various/p2/`.

### Why
- The same native turn contains multiple messages; next-message assignment is not proof of tool emission.
- Source-line coverage is an independent check against silent message loss even when total native and normalized counts differ due to mirrors.

### What worked
- Synthetic turns: 5; both genuine `continue` requests retained; orphan links: 0.
- Three private paginated sources now produce 252, 305, and 99 turns; controls produce 630, 701, and 193. All six cover every supported native message source line exactly once with no invalid associations.
- `go test ./pkg/adapters/codex -count=1` and `go test ./pkg/adapters/codex -race -count=1` pass (race rerun still needed after final edits).

### What didn't work
- First private-control inspection showed 917 rather than 630 turns in one control: legacy event messages concatenate text blocks without separators while the response display joined blocks with newlines. Fixed comparison at the typed-block boundary, without stripping arbitrary whitespace; rerun gives 630 and preserves all 1203 native message references.
- `make lint` exited 2: `pkg/adapters/codex/messages.go:199:2: QF1003: could use tagged switch on message.role (staticcheck)`, followed by `make: *** [Makefile:31: lint] Error 1`. Next action is the suggested tagged switch and a fresh lint run; no bypass or suppression.
- The full synthetic oracle still exits 1 on five execution/output checks assigned to P3. No completion of the overall ticket is claimed.

### What I learned
- Provenance coverage alone does not prove deduplication quality: inspect representation counts and content boundary behavior too.
- Legacy-mode sources contain response-only context/instruction messages; restoring them legitimately increases turns above the old event-only baseline.

### What was tricky to build
- Different-ID user mirrors need adjacent source records, the same nonempty native turn, complementary representations, and exact native text. Same-representation repeated content must survive. Canonical display text and native concatenation identity are distinct so block separators do not hide valid mirrors.
- Tool calls can appear before their explicitly referenced message. Link in a final pass using native identity, with reciprocal `tool_calls_in_turn`, rather than temporal guessing.

### What warrants a second pair of eyes
- Native ID conflicts remain diagnosed with source references and canonical response preference; verify no broad text-based deduplication slips into this rule.
- Removing speculative legacy linkage is an intentional correctness change and must be documented in P5.

### What should be done in the future
- Fix the single lint finding, run full tests and lint, review and commit P2, then print its completion and P3 start.

### Code review instructions
- Start with `collectCodexMessages`, `decodeCodexMessage`, and `linkCodexMessageCalls`, then the parser diff and `messages_test.go`.
- Run `go test ./pkg/adapters/codex -count=1`, `make lint`, and the saved private source coverage script against a fresh conversion.

### Technical details
- First private iteration: `/tmp/codex-fidelity-001-private-p2/`; corrected iteration: `/tmp/codex-fidelity-001-private-p2-reconciled/`.
- Native message-line coverage totals: 488, 1203, 602, 1341, 194, 371; no missing, invented, or duplicate source references.
- Five oracle checks remaining: typed output decoding, one failed execution, repeated distinct executions, null missing outcome, early output preservation.

### Completion checkpoint

Replaced the flagged if/else with a tagged switch; `make lint` now reports zero issues and glazed-lint passes. `make test` passes the full repository, and the final Codex race rerun passes. Reviewed and committed P2 code as `22b1f4e` — `fix(codex): restore native messages and preserve uncertain tool linkage`; pre-commit lint and full tests also passed without bypasses. P2's message/linkage requirements are verified; execution/output work remains for P3. P2 completion printed at 20:38:14Z; documentation/evidence commit `638926f` records the phase. P3 start printed at 20:39:05Z (`05-p3-start.log`).

## Step 4: Migrate nullable outcomes across consumers

Started P3 by implementing the shared outcome representation before changing Codex execution parsing. Added nullable binary success, explicit lifecycle status, and evidence-aware predicates/setters; migrated existing adapter assignments to preserve their established known outcomes. Changed SQL materialization, preview and serve projections, protobuf presence, and UI neutral rendering so a null outcome cannot silently become a passing or failing result downstream.

This checkpoint is still in progress. Protobuf regeneration succeeded with plugin versions pinned to the versions already recorded in generated headers, avoiding unrelated generator drift. The first full Go test run found one historical ticket script still using bool-only conditions; all production package tests passed, and the next step is to update that compiled script to select explicit failures.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Establish a safe outcome contract through storage and presentation before mapping native executions.

**Inferred user intent:** Missing outcomes must remain visibly unknown, and failed children must not disappear behind successful wrappers.

### What I did
- Added `pkg/minitrace/outcome.go` and outcome nullability tests; changed `ToolCallOutput.Success` to `*bool` and added `status`.
- Added `SetSuccess`, `Succeeded`, `Failed`, and `OutcomeStatus` so known outcomes stay consistent and null is never interpreted by boolean falsiness.
- Migrated existing adapter assignment/assertion sites, SQL insertion and severity, server badges and response structs, and JS import preview fields.
- Added SQLite `outcome_status`; bumped archive schema to v0.3.0 and normalized SQL schema to v4 for cache invalidation.
- Changed protobuf success to optional, added status and signed exit code, pinned existing generator versions, and ran `buf generate` successfully.
- Changed frontend decoding from missing-success=false to null and neutral pending/cancelled/unknown chips in tool rows.
- Ran `go test ./...`, capturing the first full compile/test result in `various/p3/nullable-go-tests.log`.

### Why
- An adapter-only nullable value would still become false in the API/UI without these consumer changes.
- Preserve explicit known outcomes from all other adapters rather than changing their established semantics as an accidental side effect.

### What worked
- Protobuf regeneration completed successfully; production adapter, schema, database, CLI, JS provider, and serve package tests passed in the first full run.
- New outcome tests prove JSON success=null for unknown/pending/cancelled and retain true/false for known outcomes.

### What didn't work
- Full `go test ./...` exited 1 because the historical compiled script was missed by the initial source scan excluding ttmp:
  - `ttmp/2026/04/16/bug-iserror-001--pi-adapter-iserror-not-mapped-to-output-success/scripts/01-verify-real-session.go:28:6: non-boolean condition in if statement`
  - `ttmp/2026/04/16/bug-iserror-001--pi-adapter-iserror-not-mapped-to-output-success/scripts/01-verify-real-session.go:46:6: non-boolean condition in if statement`
- This is a source migration omission, not an excuse to exclude the script or bypass repository tests; next action is evidence-aware failure predicates and a fresh full run.

### What I learned
- Historical ticket scripts participate in `go test ./...` and must remain compilable during public Go schema changes.
- Existing failure SQL already uses `success = 0`, which naturally excludes null; severity and UI conditions needed explicit predicate changes.

### What was tricky to build
- Nullable success and lifecycle status convey different evidence: status alone cannot invent a binary outcome. `OutcomeStatus` uses known binary evidence first and only recognizes pending/cancelled when success is null; unsupported or completion-only states remain unknown.
- API protobuf presence must survive protojson and TypeScript decoding; a plain bool would erase null even if Go/SQLite preserve it.

### What warrants a second pair of eyes
- Consistency of nullable success/status across API, preview, SQL, badges, and old archive input. No custom unmarshalling or compatibility adapter was introduced.
- P3 still needs native execution/output implementation plus dedicated SQL/API/UI assertions and visual inspection.

### What should be done in the future
- Fix the compiled historical script, add database/API/UI outcome tests, validate lint and builds, then commit this schema milestone before execution parsing.

### Code review instructions
- Start with `pkg/minitrace/outcome.go`, schema/builders, and `proto/.../sessions.proto`; compare generated optional success and the frontend null-coalescing change.
- Rerun `go test ./...`, then frontend build/lint and explicit nullable outcome integration checks.

### Technical details
- Generator versions retained: protoc-gen-go v1.36.11 and protoc-gen-es v2.12.0.
- Codex native wrapper defaults and lifecycle mapping are not repaired by this schema-only checkpoint; the fixture's five remaining assertions are still implementation work.

### Validation checkpoint

Fixed both historical script conditions to `!tc.Output.Failed()` and reran `go test ./...`: all packages pass. Added SQL tests for all five states, including null in tool/file projections and only one error event; added protobuf optional-success and signed-exit-code tests plus badge checks. `make lint` passes with zero issues. `pnpm install --frozen-lockfile` succeeded (existing esbuild/msw build scripts remain unapproved; no approval change was needed); `pnpm build` passes TypeScript and production bundling with the existing large-chunk advisory.

`pnpm lint` exited 1 with 10 errors and 13 warnings, saved verbatim in `various/p3/web-lint.log`. Nine story files use `@storybook/react` directly (`storybook/no-renderer-packages`), including the previously existing import in the modified tool-row story. `BlockCard.tsx:51:7` reports `react-hooks/set-state-in-effect` for `setShowAllTools(true)`. These lines existed before this work; they are validation blockers, not reasons to suppress lint. Next action: use the installed `@storybook/react-vite` framework type imports and replace the effect with a guarded render-time state adjustment that preserves the sticky show-all behavior, then rerun lint and story tests. Warnings in generated protobuf/MSW files and existing memo dependencies are recorded separately; they are not current implementation errors.

After replacing renderer-package imports and preserving sticky expansion through a guarded render-time state update, frontend lint passes with 0 errors (13 existing warnings) and build passes. The first Storybook Vitest invocation (`pnpm exec vitest run --project storybook src/components/TranscriptViewer/stories/ToolCallRow.stories.tsx`) could not launch a browser: `Executable doesn't exist at /home/manuel/.cache/ms-playwright/chromium_headless_shell-1217/chrome-headless-shell-linux64/chrome-headless-shell`. No tests ran, so this is not a test pass. Installing the matching Chromium binary is the next low-risk prerequisite step. Generated-file review also found previously mixed ES versions (2.11.0 in other files, 2.12.0 in sessions); the pinned full regeneration consistently produces 2.12.0 headers and optional `| undefined` declarations, with no manual generated edits.

Installed the matching Chromium/headless-shell 1217 using `pnpm exec playwright install chromium`; the installer also garbage-collected unused browser-cache versions. The final two-file Storybook run passes 22 tests. The Go protojson emitter (`scripts/06-proto-outcomes.go`) piped into the actual TypeScript decoder via Node 24 (`scripts/07-check-proto-outcomes.mjs`) passes all five states, retaining null, explicit false, and signed exit code -1.

Visually inspected `various/p3/outcome-states.png`: only succeeded has a green check, only failed has a red border/error icon, and unknown/pending/cancelled are neutral labeled chips. Saved the accessibility snapshot. Browser console reported only the Storybook favicon.ico 404, not an application error. Shut down the temporary Storybook server and tmux session after inspection.

Commits: `ba54414` — `fix(web): repair story framework imports and sticky focus expansion`; `80d7bd5` — `feat(schema): preserve nullable tool outcomes through SQL API and UI`. Pre-commit lint/full tests passed. This is a P3 schema milestone, not P3 completion; no phase-completion slip has been printed. Evidence/diary commit: `a40ea81`.

## Step 5: Decode typed outputs without inventing wrapper success

Continued P3 by moving output decoding into its own module, preserving text blocks and independent child outcomes before truncation. Missing results now stay pending with null success; an output without authoritative outcome remains unknown. Printed child results never determine the JavaScript wrapper's own success. Added output-before-call storage and native output-line references.

### Prompt Context

**User prompt (verbatim):** (implementation request: see Step 1)

**Additional user prompt (verbatim):** "do you parse the argument to exec_cmomand to surface the actual stuff executed as well? What is exec_command for and why is it nested tool calling ?\n\n[REMINDER] Output a <summary>...</summary> block at the VERY END of your response. This is mandatory."

**Assistant interpretation:** Explain the wrapper/subprocess distinction and expose executed commands from authoritative native events rather than scanning JavaScript arguments as execution proof.

**Inferred user intent:** Understand and query the real activity behind nested Codex tool orchestration.

### What I did
- Added `pkg/adapters/codex/outputs.go` and output regressions; removed old stringify-first parsing from convert.go.
- Recognize observed output/metadata, chunk, fulfilled-child, and renderer envelopes; preserve arbitrary JSON-looking stdout unchanged.
- Decode text/image/unsupported blocks without copying image base64, retain per-block exit/duration evidence, and truncate only decoded display text after metadata extraction.
- Store early outputs until the call arrives in both persisted and legacy parsers; set recoverable source-path/line references.
- Set newly constructed Codex response calls to pending/null instead of default success.

### Why
- Executed command records and wrapper text are different evidence. One or many printed child exit codes cannot establish the wrapper's own outcome.
- `fmt.Sprint` destroys typed output boundaries, and parsing arbitrary JSON into a permissive struct previously discarded real stdout.

### What worked
- New tests cover expected native shapes, malformed metadata, ambiguous multiple child outcomes, missing output, early output, and metadata beyond the display truncation limit; validation is underway.

### What didn't work
- First `go test ./pkg/adapters/codex -count=1` failed at compile time: `pkg/adapters/codex/convert.go:10:2: "strconv" imported and not used`. Output parsing moved that dependency into outputs.go; remove the obsolete import and rerun.
- While locating the existing deduplicator, a guessed `pkg/minitrace/dedup.go` read failed with ENOENT; symbol search found `DeduplicateToolCalls` in util.go. It deduplicates by ID, not command text.

### What I learned
- Output envelopes themselves must be validated by shape. Otherwise normal JSON stdout can be mistaken for a successful or empty tool response.

### What was tricky to build
- A direct tool with multiple result envelopes has ambiguous ownership too; keep its outcome null rather than choose the first child's code. Per-block evidence remains independently inspectable.

### What warrants a second pair of eyes
- Envelope recognition boundaries, whitespace preservation, output provenance, and full-hash/truncation semantics.
- Native CommandExecution reconciliation is still pending; do not interpret this output milestone as complete nested execution recovery.

### What should be done in the future
- Remove obsolete import, pass regression/full tests and lint, then implement native execution lifecycle normalization with conflict and identity tests.

### Code review instructions
- Read outputs.go and outputs_test.go, then the small pending-output integration changes in convert.go.
- Run the focused Codex tests and the independent synthetic CLI oracle in a fresh output directory.

### Technical details
- New output references identify the source output record (`<native-path>#L<n>`), not the invocation line.
- P3 remains active; the phase-start receipt already exists and is not reprinted for this substep.

### Validation checkpoint

Removed the obsolete import. Focused Codex tests, `make lint`, and `go test ./...` now pass. The independent synthetic oracle advances from five failures to only the two native execution assertions; missing outcomes, typed outputs, and early results now pass. Output implementation remains uncommitted while the closely related execution integration is added.

## Step 6: Surface authoritative native command executions

Added native CommandExecution lifecycle reconciliation without examining JavaScript bodies. Native IDs preserve repeated genuine commands as separate executions while repeated notifications enrich one record. Execution records surface shell scripts, original argv, decoded cwd, output streams, duration, exit status, source references, and explicit-or-unknown parent association. Conflicting command/cwd/turn/exit evidence is diagnosed and does not become confident success.

The first focused test run passes, and a fresh CLI conversion now passes all nine assertions in the independent synthetic oracle. This does not finish P3: private-source verification, additional conflict/unsupported-shape hardening, and full validation remain.

### Prompt Context

**User prompt (verbatim):** (implementation request: see Step 1; nested execution clarification: see Step 5)

**Assistant interpretation:** Surface what actually ran as independently queryable operations, not inferred JavaScript intent.

**Inferred user intent:** Query commands, locations, output, and failures accurately despite nested orchestration.

### What I did
- Added `executions.go` and `executions_test.go` and integrated the native collector before final tool linkage.
- Reconcile started/completed events by native execution identity, including duplicate completion and start-after-completion replay.
- Preserve distinct identical-command executions, parse local file-URI cwd, expose shell script or quoted non-shell argv display, and retain typed execution source references.
- Merge execution evidence into an existing direct exec_command only when native call_id explicitly identifies it; otherwise keep separate records with uncertain parentage.
- Add tests for fixture lifecycle states, conflicts, explicit direct linkage, duration across events, quoted argv, and stdout that must not be interpreted as a transport envelope.

### Why
- Structured native events supply stronger execution evidence than static JavaScript parsing and include operations from parallel wrappers.
- Separate records make child failures visible without falsely downgrading or upgrading wrapper transport outcomes.

### What worked
- `go test ./pkg/adapters/codex -count=1` passes.
- `scripts/04-check-synthetic-baseline.py` exits 0 against `/tmp/codex-fidelity-001-synthetic-execution/`; all nine assertions pass, including one failed execution and two distinct identical commands.
- The fixture yields six native execution records, not seven completion/start duplicates, with unknown, pending, and cancelled outcomes intact.

### What didn't work
- No new command/test failure in this checkpoint. Further validation is explicitly pending, not assumed from the synthetic oracle.

### What I learned
- Explicit native call identity lets a direct invocation be enriched without counting its mirrored execution twice. Shared native-turn identity alone is not enough to choose a wrapper parent.

### What was tricky to build
- Output parsing must not run on authoritative stdout: stdout can itself look like a result envelope. The native execution builder stores stdout directly and takes exit code only from the native outcome field.
- Completion replay must not downgrade a finished operation back to pending, while conflicting terminal evidence must remain uncertain and retain source references.

### What warrants a second pair of eyes
- Direct-call versus execution outcome conflict handling, missing-ID namespace collisions, unsupported status/shape diagnostics, and source-level acceptance still need additional review before P3 completion.

### What should be done in the future
- Harden remaining edge cases, audit all local native executions against independent inventory, run full tests/lint/race checks, commit code/evidence, and only then print P3 completion.

### Code review instructions
- Start with collectCodexExecutions, merge, toolCall, and appendCodexExecutions; inspect tests and fresh CLI oracle output in various/p3.

### Technical details
- Synthesized IDs are namespaced `codex-execution:<native-id>`; original native IDs and source lines remain in framework metadata.
- Record kinds currently live in framework metadata; P4 will project counting and structural file evidence through analytical consumers.

### Validation and commit checkpoint

Added bounded source-level fidelity diagnostics (complete counts, at most 100 examples and 129 count keys), visible warning events, and cleaning flags for unsupported native items/orphan outputs. Added direct-response/authoritative-execution exit conflict handling and deterministic namespace-collision avoidance. Native execution sources now retain thread IDs, timestamps, ordinals, and lifecycle timestamps.

Full tests, lint, and Codex race tests pass. An independent native execution audit against six fresh private archives accounts for all 886/1104/306 completed executions in the affected sources, including 77/60/18 failures; there are no missing/invented/duplicate native execution IDs or argv/output/outcome mismatches. Native source SHA-256 fingerprints still match all six original baseline fingerprints. Message coverage remains complete; SQL reports zero malformed map outputs and zero orphan links in every source.

Committed code as `6e656e7` — `feat(codex): normalize native executions and decode typed tool outputs`; pre-commit full tests and lint passed after the final provenance changes. Saved private audit results in various/p3 and the read-only audit script as scripts/08. Full private archives remain in `/tmp/codex-fidelity-001-private-p3/`, outside Git.

Remaining P3 review items before its completion print: legacy exec_command_end/exec-stream outcome consistency, adversarial direct-link/duplicate-output edge cases, and final diagnostics/provenance verification. P4 still owns normalized file targets and record-kind/counting projections; no completion of those requirements is implied by the successful execution audit.

Documentation checkpoint `63dcd2a` saved the execution audit. `docmgr doctor` found that the raw browser snapshot was named `.md` and therefore treated it as a managed document: `outcome-states-snapshot.md frontmatter delimiters '---' not found`. This is raw captured evidence, not an authored doc; rename it to `.txt` without changing its contents and rerun doctor. The shell sequence did not gate the audit commit on doctor's failure; the correction will be a separate traceable commit. Transient browser artifacts were moved intact out of the working tree to `/tmp/codex-fidelity-browser-artifacts.aakEqL/`.

## Step 7: Reconcile legacy terminal outcomes after response results

Legacy exec_command_end evidence previously set an exit code without updating nullable success/status, and early end notifications could disappear. Reconciliation now runs after all calls/results have been collected, preserving terminal source references and keeping conflicting terminal evidence unknown even after replay.

The older exec JSONL parser also treated completion without an exit code as success and malformed codes as failure. Both now remain unknown; numeric exit codes retain their established meaning.

### Prompt Context

**User prompt (verbatim):** (same implementation goal as Step 1; continuation requests continued work on the active objective)

**Assistant interpretation:** Harden legacy formats before completing P3.

**Inferred user intent:** New nullable outcomes must not leave older supported formats internally contradictory.

### What I did
- Added legacy_outcomes.go with a post-collection reconciliation pass.
- Removed the arrival-order-dependent exit assignment from parseSessionJSONL.
- Replaced permissive SafeInt/default-success inference for exec-stream completions with strict numeric evidence.
- Added permutation, terminal conflict/replay, absent-code, invalid-code, and known-code tests.

### Why
- Completion is lifecycle information, not proof of success. A native numeric exit must update both binary and lifecycle fields consistently.

### What worked
- `go test ./pkg/adapters/codex -count=1` passes after the compile fix.
- All three tested call/output/end orderings retain exit 7 as failure.
- Conflicting end notifications remain unknown after a repeated earlier notification.

### What didn't work
- Initial `go test ./pkg/adapters/codex -count=1` failed: `pkg/adapters/codex/legacy_outcomes.go:25:25: cannot index metadata (variable of interface type any)` (also lines 26, 34, 35, 36). mergeMetadataMap returns any; explicitly converted its result with mapValue before indexing, then reran successfully.

### What I learned
- Updating ExitCode independently of Success/Status left a contradictory state even though each field looked plausible in isolation.

### What was tricky to build
- A later repeated notification must not erase earlier terminal conflict. A per-call conflict latch retains uncertainty through the full reconciliation pass.

### What warrants a second pair of eyes
- Legacy terminal records without matching calls remain unsupported; direct execution-link adversarial cases and duplicate response outputs still need P3 review.

### What should be done in the future
- Finish the remaining P3 provenance/diagnostic review before printing its completion; continue P4 file evidence and consumers afterward.

### Code review instructions
- Review reconcileCodexLegacyEnds and the exec-jsonl command_execution branch; run `go test ./pkg/adapters/codex -count=1` and full repository checks.

### Technical details
- execution_end_sources preserves source line, timestamp, and native exit values; response_or_previous_exit_code preserves conflicting prior evidence. No historical command is evaluated.
- Committed as `9658081` — `fix(codex): reconcile legacy terminal outcomes independently of arrival order`. Pre-commit full tests and lint passed; `go test ./pkg/adapters/codex -race -count=1` also passed.

### Budget-limited handoff

The goal reached its 1,000,000-token budget (reported usage 1,037,598) and is not complete. Stop substantive work pending renewed authorization/budget. Documentation through Step 7 is committed in `3f469d2`.

Immediately before the budget stop, review found that multiple distinct native executions sharing one explicit call_id could overwrite one direct invocation during enrichment. An **uncommitted** change in `pkg/adapters/codex/executions.go` now permits enrichment only for a one-to-one original invocation/native execution association, diagnoses multiple execution references, and prevents synthesized execution IDs from being treated as original invocation IDs. It is formatted; existing Codex tests and `git diff --check` pass. Dedicated adversarial regressions and full validation for this change have NOT been run; review/test it before committing. No user changes were discarded.

Remaining: finish P3 identity/provenance/duplicate-output hardening and its completion slip; implement P4 structural file evidence, attempted/confirmed semantics, counts/materialization/presets/file-history consumers with start/done slips; complete P5 documentation and skill caveats, full build/generation/UI/CLI/private acceptance, doctor and requirement-by-requirement audit with its slips. Do not mark the goal complete based on the successful P3 native execution audit alone. Resume only with additional budget/authorization.

## Step 8: Resume with adversarial identity and output replay regressions

The user authorized another million tokens. Reviewed the pending identity guard and proved that distinct native executions sharing one invocation cannot overwrite each other. Also repaired repeated early output retention and terminal conflict replay; all result source lines and decoded block evidence now survive.

### Prompt Context

**User prompt (verbatim):** "1M more"

**Assistant interpretation:** Resume the existing objective with an additional million-token budget, without narrowing acceptance.

**Inferred user intent:** Finish the remaining phases and validation rather than stop at the prior budget boundary.

**Commits (code):** `f0d4811` — require one-to-one identity for execution enrichment; `4e61b7b` — retain replayed output evidence and terminal conflicts.

### What I did
- Tested multiple native execution IDs pointing to one invocation, synthesized-ID masquerading, and direct/native exit conflicts.
- Changed pending output maps to ordered slices in both response-item and legacy parsers.
- Preserved output_sources containing source lines and decoded block evidence; retain the first numeric output code and a persistent conflict latch.
- Kept contradictory response outcomes unknown even after native execution enrichment.
- Used constant-time conflict updates rather than rescanning prior output sources on every notification.

### Why
- Explicit identity is necessary but not sufficient for one-to-one enrichment; one invocation can have multiple native execution references.
- A later repeated result cannot erase earlier contradictory evidence, including when all results precede the invocation.

### What worked
- Focused Codex tests and Codex race tests pass for both commits.
- Commit hooks ran full repository tests and lint successfully for both commits.
- git diff --check passed.

### What didn't work
- Two exact edit attempts for pendingOutputs failed because oldText matched both parser sections; neither changed the file. Added unique neighboring context and leading newlines to avoid suffix matches across differing indentation, then applied all replacements in one call.

### What I learned
- Early-output retention must preserve every notification, not just the latest value per call ID.

### What was tricky to build
- Native enrichment and legacy terminal reconciliation could overwrite an already diagnosed response conflict. Both now honor the existing conflict latch.

### What warrants a second pair of eyes
- The last decoded result remains the display value; prior sources retain their decoded metadata and native line references, not duplicated private bodies. Conflicting binary evidence remains unknown.

### What should be done in the future
- Finish P3 provenance/diagnostics acceptance and print its completion, then implement P4 and P5 as contracted.

### Code review instructions
- Review execution_identity_test.go and output_replay_test.go, then appendCodexExecutions and codexNativeOutput.apply. Run go test ./pkg/adapters/codex -race -count=1.

### Technical details
- One-to-one enrichment requires one original invocation, one native execution reference, and an original—not synthesized—target index.
- Output source accumulation is linear in notification count; outcome conflict checking does not repeatedly scan history.

## Step 9: Close P3 and start the structural file evidence schema

Fresh P3 conversion of all six private sources passes independent execution and message audits with unchanged baseline hashes. It accounts for all 2,296 executions and 155 failures; the synthetic oracle also passes. Printed the P3 completion and P4 start slips with positive physical-print receipts.

Started P4 with a typed multi-target file evidence model and SQLite projection. File effects have their own nullable outcomes; shell success cannot silently confirm every target. The scalar path remains supported for existing adapters but does not override an explicit structural target list.

### Prompt Context

**User prompt (verbatim):** (same as Step 8; continuing the renewed objective)

**Assistant interpretation:** Advance to P4 only after fresh P3 acceptance and physical phase boundaries.

**Inferred user intent:** Turn recovered execution activity into trustworthy file evidence and analytical consumers.

**Commit (code):** `f0a5377` — represent structural file evidence and project all targets.

### What I did
- Fresh private conversion into /tmp/codex-fidelity-001-p3-final and synthetic conversion into /tmp/codex-fidelity-001-p3-final-synthetic.
- Ran scripts/08-audit-execution-coverage.py, scripts/05-audit-message-coverage.py and scripts/04-check-synthetic-baseline.py; saved final P3 reports in various/p3.
- Checked every source fingerprint against the original native inventory.
- Printed various/slips/06-p3-done and 07-p4-start via work_slip.py; both receipts report printed: true, HTTP 200, printer ok.
- Added FileTarget with path/native path, operation, evidence kind/status, nullable success, cwd/resolution, and source reference.
- Added first-class record_kind and EffectiveRecordKind, projecting kinds into SQLite.
- Materialized every structural target with its own outcome and evidence metadata; bumped normalized SQLite schema to v5.

### Why
- Tool outcome and file-effect outcome are different claims. A successful shell invocation can contain unexecuted or failed statements.
- A scalar first-path field cannot represent a multi-file operation.

### What worked
- Fresh P3 private and synthetic acceptance passed; all six source hashes match baseline.
- New SQL regression projects two targets, neither inheriting the shell's true success, and excludes an explicit-empty wrapper target list despite its scalar convenience path.
- Focused minitrace/database tests pass. Commit hooks passed full tests and lint.

### What didn't work
- No command/test failure in this step.

### What I learned
- nil versus an explicit empty target list distinguishes an older scalar-only adapter from an adapter that explicitly found no structural file evidence.

### What was tricky to build
- Preserving existing non-Codex scalar behavior without contaminating new structural evidence: legacy_scalar/reported marks that older claim, while explicit targets never inherit the tool outcome.

### What warrants a second pair of eyes
- P4 extraction, counters, API/UI projections, and file-history integration are still outstanding. The schema milestone alone does not establish Codex file fidelity.

### What should be done in the future
- Implement direct patch/native FileChange extraction and bounded shell redirects, then consumers and negative tests before P4 completion.

### Code review instructions
- Review pkg/minitrace/file_evidence.go, filesTable, MaterializeSession and TestStructuralFileTargetsDoNotInheritShellSuccess.

### Technical details
- files gains target_ordinal, evidence_kind, evidence_status, cwd, resolved, native_path and source_reference. tool_calls gains record_kind.
- Tool-record count is unchanged; file targets do not become additional tool calls.

## Step 10: Extract structural attempts and authoritative native file effects

Implemented explicit patch-header targets, bounded literal shell redirects, and native FileChange lifecycle records. Replaced Codex's regex-derived scalar path with the first structural target; opaque JavaScript and message histories never establish file evidence. Native file changes retain distinct identities, source references, move source/destination effects, and uncertain parentage.

A fresh private conversion independently matches all 472 native FileChange events and 995 targets, including five rename destinations, while the execution audit still passes. This verifies extraction, not the still-pending counter/history/API integration.

### Prompt Context

**User prompt (verbatim):** (same as Step 8)

**Assistant interpretation:** Implement P4 structural evidence against actual native shapes, with adversarial tests and independent audits.

**Inferred user intent:** Trust file history without treating code mentions or shell success as proven writes.

**Commit (code):** `24bea46` — extract structural file attempts and native file effects.

### What I did
- Added shell_targets.go: a size-bounded lexer and deliberately narrow straight-line shell grammar for literal <, > and >> redirections.
- Added files.go: typed patch/direct-path/redirect targets, lexical cwd resolution only from explicit tool/execution cwd, and structural scalar convenience paths.
- Added file_changes.go: native FileChange deduplication, source provenance, native completed effects, rename pairs, conflict preservation, and explicit file_change record kind.
- Added tests for quotes, comments, escapes, heredocs, substitutions, conditional branches, cwd mutation, search operands, multi-file patches, repeated native changes and conflicts.
- Hardened missing native identity namespaces for both executions and file changes; source-line synthetic IDs cannot collide with literal native IDs or masquerade as native identities.
- Added scripts/09-audit-file-changes.py and saved count/identity-only private audit reports under various/p4; private archives remain under /tmp.

### Why
- Native FileChange evidence can confirm applied effects; a shell exit code cannot confirm individual redirections.
- Native change maps include move_path; recording only original keys loses destinations.

### What worked
- Focused and race Codex tests pass. Commit hooks passed full tests and lint.
- Independent audit: 472 native file events, 995 targets, no missing/invented/duplicate identities, target mismatches or provenance/outcome mismatches.
- Independent execution audit remains passing on the same fresh P4 archives.

### What didn't work
- No test/build failure in this step. One discovery rg returned exit 1 because no codexCWD helper exists; inspected execution.toolCall's inline URI handling instead.

### What I learned
- All 990 source-map file paths in the local FileChange baseline are absolute; five moves create five additional target rows. None requires guessed session cwd.

### What was tricky to build
- Reject unsupported shell syntax before accepting any targets, so heredoc bodies, process substitutions and conditional branches cannot leak into the ledger. Quoted > characters remain ordinary text.
- Conflicting native file notifications retain the union of targets as unconfirmed attempts, including after later replay of an earlier notification.

### What warrants a second pair of eyes
- The shell grammar is intentionally conservative: pipelines, descriptor duplication, expansions, compound control flow and cwd-changing/eval/source commands yield diagnostics rather than inferred paths. This is not a general shell interpreter.
- Resolved means lexical absolute-path resolution, not filesystem/symlink canonicalization. Native paths remain available.
- Native FileChange status completed is treated as applied file-effect evidence; failed/cancelled changes do not assert success/failure of each individual target.

### What should be done in the future
- Project record-kind/file-touch counters and update file-history, presets, API/UI and documentation; add final consumer acceptance before P4 completion.

### Code review instructions
- Start with files_test.go and missing_identity_test.go, then review shellWords/literalShellTargets, finalizeCodexFileEvidence and appendCodexFileChanges. Re-run scripts/09 against /tmp/codex-fidelity-001-p4-extraction.

### Technical details
- Direct patch/redirect/path-argument evidence is attempted with null success. Native completed FileChange targets are confirmed/true. Contradictory native change outcomes remain unknown.
- FileChange records use codex-file-change:<native-id>, distinct from execution records; target rows do not create additional tool records.

## Step 11: Separate activity counters and switch file history to structural evidence

Added explicit counts for ordinary records, orchestration, executions, native file changes, model invocations, target rows and confirmed targets. SQLite sessions and metrics derive them from actual records, including older archives with no serialized counters. Orchestration no longer inflates operation counts.

Codex file-history now joins normalized file targets and does not inspect wrapper JavaScript/arguments. File presets use the same evidence ledger. End-to-end synthetic checks prove both redirect targets are visible as attempts, the false-branch target is absent, and neither emitter nor prior file existence is invented.

### Prompt Context

**User prompt (verbatim):** (same as Step 8)

**Assistant interpretation:** Complete analytical projections without confusing model requests, executions, targets, or confirmed effects.

**Inferred user intent:** Query trustworthy activity and file history across supported adapters.

**Commit (code):** `add1bb4` — separate activity counts and consume structural file evidence.

### What I did
- Added ActivityCounts and shared CountToolActivity; ordinary/orchestration/execution/file-change counts partition normalized records, while explicitly enriched direct executions also count as model invocations.
- Projected counters into sessions and metrics tables from actual tool records.
- Updated session-list, operation-breakdown, file-timeline and file-operations presets.
- Replaced Codex file-history regex inference with a files-table join, retaining non-Codex extraction behavior.
- Exposed evidence metadata and attempted/confirmed summary counts; Codex prior-existence inference is null.
- Added a unit test for counter separation and scripts/10-check-history-consumers.py for repeatable checkout-CLI acceptance.

### Why
- Tool-record totals are not model-invocation or executed-process totals, and target operation names alone are not proof of effects.

### What worked
- Focused core/database/command tests pass; commit hooks passed full tests and lint.
- End-to-end script passed: first and second targets each appear once, false-branch target never appears, file success stays null, confirmed-target count is zero.
- Synthetic SQL counters partition nine records into one ordinary, two orchestration and six execution records; model invocations=3, file targets=2.

### What didn't work
- Inspecting the first CLI result exposed a pre-existing JS coercion bug: null turn_index selected user turn zero as preceding instruction. Added an explicit null guard and reran successfully.
- That result also labeled a redirect target as created-before-visible-history from MODIFY alone. Removed this inference for structural Codex evidence and verified null instead.

### What I learned
- A valid SQL null association can still become a false association in a JavaScript consumer unless null handling is explicit.

### What was tricky to build
- Preserve scalar/argument inference for non-Codex adapters while ensuring old Codex archives are not silently treated as structural evidence; old Codex archives must be reconverted for file-history.

### What warrants a second pair of eyes
- Target-row counts intentionally count evidence occurrences, not unique paths. Model counts include one-to-one enriched direct invocations without counting every native child as a model call.
- API/UI counter/target projection and broad non-Codex history smoke tests remain for the next P4 work.

### What should be done in the future
- Finish API/UI and remaining consumers, then run P4 acceptance and print its completion.

### Code review instructions
- Review CountToolActivity, activityValues, the file-history SQL/conditional extraction, and scripts/10. Run the script against /tmp/codex-fidelity-001-p4-history/active/*/*.minitrace.json (quote the glob).

### Technical details
- Snapshot artifacts under various/p4 retain history-first, history-false-branch, activity-counts and history-consumer-acceptance JSON.
- created_before_visible_history is explicitly unknown for Codex structural history; attempted_targets and confirmed_effects expose what is actually supported.

## Step 12: Carry typed targets and counters through protobuf and UI

Added protobuf file targets, record kinds and activity counters, mapped them through both API response layers and actual TypeScript decoding, and displayed target evidence in expanded tool rows. Session summaries distinguish records, invocations, executions and confirmed targets.

### Prompt Context

**User prompt (verbatim):** (same as Step 8)

**Assistant interpretation:** Preserve P4 evidence across API/UI boundaries, not only SQLite.

**Inferred user intent:** Inspect the same faithful outcomes and targets in the browser.

**Commit (code):** `ccb63de` — expose structural file evidence and activity counts in API and UI.

### What I did
- Extended proto schemas, ran buf generate, updated Go/TypeScript mappings and displayed multi-target evidence.
- Derived detail counters from actual records so older serialized metrics cannot falsely report zero.
- Added API/proto regression tests and StructuralFileAttempts browser story.
- Extended scripts/06 and /07 to exercise actual Go protojson-to-TypeScript target/counter decoding alongside all five outcome states.

### Why
- A successful process must not turn nullable target outcomes into successful writes in the UI.

### What worked
- Go serve tests and commit-hook full tests/lint pass.
- pnpm build and lint pass (0 lint errors, 13 existing warnings; existing bundle-size warning).
- 23 focused Storybook browser tests pass.
- Go-to-actual-TypeScript roundtrip passes target nullability, provenance, kinds, counters and five outcome states.
- Inspected various/p4/structural-file-attempts.png: both targets show attempted/unknown despite the process success icon; cwd and source lines are visible.

### What didn't work
- Initial serve build: handlers_sessions_v2.go:98:20: undefined: minitrace. Added the missing import and reran successfully.
- Browser MCP navigation failed: Browser is already in use for /home/manuel/.cache/ms-playwright/mcp-chrome-profile. Used a separately launched isolated Playwright browser, not profile deletion or killing another session.

### What I learned
- Node's native TypeScript runner needs the explicit .ts extension for the new activityMetrics import; project compiler settings support it.

### What was tricky to build
- Two identical summary-adapter blocks needed the same spread insertion; applied a counted replacement asserting exactly two matches instead of ambiguous exact-edit calls.

### What warrants a second pair of eyes
- Finish auditing API execution framework provenance and output truncation-reference fields, plus remaining analytical/legacy consumers before P4 acceptance. This milestone exposes file-target source references but does not yet cover every execution provenance field.

### What should be done in the future
- Complete remaining consumer/provenance checks, then P4 acceptance and P5 full validation/documentation.

### Code review instructions
- Review file_evidence_test.go, protoToolCallInput, activityMetrics.ts and StructuralFileAttempts; rerun scripts/06 | scripts/07 and focused Storybook tests.

### Technical details
- Storybook ran in tmux codex-fidelity-p4-storybook on port 16006. Stopped with lsof-who -p 16006 -k and tmux cleanup; isolated browser closed normally.
- Validation logs and screenshot/text evidence are retained under various/p4.

### Second budget-limited handoff

The additional 1,000,000-token allocation is exhausted (reported usage 1,113,179; 36 minutes). The goal remains incomplete. Code through `ccb63de` and evidence through `a70b40c` are committed; no new code was changed during the final inspection.

Next concrete repair: ToolCall API responses still omit execution framework_metadata and output FullReference/FullBytes/FullHash. Carry these through Go responses, protobuf, TypeScript and relevant UI, then test real archive-to-API conversion. Existing protoStruct returns nil on structpb.NewStruct error; avoid silently dropping newly exposed provenance, and account for typed Go slices/pointers versus JSON-loaded values.

Remaining P4: finish provenance/consumer auditing, broad non-Codex history regressions and final extraction edge-case review (including native FileChange conflicting content with unchanged path/status), then full P4 acceptance and its completion print. P5 remains: adapter docs and transcript-analysis skill caveats, full make all/build/generation/logcopter checks, frontend checks, help/doctor, fresh six-source SQL/identity/receipt/private audits, requirement-by-requirement completion audit and P5 start/done prints. P3 completion and P4 start are already physically printed with receipts and must not be repeated unnecessarily.

Stop substantive work until renewed budget/authorization. No completion marking is justified by the passing intermediate audits.
