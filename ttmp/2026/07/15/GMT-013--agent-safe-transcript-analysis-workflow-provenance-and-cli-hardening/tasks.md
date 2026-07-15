# Tasks

## How to use this task plan

- Task IDs are stable and should be quoted in commits, diary steps, and changelog entries.
- Complete tasks in order within each phase unless a task explicitly says it can run in parallel.
- A task is complete only when its **Done when** condition is satisfied and its listed evidence has been recorded.
- Product implementation is intentionally separated from the completed research/design work.
- After each implementation task: format changed files, run the narrow tests, update the GMT-013 diary, relate materially changed files with docmgr, and record the commit hash.
- At each phase gate: run the full validation commands, update this checklist, and create a focused phase changelog entry.

## Completed research and design

- [x] **D.1 — Create the GMT-013 ticket workspace.**
  - Output: index, design document, diary, tasks, and changelog.
  - Evidence: ticket directory exists under `ttmp/2026/07/15/`.
- [x] **D.2 — Map current source discovery and explicit-source selection.**
  - Files reviewed: `cmd/go-minitrace/cmds/discover/`, `cmd/go-minitrace/cmds/convert/sources.go`, adapter discovery implementations.
- [x] **D.3 — Map adapter conversion and native identity handling.**
  - Files reviewed: Codex, Pi, and Claude Code conversion entrypoints and adapters.
- [x] **D.4 — Map archive publication and manifest reconciliation.**
  - Files reviewed: `pkg/minitrace/archive.go`, `pkg/minitrace/archive_test.go`.
- [x] **D.5 — Map normalized query execution and output emission.**
  - Files reviewed: `pkg/minitracedb/query.go`, query command runtime, JS runtime, and Glazed JSON formatter.
- [x] **D.6 — Map validation and embedded-help registration.**
  - Files reviewed: `pkg/validate/`, `cmd/go-minitrace/cmds/validate/`, `pkg/doc/doc.go`, and all embedded pages.
- [x] **D.7 — Reconstruct the two isolated external-agent evaluations.**
  - Evidence: preserved RAG attribution run and go-go-goja PR #95 holdout documentation.
- [x] **D.8 — Correct the Codex collision diagnosis.**
  - Finding: distinct child IDs are replaced by later replayed parent `session_meta` records during conversion.
- [x] **D.9 — Define identity, collision, batch, query, output, and attribution contracts.**
  - Output: design sections 5–7.
- [x] **D.10 — Define the documentation consolidation plan.**
  - Decision: no new feature help pages; merge overlapping pages into canonical owners.
- [x] **D.11 — Write the implementation guide and investigation diary.**
  - Output: design guide and chronological diary.
- [x] **D.12 — Validate, commit, push, and publish the design package.**
  - Validation: `docmgr doctor` passed; release/test/lint passed sequentially.
  - Commit: `b062435b8f24bc6cad8375562e9d46a98928431d`.
  - Publication: reMarkable `/ai/2026/07/15/GMT-013`.

## Phase 0 — Freeze failure contracts with regression fixtures

**Goal:** turn every observed defect into a deterministic failing test before changing production behavior.

**Phase dependency:** D.1–D.12 complete.

- [x] **P0.1 — Create the Phase 0 implementation branch and baseline record.**
  - Record `git rev-parse HEAD`, `go version`, relevant dependency versions, and clean working-tree state in the diary.
  - Run `go test ./... -count=1` before editing.
  - Done when: the baseline command output and any pre-existing failures are recorded verbatim.
- [x] **P0.2 — Inventory existing adapter and archive fixtures.**
  - Inspect `pkg/adapters/codex/testdata/`, Codex conversion tests, archive tests, query integration tests, and CLI test helpers.
  - Produce a short fixture map in the diary; do not add a separate planning document.
  - Done when: each planned fixture has an identified destination and nearest reusable test helper.
- [x] **P0.3 — Minimize a Codex child-header/parent-replay source fixture.**
  - Preserve only structural records needed to reproduce: child `session_meta`, direct/nested parent identity, representative child event, later parent `session_meta`.
  - Replace private content, paths, models, and IDs with deterministic synthetic values while preserving record shape and ordering.
  - Files: `pkg/adapters/codex/testdata/`.
  - Done when: the fixture contains no private transcript text and the current adapter still reproduces the wrong parent archive ID.
