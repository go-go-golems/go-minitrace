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
    - Path: README.md
      Note: Updated serve API docs to v2-only annotation routes in Step 3 (commit c1ca601)
    - Path: cmd/go-minitrace/cmds/serve/handlers_queries.go
      Note: Deleted legacy preset/query handler entrypoints in Step 2 (commit e44d8cf)
    - Path: cmd/go-minitrace/cmds/serve/handlers_sessions.go
      Note: Deleted legacy session handler entrypoints in Step 2 (commit e44d8cf)
    - Path: cmd/go-minitrace/cmds/serve/server.go
      Note: |-
        Initial route inventory recorded in Step 1
        Removed legacy session and query metadata route registrations in Step 2 (commit e44d8cf)
        Removed legacy annotation route registrations in Step 3 (commit c1ca601)
    - Path: cmd/go-minitrace/cmds/serve/server_test.go
      Note: |-
        Added DevMode mux 404 regression tests for removed routes in Step 2 (commit e44d8cf)
        Added mux-level 404 checks for removed annotation routes in Step 3 (commit c1ca601)
    - Path: web/src/api/minitrace.ts
      Note: Confirmed surviving frontend API contract
    - Path: web/src/mocks/handlers.ts
      Note: |-
        Captured stale mock surface during planning
        Reworked MSW handlers to v2/proto-json route shapes in Step 3 (commit c1ca601)
    - Path: web/src/types/session.ts
      Note: Updated API route comments to the v2-only surface in Step 3 (commit c1ca601)
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

## Step 3: Remove the legacy annotation surface and make the docs/mocks tell the truth

After the session and query-metadata cleanup, the remaining duplicated API family was annotations. Because the user had already clarified that we should treat this as a hard cutover, I removed the old annotation routes entirely instead of trying to preserve helper wrappers or deprecated aliases.

This slice reached a little wider than the backend. Deleting the legacy annotation file from the server was only half of the story; the frontend-facing mock layer and README still described the old contract. I updated the MSW handlers to serve v2 session and annotation shapes, rewrote the stale API comments in `web/src/types/session.ts`, and refreshed the README serve section so it now explicitly says the structured API is `/api/v2/...` while `POST /api/query` remains the lone JSON-native exception.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Finish the hard-cut cleanup by removing the last legacy route family and aligning supporting artifacts with the new route contract.

**Inferred user intent:** Make the backend, docs, and frontend support files consistently reflect a decisive v2-only API surface instead of a mixed transitional state.

**Commit (code):** `c1ca601` — `Remove legacy annotation routes and refresh API docs`

### What I did
- Removed the legacy annotation route registrations from `cmd/go-minitrace/cmds/serve/server.go`.
- Deleted `cmd/go-minitrace/cmds/serve/handlers_annotations.go` because nothing in it was still needed after the hard cutover.
- Added mux-level 404 tests for removed legacy annotation endpoints in `cmd/go-minitrace/cmds/serve/server_test.go`.
- Reworked `web/src/mocks/handlers.ts` so it mocks the active v2 session and annotation routes instead of the removed `/api/...` session/query surfaces.
- Updated route comments in `web/src/types/session.ts` from legacy `/api/...` paths to `/api/v2/...` paths.
- Updated `README.md` so the serve API section documents `/api/v2/...` annotation routes and explicitly preserves `POST /api/query` as the exception.
- Ran `go test ./cmd/go-minitrace/cmds/serve -count=1`, `golangci-lint run -v`, and `cd web && npm run build`.

### Why
- Once the backend hard-cut removed the old annotation routes, leaving legacy mock/docs references behind would create avoidable confusion for future work and for UI development.
- `handlers_annotations.go` contained only legacy entrypoints, so keeping it around after route removal would have been dead code by definition.
- The v2-first frontend deserves a mock layer that actually mirrors the active production contract.

### What worked
- Deleting the legacy annotation handler file was safe because the protobuf-backed v2 annotation handlers live in their own file and do not depend on the removed implementation.
- The new mux-level annotation absence tests behaved the same way as the earlier session/query checks when run with `DevMode: true`.
- `npm run build` succeeded after the MSW rewrite, which gave a useful sanity check that the TypeScript-side route-shape changes were coherent.

### What didn't work
- I accidentally tried to run `gofmt` on TypeScript files while batching validation commands.
- Exact failing command/output:
  - `cd go-minitrace && gofmt -w cmd/go-minitrace/cmds/serve/server.go cmd/go-minitrace/cmds/serve/server_test.go web/src/mocks/handlers.ts web/src/types/session.ts && go test ./cmd/go-minitrace/cmds/serve -count=1`
  - `web/src/mocks/handlers.ts:1:1: expected 'package', found 'import'`
  - `web/src/types/session.ts:3:1: expected 'package', found export`
- Fix: reran `gofmt` on Go files only and used the normal frontend build for TypeScript validation.

