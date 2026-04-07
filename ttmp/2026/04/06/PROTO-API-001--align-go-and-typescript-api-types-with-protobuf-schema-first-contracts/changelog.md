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
