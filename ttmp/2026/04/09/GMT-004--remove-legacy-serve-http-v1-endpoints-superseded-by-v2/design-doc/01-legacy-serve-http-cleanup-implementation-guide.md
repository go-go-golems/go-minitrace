---
Title: Legacy serve HTTP cleanup implementation guide
Ticket: GMT-004
Status: active
Topics:
    - backend
    - documentation
    - minitrace
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: README.md
      Note: Serve API docs still mention legacy annotation endpoints
    - Path: cmd/go-minitrace/cmds/serve/handlers_annotations.go
      Note: Legacy annotation handlers targeted for removal
    - Path: cmd/go-minitrace/cmds/serve/handlers_queries.go
      Note: Legacy preset/query handlers share helpers with v2
    - Path: cmd/go-minitrace/cmds/serve/handlers_sessions.go
      Note: Legacy session handlers share helpers with v2
    - Path: cmd/go-minitrace/cmds/serve/server.go
      Note: Route registration inventory and cleanup target
    - Path: cmd/go-minitrace/cmds/serve/server_test.go
      Note: Legacy handler tests to replace with route-level assertions
    - Path: web/src/api/minitrace.ts
      Note: Frontend evidence that v2 routes are already the active contract
    - Path: web/src/mocks/handlers.ts
      Note: Storybook mocks still expose stale legacy routes
ExternalSources: []
Summary: Evidence-backed plan for removing obsolete non-v2 serve HTTP endpoints while preserving the one remaining JSON-native raw-query route.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Legacy serve HTTP cleanup implementation guide

## Executive summary

`go-minitrace serve` currently registers two overlapping HTTP route families for sessions, saved-query metadata, and annotations: a legacy JSON-native `/api/...` surface and a newer protobuf-backed `/api/v2/...` surface. The first-party React app has already moved to `/api/v2/...` for all of those domains, while still intentionally using `POST /api/query` for raw SQL execution. That leaves a narrow cleanup opportunity: remove the legacy route registrations, delete dead v1 handler entrypoints, keep the shared normalization and SQL/file helper code that v2 still depends on, and update tests/mocks/docs so the repository no longer advertises or exercises dead endpoints.

This ticket should not attempt a full API redesign. The safe scope is to remove legacy session, preset/query-metadata, and annotation endpoints; preserve `POST /api/query`; preserve all `/api/v2/...` endpoints; and preserve structured query-command routes, which were introduced as v2-only from the beginning.

## Problem statement and scope

### What is wrong today

1. `cmd/go-minitrace/cmds/serve/server.go` still registers both legacy and v2 routes for sessions, saved-query metadata, and annotations, so the public HTTP surface is larger than the product actually needs.
2. `web/src/api/minitrace.ts` already calls only the v2 routes for those domains, which means the legacy handlers are now compatibility baggage rather than active product behavior.
3. `web/src/mocks/handlers.ts` still carries several legacy mock routes, so Storybook/test support code does not accurately reflect the real frontend contract.
4. `cmd/go-minitrace/cmds/serve/server_test.go` still contains direct tests for legacy handlers, which makes dead code look maintained and raises the cost of future refactors.
5. `README.md` still documents legacy annotation endpoints, which is misleading once the v2 API has become the active contract.

### In scope

- Remove legacy session routes under `/api/sessions...`.
- Remove legacy preset/query metadata routes under `/api/presets` and `/api/queries...`.
- Remove legacy annotation routes under `/api/sessions/{id}/annotations` and `/api/annotations...`.
- Keep `POST /api/query` unchanged.
- Keep all `/api/v2/...` handlers and their helper code working.
- Update tests, mocks, comments, and README references that would otherwise still point at removed routes.

### Out of scope

- Replacing `POST /api/query` with a v2/protobuf wrapper.
- Changing the data model or behavior of existing v2 endpoints.
- Removing helper structs/functions that are still used by v2 adapters.
- Query-command API changes beyond preserving the existing v2-only routes.

## Current-state analysis

### Server route registration still exposes duplicated families

`cmd/go-minitrace/cmds/serve/server.go` registers both legacy and v2 session routes, then both legacy and v2 saved-query routes, then both legacy and v2 annotation routes. The raw SQL route is separate and still registered only as `POST /api/query`.

