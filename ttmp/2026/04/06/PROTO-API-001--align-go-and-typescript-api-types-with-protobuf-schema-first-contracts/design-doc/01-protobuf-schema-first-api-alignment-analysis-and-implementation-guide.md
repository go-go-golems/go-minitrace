---
Title: Protobuf schema-first API alignment analysis and implementation guide
Ticket: PROTO-API-001
Status: active
Topics:
    - backend
    - frontend
    - minitrace
    - go-minitrace
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/go-minitrace/cmds/serve/handlers_annotations.go
      Note: |-
        Handwritten annotation request/response shapes and map-based patch decoding.
        Current handwritten annotation API request/response layer
    - Path: cmd/go-minitrace/cmds/serve/handlers_queries.go
      Note: |-
        Saved-query metadata DTOs and the dynamic query surface that is likely a poor first protobuf target.
        Current saved-query metadata and dynamic query-result surfaces
    - Path: cmd/go-minitrace/cmds/serve/handlers_sessions.go
      Note: |-
        Handwritten session summary/detail/block DTOs and route handlers.
        Current handwritten session API DTO layer
    - Path: cmd/go-minitrace/cmds/serve/server.go
      Note: |-
        Current HTTP route table plus generic query request/response DTOs and JSON helpers.
        Current route table and JSON transport helpers
    - Path: pkg/annotate/store.go
      Note: |-
        Flattened annotation row shape currently leaked directly into the frontend.
        Flattened annotation row shape currently leaked to the frontend
    - Path: pkg/minitrace/schema.go
      Note: |-
        Internal archive/domain model that should remain separate from the public API schema.
        Internal archive/domain schema that should remain distinct from the public API contract
    - Path: web/src/api/minitrace.ts
      Note: |-
        Frontend API layer currently typed entirely with handwritten interfaces.
        Frontend RTK Query layer that will eventually decode generated protobuf JSON
    - Path: web/src/types/query.ts
      Note: |-
        Saved-query metadata and dynamic query result interfaces.
        Handwritten frontend transport types for saved queries and query results
    - Path: web/src/types/session.ts
      Note: |-
        Handwritten session/annotation interfaces mirroring backend DTOs.
        Handwritten frontend transport types for sessions and annotations
ExternalSources:
    - https://tanstack.com/virtual/latest/docs/api/virtualizer
Summary: Evidence-backed analysis of how go-minitrace can align backend/frontend API types using protobuf, Buf codegen, protojson, and generated TypeScript decoders, with a phased implementation plan.
LastUpdated: 2026-04-06T20:30:00-04:00
WhatFor: Understand the current API type duplication, decide where protobuf fits and where it does not, and provide a clear phased implementation plan for aligning Go and TypeScript API contracts.
WhenToUse: Use when implementing schema-first API contracts in go-minitrace or reviewing how backend HTTP DTOs and frontend TypeScript types should be kept in sync.
---


# Protobuf schema-first API alignment analysis and implementation guide

## Executive summary

`go-minitrace` currently maintains its web API contracts in three separate forms: the internal Go domain model, handwritten Go HTTP response/request DTOs, and handwritten TypeScript interfaces in the frontend. This works, but it creates a persistent drift risk because the backend and frontend must be changed together by convention instead of by generated code.

The current duplication is especially visible in the session and annotation APIs. The backend defines dedicated transport structs in `cmd/go-minitrace/cmds/serve/handlers_sessions.go` and `cmd/go-minitrace/cmds/serve/handlers_annotations.go`, while the frontend mirrors them manually in `web/src/types/session.ts` and consumes them through RTK Query in `web/src/api/minitrace.ts`. One particularly sharp edge is the flattened annotation list shape, which currently leaks Go-exported field casing (`ID`, `SessionID`, `ScopeType`, etc.) directly into the frontend instead of presenting a consciously designed schema.

