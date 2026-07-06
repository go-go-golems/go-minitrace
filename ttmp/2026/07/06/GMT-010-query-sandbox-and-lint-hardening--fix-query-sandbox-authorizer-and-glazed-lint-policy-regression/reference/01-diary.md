---
Title: Diary
Ticket: GMT-010-query-sandbox-and-lint-hardening
Status: active
Topics:
    - tooling
    - cli
    - diagnostics
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace/pkg/minitracedb/query.go
      Note: Schema-aware authorizer implementation.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace/pkg/minitracedb/schema.go
      Note: Live annotation allowlist now uses explicit anno.* entries.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace/pkg/minitracedb/query_test.go
      Note: Regression coverage for attached-schema allowlisting.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace/pkg/minitracejs/db_builder.go
      Note: Cache-size lint fix through explicit builder setting.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace/pkg/minitracejs/db_builder_cache_test.go
      Note: Cache-size tests updated away from environment lookup.
ExternalSources: []
Summary: Chronological diary for GMT-010, covering ticket creation, lint fix, schema-aware SQLite authorizer hardening, validation, and commit status.
LastUpdated: 2026-07-06T12:30:00-04:00
WhatFor: Continue or review the GMT-010 hardening work.
WhenToUse: Read before touching the query sandbox, cache-size setting, or PR #22 merge blockers.
---

# Diary

## Goal

Capture the task-by-task hardening work for GMT-010: fix the Glazed lint regression from GMT-009 and make the SQLite authorizer schema-aware for attached annotation databases.

## Step 1: Ticket setup and scope definition

Created GMT-010 as a focused follow-up ticket to the GMT-009 PR #22 review. The ticket is intentionally narrow: unblock CI by removing direct environment lookup from the cache-size path, and close the attached-schema authorizer gap found by the review experiment.

This step produced the design-plan skeleton, task list, and this diary so the implementation can be reviewed independently from the much larger GMT-009 migration.

### Prompt Context

**User prompt (verbatim):** "ok, create a ticket to address the lint and the authorizer."

**Assistant interpretation:** Create a new go-minitrace docmgr ticket for the two concrete findings from the GMT-009 review: the Glazed lint failure and the SQLite authorizer attached-schema issue.

**Inferred user intent:** Keep the hardening work tracked separately from the large GMT-009 implementation so it can be executed and reviewed cleanly.

### What I did
- Created ticket `GMT-010-query-sandbox-and-lint-hardening`.
- Added `design-doc/01-query-sandbox-authorizer-and-cache-lint-hardening-plan.md`.
- Added `reference/01-diary.md`.
- Replaced the placeholder task list with six concrete tasks.

### Why
- The GMT-009 review produced two actionable findings that should be fixed before merge.
- A small ticket reduces review scope and keeps the audit trail clear.

### What worked
- `docmgr --root ./ttmp ticket create-ticket` created the workspace cleanly.
- The ticket inherited valid vocabulary topics.

### What didn't work
- N/A.

### What I learned
- GMT-010 should be treated as a hardening patch on top of GMT-009, not a redesign of the query engine.

### What was tricky to build
- The repository working tree already contains the full GMT-009 PR as uncommitted changes in this environment, so the new ticket must describe patch intent carefully. A local commit may include larger file diffs than the logical GMT-010 delta unless the branch history is aligned with PR #22.

### What warrants a second pair of eyes
- Whether the eventual Git commit should happen in this worktree or in a checkout where PR #22 commits are already ancestors.

### What should be done in the future
- Commit the GMT-010 patch on top of the actual PR #22 branch history if possible.

### Code review instructions
- Start with the GMT-009 code review findings, then read the GMT-010 design doc.

### Technical details
- Ticket path: `ttmp/2026/07/06/GMT-010-query-sandbox-and-lint-hardening--fix-query-sandbox-authorizer-and-glazed-lint-policy-regression/`

## Step 2: Fix lint and harden the authorizer