Evidence:
- `cmd/go-minitrace/cmds/serve/server.go:87-122`
  - legacy sessions: `/api/sessions...`
  - v2 sessions: `/api/v2/sessions...`
  - raw query: `POST /api/query`
  - legacy presets/queries: `/api/presets`, `/api/queries...`
  - v2 presets/queries: `/api/v2/presets`, `/api/v2/queries...`
  - legacy annotations: `/api/sessions/{id}/annotations`, `/api/annotations...`
  - v2 annotations: `/api/v2/sessions/{id}/annotations`, `/api/v2/annotations...`

### The first-party frontend already uses v2 for everything except raw SQL execution

`web/src/api/minitrace.ts` shows the active UI contract clearly:
- sessions hit `v2/sessions`, `v2/sessions/{id}`, `v2/sessions/{id}/summary`, `v2/sessions/{id}/blocks`
- saved queries/presets hit `v2/presets` and `v2/queries`
- annotations hit `v2/...`
- structured query commands hit `v2/query-commands`
- only `executeQuery` still posts to `query`

Evidence:
- `web/src/api/minitrace.ts:55-78`
- `web/src/api/minitrace.ts:81-115`
- `web/src/api/minitrace.ts:119-171`

### Storybook/MSW support still carries legacy route assumptions

The mock handler file still exposes old `/api/sessions`, `/api/presets`, `/api/queries`, and `POST /api/queries`, even though the production frontend no longer requests those routes. This is a drift signal: if the backend removes the legacy routes, mocks should stop pretending they matter.

Evidence:
- `web/src/mocks/handlers.ts:25-45` legacy session mocks
- `web/src/mocks/handlers.ts:63-76` legacy preset/query mocks
- `web/src/mocks/handlers.ts:102-119` legacy query-create mock

### Tests still actively exercise legacy handlers directly

`cmd/go-minitrace/cmds/serve/server_test.go` includes direct unit/integration tests for:
- legacy session handlers
- legacy preset/query handlers

Those tests are useful evidence that the old code still exists, but once the route family is intentionally removed they become a liability. The safer replacement is route-level regression coverage that the mux now returns 404 for the removed endpoints while the v2 handler tests remain intact.

Evidence:
- `cmd/go-minitrace/cmds/serve/server_test.go:202-405` legacy session tests
- `cmd/go-minitrace/cmds/serve/server_test.go:887-1048` legacy preset/query tests
- `cmd/go-minitrace/cmds/serve/server_test.go:408-886` and later v2 tests, which should remain

### README still advertises a legacy annotation surface

The top-level README serve section documents annotation endpoints only under `/api/...`, which is stale once those endpoints are intentionally removed. It also documents `POST /api/query`, which should remain.

Evidence:
- `README.md:118-136` legacy annotation table
- `README.md:161-169` raw SQL `POST /api/query` example that should stay valid

## Gap analysis

### Gap 1: Public route surface is larger than the supported product contract

Observed behavior: the backend still exposes both route families.

Desired behavior: the public contract should match the actively supported surface:
- `/api/query`
- `/api/v2/sessions...`
- `/api/v2/presets`
- `/api/v2/queries...`
- `/api/v2/query-commands...`
- `/api/v2/annotations...`

### Gap 2: Code ownership is blurred by dead handler entrypoints

Observed behavior: old handler methods still live next to shared helper code inside the same files.

Desired behavior: remove only the dead entrypoints while preserving reusable helper/model functions that v2 still calls, so the codebase becomes smaller without destabilizing shared logic.

### Gap 3: Supporting artifacts still tell the old story

Observed behavior: tests, mocks, and README content still model the removed routes.

Desired behavior: these artifacts should either disappear or be rewritten around the remaining route contract so reviewers and future contributors are not misled.

## Proposed solution

### High-level approach

1. Delete legacy route registrations from `Server.routes()`.
2. Remove dead v1 handler methods from the handler files.
3. Keep shared helper structs/functions that are still consumed by v2 handlers.
4. Replace direct legacy handler tests with route-level `404` regression tests against `server.mux`.
5. Update frontend mocks and comments to the v2-only contract.
6. Update README so it documents the surviving endpoints accurately.