This document proposes a schema-first alignment layer built around protobuf, Buf code generation, Go `protojson`, and generated TypeScript decoders. The core recommendation is to treat the **public API DTO layer** as the protobuf boundary, while keeping the internal archive/domain model in `pkg/minitrace/schema.go` separate. The first implementation phase should cover the typed REST surfaces that are already stable and highly structured: sessions, annotations, and saved-query metadata. The ad hoc SQL query result surface should be explicitly deferred because it is dynamic by nature and fits protobuf much less naturally.

The preferred rollout is a versioned `/api/v2/...` surface with protobuf-generated messages rendered as JSON via `protojson` and decoded in the frontend with generated schemas. That keeps the migration incremental, avoids breaking the existing snake_case `/api` routes immediately, and gives the frontend a generated source of truth for the most important API contracts.

## Problem statement

The web API contract in `go-minitrace` is currently duplicated across layers.

On the backend:

- `cmd/go-minitrace/cmds/serve/handlers_sessions.go` defines `SessionSummaryResponse`, `SessionSummaryDetailResponse`, `SessionDetailResponse`, `SessionBlock`, `TurnResponse`, `ToolCallResponse`, and related nested structs.
- `cmd/go-minitrace/cmds/serve/handlers_annotations.go` defines `CreateAnnotationRequest` and emits mixed response shapes, including raw `minitrace.Annotation`, ad hoc `map[string]any`, and flattened store rows.
- `cmd/go-minitrace/cmds/serve/handlers_queries.go` defines `SavedQuery`, `SaveQueryRequest`, `UpdateQueryRequest`, and uses the generic `QueryResponse` from `cmd/go-minitrace/cmds/serve/server.go`.

On the frontend:

- `web/src/types/session.ts` defines interfaces for sessions, transcript blocks, annotations, sync reports, and the flattened annotation list row.
- `web/src/types/query.ts` defines interfaces for saved queries and query execution results.
- `web/src/api/minitrace.ts` wires these handwritten interfaces into RTK Query.

This design creates several concrete problems:

1. **Backend/frontend drift risk**. A field can be renamed or widened in Go without any generated compiler guidance for the TS consumer.
2. **Inconsistent API intentionality**. Some endpoints use carefully normalized JSON shapes, while others leak storage- or implementation-driven shapes into the client.
3. **Patch/update ambiguity**. The annotation update endpoint currently decodes a generic `map[string]any` and manually reconstructs a patch object.
4. **No explicit versioned contract boundary**. The transport layer is currently “whatever the handler emits,” not a dedicated schema artifact.
5. **Internal model vs public API coupling pressure**. The archive schema in `pkg/minitrace/schema.go` is richer than the UI needs, but there is no formal distinction beyond handwritten normalization functions.

The goal of this ticket is not to protobuf-ify everything indiscriminately. The goal is to introduce a schema-first API contract that keeps the frontend and backend aligned while respecting which parts of the system are structured enough to benefit from protobuf and which parts are not.

## Scope

### In scope

- public HTTP API DTOs for sessions
- public HTTP API DTOs for annotations
- public HTTP API DTOs for saved query metadata (`/api/presets`, `/api/queries`)
- Buf scaffolding (`buf.yaml`, `buf.gen.yaml`, `proto/` layout)
- generated Go and TypeScript outputs
- Go JSON emission using protobuf-generated messages and `protojson`
- frontend decode layer using generated schemas instead of handwritten API interfaces
- versioned rollout strategy (`/api/v2/...` preferred)

### Out of scope for the first phase

- replacing the internal archive/domain structs in `pkg/minitrace/schema.go`
- replacing DuckDB archive schema definitions with protobuf
- converting ad hoc SQL execution results to strongly typed protobuf rows
- gRPC or Connect transport adoption
- CLI payload redesign unrelated to the web API

## Current-state architecture

This section explains the current API layering so a new engineer understands exactly what is being changed.

## 1. Internal model vs HTTP DTOs

The canonical archive/domain model currently lives in `pkg/minitrace/schema.go`. The top-level `Session` struct includes fields such as:

- `SchemaVersion`
- `Profile`
- `Quality`
- `Flags`
- `Condition`
- `Coordination`
- `Handover`
- `Turns`
- `ToolCalls`
- `Annotations`
- `Metrics`

That file defines the full archived session shape, not just the fields the web UI consumes.

Important evidence:

- `pkg/minitrace/schema.go` defines `Session`, `Turn`, `ToolCall`, `Annotation`, `Metrics`, and many additional nested structs.
- many of these fields do not appear in the current web API DTOs at all.

This is important because it means the current web API is already a **projection** of the archive model rather than a 1:1 reflection.

## 2. Sessions API today

The sessions API is implemented through handwritten transport structs in `cmd/go-minitrace/cmds/serve/handlers_sessions.go`.

Key evidence:

- `SessionSummaryResponse` at lines `17-26`
- `SessionSummaryDetailResponse` at lines `28-31`
- `SessionDetailResponse` at lines `33-36`
- nested timing/metrics/environment/provenance structs at lines `38-71`
- transcript block/turn/tool-call structs at lines `75-124`

Route handlers:

- `GET /api/sessions` in `handleGetSessions()` at lines `127-196`
- `GET /api/sessions/{id}` in `handleGetSession()` at lines `198-217`
- `GET /api/sessions/{id}/summary` in `handleGetSessionSummary()` at lines `219-238`
- `GET /api/sessions/{id}/blocks` in `handleGetSessionBlocks()` at lines `240-259`

The backend uses normalization functions to convert internal `minitrace.Session` values into these transport structs. That is good architectural news: a schema-first transport layer already has a logical insertion point.

## 3. Transcript block construction today

The transcript block list is built in `cmd/go-minitrace/cmds/serve/blocks.go`.

Important behavior:

- `buildSessionBlocks(session)` groups turns into transcript blocks.
- `buildRawSessionBlocks(...)` derives block boundaries from user turns.
- `normalizeTurn(...)` and `normalizeToolCall(...)` shape nested turn/tool-call DTOs.

This reinforces the design point that the public transcript route already serves a normalized view model, not raw archive JSON.

## 4. Annotations API today

The annotations HTTP surface mixes several kinds of shapes.

Evidence in `cmd/go-minitrace/cmds/serve/handlers_annotations.go`:

- `CreateAnnotationRequest` at lines `16-29`
- `GET /api/sessions/{id}/annotations` returns `map[string]any{ session_id, count, annotations }` at lines `31-58`
- `POST /api/sessions/{id}/annotations` decodes a handwritten request struct and emits raw `minitrace.Annotation` at lines `60-148`
- `GET /api/annotations` returns flattened rows from the SQLite store at lines `150-186`
- `PUT /api/annotations/{annId}` decodes `map[string]any` and reconstructs `annotate.AnnotationPatch` at lines `188-260`
- `POST /api/annotations/sync` emits the store sync report at the bottom of the file

This surface is functional, but it is not schema-uniform. Some responses are normalized message-like payloads; others are raw domain structs or anonymous maps.

## 5. Flattened annotation row leak

The strongest current evidence that the transport schema is under-specified is the flattened annotation list row.

In `pkg/annotate/store.go`:

- `AnnotationRow` is defined at lines `167-184`.
- `List(...)` returns `[]AnnotationRow` at lines `188-255`.

These fields are exported with Go names like:

- `ID`
- `SessionID`
- `ScopeType`
- `CreatedAt`

On the frontend, `web/src/types/session.ts` has to mirror that exact shape in `AnnotationListRow` at lines `180-197`, including the capitalized field names.

That is a clear sign that the frontend is currently shaped by Go implementation details instead of a contract-first schema.

## 6. Frontend types today

The frontend mirrors the backend transport structs manually.

Evidence in `web/src/types/session.ts`:

- session timing/metrics/environment/context: lines `3-36`
- session summary/detail: lines `38-125`
- annotation and sync report types: lines `127-204`
- flattened annotation list row with capitalized fields: lines `180-197`

Evidence in `web/src/types/query.ts`:

- `SavedQuery`: lines `3-10`
- `QueryResult`: lines `12-17`
- `QueryError`: lines `19-23`