- [x] **P0.4 — Add a failing Codex identity regression test.**
  - Assert child `Session.ID`, child `provenance.original_session_id`, and parent `coordination.predecessor_session`.
  - Also assert that later parent metadata does not become the archive identity.
  - Files: `pkg/adapters/codex/convert_test.go` or the nearest existing Codex test file.
  - Done when: the test fails for the expected current last-record-wins reason, not because the fixture is malformed.
- [x] **P0.5 — Add locator/header mismatch cases.**
  - Cases: matching locator/header, empty locator, conflicting locator/header, malformed first header, later replay header.
  - Define expected warning/error behavior in test names even if implementation comes in Phase 1.
  - Done when: the desired precedence table is executable as table-driven tests.
- [x] **P0.6 — Add archive collision fixtures and failing tests.**
  - Cases: destination absent; same ID/same bytes; same ID/different bytes; same ID/different source path; legacy archive without fingerprint.
  - Assert destination bytes and manifests remain unchanged after a rejected collision.
  - Files: `pkg/minitrace/archive_test.go`.
  - Done when: at least one test demonstrates the current silent overwrite.
- [x] **P0.7 — Add partial-batch failure characterization.**
  - Convert three synthetic sources where the second fails.
  - Record which files/manifests current code leaves behind.
  - Files: converter command tests or a new shared batch-runner test location selected during P0.2.
  - Done when: current partial publication is captured without yet changing behavior.
- [x] **P0.8 — Add zero-row SQL JSON integration coverage.**
  - Execute `query run` through the Cobra/Glazed command path with `--output json` and a zero-row query.
  - Assert exact desired bytes `[]\n`.
  - Done when: the test fails because current streaming output is empty, not because command construction failed.
- [x] **P0.9 — Add zero-row JS and non-query command coverage.**
  - Cover a JS command returning an empty array and one row-producing command with no matches, such as discovery or validation.
  - Separate JSON-array semantics from explicit NDJSON semantics.
  - Done when: the formatter-level scope of the defect is demonstrated.
- [x] **P0.10 — Add query error and truncation contract tests.**
  - Cases: invalid SQL, denied read, timeout, max rows reached, output processor failure.
  - Assert desired exit/non-exit and receipt behavior in test names; implementation follows in Phase 3.
  - Done when: successful-zero, failed, and incomplete/truncated outcomes are distinct test cases.
- [x] **P0.11 — Document Phase 0 findings in the diary.**
  - Record exact failing test names, commands, and failure messages.
  - Relate every fixture/test file that materially shaped the contract.
  - Done when: another engineer can reproduce each failure from the diary.
- [x] **P0.12 — Phase 0 gate.**
  - Run narrow suites for Codex, archive, query, and command integration.
  - Commit fixtures/tests without production fixes using a message such as `Test agent-safe transcript failure contracts`.
  - Done when: failures expected to remain red are explicitly isolated or marked according to repository policy, and the phase commit is recorded.

## Phase 1 — Preserve source identity and prevent destructive collisions

**Goal:** ensure native child identity survives conversion and conflicting archive writes fail before modifying output.

**Phase dependency:** P0.1–P0.12 complete.

- [x] **P1.1 — Introduce the source identity data structure.**
  - Add fields for native session ID, parent session ID, path, format, role, identity basis, SHA-256, and size.
  - Files: `pkg/adapters/types.go` and focused tests.
  - Done when: the type has documented field semantics and no adapter-specific payload fields.
- [x] **P1.2 — Implement a shared source fingerprint helper.**
  - Hash exact raw bytes with SHA-256; return size and normalized path.
  - Cover empty files, unreadable files, symlinks/path normalization, and deterministic hashes.
  - Done when: helper tests pass and adapters do not duplicate hashing logic.
