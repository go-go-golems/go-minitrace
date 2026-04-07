---
Title: Investigation diary
Ticket: PROTO-API-001
Status: active
Topics:
    - backend
    - frontend
    - minitrace
    - go-minitrace
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/go-minitrace/cmds/serve/handlers_annotations.go
      Note: |-
        Current handwritten annotation request/response layer and weakly typed patch handling.
        Annotation API analysis evidence for Step 1
    - Path: cmd/go-minitrace/cmds/serve/handlers_queries.go
      Note: |-
        Saved-query metadata DTOs and the dynamic query-result surface.
        Saved-query metadata and dynamic query-result analysis evidence for Step 1
    - Path: cmd/go-minitrace/cmds/serve/handlers_sessions.go
      Note: |-
        Current handwritten session API DTOs and normalization helpers.
        Session DTO analysis evidence for Step 1
    - Path: pkg/annotate/store.go
      Note: |-
        Flattened annotation row shape currently surfaced directly to the frontend.
        Flattened annotation row evidence for Step 1
    - Path: pkg/minitrace/schema.go
      Note: |-
        Internal archive/domain model that should remain distinct from the public API schema.
        Internal archive schema evidence for Step 1
    - Path: web/src/api/minitrace.ts
      Note: |-
        Frontend RTK Query layer built on handwritten transport types.
        Frontend API wiring analysis evidence for Step 1
    - Path: web/src/types/query.ts
      Note: |-
        Frontend handwritten transport interfaces for saved queries and query results.
        Frontend query transport type duplication evidence for Step 1
    - Path: web/src/types/session.ts
      Note: |-
        Frontend handwritten transport interfaces for sessions and annotations.
        Frontend transport type duplication evidence for Step 1
ExternalSources: []
Summary: Chronological diary for the protobuf-backed API contract alignment ticket.
LastUpdated: 2026-04-06T20:38:00-04:00
WhatFor: Record the analysis, implementation steps, and validation trail for the protobuf alignment work.
WhenToUse: Use when continuing or reviewing the protobuf-backed API type-alignment ticket.
---


# Diary

## Goal

Capture the investigation and implementation steps for aligning `go-minitrace` backend/frontend API contracts with protobuf-generated Go and TypeScript code.

## Step 1: Create the ticket, map the current API duplication, and write the implementation guide

This step established the new ticket workspace and converted the earlier conversational recommendation into an evidence-backed design document. The goal was to create a durable starting point before making any code changes, especially because the repository already contains unrelated in-progress frontend work that should not be mixed into this ticket accidentally.

I focused on the current API boundary rather than the entire internal archive model. That distinction matters because the repo already has a normalization layer from `pkg/minitrace` session data into web API DTOs, which makes the API contract the most practical protobuf boundary.

### Prompt Context

**User prompt (verbatim):** "ok, create a new docmgr ticket for that, and create a detailed analysis and implementation guide, and add tasks for it. Then work on the tasks one by one, commit at appropriate intervals, and keep a detailed diary."

**Assistant interpretation:** Create a fresh ticket workspace for protobuf-based API alignment, write the detailed design/analysis docs and task plan, then use that plan to drive incremental implementation.

**Inferred user intent:** Turn the protobuf alignment idea into a disciplined ticket with durable documentation, explicit tasks, and step-by-step implementation history rather than ad hoc coding.

**Commit (code):** pending — documentation checkpoint not committed yet at the time of writing this entry

### What I did

- Ran `docmgr status --summary-only` to confirm the doc workspace configuration.
- Created ticket `PROTO-API-001` with `docmgr ticket create-ticket`.
- Added:
  - a design doc
  - a diary doc
- Inspected the current backend/frontend API DTOs and transport boundaries in:
  - `cmd/go-minitrace/cmds/serve/server.go`
  - `cmd/go-minitrace/cmds/serve/handlers_sessions.go`
  - `cmd/go-minitrace/cmds/serve/handlers_annotations.go`
  - `cmd/go-minitrace/cmds/serve/handlers_queries.go`
  - `pkg/annotate/store.go`
  - `pkg/minitrace/schema.go`
  - `web/src/api/minitrace.ts`
  - `web/src/types/session.ts`
  - `web/src/types/query.ts`
