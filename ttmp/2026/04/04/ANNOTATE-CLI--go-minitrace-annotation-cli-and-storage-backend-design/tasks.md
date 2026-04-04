# Tasks — ANNOTATE-CLI

Implementation guide: `design-doc/02-annotation-cli-implementation-guide.md`

---

## Phase 1 — SQLite Store (pkg/annotate/)

### 1.1 Dependencies

- [ ] Add `github.com/mattn/go-sqlite3 v1.14.22` to `go.mod` (or `modernc.org/sqlite` for pure Go)
- [x] Add `github.com/google/uuid v1.6.0` to `go.mod`

### 1.2 pkg/annotate/store.go — SQLite CRUD

Reference: implementation guide Part 2 (lines ~1050–1200)

- [ ] Create `pkg/annotate/store.go`
- [x] Define `Store` struct with `db *sql.DB` and `dbPath string`
- [ ] Implement `Open(ctx, outputDir string) (*Store, error)`
  - Resolve to absolute path
  - `os.MkdirAll` to ensure directory exists
  - `sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")`
  - Call `migrate()`
  - Return `*Store`
- [ ] Implement `migrate(ctx)` — create tables with `CREATE TABLE IF NOT EXISTS`
  - `annotations` table with all columns (id, session_id, annotator, scope_type, target_id, category, title, detail, tags, taxonomy_m, taxonomy_mast, taxonomy_tm, classification, created_at, updated_at)
  - Indexes: session_id, (scope_type, target_id), category, annotator
  - `sync_state` table
  - `sessions` registry table
- [ ] Implement `Close() error`
- [ ] Implement `AddAnnotation(ctx, ann minitrace.Annotation, sessionID string) error`
  - JSON-encode tags and taxonomy arrays
  - `INSERT INTO annotations ...`
  - Call `markUnsynced(ctx, sessionID)`
- [ ] Implement `GetAnnotationsForSession(ctx, sessionID string) ([]minitrace.Annotation, error)`
  - `SELECT ... FROM annotations WHERE session_id = ? ORDER BY created_at`
  - `scanAnnotation(rows)` helper to map row → minitrace.Annotation
  - Parse JSON strings for tags and taxonomy arrays
- [ ] Implement `List(ctx, opts ListOptions) ([]AnnotationRow, error)`
  - Build WHERE clause from optional filters: SessionID, ScopeType, Category, Annotator, Taxonomy
  - For Taxonomy: LIKE pattern match across taxonomy_m, taxonomy_mast, taxonomy_tm
  - ORDER BY created_at DESC, LIMIT/OFFSET
- [ ] Implement `Update(ctx, id string, patch AnnotationPatch) error`
  - Build SET clause dynamically from non-nil patch fields
  - `UPDATE annotations SET ... WHERE id = ?`
  - `rowsAffected == 0` → return `ErrNotFound`
  - Call `markUnsynced`
- [ ] Implement `Delete(ctx, id string) error`
  - Look up session_id before delete
  - `DELETE FROM annotations WHERE id = ?`
  - Return `ErrNotFound` if not found
  - Call `markUnsynced`
- [ ] Implement `markUnsynced(ctx, sessionID)` — upsert into sync_state
- [ ] Implement `markSynced(ctx, sessionID, count)`
- [ ] Implement `GetUnsyncedSessions(ctx) ([]SyncState, error)` — JOIN annotations with sync_state to find stale sessions
- [ ] Define `ListOptions`, `AnnotationPatch`, `AnnotationRow`, `SyncState`, `ErrNotFound`

### 1.3 Unit tests

Reference: implementation guide Part 10 (lines ~2400–2500)

- [ ] Create `pkg/annotate/store_test.go`
- [ ] Test `Open` creates DB and runs migrations
- [ ] Test `AddAnnotation` inserts and `GetAnnotationsForSession` retrieves
- [ ] Test `List` with all filter combinations
- [ ] Test `Update` patches fields, returns `ErrNotFound` for unknown ID
- [ ] Test `Delete` removes row, returns `ErrNotFound`
- [ ] Test `GetUnsyncedSessions` detects unsynced and synced sessions
- [ ] Use `t.TempDir()` for all DB paths

---

## Phase 2 — CLI Commands (cmd/go-minitrace/cmds/annotate/)

Reference: implementation guide Part 5 (lines ~1200–1500)

### 2.1 Wiring

