---
Title: Cleanup implementation diary
Ticket: GMT-004
Status: active
Topics:
    - backend
    - documentation
    - minitrace
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/go-minitrace/cmds/serve/handlers_queries.go
      Note: Deleted legacy preset/query handler entrypoints in Step 2 (commit e44d8cf)
    - Path: cmd/go-minitrace/cmds/serve/handlers_sessions.go
      Note: Deleted legacy session handler entrypoints in Step 2 (commit e44d8cf)
    - Path: cmd/go-minitrace/cmds/serve/server.go
      Note: |-
        Initial route inventory recorded in Step 1
        Removed legacy session and query metadata route registrations in Step 2 (commit e44d8cf)
    - Path: cmd/go-minitrace/cmds/serve/server_test.go
      Note: Added DevMode mux 404 regression tests for removed routes in Step 2 (commit e44d8cf)
    - Path: web/src/api/minitrace.ts
      Note: Confirmed surviving frontend API contract
    - Path: web/src/mocks/handlers.ts
      Note: Captured stale mock surface during planning
ExternalSources: []
Summary: Chronological diary for the serve HTTP legacy-route cleanup work.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---



# Diary

## Goal

Capture the planning and implementation work for removing the legacy non-v2 `go-minitrace serve` HTTP endpoints that the first-party frontend no longer uses, while deliberately preserving `POST /api/query`.

## Step 1: Inventory the duplicated serve API surface and define a safe cleanup scope

I started by confirming that this cleanup request is not about query-command transport or about converting raw SQL execution to protobuf. The concrete job is narrower: identify which legacy `/api/...` route families are now dead weight, turn that understanding into a focused ticket plan, and make the eventual code changes safe by documenting what must stay alive.

The evidence was straightforward but important to collect before touching code. `server.go` still registers both legacy and v2 route families for sessions, saved-query metadata, and annotations, while the web API client already targets `/api/v2/...` for those domains and keeps only `POST /api/query` on the old surface. That means the cleanup can be aggressive about removing duplicated v1 handlers without breaking the current first-party frontend, as long as it leaves raw SQL execution alone.

### Prompt Context

**User prompt (verbatim):** "ok, do it. Make a new cleanup ticket and add tasks and a implementation guide for the above, and then work on it task by task, committing at appropriate intervals, and keeping a detailed diary."

**Assistant interpretation:** Create a new docmgr ticket for the legacy serve-API cleanup, write the implementation guide and task plan first, then carry out the cleanup in focused implementation slices with commit and diary bookkeeping throughout.

**Inferred user intent:** Reduce backend/API maintenance burden by removing dead serve endpoints and ensure the work is documented well enough that the cleanup process and rationale are reviewable later.

### What I did
- Created ticket `GMT-004` with title `Remove legacy serve HTTP v1 endpoints superseded by v2`.
- Added a primary design doc and a diary doc.
- Inspected route registrations in `cmd/go-minitrace/cmds/serve/server.go`.
- Inspected the active web API client in `web/src/api/minitrace.ts`.
- Inspected Storybook/MSW mocks in `web/src/mocks/handlers.ts`.
- Inspected the README serve API section to find stale legacy documentation.
- Wrote the first-pass task breakdown and implementation guide.

### Why
- The cleanup needs a clear non-goal: keep `POST /api/query`.
- The code mixes dead handler entrypoints with reusable helpers, so the plan needs to distinguish removable code from shared internals that v2 still needs.
- The user explicitly asked for a new ticket plus an implementation guide before coding.

### What worked
- Repo evidence was consistent: server routes still duplicate multiple families, but the frontend is already v2-first.
- The existing docmgr workflow already had the vocabulary needed for this ticket, so there was no vocabulary repair work at setup time.
- The split between route registration, handler entrypoints, and helper logic is clear enough to support focused commits.

### What didn't work
- N/A so far.

### What I learned
- The highest-value cleanup is not just deleting routes from `server.go`; it is also deleting the dead handler methods and the tests/mocks/docs that keep those endpoints looking alive.
- Query commands do not need any cleanup here because they were introduced as v2-only endpoints from the start.