- Read the protobuf Go/TS schema-exchange skill plus its templates and validation references.
- Verified there is currently no protobuf/Buf scaffolding in the repo.
- Wrote:
  - ticket index
  - detailed design/implementation guide
  - phased task list
  - changelog entry

### Why

- The protobuf migration needs a well-defined boundary and phased rollout or it will easily sprawl into a full internal-model rewrite.
- The repo currently has handwritten DTO duplication across Go and TypeScript, but not every surface is equally suitable for protobuf. The ticket needed to document where protobuf helps most and where it should be deferred.
- The working tree is currently dirty with unrelated frontend work, so the ticket had to be set up in a way that keeps this effort isolated and reviewable.

### What worked

- The repo already has a clean API normalization layer for sessions, which strongly supports protobuf at the DTO boundary.
- The evidence is clear that sessions and annotations are good first protobuf targets.
- The design doc now captures a practical phased plan instead of a vague “use protobuf” recommendation.

### What didn't work

- `npm ls @bufbuild/protobuf` returned exit code `1`, which is expected here because the dependency is not installed yet.
- The repo is not clean at the start of this ticket, so I cannot assume future commits can safely use `git add .` or broad staging patterns.

### What I learned

- The flattened annotation list row is the strongest proof that the current API layer needs a consciously owned schema. The frontend is forced to mirror Go-exported field casing today.
- The dynamic query execution result surface is a poor first protobuf target because it is intentionally open-ended.

### What was tricky to build

- The tricky part was choosing the correct protobuf boundary. It would be easy to overreach and try to replace the archive schema itself, but that would mix storage/import concerns with the public API contract.
- Another subtle point was balancing “across the board” alignment with practical rollout safety. The phased plan therefore covers all typed API surfaces while explicitly deferring arbitrary query result rows.

### What warrants a second pair of eyes

- Whether the v2 rollout should keep a full-detail `/api/v2/sessions/{id}` route or push clients toward summary + blocks only.
- Whether any token/count fields need 64-bit representation, which would affect TS `bigint` ergonomics.

### What should be done in the future

- Start Step 2 by adding Buf/proto scaffolding and generation runtime dependencies.
- Keep commits narrowly scoped because unrelated frontend work is already present in the tree.

### Code review instructions

Review in this order:

1. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/PROTO-API-001--align-go-and-typescript-api-types-with-protobuf-schema-first-contracts/design-doc/01-protobuf-schema-first-api-alignment-analysis-and-implementation-guide.md`
2. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/PROTO-API-001--align-go-and-typescript-api-types-with-protobuf-schema-first-contracts/tasks.md`
3. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/PROTO-API-001--align-go-and-typescript-api-types-with-protobuf-schema-first-contracts/index.md`
4. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/PROTO-API-001--align-go-and-typescript-api-types-with-protobuf-schema-first-contracts/changelog.md`

Validation commands used in this step:

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && docmgr status --summary-only
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && docmgr ticket create-ticket --ticket PROTO-API-001 --title "Align Go and TypeScript API types with protobuf schema-first contracts" --topics backend,frontend,minitrace,go-minitrace,documentation
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && docmgr doc add --ticket PROTO-API-001 --doc-type design-doc --title "Protobuf schema-first API alignment analysis and implementation guide"
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && docmgr doc add --ticket PROTO-API-001 --doc-type reference --title "Investigation diary"
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && buf --version
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm ls @bufbuild/protobuf
```

### Technical details

Important evidence gathered for the guide:

- handwritten session DTOs exist in `handlers_sessions.go`
- handwritten annotation request/response types exist in `handlers_annotations.go`
- saved-query metadata DTOs exist in `handlers_queries.go`
- dynamic query execution currently uses `[]map[string]any` in `server.go`
- frontend mirrors these shapes manually in `web/src/types/session.ts` and `web/src/types/query.ts`
- no existing `buf.yaml`, `buf.gen.yaml`, `proto/`, or generated protobuf runtime dependency is currently present
