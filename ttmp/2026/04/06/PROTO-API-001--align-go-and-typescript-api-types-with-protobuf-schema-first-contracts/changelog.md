# Changelog

## 2026-04-06

- Initial workspace created

## 2026-04-06

Created the `PROTO-API-001` ticket, inspected the current handwritten Go/TypeScript API DTO duplication, and wrote a detailed design and implementation guide for a protobuf-backed schema-first contract layer.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/PROTO-API-001--align-go-and-typescript-api-types-with-protobuf-schema-first-contracts/index.md — Ticket overview and key links
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/PROTO-API-001--align-go-and-typescript-api-types-with-protobuf-schema-first-contracts/tasks.md — Phased execution plan for the protobuf alignment work
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/PROTO-API-001--align-go-and-typescript-api-types-with-protobuf-schema-first-contracts/design-doc/01-protobuf-schema-first-api-alignment-analysis-and-implementation-guide.md — Primary analysis and implementation guide
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/PROTO-API-001--align-go-and-typescript-api-types-with-protobuf-schema-first-contracts/reference/01-investigation-diary.md — Diary that records ticket setup and future implementation steps

## 2026-04-06

Step 2: added protobuf/Buf scaffolding, runtime dependencies, a minimal shared `common.proto`, and generated the first Go/TypeScript protobuf outputs (commit 924ff74).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/buf.yaml — Buf module configuration with the Google APIs dependency
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/buf.gen.yaml — Go and TypeScript code generation configuration
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/proto/go_go_golems/minitrace/api/v1/common.proto — Initial shared API proto package and `ApiMeta` message
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/gen/proto/go_go_golems/minitrace/api/v1/common.pb.go — Generated Go protobuf output for the shared proto package
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/gen/proto/go_go_golems/minitrace/api/v1/common_pb.ts — Generated TypeScript protobuf output for the shared proto package
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/go.mod — Added protobuf runtime dependency for Go
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/package.json — Added `@bufbuild/protobuf` runtime dependency for the frontend

## 2026-04-06

Step 3: defined the shared session-related protobuf messages and the first full `sessions.proto` API contract, then regenerated Go and TypeScript bindings (commit ebcee29).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/proto/go_go_golems/minitrace/api/v1/common.proto — Expanded shared messages and added the `ToolCallBadge` enum plus session-related common types
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/proto/go_go_golems/minitrace/api/v1/sessions.proto — Defined session summaries, detail, blocks, turns, tool calls, and explicit response envelopes
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/gen/proto/go_go_golems/minitrace/api/v1/common.pb.go — Regenerated Go bindings for shared proto definitions
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/gen/proto/go_go_golems/minitrace/api/v1/sessions.pb.go — Generated Go bindings for the sessions API schema
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/gen/proto/go_go_golems/minitrace/api/v1/common_pb.ts — Regenerated frontend bindings for shared proto definitions
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/gen/proto/go_go_golems/minitrace/api/v1/sessions_pb.ts — Generated frontend bindings for the sessions API schema

## 2026-04-06

Step 4: added protobuf-backed `/api/v2/sessions...` endpoints, a shared `protojson` response helper, focused protojson-based handler tests, and adjusted generated frontend protobuf output to `js+dts` so it remains compatible with the repo’s TypeScript compiler settings (commit 00670d2).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/server.go — Registered the new `/api/v2/sessions...` routes
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/protojson.go — Added a shared protobuf JSON response writer
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/handlers_sessions_v2.go — Added the normalization layer from current session DTOs to generated protobuf messages
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/server_test.go — Added protojson-based tests for the new v2 session routes
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/buf.gen.yaml — Switched frontend protobuf generation from `target=ts` to `target=js+dts` to avoid `erasableSyntaxOnly` build failures
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/gen/proto/go_go_golems/minitrace/api/v1/common_pb.d.ts — New declaration output for shared protobuf bindings
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/gen/proto/go_go_golems/minitrace/api/v1/common_pb.js — New runtime JS output for shared protobuf bindings
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/gen/proto/go_go_golems/minitrace/api/v1/sessions_pb.d.ts — New declaration output for session protobuf bindings
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/gen/proto/go_go_golems/minitrace/api/v1/sessions_pb.js — New runtime JS output for session protobuf bindings

## 2026-04-06

Step 5: switched the frontend session RTK Query endpoints to `/api/v2/...`, decoded protobuf-backed JSON envelopes with generated schemas, and adapted them back into the existing session view models so the React component tree could remain stable (commit 272b937).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/api/minitrace.ts — Session API endpoints now query `/api/v2/...` and decode protobuf responses
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/api/sessionProtoAdapters.ts — New frontend decode-and-adapt layer from generated protobuf messages to existing UI-facing session models

## 2026-04-06