Implemented both code changes task-by-task. First, the disk-cache size limit no longer uses a package-level `os.Getenv` lookup; it is now an explicit `DBBuilder` field with a default fallback. Second, the SQLite authorizer now normalizes allowed reads as schema-qualified keys, so bare allowlist entries mean `main.<object>` and attached annotation tables must be explicitly allowed as `anno.annotations` and `anno.sync_state`.

This keeps the GMT-009 single-query-engine architecture intact. The query surfaces still use the same `QueryRunner`; the change is that the authorizer now distinguishes `main.sessions` from `anno.sessions`.

### Prompt Context

**User prompt (verbatim):** "then work on them, task by task, committing at appropriate intervals, keeping a detailed diary as you work"

**Assistant interpretation:** Implement the GMT-010 lint and authorizer fixes, validate each task, commit when the repository state permits, and maintain a detailed diary.

**Inferred user intent:** Move from review findings to working code with an auditable implementation narrative.

### What I did
- Updated `pkg/minitracejs/db_builder.go`:
  - added `cacheMaxBytes int64` to `DBBuilder`;
  - changed cache eviction to call `b.diskCacheMaxBytes()`;
  - removed the direct `os.Getenv("GO_MINITRACE_CACHE_MAX_BYTES")` path and `strconv` import.
- Updated `pkg/minitracejs/db_builder_cache_test.go`:
  - cache eviction test now sets `builder.cacheMaxBytes = 1` explicitly;
  - env override test became explicit override/fallback coverage.
- Updated `pkg/minitracedb/query.go`:
  - changed the internal allowlist from bare `allowedObjects` to schema-qualified `allowedReads`;
  - added `normalizeAllowedReadKey` and `readKey` helpers;
  - changed `SQLITE_READ` authorization to check `(database, object)`.
- Updated `pkg/minitracedb/schema.go`:
  - documented bare main-schema names;
  - changed `AllowedObjectNamesWithLiveAnnotations()` to return `anno.annotations` and `anno.sync_state` explicitly.
- Updated `pkg/minitracedb/query_test.go`:
  - added `TestQueryRunnerAttachmentAllowlistIsSchemaAware`.

### Why
- Removing direct env lookup restores Glazed policy compliance.
- Schema-qualified authorizer keys close the attached-table-name collision found in GMT-009.

### What worked
- Focused tests passed after updating the cache tests:
  `GOWORK=off go test ./pkg/minitracedb ./pkg/minitracejs ./cmd/go-minitrace/cmds/serve ./cmd/go-minitrace/cmds/query`.
- The attached-schema review script now reports `anno.sessions` denied.
- `make glazed-lint` now passes.

### What didn't work
- First focused test run failed because the existing cache tests still expected the removed environment override:
  - `TestDiskCacheEvictsOldestFilesWhenOverSizeLimit` did not evict with the default 2 GiB limit.
  - `TestDiskCacheMaxBytesEnvOverrideAndFallback` expected `GO_MINITRACE_CACHE_MAX_BYTES=12345` to affect `diskCacheMaxBytes()`.
- I fixed this by making tests use the explicit builder field instead of the removed env var.

### What I learned
- The authorizer callback provides enough schema information to enforce the intended policy; the bug was only that the previous wrapper discarded it.
- The cache-size override was not documented outside implementation comments/tests, so removing env support is a narrow compatibility change.

### What was tricky to build
- Backward-compatible allowlist normalization needed care. Existing callers pass bare table names, so the code now treats bare names as `main.<name>` rather than requiring every call site to change.
- Error messages needed to remain useful: `sqlite_master` still gets the `db.schema()`/`db.tables()` hint, while attached-schema denials show the qualified name such as `anno.sessions`.

### What warrants a second pair of eyes
- Whether future attached databases should use a structured allowlist type instead of strings like `anno.annotations`.
- Whether a user-facing `--cache-max-bytes` or config setting should be added now or later.

