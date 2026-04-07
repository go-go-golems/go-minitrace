---
Title: Query execution JSON-native exception and validation notes
Ticket: PROTO-API-001
Status: active
Topics:
    - backend
    - frontend
    - minitrace
    - go-minitrace
    - protobuf
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/go-minitrace/cmds/serve/server.go
      Note: |-
        `QueryRequest`/`QueryResponse` and `POST /api/query` remain JSON-native in phase 1.
        Dynamic query execution transport boundary
    - Path: cmd/go-minitrace/cmds/serve/handlers_queries.go
      Note: |-
        Legacy saved-query metadata handlers that motivated the narrower protobuf metadata migration.
        Saved-query metadata versus dynamic query execution distinction
    - Path: proto/go_go_golems/minitrace/api/v1/queries.proto
      Note: |-
        New phase-1 protobuf contract for presets and saved-query metadata only.
        Structured query-metadata protobuf boundary
    - Path: web/src/types/query.ts
      Note: |-
        Frontend `QueryResult` remains dynamic and JSON-native.
        Frontend dynamic query result shape
ExternalSources: []
Summary: Explains why `POST /api/query` remains JSON-native in phase 1, what a future protobuf path could look like, and what validation lessons matter for protojson, optional fields, and numeric choices.
LastUpdated: 2026-04-06T22:05:00-04:00
WhatFor: Preserve the rationale for leaving the dynamic query execution route outside the first protobuf migration wave.
WhenToUse: Use when reviewing the protobuf rollout boundary or considering whether to protobuf-wrap ad hoc query execution later.
---

# Query execution JSON-native exception and validation notes

## Executive summary

`go-minitrace` now uses protobuf-backed `/api/v2/...` JSON contracts for sessions, annotations, and saved-query metadata. The one deliberate exception in phase 1 is `POST /api/query`.

That endpoint remains JSON-native because its response shape is intentionally dynamic:

- `columns: string[]`
- `rows: Record<string, unknown>[]`
- execution metadata

The row schema depends on arbitrary user SQL, so protobuf would add ceremony here without delivering the same type-safety value it delivered for the structured API surfaces.

## Why `POST /api/query` was excluded from phase 1

The first protobuf migration wave focused on surfaces with stable, intentional schemas:

- sessions
- annotations
- presets and saved-query metadata

Those surfaces benefit strongly from:

- a generated cross-language contract
- clear optional/presence semantics
- field-name normalization at the transport boundary
- enum-backed categorical fields

`POST /api/query` is different. Its `rows` payload is open-ended by design. Two different SQL statements can return completely different:

- column counts
- column names
- value types
- nullability patterns

In that context, protobuf does not create a meaningful compile-time contract for the result rows unless the system also constrains or categorizes the allowed queries much more tightly than it does today.

## What protobuf would and would not buy us here

### What it could buy later

A future protobuf wrapper could standardize the envelope around dynamic results, for example:

- request metadata
- duration
- row count
- error shape
- maybe row values encoded through `google.protobuf.Struct`

That could help if the project later needs:

- uniform transport metadata across all routes
- alternate protobuf-native clients
- stronger versioning around execution metadata

### What it would not buy immediately

It would not make arbitrary SQL result rows truly “strongly typed”. If the payload is still fundamentally dynamic, the client still needs generic record handling logic.

So in phase 1, protobuf would mostly wrap a dynamic payload rather than meaningfully type it.

## Possible future direction

If the project later decides the query execution route should also use a protobuf envelope, a plausible shape would be:

```proto
message QueryExecutionRequest {
  string sql = 1;
}

message QueryExecutionError {
  string message = 1;
  optional uint32 line = 2;
  optional uint32 column = 3;
}

message QueryExecutionRow {
  google.protobuf.Struct values = 1;
}

message QueryExecutionResponse {
  ApiMeta meta = 1;
  repeated string columns = 2;
  repeated QueryExecutionRow rows = 3;
  uint32 row_count = 4;
  uint64 duration_ms = 5;
  optional QueryExecutionError error = 6;
}
```

That would standardize the envelope, but the payload would still remain effectively dynamic at the row level.

## Validation notes from the current rollout

### 1. `protojson` casing must stay intentional

The repo uses protobuf JSON with lower-camel field names because `protojson.MarshalOptions{UseProtoNames: false}` is now the shared backend default for v2 routes.

That means:

- protobuf field `session_id` becomes JSON `sessionId`
- protobuf field `taxonomy_minitrace` becomes JSON `taxonomyMinitrace`
- protobuf field `schema_version` becomes JSON `schemaVersion`

This was one of the explicit benefits of the migration: transport JSON should reflect an intentional API contract, not leaked Go-exported field names.

### 2. Presence matters; use `optional` where semantic absence matters

The migration already hit this in sessions and annotations:

- optional summaries
- optional timestamps
- optional classification
- patch semantics for repeated string fields

For repeated patch fields, plain repeated fields were not enough, so the shared `StringList` wrapper was introduced to distinguish:

- field absent
- field present but empty

That same discipline should apply to any future protobuf envelope around dynamic query execution metadata if partial-update semantics ever enter that surface.

### 3. 64-bit numeric fields affect JSON and TypeScript ergonomics

One important reason to avoid over-designing the query execution protobuf wrapper right now is that numeric choices become part of the client ergonomics.

In protobuf JSON:

- `int64` / `uint64` values are represented as strings
- TypeScript consumers often need `bigint`-aware handling

That is acceptable when the domain truly needs 64-bit values, but it is friction the current JSON-native query route does not impose.

For the current structured v2 routes, the rollout intentionally preferred:

- `uint32` for realistically bounded counts
- `double` for durations already represented as seconds
- `optional` fields where presence matters more than numeric width

If `POST /api/query` ever gets a protobuf wrapper, `duration_ms` and `row_count` should be chosen deliberately rather than automatically widened.

### 4. Build/codegen sequencing matters

During Step 9, I briefly ran `buf generate` and the frontend build in parallel. That caused a transient frontend failure because the generated `queries_pb.js` file was not yet present when the TypeScript build started.

Operationally, validation should stay sequenced like this:

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && buf generate
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && go test ./...
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build
```

## Decision summary

Phase 1 protobuf coverage now includes:

- `/api/v2/sessions...`
- `/api/v2/annotations...`
- `/api/v2/presets`
- `/api/v2/queries`

Phase 1 explicitly excludes:

- `POST /api/query`

That exclusion is intentional, documented, and consistent with the original design goal: protobuf the structured DTO surfaces first, not the inherently dynamic one.
