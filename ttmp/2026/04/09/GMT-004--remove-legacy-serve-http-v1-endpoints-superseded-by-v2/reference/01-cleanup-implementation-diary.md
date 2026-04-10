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
    - Path: cmd/go-minitrace/cmds/serve/server.go
      Note: Initial route inventory recorded in Step 1
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
