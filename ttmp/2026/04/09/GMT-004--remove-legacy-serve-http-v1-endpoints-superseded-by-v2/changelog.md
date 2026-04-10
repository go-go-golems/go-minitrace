# Changelog

## 2026-04-09

- Initial workspace created
- Added the cleanup scope, task breakdown, and implementation-plan deliverables for removing duplicate serve HTTP route families while preserving `POST /api/query`
- Step 2: removed the legacy session and saved-query/preset route registrations, deleted the dead v1 handler entrypoints, replaced direct legacy tests with mux-level 404 checks, and kept the shared v2 helper code plus `POST /api/query` intact (commit `e44d8cf`)
