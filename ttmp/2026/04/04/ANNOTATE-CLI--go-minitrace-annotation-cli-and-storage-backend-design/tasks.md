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
| 3.7 | DuckDB integration test | ✅ `09-e2e-duckdb-sqlite-live.sh` |

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
| 4.10 | API integration test | ✅ `10-e2e-api.sh` |

### Phase 5 — Web UI

| # | Task | Status |
|---|------|--------|
| 5.1 | `web/src/types/session.ts` — Annotation types | ✅ Committed `7421127` |
| 5.2 | `web/src/api/minitrace.ts` — 5 RTK Query endpoints | ✅ |
| 5.3 | `AnnotationPanel.tsx` — list + add form + delete + sync | ✅ |
| 5.4 | Transcript/Annotations tab toggle in `TranscriptViewer.tsx` | ✅ |
| 5.5 | `npm run build` passes | ✅ |
| 5.6 | `SessionBrowser` annotation badges | ❌ |
| 5.7 | `QueryEditor` cross-session annotation search | ❌ |

### Phase 6 — Polish

| # | Task | Status |
|---|------|--------|
| 6.1 | E2E: `scripts/08-e2e-annotate-cli.sh` | ✅ Committed `b663b03` |
| 6.2 | E2E: `scripts/09-e2e-duckdb-sqlite-live.sh` | ✅ |
| 6.3 | E2E: `scripts/10-e2e-api.sh` | ✅ |
| 6.4 | Bug: `extractPathParam` parts[2]→parts[1] (OOB) | ✅ Fixed `b663b03` |
| 6.5 | `pkg/validate/json.go` — annotation structure validation | ✅ Committed `5430a20` |
| 6.6 | Validation tests (12 cases) | ✅ |
| 6.7 | `## Annotations` section in README | ✅ Committed `bb3fcc5` |
| 6.8 | Classification escalation enforcement | ❌ Not enforced (levels defined, not enforced) |

### Phase 7 — Transcript-Linked Annotation UI

| # | Task | Status |
|---|------|--------|
| 7.1 | Click annotation card → switch to Transcript tab and jump to session/turn/tool-call target | ✅ |
| 7.2 | Add DOM anchors + highlight state for session top, turn rows, and tool-call rows | ✅ |
| 7.3 | Expand containing block automatically when a target turn/tool-call is focused | ✅ |
| 7.4 | Show scope label / target label on annotation cards (`Session`, `Turn #15`, `Tool call ...`) | ✅ |
| 7.5 | Inline transcript markers: show annotation chips/counts on turns and tool-call rows | ✅ |
| 7.6 | Add `Annotate` affordance on turn rows with prefilled `scope_type=turn,target_id=idx` | ✅ |
| 7.7 | Add `Annotate` affordance on tool-call rows with prefilled `scope_type=tool_call,target_id=id` | ✅ |
| 7.8 | Add basic frontend test or E2E smoke test for annotation → transcript navigation | 🔄 In progress |

---

## ❌ Remaining

### Nice-to-Have
- [ ] `SessionBrowser` annotation badges (web)
- [ ] `QueryEditor` cross-session annotation search (web)
- [ ] `annotate get` command (fetch single annotation by ID)
- [ ] `annotate stats` (counts by category/session)
- [ ] `GetAnnotationByID()` in store (single-row lookup)
- [ ] Pagination metadata in `List` response
- [ ] `--diff` flag on sync to show JSON diff before writing
- [ ] Classification escalation enforcement (public → internal → confidential → customer-confidential)

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
| `b663b03` | fix(serve): extractPathParam OOB fix + 3 E2E scripts |
| `5430a20` | pkg/validate: annotation structure validation |
| `bb3fcc5` | README: Annotations section |
