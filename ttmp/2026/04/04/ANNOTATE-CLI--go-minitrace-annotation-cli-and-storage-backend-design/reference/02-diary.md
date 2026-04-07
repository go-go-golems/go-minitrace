---
Title: ANNOTATE-CLI Implementation Diary
Ticket: ANNOTATE-CLI
Status: active
Topics:
    - minitrace
    - annotations
    - cli
    - go
    - duckdb
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/annotate/sync.go
      Note: SyncSession atomic write
    - Path: pkg/annotate/sync_test.go
      Note: 5 sync tests (commit 6c71f31)
ExternalSources: []
Summary: Step-by-step diary of implementing the annotation CLI.
LastUpdated: 2026-04-04T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Implementing the annotation CLI and SQLite storage backend for go-minitrace. Phases:
1. `pkg/annotate/store.go` — SQLite CRUD
2. `pkg/annotate/sync.go` — atomic JSON write-back
3. `pkg/annotate/duckdb.go` — DuckDB sqlite_scanner attachment
4. `cmd/go-minitrace/cmds/annotate/` — CLI commands (add, list, edit, delete, sync, import)
5. Serve HTTP endpoints
6. Web UI AnnotationPanel

---

## Step 1: pkg/annotate/store.go — SQLite CRUD

Committed: `238aba7` — "pkg/annotate: SQLite-backed annotation store"

Created `pkg/annotate/store.go` (502 lines) and `pkg/annotate/store_test.go` (384 lines, 11 tests). This is the foundation: a SQLite store at `outputDir/annotations.db` that persists minitrace annotations, tracks which sessions have been modified, and exposes CRUD operations for the CLI and HTTP API.

### Prompt Context

**User prompt (verbatim):** "Phase 1 — pkg/annotate/store.go: implement the SQLite-backed annotation store... Start working through the implementation step by step."

**Assistant interpretation:** Implement the SQLite store package with CRUD operations for annotations, wired to the minitrace schema types.

**Inferred user intent:** Build the foundation of the annotation system: a SQLite store that persists annotations and tracks sync state.

### What I did

Created `pkg/annotate/store.go` and `pkg/annotate/store_test.go`.

**Schema (`migrate()`):**
- `annotations` table: id, session_id, annotator, scope_type, target_id, category, title, detail, tags (JSON TEXT), taxonomy_m (JSON TEXT), taxonomy_mast (JSON TEXT), taxonomy_tm (JSON TEXT), classification, created_at, updated_at
- `sync_state` table: session_id (PK), synced_at, change_count
- Indexes on session_id, (scope_type, target_id), category, annotator

**API surface:**
- `Open(ctx, outputDir) (*Store, error)` — resolves to absolute path, creates dir + WAL-mode DB, runs migrations
- `AddAnnotation(ctx, ann, sessionID)` — inserts, JSON-encodes tags/taxonomy, calls markUnsynced
- `GetAnnotationsForSession(ctx, sessionID)` — selects + JSON-decodes tags/taxonomy
- `List(ctx, opts ListOptions)` — dynamic WHERE builder; taxonomy filter uses LIKE across all 3 taxonomy columns
- `Update(ctx, id, patch AnnotationPatch)` — builds dynamic SET clause from non-nil patch fields only
- `Delete(ctx, id)` — returns ErrNotFound if missing
- `GetUnsyncedSessions(ctx)` — for sync planning
- `markUnsynced(ctx, sessionID)` / `markSynced(ctx, sessionID, count)`

**Helper types:** `ListOptions`, `AnnotationPatch`, `AnnotationRow`, `SyncState`, `ErrNotFound`

### Why

SQLite is the right tool for write-heavy annotation workloads. DuckDB is analytical and read-only. The store must track which sessions have unsynced changes so the `sync` command can efficiently write annotations back to `.minitrace.json` files. Tags and taxonomy are stored as JSON strings in TEXT columns because SQLite has no native array type — this is the simplest schema that maps cleanly to/from `minitrace.Annotation`.

### What worked

