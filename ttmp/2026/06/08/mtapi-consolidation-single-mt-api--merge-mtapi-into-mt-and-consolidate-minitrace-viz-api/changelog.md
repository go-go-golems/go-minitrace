# Changelog

## 2026-06-08

- Initial workspace created


## 2026-06-08

Created intern-facing single mt API consolidation design, API assessment, clean-cutover plan, diary, and phased tasks.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/08/mtapi-consolidation-single-mt-api--merge-mtapi-into-mt-and-consolidate-minitrace-viz-api/design-doc/01-single-mt-api-consolidation-design-and-implementation-guide.md — Primary implementation guide
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/08/mtapi-consolidation-single-mt-api--merge-mtapi-into-mt-and-consolidate-minitrace-viz-api/reference/01-investigation-diary.md — Chronological investigation diary
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/08/mtapi-consolidation-single-mt-api--merge-mtapi-into-mt-and-consolidate-minitrace-viz-api/tasks.md — Phased task list


## 2026-06-08

Uploaded design bundle to reMarkable at /ai/2026/06/08/mtapi-consolidation-single-mt-api and opened tasks with md-view.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/08/mtapi-consolidation-single-mt-api--merge-mtapi-into-mt-and-consolidate-minitrace-viz-api/design-doc/01-single-mt-api-consolidation-design-and-implementation-guide.md — Uploaded in reMarkable bundle
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/08/mtapi-consolidation-single-mt-api--merge-mtapi-into-mt-and-consolidate-minitrace-viz-api/tasks.md — Opened with md-view


## 2026-06-08

Updated diary with validation, reMarkable upload, and md-view task-view handoff details.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/08/mtapi-consolidation-single-mt-api--merge-mtapi-into-mt-and-consolidate-minitrace-viz-api/reference/01-investigation-diary.md — Recorded delivery step


## 2026-06-08

Revised API design from options-map-first to staged fluent Go-owned builders and updated phased tasks accordingly.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/08/mtapi-consolidation-single-mt-api--merge-mtapi-into-mt-and-consolidate-minitrace-viz-api/design-doc/01-single-mt-api-consolidation-design-and-implementation-guide.md — Builder-first design revision
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/08/mtapi-consolidation-single-mt-api--merge-mtapi-into-mt-and-consolidate-minitrace-viz-api/reference/01-investigation-diary.md — Diary Step 4 records the correction
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/08/mtapi-consolidation-single-mt-api--merge-mtapi-into-mt-and-consolidate-minitrace-viz-api/tasks.md — Builder-composed implementation phases


## 2026-06-08

Re-uploaded revised builder-first design bundle to reMarkable at /ai/2026/06/08/mtapi-consolidation-single-mt-api.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/08/mtapi-consolidation-single-mt-api--merge-mtapi-into-mt-and-consolidate-minitrace-viz-api/design-doc/01-single-mt-api-consolidation-design-and-implementation-guide.md — Re-uploaded revised design
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/08/mtapi-consolidation-single-mt-api--merge-mtapi-into-mt-and-consolidate-minitrace-viz-api/tasks.md — Re-uploaded revised tasks


## 2026-06-08

Step 5: Added SourceSet/import/cache/limits builders and DB composition methods, with provider integration tests.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/builders.go — New builder value types and JS wrappers
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/db_builder.go — DB builder composition wiring
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/provider/provider_test.go — Integration tests


## 2026-06-08

Step 6: Added mt.importer fluent upload/import builder with save-to-session-directory integration test.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/import_builder.go — New import builder
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/provider/provider_test.go — Importer integration test


## 2026-06-08

Step 7: Added mt.query, mt.view, and mt.session builders with provider integration tests.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/provider/provider_test.go — Query/view/session integration tests
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/query_view_session.go — Query/view/session builders


## 2026-06-08

Recorded commit 5f5303d for Step 7 session/query/view builder implementation.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/08/mtapi-consolidation-single-mt-api--merge-mtapi-into-mt-and-consolidate-minitrace-viz-api/reference/01-investigation-diary.md — Updated Step 7 commit hash


## 2026-06-08

Step 8: Rewrote JS API reference and updated query-command examples/docs to builder-composed mt API.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/examples/xgoja/minitrace-command-provider/queries/reports/markdown-summary.js — Example updated for QueryCommandDefaults
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/doc/js-api-reference.md — Primary builder-composed JS API docs
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/testdata/query-repositories/js-showcase/README.md — Showcase docs updated for QueryCommandDefaults


## 2026-06-08

Step 9: Cut minitrace-viz over to go-minitrace mt builders, deleted local mtapi, and fixed role-only Pi JSONL detection.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/ClubMedMeetup/minitrace-viz/lib/session-service.js — Upload path now uses mt.importer
- /home/manuel/workspaces/2026-06-07/club-meetup-site/ClubMedMeetup/minitrace-viz/lib/timeline-data.js — Timeline data now uses mt.session and normalized SQL
- /home/manuel/workspaces/2026-06-07/club-meetup-site/ClubMedMeetup/minitrace-viz/server.js — Legacy report routes removed from mtapi path
- /home/manuel/workspaces/2026-06-07/club-meetup-site/ClubMedMeetup/minitrace-viz/xgoja.yaml — go-minitrace provider aliased as mt
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracedb/convert.go — Role-only Pi JSONL detection parity


## 2026-06-08

Recorded Step 9 commits d5e418d (go-minitrace detector parity) and 2196537 (minitrace-viz cutover).

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/08/mtapi-consolidation-single-mt-api--merge-mtapi-into-mt-and-consolidate-minitrace-viz-api/reference/01-investigation-diary.md — Step 9 commit hashes


## 2026-06-08

Marked Phase 7 validation tasks complete for go-minitrace tests and minitrace-viz smoke coverage; noted remaining broader fixture follow-up.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/08/mtapi-consolidation-single-mt-api--merge-mtapi-into-mt-and-consolidate-minitrace-viz-api/tasks.md — Validation task status


## 2026-06-08

Re-uploaded final implementation bundle to reMarkable at /ai/2026/06/08/mtapi-consolidation-single-mt-api.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/08/mtapi-consolidation-single-mt-api--merge-mtapi-into-mt-and-consolidate-minitrace-viz-api/changelog.md — Final upload entry

