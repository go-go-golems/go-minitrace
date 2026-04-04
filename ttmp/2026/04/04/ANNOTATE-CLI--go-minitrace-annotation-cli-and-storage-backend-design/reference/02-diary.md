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