- [ ] Create `cmd/go-minitrace/cmds/annotate/` directory
- [ ] Create `cmd/go-minitrace/cmds/annotate/root.go`
  - `NewCommand() (*cobra.Command, error)` — root command with `Use: "annotate"`
  - Add all subcommands
  - Long description explaining SQLite overlay pattern and sync
- [ ] Wire into `cmd/go-minitrace/main.go`
  - Import `"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/annotate"`
  - `annotateCmd, err := annotate.NewCommand()`
  - `rootCmd.AddCommand(annotateCmd)`

### 2.2 common flags

All annotate subcommands share `--output-dir` flag. Consider a shared helper or each subcommand sets it independently (implementation guide uses per-command flags for simplicity).

### 2.3 annotate add

Reference: implementation guide Part 5.3 (lines ~1250–1340)

- [ ] `cmd/go-minitrace/cmds/annotate/add.go`
- [ ] Flags: `--output-dir`, `--session` (required), `--scope` (default: session), `--target-id` (default: sessionID), `--annotator` (default: user), `--category` (required), `--title` (required), `--detail`, `--tags`, `--taxonomy-minitrace`, `--taxonomy-mast`, `--taxonomy-toolemu`, `--classification`
- [ ] `parseCommaList(s string) []string` helper
- [ ] `validateAnnotation(ann minitrace.Annotation) error` — check category, scope_type, annotator enums
- [ ] `BuildAnnotation(...)` from `pkg/minitrace`
- [ ] UUID via `uuid.New().String()`
- [ ] On success: print ID, session, scope, category, title
- [ ] Mark task [2] done

### 2.4 annotate list

Reference: implementation guide Part 5.4 (lines ~1340–1430)

- [ ] `cmd/go-minitrace/cmds/annotate/list.go`
- [ ] Flags: `--output-dir`, `--session`, `--scope`, `--category`, `--annotator`, `--taxonomy`, `--limit` (default 50), `--format` (table/json)
- [ ] `printHeader(...)` and `printRow(...)` helpers for table output
- [ ] `printJSON(...)` helper for JSON output
- [ ] Handle empty results gracefully
- [ ] Show count at end
- [ ] Mark task [3] done

### 2.5 annotate edit

Reference: implementation guide Part 5 (not fully shown, follows Update pattern)

- [ ] `cmd/go-minitrace/cmds/annotate/edit.go`
- [ ] Flags: `--output-dir`, `--id` (required), `--title`, `--detail`, `--category`, `--tags`, `--taxonomy-minitrace`, `--classification`
- [ ] Build `AnnotationPatch` from non-empty flags
- [ ] Call `store.Update(ctx, id, patch)`
- [ ] Print updated annotation on success

### 2.6 annotate delete

Reference: implementation guide Part 5 (not fully shown)

- [ ] `cmd/go-minitrace/cmds/annotate/delete.go`
- [ ] Flags: `--output-dir`, `--id` (required)
- [ ] Call `store.Delete(ctx, id)`
- [ ] Print confirmation on success

### 2.7 annotate sync

Reference: implementation guide Part 3 (lines ~950–1050) and Part 5.5 (lines ~1430–1500)

- [ ] `cmd/go-minitrace/cmds/annotate/sync.go`
- [ ] Requires: `pkg/annotate/sync.go` (Phase 2.8) — can stub this and implement later, or implement together
- [ ] Flags: `--output-dir`, `--archive-glob` (default: `./output/active/*/*.minitrace.json`), `--session` (optional filter), `--dry-run`
- [ ] Build session index: `ExpandArchiveGlobs` → read each file → extract ID → `map[string]string`
- [ ] Call `store.SyncAll(ctx, sessionIndex, SyncOptions{DryRun, SessionID})`
- [ ] Print report: synced count, skipped count, errors
- [ ] If errors: exit non-zero

### 2.8 pkg/annotate/sync.go

Reference: implementation guide Part 3 (lines ~950–1050)

- [ ] Create `pkg/annotate/sync.go`
- [ ] Define `SyncOptions` struct: `{DryRun bool, SessionID string}`
- [ ] Define `SyncReport` struct: `{Synced []string, Skipped []string, Errors []SyncError}`
- [ ] Define `SyncError` struct: `{SessionID string, Error string}`
- [ ] Implement `SyncSession(ctx, filePath string, annotations []minitrace.Annotation, dryRun bool) error`
  - `os.ReadFile(filePath)`
  - `json.Unmarshal` into `minitrace.Session`
  - `session.Annotations = annotations`
  - Ensure `session.Annotations != nil` (nil marshals to `null`, must be `[]` for spec compliance)
  - `json.MarshalIndent(session, "", "  ")` — 2-space indent
  - If dry-run: print summary and return nil
  - Write to `{file}.tmp`, `os.Rename(tmp, file)` — atomic on POSIX same-filesystem
  - `os.Remove(tmp)` on error