### Preserve these pieces intentionally

Keep:
- `POST /api/query`
- all `/api/v2/...` routes
- helper types like `SavedQuery`, session normalization DTOs, SQL-dir helpers, and block normalization helpers when still used by v2

Remove:
- `handleGetSessions`, `handleGetSession`, `handleGetSessionSummary`, `handleGetSessionBlocks`
- `handleGetPresets`, `handleGetQueries`, `handleSaveQuery`, `handleUpdateQuery`, `handleDeleteQuery`
- `handleGetSessionAnnotations`, `handleCreateAnnotation`, `handleListAnnotations`, `handleUpdateAnnotation`, `handleDeleteAnnotation`, `handleSyncAnnotations`
- any helper only needed by those removed handlers (for example the old path-extraction helper)

### Route-level end state

After cleanup, `Server.routes()` should conceptually look like:

```go
func (s *Server) routes() {
    s.mux.HandleFunc("POST /api/query", s.handleExecuteQuery)

    s.mux.HandleFunc("GET /api/v2/sessions", s.handleGetSessionsV2)
    s.mux.HandleFunc("GET /api/v2/sessions/{id}", s.handleGetSessionV2)
    s.mux.HandleFunc("GET /api/v2/sessions/{id}/summary", s.handleGetSessionSummaryV2)
    s.mux.HandleFunc("GET /api/v2/sessions/{id}/blocks", s.handleGetSessionBlocksV2)

    s.mux.HandleFunc("GET /api/v2/presets", s.handleGetPresetsV2)
    s.mux.HandleFunc("GET /api/v2/queries", s.handleGetQueriesV2)
    s.mux.HandleFunc("POST /api/v2/queries", s.handleSaveQueryV2)
    s.mux.HandleFunc("PUT /api/v2/queries/{path...}", s.handleUpdateQueryV2)
    s.mux.HandleFunc("DELETE /api/v2/queries/{path...}", s.handleDeleteQueryV2)

    s.mux.HandleFunc("GET /api/v2/query-commands", s.handleGetQueryCommandsV2)
    s.mux.HandleFunc("POST /api/v2/query-commands/{path...}", s.handleExecuteQueryCommandV2)

    s.mux.HandleFunc("GET /api/v2/sessions/{id}/annotations", s.handleGetSessionAnnotationsV2)
    s.mux.HandleFunc("POST /api/v2/sessions/{id}/annotations", s.handleCreateAnnotationV2)
    s.mux.HandleFunc("GET /api/v2/annotations", s.handleListAnnotationsV2)
    s.mux.HandleFunc("PUT /api/v2/annotations/{annId}", s.handleUpdateAnnotationV2)
    s.mux.HandleFunc("DELETE /api/v2/annotations/{annId}", s.handleDeleteAnnotationV2)
    s.mux.HandleFunc("POST /api/v2/annotations/sync", s.handleSyncAnnotationsV2)
}
```

## File-by-file implementation plan

### Phase 1 — Ticket/docs/bookkeeping

Files:
- `ttmp/2026/04/09/GMT-004--remove-legacy-serve-http-v1-endpoints-superseded-by-v2/index.md`
- `.../tasks.md`
- `.../changelog.md`
- `.../design-doc/01-legacy-serve-http-cleanup-implementation-guide.md`
- `.../reference/01-cleanup-implementation-diary.md`

Actions:
1. Record the exact cleanup scope and the explicit non-goal that `/api/query` survives.
2. Add phased tasks that map cleanly to focused commits.
3. Relate the core server/frontend files that justify the work.

### Phase 2 — Remove legacy session endpoints

Files:
- `cmd/go-minitrace/cmds/serve/server.go`
- `cmd/go-minitrace/cmds/serve/handlers_sessions.go`
- `cmd/go-minitrace/cmds/serve/server_test.go`

Actions:
1. Remove the `/api/sessions...` route registrations.
2. Delete the legacy session handler methods only.
3. Keep the session DTO/helper functions used by v2.
4. Replace direct v1 tests with a mux-level 404 regression test.

### Phase 3 — Remove legacy preset/query metadata endpoints