Evidence in `web/src/api/minitrace.ts`:

- all session, query, and annotation endpoints are wired using these handwritten TS interfaces.
- `getAnnotations` currently has a defensive `transformResponse` only to normalize `null` to `[]`, not to decode a schema-defined message.

## 7. Dynamic query surface today

The generic query execution surface is defined in `cmd/go-minitrace/cmds/serve/server.go` and mirrored in `web/src/types/query.ts`.

Evidence:

- `QueryResponse` in `server.go` lines `44-50` uses `Rows []map[string]any`
- `QueryResult` in `web/src/types/query.ts` lines `12-17` uses `rows: Record<string, unknown>[]`

This surface is fundamentally dynamic. It is valid JSON, but it is a poor first candidate for schema-first strong typing because the result columns vary arbitrarily by SQL query.

## Gap analysis

The current design gets the job done, but it leaves several gaps that protobuf can address for the structured surfaces.

### Gap 1: No generated cross-language source of truth

Today there is no artifact that both Go and TypeScript share as a contract. Instead, each side maintains a local mirror of the same idea.

### Gap 2: Public DTO semantics are informal

The distinction between:

- archive schema
- internal domain structs
- public API DTOs

is real in code, but not formalized as a standalone schema package.

### Gap 3: No route versioning for transport evolution

The API already evolved once to add summary/body splitting for sessions. A schema-first approach benefits from a versioned route surface where the new contract can coexist with the current JSON until migration is complete.

### Gap 4: Update/patch contracts are weakly typed

The annotation patch endpoint currently relies on runtime JSON key inspection. That is a sign that the transport contract should be made explicit.

### Gap 5: Dynamic and structured surfaces are not distinguished enough

Some routes are highly structured and stable (`sessions`, `annotations`, `queries metadata`), while others are intentionally dynamic (`query execution results`). The current handwritten approach treats them all as just “JSON,” which can obscure where schema-first tooling is most valuable.

## Proposed solution

The recommended solution is to introduce a dedicated protobuf-backed API contract layer for the typed HTTP surfaces.

### Core design

```text
internal archive/domain model (pkg/minitrace/schema.go)
  -> normalization layer
    -> protobuf API messages (canonical transport schema)
      -> JSON via protojson
        -> generated TypeScript schemas + fromJson()
          -> React view-model usage
```

This means:

- keep the internal archive model in Go as-is for now,
- define API transport messages in `.proto` files,
- generate Go and TS code from the same schema,
- emit JSON using `protojson` on the backend,
- decode JSON using generated schemas on the frontend,
- migrate the frontend endpoint-by-endpoint to generated types.

### Why the protobuf boundary should be the API DTO layer

The archive schema is richer than the public UI contract and includes fields that the UI does not need. If we try to make the archive struct itself the protobuf boundary immediately, we risk over-coupling transport and storage concerns.

The current backend already has an explicit normalization step (`normalizeSessionSummaryDetail`, `normalizeSessionDetail`, `buildSessionBlocks`, manual annotation request/response shaping), so the most practical migration point is the **public API DTO layer**.

### Why `/api/v2/...` is preferred

The protobuf skill strongly favors `protojson` JSON, which naturally uses camelCase names. The current API is predominantly snake_case.

We therefore have two broad choices:

1. force protobuf-generated JSON to mimic the current snake_case surface, or
2. introduce versioned `/api/v2/...` endpoints with protobuf-native JSON conventions.

The second option is cleaner because it avoids contorting the new schema around old naming assumptions. It also gives us space to move from bare arrays to explicit message envelopes.

### Why envelopes are preferred over top-level arrays

Several current endpoints return bare arrays:

- `GET /api/sessions`
- `GET /api/sessions/{id}/blocks`
- `GET /api/queries`
- `GET /api/presets`
- `GET /api/annotations`

For protobuf-backed JSON, explicit response messages are preferable. Example:

```json
{
  "schemaVersion": 1,
  "sessions": [ ... ]
}
```