- [ ] Implement `SyncAll(ctx, sessionIndex map[string]string, opts SyncOptions) (*SyncReport, error)`
  - `store.GetUnsyncedSessions(ctx)` → list of sessions needing sync
  - Filter by `opts.SessionID` if set
  - For each session: look up `sessionIndex[sessionID]` → skip if not found
  - `store.GetAnnotationsForSession(ctx, sessionID)`
  - `SyncSession(...)`
  - `store.markSynced(...)` on success
  - Collect results into `SyncReport`

### 2.9 annotate import

- [ ] `cmd/go-minitrace/cmds/annotate/import.go`
- [ ] Flag: `--output-dir`, `--file` (required JSON file)
- [ ] Read JSON file containing array of annotation objects
- [ ] For each annotation: `store.AddAnnotation(ctx, ann, ann.Scope.TargetID)` (infer sessionID from scope)
- [ ] Or accept `{session_id, annotations: [...]}` structure

### 2.10 Integration test script

Reference: implementation guide Part 10 (lines ~2500–2620)

- [ ] Create `scripts/e2e-annotate.sh` in ticket scripts folder
- [ ] Creates temp dir, writes minimal `.minitrace.json`
- [ ] `go-minitrace annotate add --output-dir $DIR --session sess-001 --category ai-failure --title "Test"`
- [ ] `sqlite3 $DIR/annotations.db "SELECT * FROM annotations;"`
- [ ] `go-minitrace annotate sync --output-dir $DIR --archive-glob "$DIR/*.minitrace.json"`
- [ ] `go-minitrace validate --path $DIR/sess-001.minitrace.json`
- [ ] Verify annotations array in JSON is non-null

---

## Phase 3 — DuckDB Integration

Reference: implementation guide Part 4 (lines ~1070–1215)

### 3.1 pkg/annotate/duckdb.go