Files:
- `cmd/go-minitrace/cmds/serve/server.go`
- `cmd/go-minitrace/cmds/serve/handlers_queries.go`
- `cmd/go-minitrace/cmds/serve/server_test.go`

Actions:
1. Remove `/api/presets` and `/api/queries...` route registrations.
2. Delete only the legacy handler methods and their v1 request structs if they become unused.
3. Keep `SavedQuery`, SQL-dir helpers, path validation helpers, and file IO helpers because v2 still uses them.
4. Add route-level 404 regression coverage for the removed endpoints.

### Phase 4 — Remove legacy annotation endpoints

Files:
- `cmd/go-minitrace/cmds/serve/server.go`
- `cmd/go-minitrace/cmds/serve/handlers_annotations.go`
- `cmd/go-minitrace/cmds/serve/server_test.go`

Actions:
1. Remove `/api/sessions/{id}/annotations` and `/api/annotations...` route registrations.
2. Delete the legacy annotation handler methods.
3. Remove dead request/path-extraction helpers that only served the legacy handlers.
4. Add mux-level 404 regression coverage for the removed annotation routes.

### Phase 5 — Update supporting artifacts

Files:
- `web/src/mocks/handlers.ts`
- `web/src/types/session.ts`
- `README.md`

Actions:
1. Remove stale legacy mock routes and replace them with the v2 shapes the adapters now expect.
2. Update comments that still describe old `/api/...` paths when the runtime now uses `/api/v2/...`.
3. Refresh the README serve API table to document the v2 annotation surface and preserve the `/api/query` example.

## Testing and validation strategy

### Automated validation

Run at minimum:

```bash
cd go-minitrace

go test ./cmd/go-minitrace/cmds/serve -count=1
go test ./... -count=1
docmgr doctor --ticket GMT-004 --stale-after 30
```

If pre-commit or local workflow expects it, also run:

```bash
golangci-lint run -v
```

### Behavioral checks to preserve

1. `POST /api/query` still accepts JSON SQL requests and enforces readonly validation.
2. All v2 session handler tests still pass.
3. All v2 preset/query metadata tests still pass.
4. All v2 annotation tests still pass.
5. Query-command v2 tests still pass untouched.
6. Removed legacy endpoints return 404 when exercised through `server.mux`.

## Risks, alternatives, and open questions

### Risks

1. **Accidentally deleting shared helpers:** `handlers_sessions.go` and `handlers_queries.go` mix reusable code with endpoint-specific code. The cleanup must remove only dead entrypoints.
2. **Mock drift after cleanup:** if Storybook/test mocks are not updated, frontend local development may still assume routes that the real server no longer exposes.
3. **Doc drift:** README and code comments can silently keep the old contract alive in readers’ heads even after the backend cleanup lands.

### Alternatives considered

1. **Keep the legacy routes forever as compatibility shims.** Rejected because there is no evidence in the repo that the first-party UI still depends on them, and they increase maintenance/test surface.
2. **Remove `POST /api/query` at the same time.** Rejected because the frontend still calls it directly and prior protobuf-rollout docs explicitly preserved it as the JSON-native exception.
3. **Only remove routes but keep dead handler methods.** Rejected because that still leaves dead code and misleading tests behind.

### Open questions

1. Whether any external clients outside this repository still depend on the legacy routes. The repo evidence does not show any.
2. Whether a future ticket should move raw SQL execution to `/api/v2/query` or a protobuf-backed equivalent; this ticket intentionally leaves that alone.

## References

- `cmd/go-minitrace/cmds/serve/server.go`
- `cmd/go-minitrace/cmds/serve/handlers_sessions.go`
- `cmd/go-minitrace/cmds/serve/handlers_sessions_v2.go`
- `cmd/go-minitrace/cmds/serve/handlers_queries.go`
- `cmd/go-minitrace/cmds/serve/handlers_queries_v2.go`
- `cmd/go-minitrace/cmds/serve/handlers_annotations.go`
- `cmd/go-minitrace/cmds/serve/handlers_annotations_v2.go`
- `cmd/go-minitrace/cmds/serve/server_test.go`
- `web/src/api/minitrace.ts`
- `web/src/mocks/handlers.ts`
- `web/src/types/session.ts`
- `README.md`