instead of:

```json
[ ... ]
```

This provides:

- schema versioning
- a stable top-level message type
- space for metadata, pagination, and diagnostics later

## Proposed proto package layout

Recommended layout:

```text
proto/go_go_golems/minitrace/api/v1/
  common.proto
  sessions.proto
  annotations.proto
  queries.proto
```

### `common.proto`

Shared message types and enums used by multiple API surfaces.

Likely contents:

- `ApiEnvelopeMeta` or a small schema version field strategy
- `SessionTiming`
- `SessionMetrics`
- `SessionEnvironment`
- `SessionOperationalContext`
- `SessionProvenance`
- `ToolCallBadge` enum
- `AnnotationCategory` enum
- `AnnotationScopeType` enum

### `sessions.proto`

Likely contents:

- `ListSessionsResponse`
- `SessionSummary`
- `GetSessionSummaryResponse`
- `GetSessionBlocksResponse`
- `GetSessionDetailResponse`
- `SessionBlock`
- `Turn`
- `ToolCall`
- `ToolCallInput`
- `ToolCallOutput`
- `BlockArtifacts`

### `annotations.proto`

Likely contents:

- `Annotation`
- `AnnotationScope`
- `AnnotationContent`
- `TaxonomyMappings`
- `GetSessionAnnotationsResponse`
- `ListAnnotationsResponse`
- `AnnotationListRow`
- `CreateAnnotationRequest`
- `UpdateAnnotationRequest`
- `UpdateAnnotationResponse`
- `DeleteAnnotationResponse` (optional; `NoContent` can stay HTTP-only if preferred)
- `SyncAnnotationsRequest`
- `SyncAnnotationsResponse`
- `SyncError`

### `queries.proto`

Likely contents for phase 1:

- `SavedQuery`
- `ListQueriesResponse`
- `ListPresetsResponse`
- `SaveQueryRequest`
- `UpdateQueryRequest`

The dynamic query execution result should be excluded from the first protobuf phase.

## Message design guidance

### Presence / optional fields

The protobuf skill notes that presence matters. This repo currently uses Go pointers for many optional fields, including:

- strings
- booleans
- integers
- floats

In proto3, use `optional` on fields where semantic absence matters.

Examples:

- `optional string summary`
- `optional string endedAt`
- `optional uint32 totalInputTokens`
- `optional bool sandbox`

### Numeric field choices

The skill warns that `int64` becomes JSON strings and decodes to `bigint` in TS. That is correct but may be ergonomically undesirable for UI-facing types.

Recommendations for this repo:

- durations already represented as seconds → `double`
- counts that are realistically bounded → `uint32`
- use `int64` only where values may truly exceed 32-bit limits

Token counts deserve explicit thought. They may fit in `uint32` for most realistic sessions, but this should be reviewed before locking the schema.

### Dynamic JSON fields

Some existing data is open-ended, especially:

- `ToolCallInput.arguments`

This should likely use `google.protobuf.Struct`.

That is acceptable because the field is genuinely dynamic. It does not need to become strongly typed if the source data is not strongly typed.

### Enums vs strings

The current API uses strings for several stable categorical fields.

Good enum candidates:

- tool call badge type (`commit`, `ticket-create`, `doc-add`, `diary-write`, `error`)
- annotation category
- annotation scope type (`session`, `turn`, `tool_call`)

Potentially still keep as strings for compatibility reasons if rollout friction becomes too high, but enums are preferable in a schema-first design.

## Surfaces to include in phase 1

### Sessions

Strong fit for protobuf because the shape is stable and already normalized.

### Annotations

Strong fit because:

- nested message shape is structured,
- categories and scope types are enum-like,
- update request currently needs better typing,
- flattened list row should stop leaking Go exported names.

### Saved-query metadata

Strong enough fit because it is structured file metadata rather than arbitrary query result rows.

## Surfaces to defer in phase 1

### Ad hoc SQL query execution

Current shape:

- `columns: string[]`
- `rows: Record<string, unknown>[]`
- execution metadata

