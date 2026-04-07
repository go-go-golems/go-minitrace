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

- [ ] Add `/api/v2/sessions`
- [ ] Add `/api/v2/sessions/{id}/summary`
- [ ] Add `/api/v2/sessions/{id}/blocks`
- [ ] Optionally add `/api/v2/sessions/{id}` if the full-detail route remains useful
- [ ] Normalize internal session values into generated protobuf messages
- [ ] Emit JSON with `protojson`
- [ ] Add/adjust serve tests for the new routes
- [ ] Commit the backend session API changes
- [ ] Update diary and changelog for the step

### Step 5 — Switch frontend session APIs to generated protobuf decoders

- [ ] Add generated TS imports and decode helpers using `fromJson(...)`
- [ ] Migrate session RTK Query endpoints to `/api/v2/...`
- [ ] Keep React component expectations stable with thin adapters if needed
- [ ] Run `cd web && npm run build`
- [ ] Smoke-test Session Browser and Transcript Viewer
- [ ] Commit the frontend session API changes
- [ ] Update diary and changelog for the step

### Step 6 — Define and implement protobuf schema for annotations

- [ ] Define `annotations.proto`
- [ ] Replace the flattened annotation list row with a consciously designed schema
- [ ] Define typed create/update/sync request and response messages
- [ ] Regenerate Go and TypeScript outputs
- [ ] Commit the annotation schema changes
- [ ] Update diary and changelog for the step

### Step 7 — Implement Go `/api/v2/annotations...` protobuf-backed JSON endpoints

- [ ] Add `/api/v2/sessions/{id}/annotations`
- [ ] Add `/api/v2/annotations`
- [ ] Add `/api/v2/annotations/{annId}` update/delete support as appropriate
- [ ] Add `/api/v2/annotations/sync`
- [ ] Remove weakly typed `map[string]any` patch decoding from the new v2 surface
- [ ] Add/adjust serve tests for the new routes
- [ ] Commit the backend annotation API changes
- [ ] Update diary and changelog for the step

### Step 8 — Switch frontend annotation APIs to generated protobuf decoders

- [ ] Migrate annotation RTK Query endpoints to `/api/v2/...`
- [ ] Replace handwritten transport typings with generated decoders
- [ ] Verify annotation panel, session badges, and sync flows
- [ ] Run `cd web && npm run build`
- [ ] Commit the frontend annotation API changes
- [ ] Update diary and changelog for the step

### Step 9 — Align saved-query metadata with protobuf

- [ ] Define `queries.proto` for presets and saved-query metadata
- [ ] Keep ad hoc query execution results out of the first protobuf phase
- [ ] Add `/api/v2/presets` and `/api/v2/queries`
- [ ] Switch frontend preset/query metadata consumers to generated decoders
- [ ] Commit the saved-query metadata changes
- [ ] Update diary and changelog for the step

### Step 10 — Document and validate the dynamic query-result exception

- [ ] Write down why `POST /api/query` remains JSON-native for now
- [ ] Optionally sketch a future `google.protobuf.Struct` path, but do not block the ticket on it
- [ ] Add validation notes for protojson casing, optional fields, and any 64-bit numeric choices
- [ ] Commit the documentation updates
- [ ] Update diary and changelog for the step

### Step 11 — Final validation, ticket cleanup, and reMarkable delivery

- [ ] Run `buf generate`
- [ ] Run `go test ./...`
- [ ] Run `cd web && npm run build`
- [ ] Run `docmgr doctor --ticket PROTO-API-001 --stale-after 30`
- [ ] Upload the final ticket bundle to reMarkable
- [ ] Verify the remote listing
- [ ] Commit the final ticket/docs updates