### What I learned
- The annotation cleanup was the easiest true hard deletion in this ticket because unlike sessions and saved queries, the old annotation file did not contain shared helper code that v2 still reused.
- The mock layer had drifted farther behind reality than I expected; it still modeled several removed routes and did not match the protobuf/v2 response envelopes the frontend adapters actually decode.

### What was tricky to build
- The tricky part was the mock rewrite, not the backend deletion. The old MSW handlers returned UI-shaped session objects directly, but the current frontend adapters decode protobuf-style v2 envelopes. That meant I had to build lightweight proto-JSON-shaped mock payloads for sessions, blocks, tool calls, badges, and annotations instead of merely renaming route paths. The symptom would have been successful route matches with decode failures or silent UI drift. The solution was to add explicit adapter helpers inside `web/src/mocks/handlers.ts` that convert the existing mock UI objects into the camelCase proto-json shape expected by the frontend decoders.

### What warrants a second pair of eyes
- `web/src/mocks/handlers.ts` because it now hand-builds proto-json-like session and annotation payloads for Storybook/MSW use.
- `README.md` to confirm the public route documentation now matches the intended hard-cut contract.
- `cmd/go-minitrace/cmds/serve/server_test.go` to ensure the removed annotation route assertions cover the right endpoints and nothing more.

### What should be done in the future
- Only the final validation/bookkeeping/delivery loop remains for this ticket.

### Code review instructions
- Start with `cmd/go-minitrace/cmds/serve/server.go` and verify that only `/api/v2/...` annotation routes remain.
- Confirm `cmd/go-minitrace/cmds/serve/handlers_annotations.go` was deleted and that `handlers_annotations_v2.go` still owns the live annotation API.
- Review `web/src/mocks/handlers.ts` and `README.md` together to ensure the supporting artifacts now describe the same route surface the backend exposes.
- Validate with:
  - `cd go-minitrace && go test ./cmd/go-minitrace/cmds/serve -count=1`
  - `cd go-minitrace && golangci-lint run -v`
  - `cd go-minitrace/web && npm run build`

### Technical details
- Commands run:
  - `cd go-minitrace && rm cmd/go-minitrace/cmds/serve/handlers_annotations.go`
  - `cd go-minitrace && gofmt -w cmd/go-minitrace/cmds/serve/server.go cmd/go-minitrace/cmds/serve/server_test.go`
  - `cd go-minitrace && go test ./cmd/go-minitrace/cmds/serve -count=1`
  - `cd go-minitrace && golangci-lint run -v`
  - `cd go-minitrace/web && npm run build`
- Related files for this step:
  - `cmd/go-minitrace/cmds/serve/server.go`
  - `cmd/go-minitrace/cmds/serve/server_test.go`
  - `web/src/mocks/handlers.ts`
  - `web/src/types/session.ts`
  - `README.md`

## Step 4: Validate, deliver, and close the cleanup ticket

With the hard-cut changes merged, the last job was to prove the repo and ticket artifacts were internally consistent. I reran the full Go test suite, reran `docmgr doctor`, checked the remaining serve route registrations directly, and then produced a bundled reMarkable upload containing the index, design doc, diary, tasks, and changelog so the cleanup can be reviewed off-device as a coherent package.

This step also flushed out one small delivery wrinkle: my first attempt to verify the remote folder used a path form that `remarquee cloud ls` did not like, even though the upload itself had succeeded. I recorded that failure and the corrected follow-up because these path-shape quirks have been easy to forget across tickets.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Finish the ticket cleanly: validate the codebase, validate the docs workspace, deliver the bundle, and make the cleanup easy to review later.

**Inferred user intent:** Do not stop at code changes; leave behind a fully validated, documented, and reviewable cleanup ticket.

### What I did
- Ran `go test ./... -count=1` from `go-minitrace`.
- Ran `docmgr doctor --ticket GMT-004 --stale-after 30`.
- Verified the remaining `Server.routes()` registrations with `rg -n 'HandleFunc\("(GET|POST|PUT|DELETE) /api/' cmd/go-minitrace/cmds/serve/server.go`.
- Confirmed reMarkable tooling/account with `remarquee status` and `remarquee cloud account --non-interactive`.
- Ran a dry-run bundled upload for the GMT-004 docs.
- Uploaded the real bundle named `GMT-004 legacy serve API cleanup`.
- Verified the uploaded entry after correcting the cloud-list path form.
- Marked the final task complete and prepared the ticket for closure.

### Why
- The user asked for an implementation guide, task-by-task execution, and a detailed diary, so the ticket needs a real finish line, not just merged code.
- `docmgr doctor` plus the reMarkable bundle are the best checks that the documentation side of the work stayed in sync with the code side.
- Route-surface verification is worthwhile here because the whole ticket is about deleting routes rather than adding behavior.

