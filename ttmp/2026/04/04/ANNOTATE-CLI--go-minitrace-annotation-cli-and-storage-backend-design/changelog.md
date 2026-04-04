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