- [x] **P1.3 — Implement Codex header identity inspection.**
  - Parse the first valid native session header before full conversion.
  - Support direct `parent_thread_id` and nested `source.subagent.thread_spawn.parent_thread_id` without assuming `.source` is always an object.
  - Done when: source descriptors correctly classify parent, subagent, and unknown shapes.
- [x] **P1.4 — Lock Codex conversion identity to the inspected header.**
  - Prevent later replayed `session_meta` IDs from replacing the native child ID.
  - Populate child ID in `Session.ID` and `provenance.original_session_id`.
  - Populate parent ID in `coordination.predecessor_session` and documented framework metadata.
  - Done when: P0.4 identity regression passes.
- [ ] **P1.5 — Define later-metadata merge precedence.**
  - Enumerate which child/header fields are immutable and which later records may update.
  - Emit a structured warning for replay/mismatch records with record index.
  - Done when: precedence is covered by table-driven tests and documented in adapter reference draft notes.
- [x] **P1.6 — Extend archive provenance additively.**
  - Add `source_fingerprint` and `identity_basis`; update normalized SQLite materialization only if these fields must be queryable in Phase 1.
  - Update schema docs/tests and JSON round-trip tests.
  - Done when: old archives still decode and new archives preserve evidence fields.
- [ ] **P1.7 — Separate archive serialization from publication.**
  - Create a deterministic serialization helper and a publication API returning created/unchanged/replaced status.
  - Keep existing callers compiling while migrating them in focused commits; avoid silent behavior shims.
  - Done when: serialization can be tested without touching disk.
- [ ] **P1.8 — Implement default collision detection.**
  - Destination absent → create.
  - Matching source fingerprint → unchanged/idempotent.
  - Different fingerprint → collision error before write.
  - Legacy missing fingerprint → conservative error unless explicit replacement policy applies.
  - Done when: P0.6 cases pass and rejected writes preserve original bytes.
- [ ] **P1.9 — Implement explicit replacement policy.**
  - Add `--collision error|replace` to converter settings.
  - Record previous and new hashes when replacement is explicit.
  - Done when: replacement is impossible without an explicit flag and is visible in result/receipt data.
- [ ] **P1.10 — Make individual archive writes atomic.**
  - Write temp file in destination filesystem, sync as appropriate, and rename.
  - Clean temp files on failures.
  - Done when: interruption/error tests never leave a truncated destination.
- [ ] **P1.11 — Introduce conversion result and warning types.**
  - Return session, source identity, and structured warnings from the shared conversion boundary.
  - Migrate Codex first; document migration path for Pi and Claude Code.
  - Done when: Codex command output and tests can report warnings without parsing error strings.
- [ ] **P1.12 — Implement a shared batch preflight.**
  - Resolve, normalize, deduplicate, sort, inspect, and fingerprint all requested sources before publication.
  - Detect duplicate paths and conflicting native IDs during preflight.
  - Done when: a collision in input N is found before output 1 is published in strict mode.
- [ ] **P1.13 — Implement staged batch publication.**
  - Stage converted archives and derived manifests under a run-specific directory.
  - Publish only after all strict-mode conversions and collision checks pass.
  - Document any remaining multi-file crash window instead of claiming unsupported global atomicity.
  - Done when: P0.7 strict-batch characterization becomes all-or-nothing under tested failure conditions.
- [ ] **P1.14 — Add partial mode and process semantics.**
  - Define `--allow-partial`; without it, any failed input yields non-zero status.
  - Ensure output rows cannot make a failed batch look successful.
  - Done when: created, unchanged, failed, collided, and skipped counts reconcile with requested inputs.
- [ ] **P1.15 — Implement conversion run receipt v1.**
  - Record tool/adapter version, settings, sorted inputs, IDs, parent IDs, hashes, statuses, outputs, warnings, summary, timestamps, and completeness.
  - Write receipt atomically on success and failure when `--run-record` is supplied.
  - Done when: JSON round-trip/schema tests and deterministic ordering tests pass.
- [ ] **P1.16 — Migrate Codex command to the shared batch runner.**
  - Preserve `--source-session` and `--source-list` behavior.
  - Remove duplicate write/manifest logic after migration.
  - Done when: existing Codex command tests plus new identity/collision/receipt tests pass.
