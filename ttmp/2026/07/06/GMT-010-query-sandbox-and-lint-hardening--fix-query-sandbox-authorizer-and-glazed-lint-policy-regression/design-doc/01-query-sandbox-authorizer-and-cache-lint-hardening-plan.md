---
Title: Query sandbox authorizer and cache lint hardening plan
Ticket: GMT-010-query-sandbox-and-lint-hardening
Status: active
Topics:
    - tooling
    - cli
    - diagnostics
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace/pkg/minitracedb/query.go
      Note: Central SQLite read-only validator and authorizer to make schema-aware.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace/pkg/minitracedb/schema.go
      Note: Main/attached allowlist helpers for normalized tables and live annotations.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace/pkg/minitracedb/query_test.go
      Note: Regression tests for schema-aware attached database reads.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace/pkg/minitracejs/db_builder.go
      Note: Disk-cache eviction limit no longer performs direct os.Getenv lookup.
    - Path: /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace/pkg/minitracejs/db_builder_cache_test.go
      Note: Cache-size tests updated for explicit builder fallback/override behavior.
ExternalSources:
    - https://github.com/go-go-golems/go-minitrace/pull/22/
Summary: Plan and implementation notes for fixing the two GMT-009 review blockers: the Glazed lint failure caused by direct cache environment lookup, and the SQLite authorizer's table-name-only allowlist for attached annotation databases.
LastUpdated: 2026-07-06T12:20:00-04:00
WhatFor: Guide the GMT-010 hardening patch on top of the GMT-009 single-query-engine migration.
WhenToUse: Read before reviewing the lint/authorizer fix patch or extending the query sandbox with more attached databases.
---

# Query sandbox authorizer and cache lint hardening plan

## Executive summary

GMT-010 fixes the two actionable findings from the GMT-009 PR #22 review:

1. `make glazed-lint` failed because `pkg/minitracejs/db_builder.go` read `GO_MINITRACE_CACHE_MAX_BYTES` directly with `os.Getenv` from CLI-reachable code.
2. The SQLite sandbox authorizer allowed reads by bare table name only. Once `serve` attached `annotations.db` as `anno`, a crafted attached database could expose `anno.sessions` and pass the allowlist because `sessions` is an allowed main-schema table.

The patch keeps the GMT-009 architecture intact. It does not reintroduce DuckDB or alter the query API. It tightens the existing single-engine implementation by making cache-size control explicit and by making authorizer allowlist keys schema-aware.

## Problem statement

### Problem 1: direct environment lookup violates Glazed CLI policy

The GMT-009 implementation added disk-cache eviction with a 2 GiB default and an environment override:

```go
raw := strings.TrimSpace(os.Getenv("GO_MINITRACE_CACHE_MAX_BYTES"))
```

The repository's Glazed policy linter rejects this pattern in CLI-reachable code. The failure is real in CI and reproducible locally:

```text
pkg/minitracejs/db_builder.go:675:27: use Glazed config/env middleware or an explicit command field instead of os.Getenv in CLI code
```

### Problem 2: attached DB allowlist is not schema-aware

The sandbox authorizer receives the SQLite database/schema name, table/object name, and column name. The GMT-009 code ignored the schema and checked only object name:

```text
allowed: sessions, turns, tool_calls, annotations, ...
query:   SELECT secret FROM anno.sessions
seen:    object=sessions, database=anno
result:  allowed
```

The intended policy is narrower:

```text
main.sessions       allowed
main.turns          allowed
main.sessions_base  allowed
anno.annotations    allowed
anno.sync_state     allowed
anno.sessions       denied
sqlite_master       denied
```

## Proposed solution

### Cache lint fix

Move cache-size control from a package-level environment lookup to an explicit `DBBuilder` setting:

```go
type DBBuilder struct {
    cacheMaxBytes int64
}

func (b *DBBuilder) diskCacheMaxBytes() int64 {
    if b != nil && b.cacheMaxBytes > 0 {
        return b.cacheMaxBytes
    }
    return defaultDiskCacheMaxBytes
}
```

The current implementation keeps the public behavior simple: default 2 GiB eviction limit, no direct env var. Tests can set `builder.cacheMaxBytes` directly because they are in package `minitracejs`. A future CLI/config field can plumb a non-default value through the builder without violating the lint policy.