Step 6: defined the annotations protobuf schema, redesigned the flattened annotation list row as an intentional contract, added presence-safe list wrappers for patch semantics, and regenerated Go/TypeScript bindings (commit d20d61c).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/proto/go_go_golems/minitrace/api/v1/common.proto — Added the shared `StringList` message for presence-safe repeated-string patch fields
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/proto/go_go_golems/minitrace/api/v1/annotations.proto — Defined annotation enums, core annotation messages, list row, create/update/sync contracts, and response envelopes
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/gen/proto/go_go_golems/minitrace/api/v1/common.pb.go — Regenerated Go bindings for the shared common schema
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/gen/proto/go_go_golems/minitrace/api/v1/annotations.pb.go — Generated Go bindings for the annotations API schema
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/gen/proto/go_go_golems/minitrace/api/v1/common_pb.d.ts — Regenerated frontend declaration output for shared protobuf definitions
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/gen/proto/go_go_golems/minitrace/api/v1/common_pb.js — Regenerated frontend runtime output for shared protobuf definitions
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/gen/proto/go_go_golems/minitrace/api/v1/annotations_pb.d.ts — Generated frontend declaration output for the annotations protobuf schema
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/gen/proto/go_go_golems/minitrace/api/v1/annotations_pb.js — Generated frontend runtime output for the annotations protobuf schema

## 2026-04-06

Step 7: added protobuf-backed `/api/v2/annotations...` handlers, replaced weakly typed annotation patch decoding on the new v2 surface with generated request messages, and added focused annotation v2 handler tests (commit f314896).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/server.go — Registered the new `/api/v2/annotations...` routes
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/protojson.go — Reused shared protobuf JSON request/response helpers for the annotation v2 routes
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/handlers_annotations_v2.go — Added the new protobuf-backed annotation handlers and enum/string normalization helpers
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/server_test.go — Added create/get/list/update/delete/sync tests for the v2 annotation routes

## 2026-04-06

Step 8: switched the frontend annotation RTK Query layer to the protobuf-backed `/api/v2/...` routes, decoded generated annotation envelopes, and adopted the new intentional lower-camel annotation list-row contract in the Session Browser path (commit e21331f).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/api/minitrace.ts — Annotation endpoints now call `/api/v2/...` and use protobuf-backed decode/encode helpers
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/api/annotationProtoAdapters.ts — New annotation decode-and-encode adapter layer for protobuf-backed annotation transport
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/types/session.ts — `AnnotationListRow` now reflects the intentional lower-camel API contract instead of Go-exported field casing
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/pages/SessionBrowserPage.tsx — Session-browser annotation aggregation updated to the new list-row contract

## 2026-04-06

Step 9: added protobuf-backed saved-query metadata contracts and `/api/v2/presets`/`/api/v2/queries` routes, migrated the frontend preset and saved-query metadata consumers to generated decoders, and kept `POST /api/query` deliberately out of the protobuf transport in this implementation slice (commit adce7eb).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/proto/go_go_golems/minitrace/api/v1/queries.proto — Defined the saved-query metadata schema and request/response envelopes for phase 1 query metadata
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/gen/proto/go_go_golems/minitrace/api/v1/queries.pb.go — Generated Go bindings for the query-metadata protobuf schema
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/handlers_queries_v2.go — Added protobuf-backed presets and saved-query metadata handlers
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/server.go — Registered the new `/api/v2/presets` and `/api/v2/queries` routes
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/server_test.go — Added protojson-based v2 query metadata handler tests
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/api/queryProtoAdapters.ts — Added frontend decode helpers for protobuf-backed preset and saved-query metadata responses
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/api/minitrace.ts — Query metadata endpoints now call the new `/api/v2/...` routes and decode generated responses
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/gen/proto/go_go_golems/minitrace/api/v1/queries_pb.d.ts — Generated frontend declaration output for the query-metadata protobuf schema
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/gen/proto/go_go_golems/minitrace/api/v1/queries_pb.js — Generated frontend runtime output for the query-metadata protobuf schema
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/mocks/handlers.ts — Updated mock handlers so the query editor’s metadata endpoints work against the new v2 envelope shapes

## 2026-04-06

Step 10: documented the deliberate phase-1 exception for `POST /api/query`, added a code comment on the JSON-native dynamic query transport, and recorded validation notes for protojson casing, optional fields, 64-bit numeric choices, and codegen sequencing (pending docs commit).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/server.go — Added an explicit code comment documenting why `QueryResponse` remains JSON-native in phase 1
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/PROTO-API-001--align-go-and-typescript-api-types-with-protobuf-schema-first-contracts/reference/02-query-execution-json-native-exception-and-validation-notes.md — New durable note explaining the dynamic query-result exception and future `google.protobuf.Struct` option
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/PROTO-API-001--align-go-and-typescript-api-types-with-protobuf-schema-first-contracts/design-doc/01-protobuf-schema-first-api-alignment-analysis-and-implementation-guide.md — Added an implementation note linking the dynamic query exception record
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/PROTO-API-001--align-go-and-typescript-api-types-with-protobuf-schema-first-contracts/index.md — Linked the new exception note from the ticket index

## 2026-04-06

Step 11: completed final validation, ran `docmgr doctor`, uploaded the ticket bundle to reMarkable under `/ai/2026/04/06/PROTO-API-001/`, and verified the remote listing (pending final docs commit).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/PROTO-API-001--align-go-and-typescript-api-types-with-protobuf-schema-first-contracts/tasks.md — Final execution checklist is now fully checked off
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/PROTO-API-001--align-go-and-typescript-api-types-with-protobuf-schema-first-contracts/index.md — Ticket status updated to complete and final validation/upload status recorded
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/PROTO-API-001--align-go-and-typescript-api-types-with-protobuf-schema-first-contracts/reference/01-investigation-diary.md — Added the final validation and delivery diary entry