- [ ] **P1.17 — Migrate Pi and Claude Code commands.**
  - Reuse batch/collision/receipt behavior without changing adapter-specific parsing semantics.
  - Done when: existing adapter suites pass and collision behavior is consistent across all three primary JSONL adapters.
- [ ] **P1.18 — Phase 1 documentation and phase gate.**
  - Update diary, changelog, `convert.md`, and adapter reference sections owned by this behavior.
  - Run `gofmt`, targeted tests, `go test ./... -count=1`, race tests for changed packages, lint, and archive fixture checks.
  - Done when: phase tasks are checked, files are related, commit hashes are recorded, and no destructive collision path remains.

## Phase 2 — Extend validation to archives, manifests, and receipts

**Goal:** replace directory-shape-sensitive audit scripts with native, machine-readable archive integrity checks.

**Phase dependency:** P1.1–P1.18 complete.

- [ ] **P2.1 — Define validation finding codes and severities.**
  - Add stable code, severity, path, session ID, and details fields.
  - Preserve current human-readable columns while making JSON reliable.
  - Done when: finding serialization and ordering tests pass.
- [ ] **P2.2 — Add selectable validation checks.**
  - Implement `--checks syntax,schema,archive` with explicit validation of unknown names.
  - Done when: existing syntax behavior remains available and archive checking is opt-in until documented rollout.
- [ ] **P2.3 — Detect archive roots robustly.**
  - Support a direct root containing `manifest.json` and `active/` and a parent containing multiple framework roots.
  - Avoid constructing `active/active` paths.
  - Done when: fixtures cover both layouts and the holdout's wrong-level invocation succeeds or reports a clear root error.
- [ ] **P2.4 — Validate archive filenames and payload identities.**
  - Check sanitized filename against `Session.ID` and detect duplicate IDs across files.
  - Done when: mismatches and duplicates produce deterministic error findings.
- [ ] **P2.5 — Validate period placement.**
  - Compare `timing.started_at` month with `active/YYYY-MM`; handle `unknown` explicitly.
  - Done when: misplaced archives are reported without modifying them.
- [ ] **P2.6 — Validate root and period manifests.**
  - Compare period list, counts, file names, IDs, selected metadata, and total statistics against actual archives.
  - Done when: stale counts, orphan entries, missing manifests, and wrong file paths each have dedicated finding codes.
- [ ] **P2.7 — Detect orphan archives and orphan manifest entries.**
  - Report both directions and preserve enough path detail for repair.
  - Done when: a mixed fixture yields exactly the expected findings.
- [ ] **P2.8 — Validate source identity consistency.**
  - Detect conflicting fingerprints for one session ID and suspicious reuse of one fingerprint under different IDs.
  - Treat legacy missing fingerprints as informational or warning according to documented policy.
  - Done when: identity findings match Phase 1 collision semantics.
- [ ] **P2.9 — Validate conversion receipts.**
  - Check receipt schema/version, output paths, output hashes, summary counts, completeness, and archive existence.
  - Done when: a valid Phase 1 receipt passes and tampered/missing output cases fail predictably.
- [ ] **P2.10 — Add non-zero exit behavior for error findings.**
  - Warnings remain inspectable without masking errors.
  - Add an explicit override only if a real workflow requires it.
  - Done when: machine tests assert rows plus process status.
- [ ] **P2.11 — Make manifest writes atomic and add a rebuild path.**
  - Reuse validated archive scanning; decide whether repair is a flag on `validate` or remains a separate future operation.
  - Do not silently repair during validation.
  - Done when: interrupted writes preserve the previous valid manifest.
- [ ] **P2.12 — Retire skill dependence on `audit_manifests.sh`.**
  - Update the skill to call native validation; retain the helper only if needed for older released binaries and label that scope.
  - Done when: the isolated holdout validation sequence uses no directory-shape-specific audit script.
- [ ] **P2.13 — Phase 2 documentation and phase gate.**
  - Update `validate.md`, `troubleshooting.md`, diary, tasks, and changelog.
  - Run validation package tests, CLI integration tests, full Go tests, lint, and real ticket-local archive checks.
  - Done when: one command validates single-root and multi-root analysis layouts with machine-readable findings.