This is intentionally dynamic. It can be modeled later with `google.protobuf.Struct`, but doing so in the first phase would add complexity while giving limited practical type safety.

The recommended design decision is therefore:

> Keep the query execution endpoint JSON-native in phase 1 and document that this is an intentional exception.

## Proposed implementation phases

## Phase 1 — Ticket documentation and scope confirmation

Goal:

- create the ticket
- write the analysis/design/implementation guide
- define the phased task list
- record the current architecture and rollout strategy

Deliverables:

- ticket workspace
- design doc
- diary
- tasks
- changelog

## Phase 2 — Add protobuf/Buf scaffolding

Goal:

Create the repository scaffolding required for schema-first development.

Expected files:

- `buf.yaml`
- `buf.gen.yaml`
- `proto/go_go_golems/minitrace/api/v1/*.proto`
- generated Go output directory (for example `gen/proto/...`)
- generated TS output directory (for example `web/src/gen/proto/...`)

Expected dependency changes:

- Go runtime support for protobuf / protojson if not already present
- `@bufbuild/protobuf` in `web/package.json`

Validation:

- `buf generate` runs successfully
- repo builds without yet wiring handlers to the new schema

## Phase 3 — Define sessions API protobuf schema

Goal:

Define the session list, summary, blocks, and detail messages.

Key design decisions:

- use explicit response envelopes
- use camelCase JSON via `protojson`
- represent transcript blocks as the public API view model, not raw archive turns/tool calls only

Validation:

- generated Go and TS code compiles
- message shapes cover current session endpoints adequately

## Phase 4 — Implement Go `/api/v2/sessions...` endpoints

Goal:

Add parallel protobuf-backed JSON routes for session list/summary/blocks/detail.

Approach:

- keep current `/api/...` endpoints intact initially
- add `/api/v2/sessions`
- add `/api/v2/sessions/{id}/summary`
- add `/api/v2/sessions/{id}/blocks`
- optionally add `/api/v2/sessions/{id}` if still useful

Implementation pattern:

```text
load internal session or list row
  -> normalize to generated protobuf message
  -> protojson marshal
  -> write JSON response
```

Validation:

- Go tests for new handlers
- endpoint JSON matches generated TS decoder expectations

## Phase 5 — Switch frontend sessions API to generated decoders

Goal:

Move session routes in `web/src/api/minitrace.ts` away from handwritten interfaces and onto generated protobuf decode.

Approach:

- add generated imports
- decode RTK Query responses with `fromJson(...)`
- optionally add thin adapter helpers if React components should keep existing local prop names temporarily

Validation:

- `npm run build`
- transcript/session routes still render correctly
- no field-casing mismatch remains in the session flow

## Phase 6 — Define and implement annotations API protobuf schema + v2 handlers

Goal:

Migrate the annotations API to explicit generated contracts.

High-value targets:

- `GetSessionAnnotationsResponse`
- `CreateAnnotationRequest`
- `UpdateAnnotationRequest` with optional fields
- `ListAnnotationsResponse` with a consciously designed row shape
- `SyncAnnotationsRequest` / `SyncAnnotationsResponse`

Validation:

- current annotation UI still works after frontend migration
- flattened list row no longer leaks capitalized Go storage field names

## Phase 7 — Migrate frontend annotations API to generated decoders

Goal:

Use generated schemas for annotation responses and requests.

Validation:

- annotation panel works
- session browser annotation badges still work
- update/delete/sync flows remain correct

## Phase 8 — Saved-query metadata protobuf alignment

Goal:

Migrate `/api/presets` and `/api/queries` metadata surfaces.

This is lower risk than transcript or annotation behavior because it does not involve dynamic nested transcript data.

## Phase 9 — Explicitly document the query-execution exception

Goal:

Record, in code and docs, that dynamic query execution results remain JSON-native in phase 1.

Possible future direction:

- later evaluate `google.protobuf.Struct` if a schema-first wrapper becomes valuable

## Phase 10 — Validation, docs, and operator guidance

