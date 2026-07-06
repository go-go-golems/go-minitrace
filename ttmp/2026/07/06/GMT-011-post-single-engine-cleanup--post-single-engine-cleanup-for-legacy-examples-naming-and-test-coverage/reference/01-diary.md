---
Title: Diary
Ticket: GMT-011-post-single-engine-cleanup
Status: active
Topics:
    - tooling
    - cli
    - documentation
    - diagnostics
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/Makefile
      Note: Removed deprecated --db-path from dev serve commands.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/testdata/query-repositories/README.md
      Note: Replaced DuckDB JSON guidance with SQLite/normalized guidance.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/web/src/components/QueryEditor/stories/QueryEditor.stories.tsx
      Note: Storybook SQL examples now use normalized tables.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/web/src/mocks/handlers.ts
      Note: MSW query-command render mocks now use normalized SQL.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/cmd/go-minitrace/cmds/query/command_runtime_javascript_test.go
      Note: Renamed JS runtime tests now compile on normal platforms.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/cmd/go-minitrace/cmds/query/sqlite_fixtures_test.go
      Note: Shared advanced fixture now covers the restored JS runtime tests.
ExternalSources: []
Summary: Chronological diary for GMT-011 cleanup: stale legacy examples, JS test restoration, fixture consolidation, README migration checklist, validation, and commit status.
LastUpdated: 2026-07-06T13:10:00-04:00
WhatFor: Continue or review post-single-engine cleanup work.
WhenToUse: Read before touching query examples, JS runtime tests, or migration docs after GMT-009/GMT-010.
---

# Diary

## Goal

Track the GMT-011 cleanup work after the single-query-engine migration: remove stale DuckDB-era examples, restore ignored JS runtime tests, consolidate fixtures, and make user migration guidance easier to find.

## Step 1: Ticket setup and cleanup scope

Created a single follow-up cleanup ticket instead of scattering the findings across multiple tickets. The scope is intentionally bounded to high-confidence cleanup: examples/docs/test coverage/naming, not large package decomposition.

The main implementation guide records what stays as compatibility (`sessions_base`, `{{TABLE_NAME}}`, deprecated flags) versus what should stop appearing as normal examples (`UNNEST`, `--db-path`, DuckDB wording in testdata/web stories).

### Prompt Context

**User prompt (verbatim):** "make a single cleanup follow up ticket. and then add tasks, a clear implementation guide, and work on them task by task, keeping a detailed diary as you work, and committing at appropriate intervals"

**Assistant interpretation:** Create one docmgr ticket for the cleanup concerns raised in the prior audit, add concrete tasks and a reviewable implementation guide, then implement the tasks with diary updates and commits.

**Inferred user intent:** Convert the code-quality observations into tracked cleanup work and land small, concrete improvements instead of only producing another report.

### What I did
- Created ticket `GMT-011-post-single-engine-cleanup`.
- Added `design-doc/01-post-single-engine-cleanup-implementation-guide.md`.
- Added this diary.
- Replaced the placeholder task list with concrete cleanup tasks.

### Why
- The single-engine migration is functionally complete, but stale examples and ignored tests would confuse users and reduce trust in the codebase.

### What worked
- The ticket workspace was created cleanly under go-minitrace-pr's `ttmp`.

### What didn't work
- N/A.

### What I learned
- The right unit of work is not “split all big files now”; it is “remove misleading legacy examples and restore test coverage now, record larger refactors for later.”

### What was tricky to build
- The cleanup had to preserve deliberate compatibility while removing stale teaching material. `sessions_base` is still valid, but new examples should not teach it as the default path.

### What warrants a second pair of eyes
- Whether any remaining `sessions_base` example outside tests/docs migration pages should stay as an explicit compatibility fixture or be rewritten.

### What should be done in the future
- Open a separate architecture ticket for package splitting if desired.

### Code review instructions
- Read the design doc before reviewing the diff to understand the compatibility boundary.

### Technical details
- Initial inventory used `rg` for `DuckDB`, `UNNEST`, `sessions_base`, `--db-path`, `TABLE_NAME`, and `GOWORK=off go list -f '{{.IgnoredGoFiles}}'`.

## Step 2: Remove stale examples and add migration checklist

