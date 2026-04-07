# Tasks

## Complete

- [x] Create a new docmgr ticket for protobuf-backed API type alignment
- [x] Inspect the current backend/frontend API type duplication and identify the best protobuf boundary
- [x] Write a detailed analysis and implementation guide for the ticket

## Execution plan

### Step 2 — Add protobuf/Buf scaffolding

- [x] Add `buf.yaml` and `buf.gen.yaml`
- [x] Add a `proto/go_go_golems/minitrace/api/v1/` layout
- [x] Add protobuf runtime dependencies for Go and TypeScript
- [x] Choose generated output directories for Go and TypeScript
- [x] Run `buf generate`
- [x] Run `go test ./...` and `cd web && npm run build`
- [x] Commit the scaffolding changes
- [x] Update diary and changelog for the step

### Step 3 — Define protobuf schema for the typed sessions API

- [x] Define `common.proto` message types needed by sessions
- [x] Define `sessions.proto` with explicit response envelopes
- [x] Keep transcript blocks as the public API projection rather than raw archive-only structs
- [x] Regenerate Go and TypeScript outputs
- [x] Commit the schema changes
- [x] Update diary and changelog for the step

### Step 4 — Implement Go `/api/v2/sessions...` protobuf-backed JSON endpoints

- [x] Add `/api/v2/sessions`
- [x] Add `/api/v2/sessions/{id}/summary`
- [x] Add `/api/v2/sessions/{id}/blocks`
- [x] Optionally add `/api/v2/sessions/{id}` if the full-detail route remains useful
- [x] Normalize internal session values into generated protobuf messages
- [x] Emit JSON with `protojson`
- [x] Add/adjust serve tests for the new routes
- [x] Commit the backend session API changes
- [x] Update diary and changelog for the step

### Step 5 — Switch frontend session APIs to generated protobuf decoders

- [x] Add generated TS imports and decode helpers using `fromJson(...)`
- [x] Migrate session RTK Query endpoints to `/api/v2/...`
- [x] Keep React component expectations stable with thin adapters if needed
- [x] Run `cd web && npm run build`
- [x] Smoke-test Session Browser and Transcript Viewer
- [x] Commit the frontend session API changes
- [x] Update diary and changelog for the step

### Step 6 — Define and implement protobuf schema for annotations

- [x] Define `annotations.proto`
- [x] Replace the flattened annotation list row with a consciously designed schema
- [x] Define typed create/update/sync request and response messages
- [x] Regenerate Go and TypeScript outputs
- [x] Commit the annotation schema changes
- [x] Update diary and changelog for the step

### Step 7 — Implement Go `/api/v2/annotations...` protobuf-backed JSON endpoints

- [x] Add `/api/v2/sessions/{id}/annotations`
- [x] Add `/api/v2/annotations`
- [x] Add `/api/v2/annotations/{annId}` update/delete support as appropriate
- [x] Add `/api/v2/annotations/sync`
- [x] Remove weakly typed `map[string]any` patch decoding from the new v2 surface
- [x] Add/adjust serve tests for the new routes
- [x] Commit the backend annotation API changes
- [x] Update diary and changelog for the step

### Step 8 — Switch frontend annotation APIs to generated protobuf decoders

- [x] Migrate annotation RTK Query endpoints to `/api/v2/...`
- [x] Replace handwritten transport typings with generated decoders
- [x] Verify annotation panel, session badges, and sync flows
- [x] Run `cd web && npm run build`
- [x] Commit the frontend annotation API changes
- [x] Update diary and changelog for the step

### Step 9 — Align saved-query metadata with protobuf

- [x] Define `queries.proto` for presets and saved-query metadata
- [x] Keep ad hoc query execution results out of the first protobuf phase
- [x] Add `/api/v2/presets` and `/api/v2/queries`
- [x] Switch frontend preset/query metadata consumers to generated decoders
- [x] Commit the saved-query metadata changes
- [x] Update diary and changelog for the step

### Step 10 — Document and validate the dynamic query-result exception

- [x] Write down why `POST /api/query` remains JSON-native for now
- [x] Optionally sketch a future `google.protobuf.Struct` path, but do not block the ticket on it
- [x] Add validation notes for protojson casing, optional fields, and any 64-bit numeric choices
- [x] Commit the documentation updates
- [x] Update diary and changelog for the step

### Step 11 — Final validation, ticket cleanup, and reMarkable delivery

- [x] Run `buf generate`
- [x] Run `go test ./...`
- [x] Run `cd web && npm run build`
- [x] Run `docmgr doctor --ticket PROTO-API-001 --stale-after 30`
- [x] Upload the final ticket bundle to reMarkable
- [x] Verify the remote listing
- [x] Commit the final ticket/docs updates