Goal:

- tests
- generation instructions
- API versioning docs
- reMarkable upload of the final guide or bundle

## File-by-file implementation guide

### `go.mod`

Likely needs protobuf runtime dependencies if not already present.

Expected additions:

- `google.golang.org/protobuf`

### `web/package.json`

Expected additions:

- `@bufbuild/protobuf`

Optional tooling could be added later, but the minimal runtime decoder dependency is enough for phase 1.

### `buf.yaml`

Add Buf module configuration.

The skill recommends Buf v2 config with the Google APIs dependency.

### `buf.gen.yaml`

Use remote plugins for:

- Go generation
- TypeScript generation (`bufbuild/es`)

### `proto/go_go_golems/minitrace/api/v1/common.proto`

First place to centralize reusable message definitions.

### `proto/go_go_golems/minitrace/api/v1/sessions.proto`

Primary initial schema file.

### `proto/go_go_golems/minitrace/api/v1/annotations.proto`

Second major schema file.

### `proto/go_go_golems/minitrace/api/v1/queries.proto`

Saved-query metadata only in phase 1.

### `cmd/go-minitrace/cmds/serve/server.go`

Add `/api/v2/...` route registrations.

### `cmd/go-minitrace/cmds/serve/handlers_sessions.go`

Either:

- add new protobuf-specific handler functions here, or
- extract them into a new file if the file becomes too crowded.

### `cmd/go-minitrace/cmds/serve/handlers_annotations.go`

Add protobuf-backed v2 handlers and replace weakly typed patch decoding for the new surface.

### `cmd/go-minitrace/cmds/serve/handlers_queries.go`

Add v2 preset/query metadata handlers only.

### `web/src/api/minitrace.ts`

This is the main frontend integration point.

It will likely need:

- generated schema imports
- `transformResponse` calls using `fromJson(...)`
- temporary adapters if component prop names are migrated gradually

### `web/src/types/session.ts` and `web/src/types/query.ts`

These should shrink over time as generated schemas take over the API contract role.

Not every type here must disappear immediately. Some component-local view models may still be worth keeping. But the API-facing transport types should stop being handwritten duplicates.

## Pseudocode sketches

### Backend emission

```go
msg := &apiv1.ListSessionsResponse{
    SchemaVersion: 1,
    Sessions: sessions,
}

marshaler := protojson.MarshalOptions{
    UseProtoNames: false, // camelCase JSON
}

bytes, err := marshaler.Marshal(msg)
if err != nil {
    // handle
}
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
_, _ = w.Write(bytes)
```

### Frontend decode

```ts
import { fromJson } from "@bufbuild/protobuf";
import { ListSessionsResponseSchema } from "../gen/proto/.../sessions_pb";

transformResponse: (response: unknown) => {
  const decoded = fromJson(ListSessionsResponseSchema, response as any);
  return decoded.sessions;
}
```

### Annotation patch message

```proto
message UpdateAnnotationRequest {
  optional string title = 1;
  optional string detail = 2;
  optional AnnotationCategory category = 3;
  repeated string tags = 4;
  repeated string taxonomyMinitrace = 5;
  repeated string taxonomyMast = 6;
  repeated string taxonomyToolemu = 7;
  optional string classification = 8;
}
```

This is cleaner than decoding into `map[string]any` and reconstructing a patch manually.

## Testing and validation strategy

### Code generation

- `buf generate`
- ensure generated Go and TS outputs land in expected directories

### Go validation

- `go test ./cmd/go-minitrace/cmds/serve/...`
- add focused tests for new v2 handlers
- verify `protojson` output shape is stable and decodable

### Frontend validation

- `cd web && npm run build`
- route smoke tests for:
  - `/sessions`
  - `/sessions/:id`
  - annotations workflow
  - saved queries/presets loading

### Contract validation

Following the protobuf skill guidance:

- generated JSON uses expected casing
- optional fields round-trip correctly
- `Struct` fields decode to JS objects where used
- any 64-bit numeric fields are intentionally chosen and handled