## Phase 3 — Make query evidence reproducible and structured output total

**Goal:** ensure saved queries, exact archive inputs, result completeness, valid empty output, and errors are all durable and machine-readable.

**Phase dependency:** P0 query/output tests complete; Phase 1 receipt primitives available where reusable.

- [ ] **P3.1 — Reproduce and isolate the Glazed empty-array formatter defect upstream.**
  - Add a formatter test where `Close` occurs before any `OutputRow` call in JSON array mode.
  - Confirm non-streaming table and individual-row modes are unaffected.
  - Done when: the upstream test fails for the exact empty-stream reason.
- [ ] **P3.2 — Implement the formatter fix at the correct boundary.**
  - Emit `[]\n` only for JSON array mode with zero rows.
  - Do not emit fake rows and do not change explicit JSONL/individual-row semantics.
  - Done when: upstream formatter tests pass.
- [ ] **P3.3 — Update/pin the Glazed dependency.**
  - Record upstream commit/release and dependency diff.
  - Done when: go-minitrace zero-row integration tests pass against the pinned version.
- [ ] **P3.4 — Add go-minitrace cross-command empty-output tests.**
  - Cover SQL, JS empty array, discover no matches, and validate no targets/findings as applicable.
  - Assert exact bytes and exit status.
  - Done when: no JSON array command emits an invalid empty byte stream.
- [ ] **P3.5 — Implement deterministic archive inventory resolution.**
  - Expand globs, normalize paths, deduplicate, sort, hash files, and compute an inventory hash.
  - Record unmatched globs explicitly.
  - Done when: glob order does not change the inventory hash.
- [ ] **P3.6 — Define query run receipt v1.**
  - Include tool/engine version, query kind/path/hash, optional inline text policy, archive globs/inventory, limits, timestamps, result columns/count/truncation, status, and stable error code.
  - Done when: Go round-trip and deterministic-order tests pass.
- [ ] **P3.7 — Add `--run-record` to `query run`.**
  - Write success and failure receipts atomically after output processing finishes.
  - Done when: processor/output failures produce failure receipts rather than false success.
- [ ] **P3.8 — Capture saved SQL provenance.**
  - Resolve absolute SQL-file path and SHA-256 before execution.
  - Named presets record preset name plus embedded SQL hash/version.
  - Done when: changing SQL bytes changes the receipt hash.
- [ ] **P3.9 — Define inline SQL behavior for strict runs.**
  - Reject inline `--sql` in strict mode unless its exact text is captured in a receipt.
  - Keep interactive inline SQL available.
  - Done when: tests distinguish interactive and strict profiles.
- [ ] **P3.10 — Add truncation completeness policy.**
  - Interactive mode reports `truncated=true` while returning results.
  - Strict mode exits non-zero unless `--allow-truncated` is explicit.
  - Done when: truncation cannot be mistaken for a complete report.
- [ ] **P3.11 — Unify SQL and JS machine-readable failures.**
  - Define one envelope shape and stable error categories while retaining concise stderr.
  - Ensure stdout has either result data or one documented error object, never mixed prose.
  - Done when: SQL validation, SQLite execution, JS runtime, timeout, and output failures have consistent tests.
- [ ] **P3.12 — Expose result metadata to receipt generation before row flattening.**
  - Preserve columns, count, and truncation across the `RunQueryTargetIntoProcessor` boundary.
  - Done when: receipt generation does not infer metadata from emitted rows.
- [ ] **P3.13 — Add direct-output-file examples and tests.**
  - Use Glazed output-file support; avoid producer pipelines in evidence workflows.
  - Done when: documented commands create parseable files for zero, one, and many rows.
- [ ] **P3.14 — Add a saved-query enforcement test fixture.**
  - Re-run representative session-profile, commit-candidate, and commit-verification SQL from files.
  - Confirm each receipt points to the correct path/hash.
  - Done when: the previous “six unsaved report queries” failure cannot occur in strict examples.