### What should be done in the future
- Consider adding a Glazed-backed cache settings section if users need to tune disk cache size.
- Copy or promote the GMT-009 shell review scripts into a reusable regression playbook if this pattern recurs.

### Code review instructions
- Review `pkg/minitracedb/query.go` first, especially `normalizeAllowedReadKey`, `readKey`, and `newReadOnlyAuthorizer`.
- Then review `pkg/minitracedb/query_test.go::TestQueryRunnerAttachmentAllowlistIsSchemaAware`.
- Review `pkg/minitracejs/db_builder.go` / `db_builder_cache_test.go` for the lint fix.

### Technical details
- Before fix: `anno.sessions` returned the sentinel row in the GMT-009 review script.
- After fix: `anno.sessions error="query references disallowed table/view \"anno.sessions\"" rows=0`.

## Step 3: Validation and commit status

Validated the hardening patch with focused tests, full tests, lint, and the original GMT-009 review scripts. The functional findings are fixed: `make glazed-lint` passes, and the attached-schema experiment denies `anno.sessions` while still allowing `anno.annotations` and `main.sessions`.

I committed the implementation in the clean PR worktree (`go-minitrace-pr`), whose HEAD already included the GMT-009 PR commits. This avoided creating a misleading giant commit in the dirty `go-minitrace` worktree where the entire GMT-009 diff appears uncommitted against `main`.

**Commit (code):** `4065f0c0d272e82aafc2f1bf0c261afca8796cfa` — "GMT-010: harden query sandbox and fix lint"

### Prompt Context

**User prompt (verbatim):** (see Step 2)

**Assistant interpretation:** Validate the implementation and commit when practical.

**Inferred user intent:** Ensure the fixes are actually merge-ready, not just edited.

### What I did
- Ran focused test set:
  `GOWORK=off go test ./pkg/minitracedb ./pkg/minitracejs ./cmd/go-minitrace/cmds/serve ./cmd/go-minitrace/cmds/query`.
- Ran `make glazed-lint`.
- Ran full test suite:
  `GOWORK=off go test ./...`.
- Reran GMT-009 scripts:
  - `scripts/02-review-authorizer-attach-schema.sh` now denies `anno.sessions`.
  - `scripts/01-review-ci-and-query-smoke.sh` now passes tests, query smoke, and `make glazed-lint`.

### Why
- These commands directly correspond to the two GMT-009 review findings and the normal repository merge gates.

### What worked
- Focused tests: pass.
- Full tests: pass.
- Glazed lint: pass.
- Query smoke: pass.
- Authorizer script: pass with the expected denial.

### What didn't work
- Initial implementation and docs commit completed in `go-minitrace-pr` as `4065f0c0d272e82aafc2f1bf0c261afca8796cfa`.

### What I learned
- The original GMT-009 review scripts are useful before/after regression checks and should be kept.

### What was tricky to build
- Validating in this worktree requires remembering that `git status` shows the entire GMT-009 PR as uncommitted. The logical GMT-010 patch is only a small subset of those files.

### What warrants a second pair of eyes
- The exact git staging strategy in this environment. The safe logical patch files are:
  - `pkg/minitracedb/query.go`
  - `pkg/minitracedb/schema.go`
  - `pkg/minitracedb/query_test.go`
  - `pkg/minitracejs/db_builder.go`
  - `pkg/minitracejs/db_builder_cache_test.go`
  - `ttmp/2026/07/06/GMT-010...`

### What should be done in the future
- Push `4065f0c0d272e82aafc2f1bf0c261afca8796cfa` from `go-minitrace-pr` if the PR branch should be updated immediately.

### Code review instructions
- Re-run the exact validation commands listed above.
- Confirm `make glazed-lint` no longer reports `os.Getenv` in `db_builder.go`.
- Confirm `anno.sessions` is denied by either the unit test or the GMT-009 shell script.

### Technical details
- `make glazed-lint` now exits 0.
- `GOWORK=off go test ./...` now exits 0.