- All 11 tests pass cleanly
- golangci-lint reports 0 issues on `./pkg/annotate/...`
- `mattn/go-sqlite3` and `google/uuid` already in `go.mod` — no new dependencies needed
- `buildPatchSET` correctly handles partial patches (only non-nil fields generate SET entries)
- `closeRows`/`closeStore` helpers cleanly silence errcheck without verbose anonymous functions

### What didn't work

**Bug 1: SQLite VALUES column restriction** — First attempt at `markUnsynced` used:

```sql
INSERT INTO sync_state (session_id, synced_at, change_count)
VALUES (?, '', change_count + 1)
ON CONFLICT(session_id) DO UPDATE SET change_count = change_count + 1
```

Runtime error: `no such column: change_count`. SQLite does not support referencing a column in the VALUES clause of an INSERT that also has an ON CONFLICT clause.

Fix:
```go
result, err := s.db.ExecContext(ctx, `UPDATE sync_state SET change_count = change_count + 1 WHERE session_id = ?`, sessionID)
rowsAffected, _ := result.RowsAffected()
if rowsAffected == 0 {
    _, err = s.db.ExecContext(ctx, `INSERT INTO sync_state (session_id, synced_at, change_count) VALUES (?, '', 1)`, sessionID)
}
```

**Bug 2: json.Unmarshal leaves nil slice for `[]`** — `var out []string` creates nil slice. `json.Unmarshal([]byte("[]"), &out)` appends nothing for an empty array. Result: nil, not `[]string{}`.

Test `TestNilTagsAndTaxonomy` caught this. Fix:
```go
func parseJSONArray(s string) []string {
    if s == "" || s == "null" || s == "[]" {
        return []string{}
    }
    var out []string
    _ = json.Unmarshal([]byte(s), &out)
    return out
}
```

**Lint: errcheck failures** — `defer rows.Close()` and `defer store.Close()` flagged. Fix: helper functions:
```go
func closeRows(rows *sql.Rows)  { _ = rows.Close() }
func closeStore(s *Store)       { _ = s.Close() }
```

### What I learned

- SQLite's ON CONFLICT clause has a subtle restriction: you cannot read a column value in the VALUES clause if the INSERT also has ON CONFLICT. The error message ("no such column") is misleading — it suggests a typo rather than revealing the restriction.
- `json.Unmarshal` with a nil destination slice is safe (appends to nil), but for `[]` input it appends nothing, leaving the slice nil. For `null` it also leaves nil. Need explicit `"[]"` check.
- The `errcheck` linter flags `defer x.Close()` but not named-result-ignored calls or inline `(_ = x.Close())`. Helper functions are the cleanest solution.

### What was tricky to build

- **markUnsynced** needed to be idempotent (upsert semantics) but SQLite's ON CONFLICT VALUES restriction forced a two-step approach. The UPDATE-then-INSERT sequence is correct but the original single-statement approach felt cleaner.
- **buildPatchSET** must not produce trailing commas or empty SET clauses. The `appendSet` helper pattern handles this cleanly by checking `set != ""` before prepending ", ".
- **gofmt formatting** in the test file — the `TestNilTagsAndTaxonomy` assertion originally used `string(rune('a'+i))` to generate IDs which produced a gofmt diff. Simplified to fixed string prefixes.

### What warrants a second pair of eyes

- **markUnsynced race condition**: If two goroutines call `markUnsynced` for the same session_id concurrently, the UPDATE could both return rowsAffected=0, causing two INSERT attempts. SQLite's UNIQUE constraint on session_id would cause one to fail. This is a rare edge case (unlikely in single-user CLI) but worth noting. A transaction would serialize it.
- **List taxonomy filter** uses three LIKE patterns — performance degrades with many annotations. Fine for v1.
- **nil vs empty in AnnotationPatch**: `Tags *[]string` — callers must understand the pointer semantics. No compile-time enforcement of which fields are set.

### What should be done in the future