- [ ] **P3.15 — Phase 3 documentation and phase gate.**
  - Update `query.md`, `output-formats.md`, `writing-queries.md`, troubleshooting, diary, tasks, and changelog.
  - Run upstream formatter tests, go-minitrace query/JS integration tests, full tests, race tests, and lint.
  - Done when: zero-row, error, and truncated runs have valid bytes, correct status, and durable receipts.

## Phase 4 — Add attribution-oriented discovery and a strict execution profile

**Goal:** let agents produce deterministic bounded candidate inventories without format-specific grep scripts.

**Phase dependency:** Phase 1 source identities and Phase 3 receipts available.

- [ ] **P4.1 — Define shared discovery result fields.**
  - Native ID, parent ID, role, path, format, cwd, started time, optional fingerprint, and parse warnings.
  - Done when: Codex, Pi, and Claude Code can map to one documented result schema.
- [ ] **P4.2 — Add common time filters.**
  - Implement inclusive/exclusive boundary semantics for `--since` and `--until` and document timezone handling.
  - Done when: table-driven tests cover boundaries and missing timestamps.
- [ ] **P4.3 — Add normalized cwd filtering.**
  - Support `exact` and `descendant` modes; normalize absolute paths without resolving unrelated nonexistent paths incorrectly.
  - Done when: prefix-collision cases such as `/repo` vs `/repo-old` are tested.
- [ ] **P4.4 — Add source-role filtering.**
  - Support parent, subagent, fork, and unknown where adapters can prove the role.
  - Done when: unknown remains explicit rather than being guessed as parent.
- [ ] **P4.5 — Add opt-in discovery fingerprinting.**
  - Avoid hashing an entire store unless requested.
  - Done when: output and performance behavior are documented and tested.
- [ ] **P4.6 — Implement Codex enriched discovery.**
  - Reuse Phase 1 header inspection; handle polymorphic source metadata.
  - Done when: no external jq source-shape logic is required for Codex candidate inventory.
- [ ] **P4.7 — Implement Pi enriched discovery.**
  - Preserve Pi native ID/lineage semantics and cwd evidence without reading live stores outside the requested root.
  - Done when: bounded fixture tests produce the shared fields.
- [ ] **P4.8 — Implement Claude Code enriched discovery.**
  - Map cwd, timestamps, identity, and role only when source evidence supports them.
  - Done when: missing data is null/unknown, not fabricated.
- [ ] **P4.9 — Define shared `execution-profile` settings.**
  - Profiles: `interactive` and `agent-strict`.
  - Record the exact expansion of sorting, collision, partial, receipt, inline SQL, truncation, and non-interactive behavior.
  - Done when: profile expansion is one shared implementation, not duplicated per command.
- [ ] **P4.10 — Wire strict profile into convert, validate, and query.**
  - Require or deterministically choose run-record paths according to the final decision.
  - Done when: strict settings appear in receipts and explicit overrides are visible.
- [ ] **P4.11 — Add capability/version output if flag probing remains necessary.**
  - First evaluate whether shared profile plus existing help is sufficient.
  - If added, expose a stable machine-readable object without adding a help page.
  - Done when: agents can determine support without scraping prose.
- [ ] **P4.12 — Re-run a bounded attribution scenario.**
  - Use an answer-free snapshot, enriched discovery, explicit source list, strict conversion, native validation, saved SQL, and receipts.
  - Done when: candidate roles and repository verification remain correct with no format-specific audit scripts.
- [ ] **P4.13 — Phase 4 documentation and phase gate.**
  - Update `discover.md`, command help, analysis guide, diary, tasks, and changelog.
  - Run adapter discovery tests, strict-profile integration tests, full tests, lint, and the bounded scenario acceptance suite.
  - Done when: a repository/time-bounded source inventory is deterministic and independently auditable.

## Phase 5 — Consolidate help and thin the transcript-analysis skill

**Goal:** document the new contracts in a smaller, canonical help tree instead of adding feature pages.

**Phase dependency:** command behavior from Phases 1–4 stable enough to document.

- [ ] **P5.1 — Add embedded-help registry tests before moving content.**
  - Reject duplicate slugs and unresolved internal `go-minitrace help <slug>` references.
  - Done when: current catalog is captured and moves can fail safely.
