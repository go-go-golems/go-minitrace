# Changelog

## 2026-04-04

- Initial workspace created


## 2026-04-04

Created ticket with schema research document and annotation storage backend design decision. Analyzed 4 options (direct JSON edit, DuckDB writes, parallel SQLite, sidecar files). Recommended parallel SQLite approach.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/04/ANNOTATE-CLI--go-minitrace-annotation-cli-and-storage-backend-design/design-doc/01-annotation-storage-backend-and-cli-design-decision.md — Design decision with 4 options analyzed
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/04/ANNOTATE-CLI--go-minitrace-annotation-cli-and-storage-backend-design/reference/01-minitrace-schema-full-affordances-research.md — Full minitrace schema affordances research


## 2026-04-04

Added comprehensive implementation guide covering all 12 parts: existing system anatomy (pkg structure, schema, builders, DuckDB engine, serve, command wiring), SQLite store (CRUD, sync state, atomic writes), DuckDB integration (annotations_flat temp table), CLI commands (add/list/edit/delete/sync), HTTP API endpoints, Web UI (React AnnotationPanel), testing strategy, directory structure, and 6-phase implementation plan with open questions.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/04/ANNOTATE-CLI--go-minitrace-annotation-cli-and-storage-backend-design/design-doc/02-annotation-cli-implementation-guide.md — 83KB detailed implementation guide with prose


## 2026-04-04

Investigated DuckDB sqlite_scanner extension. Key findings: (1) built-in, no external dependency, (2) uses NAMED parameters not positional, (3) attached tables land in main schema, (4) annotations are live — no refresh needed, (5) annotations_flat JSON approach is unnecessary. Updated Part 4 of implementation guide and Phase 3 tasks. Saved 7 investigation scripts to scripts/.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/04/ANNOTATE-CLI--go-minitrace-annotation-cli-and-storage-backend-design/scripts/00-README.md — Scripts README
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/04/ANNOTATE-CLI--go-minitrace-annotation-cli-and-storage-backend-design/scripts/07-sqlite-attach-working-join.sql — Working end-to-end test with join across SQLite and DuckDB sessions


## 2026-04-04

Replaced high-level tasks with detailed implementation tasks (18KB): 6 phases, 47 concrete tasks with checkboxes, dependency order diagram, open questions to resolve before starting, organized by Phase 1 (SQLite store), Phase 2 (CLI commands), Phase 3 (DuckDB integration), Phase 4 (HTTP API), Phase 5 (web UI), Phase 6 (polish).


## 2026-04-04

Committed Phase 1: pkg/annotate/store.go + store_test.go (14KB + 10KB, 11 tests, 0 lint issues).

Bug found and fixed: SQLite does not support reading a column in VALUES of an ON CONFLICT DO UPDATE clause — the original markUnsynced query tried to use `change_count + 1` in VALUES which caused 'no such column' at runtime. Fixed with separate UPDATE then INSERT.

TestNilTagsAndTaxonomy revealed a second bug: json.Unmarshal appends to a nil slice but nothing is appended for empty [], leaving the slice nil. Fixed parseJSONArray to return []string{} for '[]' string.

All 11 tests pass, golangci-lint reports 0 issues on ./pkg/annotate/...


## 2026-04-04

Step 1: pkg/annotate/store.go (commit 238aba7). SQLite CRUD: Open, AddAnnotation, GetAnnotationsForSession, List (dynamic WHERE, taxonomy LIKE), Update (buildPatchSET), Delete, GetUnsyncedSessions, markUnsynced/markSynced. Two bugs caught by tests: (1) SQLite VALUES column restriction in ON CONFLICT — fixed with separate UPDATE then INSERT; (2) json.Unmarshal leaves nil slice for empty JSON array — fixed parseJSONArray to return []string{} for "[]". 11 tests pass, 0 lint issues.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/annotate/store.go — Store struct
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/annotate/store_test.go — 11 tests (commit 238aba7)
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/04/ANNOTATE-CLI--go-minitrace-annotation-cli-and-storage-backend-design/reference/02-diary.md — Diary Step 1


## 2026-04-04

Step 2: pkg/annotate/sync.go (commit 6c71f31). Atomic JSON write-back: SyncSession reads .minitrace.json, patches annotations field using map[string]any to preserve all other fields, writes to .tmp then Rename (atomic on POSIX). SyncAll iterates unsynced sessions from store, looks up file paths in sessionIndex, calls SyncSession, marks synced. Handles nil annotations (produces [] not null). 5 tests pass, 0 lint issues.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/annotate/sync.go — SyncSession
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/annotate/sync_test.go — 5 tests (commit 6c71f31)


## 2026-04-04