- Add `GetAnnotationByID(ctx, id)` — single-row lookup (currently only session-scoped lookup exists)
- Add pagination metadata to `List` — total count requires a separate `SELECT COUNT(*) WHERE ...` query
- Consider a transaction wrapper around `markUnsynced` to serialize concurrent upserts

### Code review instructions

**Start at:** `pkg/annotate/store.go`

Key symbols to review:
- `Open()` — dir creation, WAL mode, migration order
- `migrate()` — schema correctness (column names, index coverage)
- `AddAnnotation()` — JSON encoding of tags/taxonomy
- `scanAnnotations()` — all fields mapped correctly
- `Update()` — dynamic SET clause, ErrNotFound handling
- `markUnsynced()` — UPDATE-then-INSERT pattern
- `List()` — WHERE clause builder, taxonomy LIKE across 3 columns

**Validate:**
```bash
go test -v ./pkg/annotate/
golangci-lint run ./pkg/annotate/...
```

### Technical details

**SQLite pragma string:** `?_journal_mode=WAL&_busy_timeout=5000`
- WAL mode allows concurrent readers + serialized writer
- busy_timeout=5000ms waits up to 5s before returning "database is locked"

**JSON column encoding:** Tags (`[]string`), taxonomy (`[]string` for Minitrace/Mast/Toolemu) stored as JSON strings. Encoding: `json.Marshal`. Decoding: `json.Unmarshal` + `"[]"` guard in `parseJSONArray`.

**Dynamic WHERE in List:** Each optional field appends `AND col = ?` to base `WHERE 1=1`. Taxonomy filter: `LIKE '%pattern%'` applied to all 3 taxonomy columns.

**buildPatchSET:** Walks AnnotationPatch fields in order. Only non-nil pointers generate a SET entry. Returns `("", nil)` if patch is empty (Update returns early).

**Commit:** `238aba7` — "pkg/annotate: SQLite-backed annotation store"

---

## Step 2: pkg/annotate/sync.go — Atomic JSON Write-back

Committed: `6c71f31` — "pkg/annotate: atomic JSON sync for .minitrace.json write-back"

Added `pkg/annotate/sync.go` (185 lines) and `pkg/annotate/sync_test.go` (180 lines, 5 tests). SyncSession reads a `.minitrace.json` file, unmarshals it generically (to avoid re-serializing the whole session), replaces the `annotations` field with the provided `[]Annotation`, and writes back atomically via write-to-temp then `os.Rename`. `SyncAll` iterates over unsynced sessions from the store, looks up their file paths, and calls `SyncSession`.

### Prompt Context

**User prompt (verbatim):** "Phase 2..."

**Assistant interpretation:** Implement the atomic JSON write-back module and sync logic.

**Inferred user intent:** Enable writing annotations back to `.minitrace.json` files after editing.

### What I did

**SyncSession(filePath, annotations, opts):**
1. Read file as `[]byte`, unmarshal into `map[string]any`
2. Replace `annotations` key with JSON-marshaled annotation slice
3. Re-marshal with 2-space indent, append `\n`
4. If `DryRun`: print summary to stdout
5. Else: `WriteFile(tmp)`, then `Rename(tmp, file)` — atomic on POSIX same-filesystem

**SyncAll(ctx, sessionIndex, opts):**
1. Get unsynced sessions from store (or single session if `opts.SessionID` set)
2. For each session: look up `sessionIndex[sessionID]`, fetch annotations, call `SyncSession`
3. On success: `markSynced(ctx, sessionID, count)`
4. Collect results into `SyncReport`

**Types added:** `SyncOptions{DryRun, SessionID}`, `SyncReport{Synced, Skipped, Errors}`, `SyncError{SessionID, Error}`

### Why

The atomic write pattern (temp file + rename) is required for crash safety. If the process is killed mid-write, the original file is untouched. `os.Rename` is atomic on POSIX when source and destination are on the same filesystem. Using `map[string]any` to patch the annotations field avoids re-serializing the entire session (which could lose formatting or comments).

### What worked