- [ ] **P5.2 — Add command/frontmatter reference tests.**
  - Verify `Commands:` entries map to real command paths where practical.
  - Done when: stale command references fail tests.
- [ ] **P5.3 — Add executable documentation smoke tests.**
  - Select bounded fixture-backed examples for discover, convert, validate, and query.
  - Done when: examples verify current flags/defaults without accessing live user stores.
- [ ] **P5.4 — Refocus `getting-started.md`.**
  - Keep only the shortest successful journey and links to rigorous analysis.
  - Remove advanced methodology duplicated elsewhere.
  - Done when: a new user can complete the basic path without encountering strict-mode complexity.
- [ ] **P5.5 — Merge `end-to-end-analysis.md` into `analysis-guide.md`.**
  - Preserve unique material, add bounded-source attribution and receipt workflow, then remove the old page.
  - Done when: no repository reference points to the retired slug.
- [ ] **P5.6 — Merge framework metadata mappings into `adapter-reference.md`.**
  - Add identity extraction, lineage, warning, and outcome-evidence sections.
  - Remove `framework-metadata-mappings.md` after link checks.
  - Done when: adapter reference is the single owner of source-specific field semantics.
- [ ] **P5.7 — Consolidate DuckDB migration stubs.**
  - Keep `query-duckdb.md` as the one migration reference for the agreed release window.
  - Remove `writing-duckdb-queries.md` and `duckdb-query-recipes.md` after checking published/in-repository links.
  - Done when: current query tutorial and recipes contain no legacy-engine ambiguity.
- [ ] **P5.8 — Update command contract pages.**
  - `discover.md`: identity/cwd/time/role fields.
  - `convert.md`: collisions, batch outcomes, receipts.
  - `validate.md`: syntax/schema/archive checks.
  - `query.md`: receipts, strict mode, truncation.
  - `output-formats.md`: zero rows, JSON vs NDJSON, errors, output files, shell safety.
  - Done when: each behavior has one canonical owner.
- [ ] **P5.9 — Update troubleshooting.**
  - Add collision, replayed identity, partial conversion, manifest mismatch, zero-row old-version behavior, truncation, and receipt diagnostics.
  - Done when: each known symptom points to one corrective command sequence.
- [ ] **P5.10 — Remove duplicated methodology from query pages.**
  - Keep `writing-queries.md` focused on authoring and `query-recipes.md` focused on runnable SQL.
  - Link to `analysis-guide` for attribution/evidence method.
  - Done when: the same workflow is not maintained in three pages.
- [ ] **P5.11 — Thin the transcript-analysis skill.**
  - Retain bounded-source operational steps, automation scripts, role classification, repository verification, and links.
  - Remove generic query/schema tutorials now owned by embedded help.
  - Done when: the skill remains executable but no longer duplicates canonical product docs.
- [ ] **P5.12 — Review helper scripts against native capabilities.**
  - Retire or clearly version-gate source audit, staging, and manifest helpers superseded by CLI behavior.
  - Run ShellCheck on retained scripts.
  - Done when: strict workflow prefers native commands and fallbacks declare supported binary versions.
- [ ] **P5.13 — Run retired-slug and content-ownership audit.**
  - Search repository, README, examples, skills, and ticket docs for retired slugs and duplicated contract tables.
  - Done when: only intentional historical ticket references remain.
- [ ] **P5.14 — Phase 5 gate.**
  - Run help registry/link/example tests, `go test ./...`, lint, and manually render canonical pages.
  - Compare help page count before/after; expected net reduction is four pages.
  - Done when: no new feature help page exists and all canonical pages resolve.

## Phase 6 — Make historical tool outcomes evidence-bearing

**Goal:** distinguish succeeded, failed, cancelled, and unknown tool calls across adapters without framework-specific interpretation.

**Phase dependency:** Phases 1–5 complete; archive/schema migration impact reviewed separately.