### What was tricky to build
- The tricky part in planning is avoiding an over-broad cleanup. `handlers_sessions.go` and `handlers_queries.go` are not pure “legacy files”; they also host helper structs and normalization/file-loading logic that the v2 handlers still depend on. The plan therefore targets handler method removal, not wholesale file deletion.

### What warrants a second pair of eyes
- The exact set of helper types/functions that remain referenced by v2 after legacy handlers are removed.
- Whether any external, non-repo clients still depend on the old routes. The repo itself does not show any.

### What should be done in the future
- After this ticket, decide separately whether `POST /api/query` should eventually get a v2/protobuf wrapper.

### Code review instructions
- Start with `cmd/go-minitrace/cmds/serve/server.go` to see the duplicated route registrations.
- Then compare `web/src/api/minitrace.ts` to confirm which routes the first-party frontend still hits.
- Finally review `web/src/mocks/handlers.ts` and `README.md` to see the stale supporting artifacts that should change along with the backend cleanup.

### Technical details
- Evidence commands run:
  - `docmgr status --summary-only`
  - `docmgr ticket create-ticket --ticket GMT-004 --title "Remove legacy serve HTTP v1 endpoints superseded by v2" --topics backend,documentation,minitrace`
  - `docmgr doc add --ticket GMT-004 --doc-type design-doc --title "Legacy serve HTTP cleanup implementation guide"`
  - `docmgr doc add --ticket GMT-004 --doc-type reference --title "Cleanup implementation diary"`
  - `nl -ba cmd/go-minitrace/cmds/serve/server.go | sed -n '70,125p'`
  - `nl -ba web/src/api/minitrace.ts | sed -n '50,120p'`
  - `nl -ba web/src/mocks/handlers.ts | sed -n '1,140p'`

## Step 2: Hard-cut the legacy session and saved-query route families

Once the ticket plan was in place, I moved directly into the backend cleanup instead of preserving any compatibility shims. The user explicitly clarified that we can assume a hard cutover, so I treated the old `/api/sessions...`, `/api/presets`, and `/api/queries...` surfaces as dead code rather than as something to deprecate gently.

The actual code change was mostly subtraction. I removed the legacy route registrations from `server.go`, deleted the dead session and saved-query handler entrypoints, and kept only the shared normalization/file-loading helper code that the v2 handlers still call. I also replaced the old direct-handler tests with mux-level tests that prove the removed routes are now absent when the server runs in `DevMode`, which avoids the SPA fallback turning missing API paths into HTML 200 responses.

### Prompt Context

**User prompt (verbatim):** "I don't wnat to keep any legacy stuff over, we can pretty much assume hard cutover"

**Assistant interpretation:** Treat the remaining duplicated serve endpoints as a hard removal rather than as a compatibility/deprecation exercise.

**Inferred user intent:** Keep the cleanup decisive and avoid carrying dead routes or shims just because they used to exist.

**Commit (code):** `e44d8cf` — `Remove legacy session and query metadata routes`

### What I did
- Removed legacy session route registrations from `cmd/go-minitrace/cmds/serve/server.go`.
- Removed legacy preset/query route registrations from `cmd/go-minitrace/cmds/serve/server.go`.
- Deleted the v1 session handlers from `cmd/go-minitrace/cmds/serve/handlers_sessions.go`.
- Deleted the v1 preset/query handlers plus unused request structs from `cmd/go-minitrace/cmds/serve/handlers_queries.go`.
- Removed the now-unused `decodeRequest(...)` helper from `cmd/go-minitrace/cmds/serve/server.go` after lint reported it as dead.
- Replaced direct legacy session/query tests in `cmd/go-minitrace/cmds/serve/server_test.go` with mux-level 404 checks that run with `DevMode: true`.
- Ran `go test ./cmd/go-minitrace/cmds/serve -count=1`, then the pre-commit `go test ./...` and `golangci-lint run -v` via commit hooks.