Step 3: cmd/annotate CLI (commit eec4611). 6 subcommands: add, list, edit, delete, sync, import. Wired into main.go. annotate sync builds session index by expanding archive-glob, reads each file to extract session IDs, calls store.SyncAll. annotate list supports table/json output with all filter options. Smoke test passed: add + list + delete work end-to-end.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/annotate/add.go — add command (commit eec4611)
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/annotate/delete.go — delete command (commit eec4611)
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/annotate/edit.go — edit command (commit eec4611)
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/annotate/import.go — import command (commit eec4611)
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/annotate/list.go — list command (commit eec4611)
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/annotate/synccmd.go — sync command with session index builder (commit eec4611)


## 2026-04-04

Step 4: DuckDB sqlite_scanner integration (commit 4116a58). pkg/annotate/duckdb.go: AttachAnnotationsToDuckDB — INSTALL/LOAD sqlite_scanner + sqlite_attach(path, overwrite => true). Annotations are live — no refresh needed. serve.go wired after OpenConnection, before LoadArchive. outputDir inferred from first archive glob. queries/annotations.sql updated to join SQLite annotations table with DuckDB sessions_base.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/serve.go — wired DuckDB attachment after OpenConnection (commit 4116a58)
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/annotate/duckdb.go — AttachAnnotationsToDuckDB — sqlite_scanner install/load/attach (commit 4116a58)
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/queries/annotations.sql — updated to join SQLite annotations with sessions_base (commit 4116a58)


## 2026-04-04

Step 5: HTTP API handlers (commit f155b6e). handlers_annotations.go: 6 handlers (GET/POST/Put/Delete for sessions and annotations + sync). NewServer accepts annoStore + annoIndex. serve.go opens Store at startup. handlers handle nil store gracefully (503). All serve tests pass.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/handlers_annotations.go — 6 annotation handlers (commit f155b6e)
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/server.go — annoStore field


## 2026-04-04

Step 6: Web UI AnnotationPanel (commit 7421127). web/src/types/session.ts: Annotation types. web/src/api/minitrace.ts: 5 RTK Query annotation endpoints. AnnotationPanel.tsx: panel with list, add form (category select, title, detail, tags), delete, sync button. TranscriptViewer.tsx: added Transcript/Annotations tab toggle. npm build passes.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/api/minitrace.ts — 5 annotation RTK Query endpoints (commit 7421127)
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/AnnotationPanel.tsx — annotation panel component (commit 7421127)
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/TranscriptViewer.tsx — Transcript/Annotations tab toggle (commit 7421127)
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/types/session.ts — Annotation types (commit 7421127)


## 2026-04-04

**ANNOTATE-CLI feature complete.** All 10 feature commits merged. Full postmortem report written and uploaded to reMarkable.

### Summary

- 16 Go commits: 6 core packages + CLI + HTTP API + web UI + E2E tests + validation + README
- SQLite working store at `outputDir/annotations.db` (WAL mode, crash-safe)
- DuckDB live queries via `sqlite_scanner` (no refresh needed)
- Atomic JSON write-back via `go-minitrace annotate sync`
- 6 CLI subcommands, 6 HTTP API endpoints, web UI with RTK Query
- 12 validation tests, 3 E2E shell scripts

### Commit graph

`238aba7` pkg/annotate: SQLite-backed annotation store
`6c71f31` pkg/annotate: atomic JSON sync
`eec4611` cmd/annotate: 6 CLI commands
`4116a58` pkg/annotate: DuckDB sqlite_scanner
`f155b6e` serve: 6 annotation HTTP API handlers
`7421127` web: AnnotationPanel + RTK Query
`b663b03` fix: extractPathParam OOB + 3 E2E scripts
`5430a20` pkg/validate: annotation structure validation
`bb3fcc5` README: Annotations section
`2e15017` serve: open Store at startup, pass to NewServer
`b2bc104` docs: full postmortem report

### Key bugs caught

1. SQLite ON CONFLICT VALUES restriction → UPDATE+INSERT workaround
2. DuckDB positional boolean in sqlite_attach → named parameter required
3. extractPathParam parts[2] OOB → parts[1]
4. json.Unmarshal nil slice for "[]" → explicit parseJSONArray guard
5. Shell variable leakage in E2E test → _BODY/_CODE globals

### Related Files

- `POSTMORTEM.md` — Full postmortem report (intern guide, architecture, API reference, bug log)
- `reference/02-diary.md` — Step-by-step implementation diary
- `tasks.md` — Fine-grained task status table
- `design-doc/01-annotation-storage-backend-and-cli-design-decision.md` — Design rationale
- `design-doc/02-annotation-cli-implementation-guide.md` — Implementation guide

## 2026-04-04

Added design doc 03 for transcript-linked annotation UX: clickable annotation cards that jump to transcript targets, inline turn/tool-call markers, and in-context scoped annotation creation. Included ASCII wireframes and a YAML DSL component sketch; uploaded to reMarkable.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/04/ANNOTATE-CLI--go-minitrace-annotation-cli-and-storage-backend-design/design-doc/03-transcript-linked-annotation-ui-design.md — New UI design doc for annotation-to-transcript navigation and inline markers