- [ ] **P6.1 — Measure current outcome semantics by adapter.**
  - Use redacted/sample archives to count success values, missing results, exit codes, interrupts, and adapter annotations.
  - Save report-bearing SQL and receipts.
  - Done when: schema decisions are supported by measured source/output evidence.
- [ ] **P6.2 — Define status vocabulary and evidence codes.**
  - Status: succeeded, failed, cancelled, unknown.
  - Evidence examples: native-is-error, exit-code, structured-result, adapter-scrape, missing-result, user-interrupt.
  - Done when: each primary adapter source shape maps without inventing certainty.
- [ ] **P6.3 — Decide compatibility representation.**
  - Evaluate nullable derived `success`, schema-version change, normalized view compatibility, and API impact.
  - Record an accepted decision before code changes.
  - Done when: archive, SQLite, proto/web, and query migration consequences are explicit.
- [ ] **P6.4 — Update archive schema and builders.**
  - Add status/evidence fields and central derivation helper.
  - Done when: invalid status/evidence combinations are rejected or normalized predictably.
- [ ] **P6.5 — Update normalized SQLite schema/materialization.**
  - Add indexed/queryable status and evidence representation as justified by query needs.
  - Update cache/schema identity.
  - Done when: old/new archive fixture behavior is tested.
- [ ] **P6.6 — Map Codex outcomes.**
  - Distinguish structured exit code, scraped exit code, missing result, cancellation, and unknown.
  - Done when: Codex mapping tests cover each evidence path.
- [ ] **P6.7 — Map Pi outcomes.**
  - Preserve native error flags and missing-result semantics; do not infer exit codes that are absent.
  - Done when: Pi tests distinguish failure from unknown.
- [ ] **P6.8 — Map Claude Code outcomes.**
  - Use `toolUseResult`, interruption, stderr/result, and native error evidence where available.
  - Done when: cancellation is not collapsed into ordinary failure.
- [ ] **P6.9 — Audit remaining adapters.**
  - Copilot, ChatGPT, claude.ai, and turnsdb each document native/derived/unknown support.
  - Done when: no adapter silently defaults unknown outcomes to success.
- [ ] **P6.10 — Update query presets and recipes.**
  - Port `tool-failures` and cross-framework summaries to status/evidence.
  - Preserve a clearly documented compatibility query if needed.
  - Done when: failure reports no longer require adapter-specific caveats for basic status.
- [ ] **P6.11 — Update API and web UI.**
  - Regenerate protobuf/TypeScript where required and render unknown/cancelled distinctly.
  - Done when: API round-trip and Storybook/UI tests cover all statuses.
- [ ] **P6.12 — Update adapter fidelity documentation.**
  - Replace design assumptions with measured support and known limitations.
  - Done when: the matrix states sample scope and evidence source.
- [ ] **P6.13 — Re-run fidelity and attribution evaluations.**
  - Confirm tool-outcome changes do not alter identity, role attribution, or repository verification.
  - Done when: acceptance suites pass and outcome metrics are explainable.
- [ ] **P6.14 — Phase 6 final gate and ticket closure recommendation.**
  - Run full Go/race/lint/release checks, web tests if changed, docmgr doctor, and final bounded workflow.
  - Update diary, tasks, changelog, design decisions, and reMarkable bundle.
  - Done when: all phase acceptance criteria pass and unresolved work is moved to explicitly linked follow-up tickets.

## Cross-phase review checkpoints

- [ ] **R.1 — Identity review after Phase 1.** No parent ID can replace a child archive ID.
- [ ] **R.2 — Publication review after Phase 1.** No conflicting destination changes before collision approval.
- [ ] **R.3 — Integrity review after Phase 2.** Archive files, manifests, and receipts reconcile from one native command.
- [ ] **R.4 — Output review after Phase 3.** Zero-row, error, and truncated runs are machine-readable and semantically distinct.
- [ ] **R.5 — Attribution review after Phase 4.** Cwd/title/mentions remain shortlist signals; Git remains ground truth.
- [ ] **R.6 — Documentation review after Phase 5.** Each contract has one canonical owner and the help catalog is smaller.
- [ ] **R.7 — Schema review after Phase 6.** Tool outcome uncertainty is preserved rather than guessed.