- [ ] Create `pkg/annotate/duckdb.go`
- [ ] `AttachAnnotationsToDuckDB(ctx, conn *sql.Conn, outputDir string) error`
  - `conn.ExecContext(ctx, "INSTALL sqlite_scanner")`
  - `conn.ExecContext(ctx, "LOAD sqlite_scanner")`
  - Resolve `annotations.db` path
  - Check if file exists: `os.Stat(dbPath)` → if not found, return nil (DB created on first write)
  - `conn.ExecContext(ctx, "CALL sqlite_attach($1, overwrite => true)", dbPath)` — use `$1` for path, named param for overwrite
  - If attach fails (DB doesn't exist): log warning, return nil

### 3.2 Wire into serve startup

Reference: implementation guide Part 4.6 (lines ~1215–1220)

- [ ] In `cmd/go-minitrace/cmds/serve/serve.go`, modify `ServeSettings`:
  - Add `OutputDir string` field (or infer from first `--archive-glob`)
- [ ] In `ServeCommand.Run(...)`, after `OpenConnection`, before `LoadArchive`:
  - `annotate.AttachAnnotationsToDuckDB(signalCtx, conn, outputDir)` — log warning on error, don't fail serve
- [ ] Note: `ServeSettings` already has `--archive-glob`. Use that to infer output dir:
  ```go
  outputDir := filepath.Dir(filepath.Dir(settings.ArchiveGlob[0]))
  ```

### 3.3 Update queries/annotations.sql

Reference: implementation guide Part 4.4 (lines ~1200–1210)

- [ ] Update `queries/annotations.sql`
- [ ] Replace `UNNEST(annotations)` approach with SQLite join:
  ```sql
  SELECT
    a.session_id,
    sb.environment->>'agent_framework' AS framework,
    a.annotator,
    a.category,
    a.title,
    a.scope_type,
    a.target_id,
    a.created_at,
    a.taxonomy_m AS taxonomy_minitrace,
    a.tags
  FROM annotations a
  JOIN sessions_base sb ON sb.id = a.session_id
  ORDER BY a.created_at DESC;
  ```

### 3.4 DuckDB + SQLite integration test

- [ ] `scripts/e2e-duckdb-sqllite.sh` in ticket scripts folder
- [ ] Start `go-minitrace serve` with annotations.db and sessions loaded
- [ ] `go-minitrace annotate add` to create annotation
- [ ] `curl http://localhost:8080/api/query` with annotations.sql → verify annotation appears
- [ ] Delete annotation via CLI
- [ ] Query again → verify it's gone (DuckDB reads live SQLite)

---

## Phase 4 — HTTP API

Reference: implementation guide Part 6 (lines ~1700–1920)

### 4.1 Extend Server struct

Reference: implementation guide Part 6.1 (lines ~1705–1730)

- [ ] Add `annoStore *annotate.Store` and `annoIndex map[string]string` to `Server` struct
- [ ] Update `NewServer` to accept these new fields
- [ ] Add to `ServeSettings`: `AnnotationsDBPath string` (default: inferred from archive glob)

### 4.2 New HTTP routes

Reference: implementation guide Part 6.2 (lines ~1735–1750)

- [ ] Add to `routes()`:
  ```
  GET  /api/sessions/{id}/annotations
  POST /api/sessions/{id}/annotations
  GET  /api/annotations
  PUT  /api/annotations/{annId}
  DELETE /api/annotations/{annId}
  POST /api/annotations/sync
  ```

### 4.3 GET /api/sessions/{id}/annotations

Reference: implementation guide Part 6.3 (lines ~1755–1770)

- [ ] `handleGetSessionAnnotations(w, r)`
- [ ] Extract `sessionID` from path
- [ ] `s.annoStore.GetAnnotationsForSession(ctx, sessionID)`
- [ ] Return `{"session_id": "...", "count": N, "annotations": [...]}`
- [ ] 500 on error

### 4.4 POST /api/sessions/{id}/annotations

Reference: implementation guide Part 6.3 (lines ~1770–1820)

- [ ] `handleCreateAnnotation(w, r)`
- [ ] `CreateAnnotationRequest` struct
- [ ] Validate: title required, category required
- [ ] Default `scope_type = "session"`, `target_id = sessionID`, `annotator = "user"`
- [ ] `BuildAnnotation(...)` with UUID
- [ ] `s.annoStore.AddAnnotation(ctx, ann, sessionID)`
- [ ] Return `201 Created` with annotation body
- [ ] **No DuckDB refresh needed** — annotations are live via sqlite_scanner

### 4.5 GET /api/annotations

- [ ] `handleListAnnotations(w, r)`
- [ ] Parse query params: `session`, `category`, `annotator`, `taxonomy`, `limit`
- [ ] `s.annoStore.List(ctx, annotate.ListOptions{...})`
- [ ] Return JSON array

### 4.6 PUT /api/annotations/{annId}

- [ ] `handleUpdateAnnotation(w, r)`
- [ ] `AnnotationPatchRequest` struct
- [ ] `s.annoStore.Update(ctx, annId, patch)`
- [ ] Return `404` if `ErrNotFound`
- [ ] Return `200` with updated annotation

### 4.7 DELETE /api/annotations/{annId}

- [ ] `handleDeleteAnnotation(w, r)`
- [ ] `s.annoStore.Delete(ctx, annId)`
- [ ] Return `204 No Content` on success
- [ ] Return `404` if `ErrNotFound`

### 4.8 POST /api/annotations/sync

Reference: implementation guide Part 6.3 (lines ~1850–1890)

- [ ] `handleSyncAnnotations(w, r)`
- [ ] `SyncRequest` struct: `{session_id?: string, dry_run?: bool}`
- [ ] `s.annoStore.SyncAll(ctx, s.annoIndex, annotate.SyncOptions{...})`
- [ ] Return `SyncResponse`: `{synced: [], skipped: [], errors: []}`
- [ ] Return `206 Partial Content` if errors present, `200 OK` otherwise

### 4.9 API integration test

- [ ] `scripts/e2e-api.sh` in ticket scripts folder
- [ ] Start serve
- [ ] `curl POST /api/sessions/{id}/annotations` → verify 201
- [ ] `curl GET /api/sessions/{id}/annotations` → verify annotation present
- [ ] `curl PUT /api/annotations/{annId}` → verify update
- [ ] `curl DELETE /api/annotations/{annId}` → verify 204
- [ ] `curl POST /api/annotations/sync` → verify JSON updated

---

## Phase 5 — Web UI

Reference: implementation guide Part 7 (lines ~1920–2100)

### 5.1 TypeScript types

Reference: implementation guide Part 7.1 (lines ~1925–2000)

- [ ] `web/src/api/minitrace.ts`: add `Annotation`, `CreateAnnotationPayload`, `SyncResponse` types
- [ ] Add API functions: `getSessionAnnotations`, `createAnnotation`, `updateAnnotation`, `deleteAnnotation`, `syncAnnotations`

### 5.2 AnnotationPanel component

Reference: implementation guide Part 7.2 (lines ~2000–2140)

- [ ] `web/src/components/TranscriptViewer/AnnotationPanel.tsx`
- [ ] Props: `sessionId`, `selectedScope?: {type, targetId}`, `onClose`
- [ ] State: `annotations`, `loading`, `showForm`, form fields
- [ ] `useEffect` on `sessionId` → `getSessionAnnotations`
- [ ] Form: category select, title input, detail textarea, tags input, taxonomy input
- [ ] Submit → `createAnnotation` → reload
- [ ] Delete button per annotation
- [ ] Color-coded category badges

### 5.3 Integrate into TranscriptViewer

- [ ] `web/src/pages/TranscriptViewerPage.tsx`
- [ ] Add "Annotations" button / tab alongside "Transcript"
- [ ] Toggle between transcript view and annotation panel
- [ ] On turn/tool call click: set `selectedScope` and open panel

### 5.4 Annotation badges in SessionBrowser

Reference: implementation guide Part 7.3 (lines ~2140–2180)

- [ ] `web/src/components/SessionBrowser/SessionBrowser.tsx`
- [ ] Extend `SessionSummary` to include annotation counts (fetch from API or embed in session manifest)
- [ ] Display badges: `N failures`, `N observations` per session row

### 5.5 Cross-session annotation search in QueryEditor

- [ ] `web/src/pages/QueryEditorPage.tsx`
- [ ] Add "Annotations" tab in query sidebar
- [ ] Preset query buttons: "All ai-failures", "All F-AUT", "Annotations by X"
- [ ] Uses the same query editor as sessions

---

## Phase 6 — Polish

### 6.1 Validate command enhancement

- [ ] Extend `pkg/validate/json.go` to validate annotation structure
- [ ] Check: `annotations` is array, each annotation has required fields, enum values valid, taxonomy codes match known set

### 6.2 Classification escalation validation

- [ ] In `store.Update`, enforce: classification can only escalate (public → internal → confidential → customer-confidential), never de-escalate
- [ ] On API level: reject requests that try to de-escalate

### 6.3 go-minitrace validate with annotation check

- [ ] `go-minitrace validate --path session.minitrace.json` should validate annotation arrays if present

### 6.4 Documentation

- [ ] Add `## Annotations` section to README
- [ ] Document the SQLite + DuckDB pattern
- [ ] Document `--dry-run` for sync
- [ ] Document the SQLite path convention (colocated at output root)

---

## Open Questions (resolve before starting)

- [ ] **SQLite driver**: `mattn/go-sqlite3` (CGO) or `modernc.org/sqlite` (pure Go)? Check if go-minitrace already builds with CGO.
- [ ] **Multiple output dirs**: If sessions live in multiple output roots, how is `annotations.db` found? (Current design: one per output root.)
- [ ] **Serve + output-dir**: Does `--archive-glob` reliably give us the output dir, or do we need a new `--output-dir` flag for serve?

---

## Dependency order

```
Phase 1 ────────────────────── Phase 2 ────────────── Phase 3 ──── Phase 4 ─── Phase 5
  1.1 go.mod deps ──────────────────┐
  1.2 store.go ────────────────────┼─── 2.3 add ─────────────────┐
  1.3 store tests ─────────────────┼─── 2.4 list ────────────────┤
  ────────────────────────────────── 2.5 edit ────────────────┤
  ────────────────────────────────── 2.6 delete ───────────────┤
  ────────────────────────────────── 2.7 sync ─────────────────┼── 3.1 duckdb.go
  ────────────────────────────────── 2.8 sync.go ──────────────┤    3.2 serve wiring
  ────────────────────────────────── 2.9 import ───────────────┤    3.3 queries/annotations.sql
  ────────────────────────────────── 2.10 e2e script ──────────┘    3.4 e2e script
                                                                 Phase 4: all sub-tasks depend on Phase 3
                                                                 Phase 5: all sub-tasks depend on Phase 4
```
