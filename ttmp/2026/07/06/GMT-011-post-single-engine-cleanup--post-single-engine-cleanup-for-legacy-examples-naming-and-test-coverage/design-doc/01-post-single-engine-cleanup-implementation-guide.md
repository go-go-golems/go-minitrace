---
Title: Post single-engine cleanup implementation guide
Ticket: GMT-011-post-single-engine-cleanup
Status: active
Topics:
    - tooling
    - cli
    - documentation
    - diagnostics
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/Makefile
      Note: Dev commands must stop showing deprecated --db-path usage.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/testdata/query-repositories/README.md
      Note: Testdata docs should teach SQLite/normalized query rules, not DuckDB rules.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/web/src/components/QueryEditor/stories/QueryEditor.stories.tsx
      Note: Storybook examples should use normalized SQL, not UNNEST/sessions_base defaults.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/cmd/go-minitrace/cmds/query/command_runtime_javascript_test.go
      Note: Renamed JS runtime tests now compile on normal GOOS targets.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/cmd/go-minitrace/cmds/query/sqlite_fixtures_test.go
      Note: Shared advanced fixture now covers JS and SQL query-path tests.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/README.md
      Note: Adds a short migration checklist for pre-single-engine users.
ExternalSources: []
Summary: Cleanup plan for the post-GMT-009/GMT-010 state: remove stale DuckDB examples, restore ignored JS runtime tests, consolidate duplicated fixtures, improve one small naming drift, and document user migration at the README level.
LastUpdated: 2026-07-06T13:00:00-04:00
WhatFor: Guide and review the GMT-011 cleanup patch after the single-query-engine migration.
WhenToUse: Use before reviewing cleanup changes that touch examples, query tests, or migration docs.
---

# Post single-engine cleanup implementation guide

## Executive summary

GMT-009 correctly moved go-minitrace to a single normalized SQLite query engine, and GMT-010 hardened the two concrete review blockers. GMT-011 is the follow-up cleanup ticket: remove stale DuckDB-era examples that could confuse users, restore test coverage hidden by a `_js_test.go` filename, consolidate duplicated query fixtures, add an explicit README migration checklist, and tighten one small naming mismatch in the authorizer API.

This ticket intentionally does **not** split the largest packages (`serve`, `db_builder.go`, adapters). Those are valid future cleanup topics, but they are larger refactors. GMT-011 focuses on small, high-confidence cleanup that reduces user confusion and improves CI coverage.

## Problem statement

The post-single-engine codebase still had several rough edges:

1. **Stale examples:** Makefile dev commands, testdata docs, and web mocks/stories still showed `--db-path`, `sessions_base`, `UNNEST`, or DuckDB wording as if those were normal current usage.
2. **Hidden tests:** `cmd/go-minitrace/cmds/query/command_runtime_js_test.go` was ignored by the Go toolchain because `_js_test.go` is interpreted as GOOS `js`.
3. **Duplicated test fixtures:** Once the JS runtime test file was renamed, it collided with newer shared fixture helpers in `sqlite_fixtures_test.go`.
4. **Migration discoverability:** The docs had good detailed migration pages, but the top-level README only linked them indirectly.
5. **Naming drift:** GMT-010 made the authorizer schema-aware internally, but the `NewQueryRunner` parameter still said `allowedObjects` even though it now means allowed read keys.

## Proposed solution

### 1. Clean examples without removing compatibility

Keep these intentional compatibility surfaces:

- `sessions_base` compatibility view;
- `{{TABLE_NAME}}` support;
- deprecated runtime flags accepted for one transition period;
- migration docs mentioning DuckDB.

Remove stale examples that teach those surfaces as the default path:

- Makefile serve/dev commands no longer pass `--db-path`.
- Testdata docs use SQLite/normalized JSON guidance.
- Storybook and MSW mocks use `sessions`, `turns`, and normalized columns.

### 2. Rename JS runtime tests and consolidate fixtures

Rename:

```text
command_runtime_js_test.go -> command_runtime_javascript_test.go
```

Then remove duplicated local helpers from the renamed file and use the shared helpers in `sqlite_fixtures_test.go`. Expand the shared advanced fixture so it still covers the JS showcase assertions: annotations, spawned-agent metadata, handover fields, system prompt text, and image/read tool variations.

### 3. Add README migration checklist

Add a small checklist near the query section:

- replace `query duckdb` with `query run`;
- remove deprecated runtime flags;
- rewrite `UNNEST` against normalized child tables;
- use `mt.db().RuntimeArchives().QueryCommandDefaults().Build()` in JS.

### 4. Small naming cleanup

Rename the `NewQueryRunner` argument from `allowedObjects` to `allowedReads` to match the schema-aware implementation introduced in GMT-010.

## Design decisions

### Decision: do not remove `sessions_base` in this cleanup

- **Context:** `sessions_base` is still valuable compatibility for simple session-level SQL.
- **Decision:** Keep compatibility, but stop presenting it as the default in examples.
- **Rationale:** Users get a migration bridge without new users learning the old shape first.
- **Status:** accepted.

### Decision: restore JS tests now, package splitting later

- **Context:** The renamed JS runtime tests immediately compile and run after fixture consolidation. Splitting `serve` or `db_builder.go` is larger and riskier.
- **Decision:** Restore test coverage in GMT-011; leave package decomposition for a future design ticket.
- **Rationale:** This yields direct correctness value with bounded diff size.
- **Status:** accepted.

## Implementation plan

1. Inventory legacy references with `rg` and file/package size reports.
2. Edit Makefile/testdata/web examples away from stale DuckDB-era defaults.
3. Rename `command_runtime_js_test.go` to `command_runtime_javascript_test.go`.
4. Delete duplicated helpers from the renamed JS test and enhance the shared fixture.
5. Update golden expectations that legitimately change with the richer shared fixture.
6. Add README migration checklist.
7. Rename `NewQueryRunner` parameter to `allowedReads`.
8. Validate:
   - `GOWORK=off go test ./cmd/go-minitrace/cmds/query ./pkg/minitracedb ./pkg/minitracejs`
   - `make glazed-lint`
   - `GOWORK=off go test ./...`
   - `GOWORK=off go list -f '{{.IgnoredGoFiles}}' ./cmd/go-minitrace/cmds/query` should return `[]`.
9. Attempt frontend validation; record if local dependencies are missing.
10. Commit code and docs.

## Validation notes

Go validation passed after implementation:

```bash
GOWORK=off go test ./cmd/go-minitrace/cmds/query ./pkg/minitracedb ./pkg/minitracejs
make glazed-lint
GOWORK=off go test ./...
GOWORK=off go list -f '{{.IgnoredGoFiles}}' ./cmd/go-minitrace/cmds/query
```

Frontend validation could not run in this checkout because `web/node_modules` is missing:

- `pnpm lint` fails before linting project files because ESLint 6 cannot find a config.
- `pnpm build` fails because `tsc` is not installed.

The edited TypeScript is simple string-literal story/mock content, but a full frontend validation still requires installing dependencies or using the repository's frontend build pipeline.

## Future cleanup left intentionally out of scope

- Split `pkg/minitracejs/db_builder.go` into builder/cache/handle/source files.
- Split `pkg/minitracedb/schema.go` by table/view/allowlist definitions.
- Decompose `cmd/go-minitrace/cmds/serve` into smaller route/runtime packages.
- Replace file-level `glazedclilint:file-ignore` comments with migrated Glazed command definitions.