### What worked
- `go test ./... -count=1` passed after the cleanup.
- `docmgr doctor --ticket GMT-004 --stale-after 30` reported all checks passed.
- The route inventory now shows only `/api/v2/...` families plus `POST /api/query`.
- The reMarkable bundle uploaded successfully.

### What didn't work
- My first verification command for the uploaded bundle failed even though the upload had succeeded.
- Exact failing command/output:
  - `remarquee cloud ls '/ai/2026/04/09/GMT-004/' --long --non-interactive`
  - `Error: entry 'GMT-004' doesnt exist`
- Fix:
  - listed the parent directory first with `remarquee cloud ls '/ai/2026/04/09/' --long --non-interactive`
  - then verified the uploaded entry with `remarquee cloud ls '/ai/2026/04/09/GMT-004' --long --non-interactive`

### What I learned
- `remarquee cloud ls` can be picky about whether a remote target is treated as an entry versus a directory path; when in doubt, list the parent first and then the exact entry name.
- The final route inventory is now pleasantly simple: all structured serve APIs are on `/api/v2/...`, and `POST /api/query` is the only intentional holdout.

### What was tricky to build
- The tricky part here was not code, but validation sequencing. Because the ticket mixes backend deletions, frontend mock/docs updates, and delivery artifacts, I needed to validate in layers: backend tests/lint first, then docmgr health, then reMarkable upload, then remote verification. Skipping that ordering would have made it harder to tell whether a failure came from code, docs, or delivery tooling.

### What warrants a second pair of eyes
- The remote reMarkable listing path and bundle naming if anyone wants to verify the upload independently.
- The final serve route inventory in `server.go`, because that is the core behavioral output of the ticket.

### What should be done in the future
- If the project later decides to protobuf-wrap raw SQL execution too, create a separate ticket instead of reopening this cleanup one.

### Code review instructions
- Run:
  - `cd go-minitrace && go test ./... -count=1`
  - `cd go-minitrace && docmgr doctor --ticket GMT-004 --stale-after 30`
- Inspect `cmd/go-minitrace/cmds/serve/server.go` and confirm only `/api/v2/...` routes plus `POST /api/query` remain.
- If needed, verify the uploaded bundle with:
  - `remarquee cloud ls '/ai/2026/04/09/GMT-004' --long --non-interactive`

### Technical details
- Validation commands run:
  - `cd go-minitrace && go test ./... -count=1`
  - `cd go-minitrace && docmgr doctor --ticket GMT-004 --stale-after 30`
  - `cd go-minitrace && rg -n 'HandleFunc\("(GET|POST|PUT|DELETE) /api/' cmd/go-minitrace/cmds/serve/server.go`
- Delivery commands run:
  - `remarquee status`
  - `remarquee cloud account --non-interactive`
  - `cd go-minitrace && remarquee upload bundle --dry-run ttmp/2026/04/09/GMT-004--remove-legacy-serve-http-v1-endpoints-superseded-by-v2/index.md ttmp/2026/04/09/GMT-004--remove-legacy-serve-http-v1-endpoints-superseded-by-v2/design-doc/01-legacy-serve-http-cleanup-implementation-guide.md ttmp/2026/04/09/GMT-004--remove-legacy-serve-http-v1-endpoints-superseded-by-v2/reference/01-cleanup-implementation-diary.md ttmp/2026/04/09/GMT-004--remove-legacy-serve-http-v1-endpoints-superseded-by-v2/tasks.md ttmp/2026/04/09/GMT-004--remove-legacy-serve-http-v1-endpoints-superseded-by-v2/changelog.md --name "GMT-004 legacy serve API cleanup" --remote-dir "/ai/2026/04/09/GMT-004" --toc-depth 2`
  - `cd go-minitrace && remarquee upload bundle ttmp/2026/04/09/GMT-004--remove-legacy-serve-http-v1-endpoints-superseded-by-v2/index.md ttmp/2026/04/09/GMT-004--remove-legacy-serve-http-v1-endpoints-superseded-by-v2/design-doc/01-legacy-serve-http-cleanup-implementation-guide.md ttmp/2026/04/09/GMT-004--remove-legacy-serve-http-v1-endpoints-superseded-by-v2/reference/01-cleanup-implementation-diary.md ttmp/2026/04/09/GMT-004--remove-legacy-serve-http-v1-endpoints-superseded-by-v2/tasks.md ttmp/2026/04/09/GMT-004--remove-legacy-serve-http-v1-endpoints-superseded-by-v2/changelog.md --name "GMT-004 legacy serve API cleanup" --remote-dir "/ai/2026/04/09/GMT-004" --toc-depth 2`
  - `remarquee cloud ls '/ai/2026/04/09/' --long --non-interactive`
  - `remarquee cloud ls '/ai/2026/04/09/GMT-004' --long --non-interactive`
