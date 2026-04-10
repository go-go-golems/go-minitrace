# Changelog

## 2026-04-09

- Initial workspace created
- Added the cleanup scope, task breakdown, and implementation-plan deliverables for removing duplicate serve HTTP route families while preserving `POST /api/query`
- Step 2: removed the legacy session and saved-query/preset route registrations, deleted the dead v1 handler entrypoints, replaced direct legacy tests with mux-level 404 checks, and kept the shared v2 helper code plus `POST /api/query` intact (commit `e44d8cf`)
- Step 3: removed the legacy annotation route family, deleted the now-unused v1 annotation handler file entirely, updated frontend mocks and API-facing comments to the v2-only contract, and refreshed the README serve API section to document the hard cutover accurately (commit `c1ca601`)
- Step 4: ran full backend validation, confirmed the remaining serve route surface is v2-only except for `POST /api/query`, passed `docmgr doctor`, uploaded the updated ticket bundle to reMarkable, and verified the remote entry after correcting an initial cloud-list path mismatch

## 2026-04-09

Ticket closed