### Why
- The first-party frontend already uses the v2 routes for these domains, so keeping the old route families only increases maintenance surface.
- The user explicitly said not to preserve legacy compatibility.
- Route-level absence tests are a better long-term fit than continuing to exercise deleted handlers directly.

### What worked
- The handler files separated cleanly into "dead entrypoints" and "shared helpers still used by v2", so most of the change was straightforward deletion.
- The v2 tests continued to pass without code changes, which confirmed that the shared session normalization and saved-query helper code remained intact.
- Running the mux-level absence tests in `DevMode` avoided accidental SPA fallback interference.

### What didn't work
- My first attempt at the absence tests expected 404s from the default server config, but the non-dev `spaHandler(frontendFS)` fallback returned the frontend HTML with status 200 instead.
- Exact failing command/output:
  - `cd go-minitrace && go test ./cmd/go-minitrace/cmds/serve -count=1`
  - `--- FAIL: TestLegacySessionRoutesReturnNotFound (0.00s)`
  - `server_test.go:223: expected 404, got 200 with body <!doctype html> ...`
- Fix: instantiate the test server with `DevMode: true` so the SPA catch-all route is not registered when asserting missing API endpoints.

### What I learned
- For route-removal tests in this server, `DevMode` is the easiest way to assert true mux absence because otherwise the SPA fallback masks missing endpoints.
- Removing one family of legacy handlers can expose additional dead helpers immediately; in this slice, `decodeRequest(...)` became unused as soon as the old JSON query-metadata handlers disappeared.

### What was tricky to build
- The tricky part was the interaction between API route removal and the SPA catch-all. A missing API route does not naturally produce a 404 in the default test server because the embedded frontend is mounted at `/`. The symptom was an unexpected HTML document with status 200 in the new regression test. The solution was to run the absence assertions with `ServeSettings{DevMode: true}`, which skips the SPA route and lets the mux report 404 for genuinely missing API endpoints.

### What warrants a second pair of eyes
- `cmd/go-minitrace/cmds/serve/server.go` to confirm the intended route families are the only ones removed.
- `cmd/go-minitrace/cmds/serve/server_test.go` to confirm the new absence tests cover the intended hard-cut endpoints without accidentally asserting behavior hidden by the SPA fallback.
- `cmd/go-minitrace/cmds/serve/handlers_queries.go` to confirm only dead entrypoints were removed and not the helper code reused by v2 CRUD handlers.

### What should be done in the future
- Finish the same hard-cut removal for the legacy annotation route family and then update mocks/docs so the supporting artifacts match the backend cleanup.

### Code review instructions
- Start with `cmd/go-minitrace/cmds/serve/server.go` and verify that legacy session/query metadata registrations are gone while `/api/query` and `/api/v2/...` remain.
- Review `cmd/go-minitrace/cmds/serve/handlers_sessions.go` and `cmd/go-minitrace/cmds/serve/handlers_queries.go` to ensure only dead handler entrypoints were deleted.
- Run:
  - `cd go-minitrace && go test ./cmd/go-minitrace/cmds/serve -count=1`
  - `cd go-minitrace && golangci-lint run -v`

### Technical details
- Commands run:
  - `cd go-minitrace && gofmt -w cmd/go-minitrace/cmds/serve/server.go cmd/go-minitrace/cmds/serve/handlers_sessions.go cmd/go-minitrace/cmds/serve/handlers_queries.go cmd/go-minitrace/cmds/serve/server_test.go`
  - `cd go-minitrace && go test ./cmd/go-minitrace/cmds/serve -count=1`
  - `cd go-minitrace && golangci-lint run -v`
- Related files for this step:
  - `cmd/go-minitrace/cmds/serve/server.go`
  - `cmd/go-minitrace/cmds/serve/handlers_sessions.go`
  - `cmd/go-minitrace/cmds/serve/handlers_queries.go`
  - `cmd/go-minitrace/cmds/serve/server_test.go`