- All 5 sync tests pass
- golangci-lint 0 issues
- `TestSyncSessionNilAnnotationsProducesEmptyArray` correctly verifies that `nil` annotations produce `"annotations": []` in JSON (not `null`)

### What didn't work

**errcheck in sync_test.go:** `json.Unmarshal(data, &after)` error was ignored after `os.ReadFile`. Fix: added explicit `if err := json.Unmarshal(...)`. Similarly, `os.Remove(tmpPath)` in the error path of `SyncSession` needed `_ = os.Remove(tmpPath)`.

### What I learned

- `os.Rename` on POSIX is atomic only when source and destination are on the same filesystem. For cross-filesystem renames, Go falls back to copy+delete which is not atomic. This is fine for our use case since `outputDir/annotations.db` and `outputDir/active/.../*.minitrace.json` are on the same filesystem.
- Using `map[string]any` to patch JSON avoids the problem of re-serializing the entire session with `json.Marshal` (which would lose comments, key ordering, and any non-standard fields).

### What was tricky to build

- **Annotations field replacement**: The session file has `annotations []Annotation` as a typed field. Unmarshaling into `map[string]any` and then setting the key loses type information but preserves all other fields. Re-marshaling with `MarshalIndent` produces clean output.
- **Nil annotations → empty array**: `annotations = nil` in the map produces `"annotations": null`. Need to explicitly set `session["annotations"] = []any{}` for nil.

### What warrants a second pair of eyes

- **Cross-filesystem Rename**: If the output directory spans multiple filesystems (unlikely), `os.Rename` is not atomic. Not a concern for typical setups but worth documenting.
- **markSynced failure after successful file write**: In `SyncAll`, if `SyncSession` succeeds but `markSynced` fails, the session is marked as an error but the file is already updated. This means the next sync will re-write the same data — harmless but noisy.

### What should be done in the future

- Consider a combined transaction: write file + mark synced in one step, with rollback on failure
- Consider adding `--diff` flag to show the exact JSON diff before syncing

### Code review instructions

**Start at:** `pkg/annotate/sync.go`

Key symbols to review:
- `SyncSession` — atomic write pattern, nil annotation handling
- `SyncAll` — session index lookup, error aggregation

**Validate:**
```bash
go test -v ./pkg/annotate/ -run Sync
golangci-lint run ./pkg/annotate/...
```

### Technical details

**Atomic write sequence:**
```go
tmpPath := filePath + ".tmp"
os.WriteFile(tmpPath, out, 0644)
os.Rename(tmpPath, filePath)  // atomic on POSIX same filesystem
```

**JSON patching:** Unmarshal to `map[string]any`, set `annotations` key, re-marshal. This preserves all other fields and formatting.

**Commit:** `6c71f31` — "pkg/annotate: atomic JSON sync for .minitrace.json write-back"
---

## Step 3: cmd/annotate — CLI Commands

Committed: `eec4611` — "cmd/annotate: add CLI commands for annotation management"

Created `cmd/go-minitrace/cmds/annotate/` with 6 subcommands wired into `main.go`. All commands share `--output-dir` flag (default `./output`). Each subcommand opens the SQLite store via a shared `openStore(cmd)` helper.

### Prompt Context

**User prompt (verbatim):** "Phase 2..."

**Assistant interpretation:** Implement the CLI commands for annotation management.

**Inferred user intent:** Provide a command-line interface for creating, listing, editing, deleting, syncing, and importing annotations.

### What I did

**Commands:**
- `annotate add` — validates category against known set, uses `uuid.New().String()` for ID, calls `store.AddAnnotation`, prints summary
- `annotate list` — calls `store.List` with all filter options, prints table (tabwriter) or JSON
- `annotate edit` — uses `flagIsSet` helper to detect which flags were passed, builds `AnnotationPatch`, calls `store.Update`
- `annotate delete` — calls `store.Delete`, prints confirmation
- `annotate sync` — infers archive-glob from output-dir (`<output-dir>/active/*/*.minitrace.json`), calls `ExpandArchiveGlobs`, reads each file to extract session IDs, calls `store.SyncAll`, reports synced/skipped/errors
- `annotate import` — reads JSON file/stdin, converts to `[]Annotation`, calls `store.AddAnnotation` for each