## Risks

### Risk 1 — Over-scoping into the entire archive model

If the ticket tries to replace `pkg/minitrace/schema.go` with protobuf immediately, the change will grow too large and couple storage/import logic to the web contract unnecessarily.

Mitigation:

- keep the protobuf boundary at the public API DTO layer first.

### Risk 2 — Frontend churn from casing changes

Moving from snake_case handwritten JSON to camelCase protobuf JSON can create widespread UI churn if done in-place.

Mitigation:

- version the API under `/api/v2`
- migrate endpoint by endpoint
- use adapters temporarily where helpful

### Risk 3 — Dynamic query result overreach

Trying to protobuf-type arbitrary SQL result rows early will create extra complexity with limited benefit.

Mitigation:

- explicitly defer that surface.

### Risk 4 — Generated code layout friction

Poor choices for output paths can create noisy imports or difficult module boundaries.

Mitigation:

- keep output locations simple and source-relative
- align `go_package` and Buf config carefully

### Risk 5 — BigInt ergonomics in TS

Using `int64` too casually can make frontend handling more awkward.

Mitigation:

- prefer `uint32`/`int32`/`double` where semantically appropriate.

## Alternatives considered

### Alternative 1 — Keep handwritten DTOs and add more tests

Pros:

- no new generation tooling
- minimal infra change

Cons:

- does not remove duplication
- does not create a shared schema artifact
- still relies on discipline instead of generated alignment

Verdict:

- acceptable short-term, but does not solve the core alignment problem.

### Alternative 2 — Reuse internal archive structs as the API contract

Pros:

- fewer transport-layer types

Cons:

- exposes more internal shape than the UI needs
- weakens the distinction between archive schema and public API projection
- makes API evolution harder

Verdict:

- not recommended.

### Alternative 3 — Convert everything, including dynamic query results, to protobuf immediately

Pros:

- one transport technology for all routes

Cons:

- query execution results are inherently dynamic
- would likely force `Struct` into places where it adds little real type safety
- increases migration scope significantly

Verdict:

- not recommended for phase 1.

## Open questions

1. Should the generated TS types be consumed directly by React components, or should the frontend keep a thin view-model adapter layer?
2. Do we want to retain `GET /api/sessions/{id}` in v2, or encourage the summary + blocks split only?
3. Which token/count fields, if any, truly need 64-bit representation?
4. Should annotation categories/scope types become enums immediately, or remain strings for the first migration wave?
5. Should saved queries and presets share one common list response, or remain separate envelope messages?

## Recommended first implementation slice

If the goal is to de-risk the migration while making real progress, the best first code slice is:

1. add Buf/proto scaffolding,
2. define the sessions v1 schema,
3. generate Go + TS,
4. add `/api/v2/sessions`, `/api/v2/sessions/{id}/summary`, and `/api/v2/sessions/{id}/blocks`,
5. switch the frontend session routes to decode generated responses,
6. leave annotations and saved queries for the next slices.

That slice gives the team an end-to-end proof that the schema-first pattern works without immediately touching every API in the system.

## References

- `cmd/go-minitrace/cmds/serve/server.go`
- `cmd/go-minitrace/cmds/serve/handlers_sessions.go`
- `cmd/go-minitrace/cmds/serve/handlers_annotations.go`
- `cmd/go-minitrace/cmds/serve/handlers_queries.go`
- `cmd/go-minitrace/cmds/serve/blocks.go`
- `pkg/annotate/store.go`
- `pkg/minitrace/schema.go`
- `web/src/api/minitrace.ts`
- `web/src/types/session.ts`
- `web/src/types/query.ts`
- protobuf skill: `/home/manuel/.pi/agent/skills/protobuf-go-ts-schema-exchange/SKILL.md`
- protobuf templates: `/home/manuel/.pi/agent/skills/protobuf-go-ts-schema-exchange/references/templates.md`
- protobuf validation checklist: `/home/manuel/.pi/agent/skills/protobuf-go-ts-schema-exchange/references/validation.md`