Updated user-facing and demo-facing examples so the default path is normalized SQLite. The Makefile no longer passes deprecated `--db-path` to `serve`, testdata documentation now describes SQLite JSON rules, and web stories/mocks no longer use `UNNEST` or `sessions_base` as current examples. The README now has a short migration checklist for pre-single-engine users.

This preserves the detailed migration docs while making the top-level and demo surfaces less surprising for new users.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement the cleanup tasks in concrete files and keep a diary.

**Inferred user intent:** Make the codebase read as if the normalized SQLite migration is complete, while still documenting compatibility.

### What I did
- `Makefile`: removed `--db-path $(DEV_DB)` from dev serve commands and log instructions.
- `testdata/query-repositories/README.md`: replaced “DuckDB JSON predicate rule of thumb” with SQLite/normalized guidance.
- `testdata/query-repositories/js-showcase/overview/runtime-playground.js` and README: replaced “DuckDB query” wording with database-neutral wording.
- `web/src/components/QueryEditor/stories/QueryEditor.stories.tsx`: rewrote story SQL to use `sessions` and `turns`, with SQLite `substr` / `unixepoch` instead of DuckDB `LEFT` / `UNNEST` / `EXTRACT`.
- `web/src/components/QueryEditor/stories/QueryCommandForm.stories.tsx`: rewrote rendered SQL to use `sessions`.
- `web/src/mocks/handlers.ts`: rewrote mocked rendered SQL to use normalized columns.
- `README.md`: added a short migration checklist.

### Why
- New users often learn from examples and stories before reading the full docs. These files should not imply that DuckDB-era syntax is still the current way.

### What worked
- After edits, `rg` over Makefile/testdata/web no longer found stale `DuckDB JSON`, `querying DuckDB`, `UNNEST`, `--db-path`, or `sessions_base` examples.

### What didn't work
- Frontend validation could not run locally because dependencies are not installed:
  - `pnpm lint` fails because ESLint cannot find a config in the dependency state.
  - `pnpm build` fails because `tsc` is not installed.

### What I learned
- Most stale examples were not in the embedded help docs; they were in developer/demo surfaces (Makefile, testdata, Storybook/MSW).

### What was tricky to build
- The Storybook long-query example needed a real SQLite equivalent, not just a superficial table rename. The rewrite uses `turns`, `turn_index`, `timestamp`, `substr`, and `unixepoch`.

### What warrants a second pair of eyes
- The TypeScript stories should be validated in an environment with `web/node_modules` installed or via the Dagger/frontend build pipeline.

### What should be done in the future
- Add a lightweight frontend validation command to CI if one is not already reliable.

### Code review instructions
- Review only the SQL string changes in web files; they are demo/mock content, not runtime query code.
- Check README migration checklist for accuracy.

### Technical details
- Migration checklist includes `query duckdb -> query run`, deprecated flag removal, `UNNEST -> child tables`, and JS `mt.db()` builder usage.

## Step 3: Restore JS runtime tests and consolidate fixtures

Renamed the hidden JS runtime test file so Go compiles it on normal platforms. This immediately surfaced duplicate helper definitions and stale expectations, which is exactly why the file should not have been ignored.

The fix was to remove duplicated helpers from the renamed JS test file and enrich the shared SQLite fixture so both SQL and JS command-runtime tests use one source of truth.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Restore ignored test coverage as one of the cleanup tasks and repair the suite until it passes.

**Inferred user intent:** Improve maintainability, not just polish examples.

### What I did
- Renamed `cmd/go-minitrace/cmds/query/command_runtime_js_test.go` to `command_runtime_javascript_test.go`.
- Removed duplicate local `captureProcessor`, `rowToMap`, `writeAdvancedFixtureArchive`, and `buildAdvancedFixtureSessions` helpers from the renamed file.
- Extended `sqlite_fixtures_test.go` with the data the JS showcase tests need: system prompt, annotation, handover, spawned-agent metadata, image/read tool mix, token metrics.
- Updated tests that legitimately changed because the shared beta fixture now has 4 tools instead of 3 and 5 read operations instead of 4.
- Fixed an alias lookup in the restored test to use `catalog.ByPath[...]` for alias files.

### Why
- `command_runtime_js_test.go` was hidden by the `_js_test.go` GOOS suffix. Restoring it gives real coverage for command-scoped JS loader behavior, aliases, and showcase query commands.

