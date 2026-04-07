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