### Authorizer fix

Normalize allowlist entries to schema-qualified keys:

- bare `sessions` becomes `main.sessions`, preserving current callers;
- explicit `anno.annotations` stays `anno.annotations`;
- empty database names from SQLite default to `main`.

The `SQLITE_READ` callback then checks `(database, object)`, not just `object`:

```go
key := readKey(database, object)
if _, ok := allowedReads[key]; ok {
    return sqlite3.SQLITE_OK
}
return sqlite3.SQLITE_DENY
```

`AllowedObjectNamesWithLiveAnnotations()` now returns:

```go
append(AllowedObjectNames(), "anno.annotations", "anno.sync_state")
```

not bare `sync_state`.

## Design decisions

### Decision: bare allowlist names mean `main.<name>`

- **Context:** Existing callers and tests pass `AllowedTableNames()` or `AllowedObjectNames()` as bare table names.
- **Options considered:** require every caller to update to `main.*`; keep bare names as wildcard-any-schema; interpret bare names as main only.
- **Decision:** bare names normalize to `main.<name>`.
- **Rationale:** This preserves source compatibility while closing the attached-schema bypass. A wildcard interpretation would preserve the bug.
- **Consequences:** New attached databases must be explicit in allowlist helpers.
- **Status:** accepted.

### Decision: no direct env fallback for cache size in this patch

- **Context:** The immediate blocker is the Glazed linter, not a user requirement for the env var.
- **Options considered:** delete eviction; keep env and suppress lint; introduce a full CLI flag/config flow; keep default and explicit builder setting.
- **Decision:** keep default eviction and explicit builder override only.
- **Rationale:** This is the smallest safe change that restores CI. A user-facing flag can be added later with normal Glazed config plumbing.
- **Consequences:** `GO_MINITRACE_CACHE_MAX_BYTES` is not supported by this patch. There were no docs for it outside the implementation comments/tests.
- **Status:** accepted.

## Implementation plan

1. Add GMT-010 ticket docs and tasks.
2. Reproduce the GMT-009 findings using the review scripts.
3. Replace `diskCacheMaxBytes()` package env lookup with `(*DBBuilder).diskCacheMaxBytes()`.
4. Update cache tests to use explicit builder override/fallback.
5. Change `QueryRunner` internals from `allowedObjects` to schema-qualified `allowedReads`.
6. Update live annotation allowlist to include `anno.annotations` and `anno.sync_state` explicitly.
7. Add a regression test proving:
   - `anno.annotations` is allowed;
   - `main.sessions` is allowed;
   - `anno.sessions` is denied;
   - sqlite catalog denial still gives the introspection hint.
8. Validate with focused tests, full `go test`, `make glazed-lint`, and the GMT-009 review scripts.

## Validation results

Validated after implementation:

```bash
GOWORK=off go test ./pkg/minitracedb ./pkg/minitracejs ./cmd/go-minitrace/cmds/serve ./cmd/go-minitrace/cmds/query
make glazed-lint
GOWORK=off go test ./...
ttmp/.../GMT-009.../scripts/02-review-authorizer-attach-schema.sh
ttmp/.../GMT-009.../scripts/01-review-ci-and-query-smoke.sh
```

Key outcomes:

- focused tests pass;
- full tests pass;
- glazed-lint passes;
- `anno.sessions` is now denied with `query references disallowed table/view "anno.sessions"`;
- `query run` fixture smoke still succeeds.

## Open questions

- Should a future PR add a user-facing `--cache-max-bytes` / config setting? The builder is ready for explicit plumbing, but this ticket intentionally avoids growing CLI surface area.
- Should the GMT-009 review scripts be copied into GMT-010 or left as review evidence in GMT-009? For now they remain in GMT-009 because they were created as part of the review and now double as before/after regression smoke scripts.

## References

- GMT-009 code review: `../GMT-009.../code-review/01-gmt-009-pr-22-project-and-code-review.md`
- PR #22: <https://github.com/go-go-golems/go-minitrace/pull/22/>
- Key implementation files: `pkg/minitracedb/query.go`, `pkg/minitracedb/schema.go`, `pkg/minitracejs/db_builder.go`