### What worked
- `GOWORK=off go list -f '{{.IgnoredGoFiles}}' ./cmd/go-minitrace/cmds/query` now returns `[]`.
- `GOWORK=off go test ./cmd/go-minitrace/cmds/query -count=1` passes.

### What didn't work
- First compile after rename failed with duplicate helper definitions against `sqlite_fixtures_test.go`.
- After removing duplicates, tests failed because the shared fixture lacked the richer data the old hidden JS fixture had.
- After enriching the shared fixture, a few golden expectations needed updating to match the richer fixture.

### What I learned
- The hidden JS tests were not dead; they covered useful behavior and caught fixture drift immediately once compiled.

### What was tricky to build
- Consolidating fixtures without weakening tests required carrying over the richer old fixture behavior rather than simply changing assertions to empty/zero results.

### What warrants a second pair of eyes
- The shared fixture is now more featureful and may become another mini-monolith. If it grows again, split fixture builders by concern.

### What should be done in the future
- Avoid `_js_test.go`, `_linux_test.go`, etc. suffixes unless intentionally creating GOOS-specific tests.

### Code review instructions
- Start with the rename and fixture diff.
- Confirm the removed helpers are covered by `sqlite_fixtures_test.go`.
- Run `GOWORK=off go test ./cmd/go-minitrace/cmds/query -count=1`.

### Technical details
- The restored tests now exercise JS aliases, runtime archive defaults, recipe/showcase commands, and SQL/JS command paths on the normalized engine.

## Step 4: Validation and final cleanup

Ran the Go validation suite and lint after the cleanup. The Go side passes, and the ignored-test check confirms that the JS runtime tests are now active. Frontend validation was attempted and failed before reaching project code because dependencies are missing in the local web workspace.

The implementation and initial ticket docs were committed as `d973bed394c1d14bd2392f3a7a384ebeef44136f`. A final docs-only bookkeeping commit records the checked task and changelog update.

**Commit (code):** `d973bed394c1d14bd2392f3a7a384ebeef44136f` — "GMT-011: clean up single-engine migration leftovers"

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Validate the cleanup implementation, record exact outcomes, and commit at an appropriate checkpoint.

**Inferred user intent:** Make the cleanup independently reviewable and safe to push.

### What I did
- Ran `GOWORK=off go test ./cmd/go-minitrace/cmds/query ./pkg/minitracedb ./pkg/minitracejs`.
- Ran `make glazed-lint`.
- Ran `GOWORK=off go test ./...`.
- Ran `GOWORK=off go list -f '{{.IgnoredGoFiles}}' ./cmd/go-minitrace/cmds/query`.
- Attempted `pnpm lint` and `pnpm build` in `web/` and recorded the dependency limitation.
- Committed the implementation and initial ticket docs as `d973bed394c1d14bd2392f3a7a384ebeef44136f`.

### Why
- The cleanup touches tests, docs/examples, and TypeScript story strings. Go validation is required; frontend validation should be attempted even if dependency setup blocks it.

### What worked
- Focused Go tests passed.
- Full Go tests passed.
- Glazed lint passed.
- IgnoredGoFiles is now `[]` for the query command package.

### What didn't work
- `web/pnpm lint` failed with ESLint configuration discovery under missing dependencies.
- `web/pnpm build` failed with `tsc: not found`, also because `node_modules` is missing.

### What I learned
- The repository's Go validation is self-contained in this checkout, but frontend validation requires dependency installation or the Dagger build path.

### What was tricky to build
- The test rename changed the effective compiled test set. The cleanup therefore had to fix real failures rather than only move files.

### What warrants a second pair of eyes
- Review the web SQL strings manually or run the frontend build in a prepared environment.

### What should be done in the future
- Consider adding a documented one-command frontend validation target that installs/uses pnpm dependencies consistently.
- Push `d973bed394c1d14bd2392f3a7a384ebeef44136f` and the follow-up diary bookkeeping commit when updating the remote PR branch.

### Code review instructions
- Run the Go validation commands listed in the design doc.
- If frontend dependencies are installed, run `cd web && pnpm build`.

### Technical details
- `pnpm lint` failure: ESLint 6 could not find a config file and warned that local `node_modules` is missing.
- `pnpm build` failure: `tsc: not found` and local `node_modules` missing.
