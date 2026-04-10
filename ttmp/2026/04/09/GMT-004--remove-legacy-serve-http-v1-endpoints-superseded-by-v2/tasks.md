# Tasks

## TODO

- [x] Inventory the currently registered legacy `/api/...` serve routes and confirm which ones are still used by the first-party web app
- [x] Write the cleanup implementation guide and relate the key server/frontend files that justify the scope
- [x] Update the ticket index/changelog/diary with the cleanup plan and explicit non-goal that `POST /api/query` remains
- [x] Remove the legacy session route registrations from `cmd/go-minitrace/cmds/serve/server.go`
- [x] Remove the unused v1 session HTTP handlers while keeping the shared normalization/helper code needed by v2
- [x] Replace direct v1 session handler tests with route-level coverage that proves those legacy endpoints now return 404 while v2 handlers remain covered
- [x] Remove the legacy preset/query route registrations from `cmd/go-minitrace/cmds/serve/server.go`
- [x] Remove the unused v1 preset/query handlers while keeping the shared SQL file helper/types used by v2
- [x] Replace direct v1 preset/query tests with route-level coverage for removed endpoints and keep v2 CRUD coverage intact
- [x] Remove the legacy annotation route registrations from `cmd/go-minitrace/cmds/serve/server.go`
- [x] Remove the unused v1 annotation handlers and dead path-parsing helpers while keeping the protobuf-backed v2 annotation flow intact
- [x] Add route-level regression coverage that removed legacy annotation endpoints return 404
- [x] Update frontend mocks and API-facing comments/docs to reflect the v2-only route surface, while preserving `/api/query`
- [x] Update the top-level README serve API section so it documents the surviving route families accurately
- [x] Run focused and full validation (`go test`, lint if needed, `docmgr doctor`) and upload the updated ticket bundle to reMarkable
