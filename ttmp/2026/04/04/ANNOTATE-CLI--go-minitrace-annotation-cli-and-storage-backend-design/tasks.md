# Tasks — ANNOTATE-CLI

Implementation status as of 2026-04-04.

---

## ✅ Done

### Phase 1 — SQLite Store (pkg/annotate/)

| # | Task | Status |
|---|------|--------|
| 1.1 | Add `github.com/mattn/go-sqlite3` to go.mod | ✅ Already present |
| 1.2 | Add `github.com/google/uuid` to go.mod | ✅ Already present |
| 1.3 | Create `pkg/annotate/store.go` | ✅ Committed `238aba7` |
| 1.4 | Create `pkg/annotate/store_test.go` | ✅ 11 tests, committed |
| 1.5 | `migrate()` — annotations + sync_state tables + indexes | ✅ |
| 1.6 | `Open()` — WAL mode, busy_timeout=5000 | ✅ |
| 1.7 | `AddAnnotation()` — JSON-encode tags/taxonomy, mark unsynced | ✅ |
| 1.8 | `GetAnnotationsForSession()` — JSON-decode on read | ✅ |
| 1.9 | `List()` — dynamic WHERE, taxonomy LIKE across 3 columns | ✅ |
| 1.10 | `Update()` — buildPatchSET, ErrNotFound | ✅ |
| 1.11 | `Delete()` — ErrNotFound | ✅ |
| 1.12 | `markUnsynced()` — UPDATE then INSERT (SQLite limitation) | ✅ |
| 1.13 | `markSynced()` — upsert | ✅ |
| 1.14 | `GetUnsyncedSessions()` — JOIN sync_state | ✅ |
| 1.15 | All unit tests | ✅ 11 tests pass, 0 lint issues |

### Phase 2 — Sync + CLI

| # | Task | Status |
|---|------|--------|
| 2.1 | Create `pkg/annotate/sync.go` | ✅ Committed `6c71f31` |
| 2.2 | Create `pkg/annotate/sync_test.go` | ✅ 5 tests, committed |
| 2.3 | `SyncSession()` — atomic write (tmp-then-rename), nil → `[]` | ✅ |
| 2.4 | `SyncAll()` — session index iteration, report aggregation | ✅ |
| 2.5 | `annotate add` — validate category, UUID, print summary | ✅ Committed `eec4611` |
| 2.6 | `annotate list` — table/json output, all filters | ✅ |
| 2.7 | `annotate edit` — flagIsSet partial patching | ✅ |
| 2.8 | `annotate delete` — ErrNotFound handling | ✅ |
| 2.9 | `annotate sync` — archive-glob inference, session index | ✅ |
| 2.10 | `annotate import` — JSON file/stdin | ✅ |
| 2.11 | Wire into `main.go` | ✅ |
| 2.12 | Smoke test (add + list + delete) | ✅ Passed |

### Phase 3 — DuckDB Integration

| # | Task | Status |
|---|------|--------|
| 3.1 | Create `pkg/annotate/duckdb.go` | ✅ Committed `4116a58` |
| 3.2 | `AttachAnnotationsToDuckDB()` — INSTALL/LOAD sqlite_scanner + sqlite_attach | ✅ |
| 3.3 | Wire into serve startup (`serve.go`) | ✅ |
| 3.4 | `outputDirFromGlobs()` helper | ✅ |
| 3.5 | Errors as warnings (non-fatal) | ✅ |
| 3.6 | Update `queries/annotations.sql` | ✅ JOIN annotations with sessions_base |
| 3.7 | DuckDB integration test | ❌ Not written |

### Phase 4 — HTTP API

| # | Task | Status |
|---|------|--------|
| 4.1 | `handlers_annotations.go` | ✅ Committed `f155b6e` |
| 4.2 | `GET /api/sessions/{id}/annotations` | ✅ |
| 4.3 | `POST /api/sessions/{id}/annotations` | ✅ |
| 4.4 | `GET /api/annotations` — list with filters | ✅ |
| 4.5 | `PUT /api/annotations/{annId}` — patch | ✅ |
| 4.6 | `DELETE /api/annotations/{annId}` | ✅ |
| 4.7 | `POST /api/annotations/sync` | ✅ |
| 4.8 | `NewServer` accepts `annoStore, annoIndex` | ✅ |
| 4.9 | `Server.annoStore == nil` → 503 on all handlers | ✅ |
| 4.10 | API integration test | ❌ Not written |

### Phase 5 — Web UI

| # | Task | Status |
|---|------|--------|
| 5.1 | `web/src/types/session.ts` — Annotation types | ✅ Committed `7421127` |
| 5.2 | `web/src/api/minitrace.ts` — 5 RTK Query endpoints | ✅ |
| 5.3 | `AnnotationPanel.tsx` — list + add form + delete + sync | ✅ |
| 5.4 | Transcript/Annotations tab toggle in `TranscriptViewer.tsx` | ✅ |
| 5.5 | `npm run build` passes | ✅ |
| 5.6 | `SessionBrowser` annotation badges | ❌ Not done |
| 5.7 | `QueryEditor` cross-session annotation search | ❌ Not done |

---

## ❌ Remaining

### E2E Test Scripts
- [ ] `scripts/e2e-annotate.sh` — CLI smoke test: add → sqlite3 verify → sync → validate
- [ ] `scripts/e2e-duckdb-sqlite.sh` — DuckDB live query after annotate add/delete
- [ ] `scripts/e2e-api.sh` — HTTP API: curl POST/GET/PUT/DELETE + sync

### Phase 6 — Polish
- [ ] Extend `pkg/validate/json.go` to validate annotation structure
- [ ] Classification escalation enforcement (public → internal → confidential → customer-confidential)
- [ ] `go-minitrace validate` with annotation check
- [ ] Add `## Annotations` section to README

### Nice-to-Have
- [ ] `annotate get` command (fetch single annotation by ID)
- [ ] `annotate stats` (counts by category/session)
- [ ] `GetAnnotationByID()` in store (single-row lookup)
- [ ] Pagination metadata in `List` response
- [ ] `--diff` flag on sync to show JSON diff before writing

---

## Commit Summary

| Commit | Description |
|--------|-------------|
| `238aba7` | pkg/annotate: SQLite-backed annotation store |
| `6c71f31` | pkg/annotate: atomic JSON sync |
| `eec4611` | cmd/annotate: 6 CLI commands |
| `4116a58` | pkg/annotate: DuckDB sqlite_scanner attachment |
| `f155b6e` | serve: 6 annotation HTTP API handlers |
| `7421127` | web: AnnotationPanel + RTK Query API |