**Helpers shared across commands:**
- `openStore(cmd) (*annotate.Store, string, error)` — reads `--output-dir` flag, calls `annotate.Open`
- `closeStore(store)` — `_ = store.Close()`
- `parseCommaList(s string) []string` — splits on comma, trims whitespace
- `flagIsSet(cmd, name) bool` — detects whether a flag was explicitly set vs. using its default

### Why

Using a shared `openStore` helper avoids repeating the flag-reading and error-handling logic in each command. The `flagIsSet` helper is needed because Cobra's `cmd.Flags().GetString()` returns the default value even when the flag was not set — for `edit`, we only want to patch fields the user actually specified.

### What worked

- Smoke test: `go-minitrace annotate add --session sess-test --category observation --title "Test"` → printed summary → `go-minitrace annotate list` showed the annotation
- All lint checks pass (gofmt, errcheck, staticcheck)

### What didn't work

**Unused imports:** `os` in `add.go` (imported but not used), `pkg/annotate` in `import.go` (used via root.go's shared helper, not directly). Fixed with `sed -i` to remove the unused imports.

**MarkFlagRequired errcheck failures:** `cmd.MarkFlagRequired(...)` return values were ignored. Fixed: `_ = cmd.MarkFlagRequired(...)` in `add.go`, `edit.go`, `delete.go`.

### What was tricky to build

**inferring the archive-glob in `sync`:** Since output-dir can be any path, the glob needs to be constructed as `<output-dir>/active/*/*.minitrace.json`. This assumes the standard output directory layout with `active/` subdirectory. If the user uses a non-standard layout, they must specify `--archive-glob` explicitly.

**flagIsSet:** Cobra's `flag.Changed` tracks whether the flag was set explicitly. The helper walks all flags and checks `flag.Name == name && flag.Changed`.

### What warrants a second pair of eyes

- **Non-standard output layouts:** `sync` infers `--archive-glob` as `<output-dir>/active/*/*.minitrace.json`. If the user's output directory structure differs, sync won't find any files. Document this assumption.
- **No validation of taxonomy codes:** `add` accepts any taxonomy string and passes it through. Taxonomy codes are not validated.

### What should be done in the future

- Add `--format table` / `--format json` to all commands consistently (currently only `list` has it)
- Add `annotate get` command to fetch a single annotation by ID
- Consider `annotate stats` to summarize annotation counts by category/session

### Code review instructions

**Start at:** `cmd/go-minitrace/cmds/annotate/root.go`

Key files to review:
- `add.go` — flag validation, category enum check
- `synccmd.go` — session index building, archive-glob inference
- `edit.go` — flagIsSet usage for partial patching

**Validate:**
```bash
go build ./cmd/go-minitrace/
./go-minitrace annotate --help
./go-minitrace annotate add --help
./go-minitrace annotate sync --help
```

### Technical details

**Category validation:** `validCategories` map with known values: `observation`, `ai-failure`, `user-error`, `environment-issue`, `success`, `question`, `to-discuss`, `to-improve`.

**Default output-dir inference:** If `--output-dir` is not set, commands use `./output`. The `sync` command then infers the glob as `./output/active/*/*.minitrace.json`.

**Commit:** `eec4611` — "cmd/annotate: add CLI commands for annotation management"
---

## Step 4: pkg/annotate/duckdb.go — DuckDB sqlite_scanner Attachment

Committed: `4116a58` — "pkg/annotate: DuckDB sqlite_scanner attachment + updated annotations.sql"

Added `pkg/annotate/duckdb.go` and wired it into the serve command. DuckDB's `sqlite_scanner` extension makes SQLite tables directly queryable from DuckDB SQL — no JSON export/import, no refresh needed.

### Prompt Context

**User prompt (verbatim):** "Phase 3: DuckDB Integration..."

**Assistant interpretation:** Wire DuckDB sqlite_scanner attachment into the serve startup.

**Inferred user intent:** Make annotations queryable from DuckDB alongside session data without a separate refresh step.

### What I did

**pkg/annotate/duckdb.go:** `AttachAnnotationsToDuckDB(conn, outputDir)`:
1. `INSTALL sqlite_scanner` + `LOAD sqlite_scanner`
2. Resolve `annotations.db` path (`outputDir/annotations.db`)
3. Check if file exists (skip if not yet created)
4. `CALL sqlite_attach($1, overwrite => true)` — named parameter required

**serve.go wiring:**
- Added `annotate` import
- Added `outputDirFromGlobs()` helper: expands globs, takes first file, returns `filepath.Dir(filepath.Dir(firstFile))`
- Called between `OpenConnection` and `LoadArchive`
- Errors logged as warnings (non-fatal — serve continues without annotations)

**queries/annotations.sql:** updated to `JOIN annotations a ON sb.id = a.session_id` instead of `UNNEST(annotations)`.

### What was tricky to build

**Named parameters in sqlite_attach:** `CALL sqlite_attach('/path', true)` fails with Binder Error. The correct syntax is `CALL sqlite_attach($1, overwrite => true)` — DuckDB requires named parameters for boolean flags.

**Inferring outputDir from globs:** The glob pattern contains wildcards, so `filepath.Dir` on the glob string doesn't work. Instead, `ExpandArchiveGlobs` resolves to actual file paths, then `filepath.Dir(filepath.Dir(file[0]))` gives the output root.

### What I learned

- `sqlite_scanner` is built into DuckDB — no external dependency, no CGO
- Attached SQLite tables land in the `main` schema — query as `SELECT * FROM annotations`, no schema prefix
- Annotations are live — DuckDB re-reads the SQLite file on every query, no refresh needed after writes

### Technical details

**sqlite_scanner call sequence:**
```sql
INSTALL sqlite_scanner;
LOAD sqlite_scanner;
CALL sqlite_attach('/abs/path/to/annotations.db', overwrite => true);
```

Named parameter `overwrite => true` is required — positional arguments cause a Binder Error.

**outputDirFromGlobs:** `ExpandArchiveGlobs` always returns absolute paths. `filepath.Dir(filepath.Dir(absFile))` gives the output root (e.g. `/home/.../output` from `/home/.../output/active/2026-04/sess-001.minitrace.json`).

**Commit:** `4116a58` — "pkg/annotate: DuckDB sqlite_scanner attachment + updated annotations.sql"
---

## Step 5: HTTP API Handlers

Committed: `f155b6e` — "serve: add annotation HTTP API handlers"

Added `cmd/go-minitrace/cmds/serve/handlers_annotations.go` (340 lines) with 6 HTTP handlers, wired into the serve command. The `Store` is opened at serve startup and passed to `NewServer`. Handlers check `s.annoStore == nil` and return 503 if the store is unavailable.

### Prompt Context

**User prompt (verbatim):** "Phase 4: HTTP API..."

**Assistant interpretation:** Implement the HTTP API handlers for annotation CRUD.

**Inferred user intent:** Provide a REST API for annotation management alongside the CLI.

### What I did

**6 HTTP handlers:**
- `GET /api/sessions/{id}/annotations` — calls `store.GetAnnotationsForSession`, returns `{session_id, count, annotations}`
- `POST /api/sessions/{id}/annotations` — validates category/title, builds Annotation, calls `store.AddAnnotation`, returns 201
- `GET /api/annotations` — calls `store.List` with query params (session, scope, category, annotator, taxonomy, limit)
- `PUT /api/annotations/{annId}` — decodes JSON into AnnotationPatch, calls `store.Update`
- `DELETE /api/annotations/{annId}` — calls `store.Delete`, returns 204
- `POST /api/annotations/sync` — calls `store.SyncAll`, returns 200 (or 206 if errors)

**server.go changes:**
- Added `annoStore *annotate.Store` and `annoIndex map[string]string` to `Server` struct
- `NewServer` signature changed: now accepts `(conn, settings, sessionIndex, annoStore, annoIndex)`
- `writeError(w, status, msg)` helper added
- Annotation routes registered in `routes()`
- serve.go opens `annotate.Open(outputDir)` at startup and passes it to `NewServer`

**server_test.go:** Updated all `NewServer` calls to pass `nil, nil` for the new parameters.

### What didn't work

**`NewServer` signature change:** broke 8 test calls in `server_test.go`. Fixed by adding `nil, nil` to all `NewServer` invocations.

**Lint failures:** gofmt formatting issues (field alignment with trailing spaces), staticcheck "empty branch" for the best-effort sync body decode. Fixed: removed `if err != nil {}` block, replaced with `_ = json.NewDecoder(r.Body).Decode(&syncReq)`.

### Technical details

**Handlers check for nil store:** Every handler starts with:
```go
if s.annoStore == nil {
    writeError(w, http.StatusServiceUnavailable, "annotation store not available")
    return
}
```
This ensures serve continues to work even if the annotation store can't be opened.

**Annotation patch from JSON:** Uses type assertions (`if s, ok := v.(string); ok`) to extract typed values from `map[string]any`. Arrays handled similarly with `[]any` → `[]string` conversion.

**Commit:** `f155b6e` — "serve: add annotation HTTP API handlers"
---

## Step 6: Web UI — AnnotationPanel

Committed: `7421127` — "web: add annotation panel and API to React frontend"

Added annotation support to the React frontend using Redux Toolkit Query (RTK Query). The AnnotationPanel is integrated into the TranscriptViewer as a "Transcript/Annotations" tab.

### Prompt Context

**User prompt (verbatim):** "Phase 5: Web UI..."

**Assistant interpretation:** Add the annotation panel to the React frontend.

**Inferred user intent:** Provide a UI for creating, viewing, and deleting annotations directly in the browser.

### What I did

**web/src/types/session.ts:** Added `Annotation`, `AnnotationCategory`, `ANNOTATION_CATEGORY_COLORS`, `SessionAnnotationsResponse`, `SyncReport` types. These mirror the Go HTTP API response shapes.

**web/src/api/minitrace.ts:** Added 5 RTK Query endpoints:
- `useGetSessionAnnotationsQuery` — fetches annotations for a session
- `useCreateAnnotationMutation` — creates annotation, invalidates session's annotation cache
- `useUpdateAnnotationMutation` — patches annotation
- `useDeleteAnnotationMutation` — deletes annotation
- `useSyncAnnotationsMutation` — syncs annotations back to .minitrace.json

**AnnotationPanel.tsx (351 lines):** MUI-based panel with:
- Annotation list rendered as Cards with color-coded category chips
- Add form: category Select, title TextField, detail multiline TextField, tags TextField
- Delete button per annotation
- "Sync to JSON" button in footer
- Loading/error states using RTK Query hooks

**TranscriptViewer.tsx:** Added `Tabs` component with "Transcript" and "Annotations" tabs. When "Annotations" tab is active, renders `<AnnotationPanel>`.

### What didn't work

**TypeScript lint errors:** `Divider` was imported but never used, and `ANNOTATION_CATEGORY_COLORS` was imported twice (once as type import, once as value). Fixed by removing the unused `Divider` import and consolidating to the single `CATEGORY_COLORS` alias.

### Technical details

**RTK Query invalidation:** `createAnnotation` and `deleteAnnotation` invalidate the `Annotations` tag for the specific session ID, causing `useGetSessionAnnotationsQuery` to re-fetch automatically when the cache is invalidated.

**MUI color mapping:** `CATEGORY_COLORS` maps each category to a MUI `Chip` color variant (`error`, `warning`, `success`, `info`, etc.).

**npm build:** `tsc -b && vite build` passes (685 modules, 962KB bundle).

**Commit:** `7421127` — "web: add annotation panel and API to React frontend"

---

## Step 7: E2E tests + Phase 6 polish (validate + README)

### What I did

**E2E test scripts (3):**
- `08-e2e-annotate-cli.sh` — CLI smoke test: add → sqlite3 verify → sync → validate → list → edit → delete → validate JSON
- `09-e2e-duckdb-sqlite-live.sh` — DuckDB sqlite_scanner live query: starts serve in background, adds annotation via CLI, queries DuckDB with `sqlite_attach` via `-set` flag to avoid heredoc variable escaping
- `10-e2e-api.sh` — HTTP API E2E: POST/GET/PUT/DELETE/sync all annotation endpoints via curl

**Bug fix: extractPathParam index OOB** (`handlers_annotations.go`): `annId` case used `parts[2]` but after `strings.TrimPrefix(r.URL.Path, "/api/")`, URL `/api/annotations/{id}` splits into `["annotations", UUID]` (2 parts), so `parts[2]` is out of range. Fixed to `parts[1]` for both `id` and `annId` cases.

**Bug fix: shell variable leakage in 10-e2e-api.sh**: `get()` function returned `RESP` (body only) but did NOT capture `HTTP_CODE`. DELETE step set `HTTP_CODE=204`, then GET-after-delete step called `get()` which didn't update `HTTP_CODE`, so it retained "204" and `[ "$HTTP_CODE" = "200" ]` failed. Rewrote with `_BODY` and `_CODE` global variables and `capture()` helper.

**pkg/validate extension** (`json.go`): Extended `ValidatePath()` to validate annotation arrays:
- `validateAnnotations(field)` — checks null, array type, iterates items
- `validateAnnotation(ann)` — id, timestamp, annotator, scope.type (session|turn|tool_call), scope.target_id, content.category (known set), content.title, content.tags (array of strings), taxonomy_mappings.* (array of strings), classification (known levels)
- Bug: tags were read from `ann["tags"]` (top-level) but should be `content["tags"]` per minitrace schema. Fixed.

**README additions**: 104-line Annotations section covering storage model, CLI commands, categories, serve + HTTP API table, DuckDB queries.

### What didn't work

**gofmt field alignment**: `ValidAnnotationCategories` map had misaligned columns (`observation:` with extra space). Fixed with `gofmt -w`.

**sed substitution bug**: Attempted `sed -i 's/contains(...)/strContains(...)/g'` but sed substituted `result.Error` → `results[0].Error` in all occurrences. Fixed by reading the file and manually rewriting.

**test file `t` parameter**: `strContains` and `strContainsIndex` helper functions used `t` as parameter name, shadowing the outer `*testing.T`. Renamed to `strContains` using `strings.Contains` instead.

### Technical details

**DuckDB `-set` flag for SQLite path**: `duckdb -set db_path /tmp/annotations.db "INSTALL sqlite_scanner; LOAD sqlite_scanner; CALL sqlite_attach($db_path, overwrite => true); SELECT COUNT(*) FROM annotations;"` — named parameter avoids positional-boolean Binder Error.

**ValidClassificationLevels**: `[]string{"public", "internal", "confidential", "customer-confidential"}` — ordered for potential de-escalation enforcement in future.

**gofmt output for map literals**: When all values are `true`, gofmt aligns the RHS at column 12. `"observation"` is 11 chars, so one space before `true`.

**Commit graph:** `238aba7` → `6c71f31` → `eec4611` → `4116a58` → `f155b6e` → `7421127` → `2e15017` → `82017db` → `b663b03` → `5430a20` → `bb3fcc5`

**New commits:**
- `b663b03` fix(serve): extractPathParam parts[2]→parts[1] (index OOB), gofmt fix
- `5430a20` pkg/validate: annotate annotation structure in go-minitrace validate
- `bb3fcc5` README: add Annotations section covering CLI, HTTP API, storage model, DuckDB queries
- `2e15017` serve: open annotate.Store at startup, pass to NewServer
- `82017db` diary + tasks: Step 7 complete
