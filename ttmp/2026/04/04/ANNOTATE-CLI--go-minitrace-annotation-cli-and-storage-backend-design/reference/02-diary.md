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
    - Path: pkg/annotate/store.go
      Note: Store struct
    - Path: pkg/annotate/store_test.go
      Note: 11 tests covering all public API (commit 238aba7)
ExternalSources: []
Summary: 'Step-by-step diary of implementing the annotation CLI. Phase 1: SQLite store. Phase 2: sync + CLI. Phase 3: DuckDB integration. Phase 4: HTTP API. Phase 5: web UI.'
LastUpdated: 2026-04-04T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Diary — ANNOTATE-CLI Implementation

## Goal

Implementing the annotation CLI and SQLite storage backend for go-minitrace:
1. `pkg/annotate/store.go` — SQLite CRUD
2. `pkg/annotate/sync.go` — atomic JSON write-back
3. `pkg/annotate/duckdb.go` — DuckDB sqlite_scanner attachment
4. `cmd/go-minitrace/cmds/annotate/` — CLI commands (add, list, edit, delete, sync, import)
5. Serve HTTP endpoints
6. Web UI AnnotationPanel

## Step 1: pkg/annotate/store.go — SQLite CRUD

Committed: `238aba7` — "pkg/annotate: SQLite-backed annotation store"

### Prompt Context

**User prompt (verbatim):** "Phase 1 — pkg/annotate/store.go: implement the SQLite-backed annotation store... Start working through the implementation step by step."

**Assistant interpretation:** Implement the SQLite store package with CRUD operations for annotations, wired to the minitrace schema types.

**Inferred user intent:** Build the foundation of the annotation system: a SQLite store that can persist and query annotations, with a sync_state table to track which sessions need their .minitrace.json files updated.

### What I did

Created `pkg/annotate/store.go` (502 lines) and `pkg/annotate/store_test.go` (384 lines, 11 tests).

Files created:
- `pkg/annotate/store.go` — Store struct, Open/Close, migrate, AddAnnotation, GetAnnotationsForSession, List, Update, Delete, GetUnsyncedSessions, markUnsynced, markSynced
- `pkg/annotate/store_test.go` — 11 tests covering all public API surfaces

Schema created by `migrate()`:
- `annotations` table: id, session_id, annotator, scope_type, target_id, category, title, detail, tags (JSON), taxonomy_m (JSON), taxonomy_mast (JSON), taxonomy_tm (JSON), classification, created_at, updated_at
- `sync_state` table: session_id (PK), synced_at, change_count
- Indexes on session_id, (scope_type, target_id), category, annotator

Key implementation decisions:
- `Open(ctx, outputDir)` resolves to absolute path, creates dir, opens SQLite with `?_journal_mode=WAL&_busy_timeout=5000`
- Tags/taxonomy stored as JSON strings in TEXT columns
- `parseJSONArray` returns `[]string{}` for `"[]"` to avoid nil-slice problem
- `buildPatchSET` dynamically builds SET clause from non-nil AnnotationPatch fields
- `markUnsynced` uses separate UPDATE then INSERT (SQLite limitation)
- `closeRows` helper silences errcheck linter

### What worked

- All 11 tests pass
- golangci-lint reports 0 issues on `./pkg/annotate/...`
- Both `mattn/go-sqlite3` and `google/uuid` already in `go.mod` — no new dependencies needed
- gofmt formatting stable after fix

### What didn't work

**Bug 1: SQLite VALUES column restriction** — First attempt at `markUnsynced` used:

```sql
INSERT INTO sync_state (session_id, synced_at, change_count)
VALUES (?, '', change_count + 1)
ON CONFLICT(session_id) DO UPDATE SET change_count = change_count + 1
```

Runtime error: `no such column: change_count` — SQLite does not support referencing a column in the VALUES clause of an INSERT that also has an ON CONFLICT clause.

Fix: separate UPDATE then conditional INSERT:

```go
result, err := s.db.ExecContext(ctx, `UPDATE sync_state SET change_count = change_count + 1 WHERE session_id = ?`, sessionID)
rowsAffected, _ := result.RowsAffected()
if rowsAffected == 0 {
    _, err = s.db.ExecContext(ctx, `INSERT INTO sync_state (session_id, synced_at, change_count) VALUES (?, '', 1)`, sessionID)
}
```

**Bug 2: json.Unmarshal + nil slice** — `var out []string` creates nil slice. `json.Unmarshal([]byte("[]"), &out)` appends nothing for empty array. Result: nil, not `[]string{}`.

Test `TestNilTagsAndTaxonomy` caught this by asserting `got[0].Content.Tags == nil` is false (expecting non-nil).

Fix: `parseJSONArray` returns `[]string{}` for both `""`, `"null"`, and `"[]"`:

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

**Lint: errcheck failures** — `defer rows.Close()` and `defer store.Close()` flagged by errcheck linter.

Fix: helper functions that discard the error:

```go
func closeRows(rows *sql.Rows) { _ = rows.Close() }
func closeStore(s *Store)     { _ = s.Close() }
```

### What was tricky to build

- The SQLite ON CONFLICT VALUES limitation is not well-documented. The error message ("no such column") is misleading — it suggests the column doesn't exist rather than revealing the SQLite restriction.
- `errcheck` flags deferred Close calls but not explicit ones. I initially tried adding `defer func() { _ = rows.Close() }()` pattern but `closeRows` helper is cleaner.
- `buildPatchSET` needs to handle all optional fields without generating trailing commas or empty SET clauses.

### What should be done in the future

- Consider adding a `GetAnnotationByID` method (single-row lookup by ID, not just by session)
- Consider adding pagination metadata to `List` response (total count for pagination)
- The sync_state design (separate UPDATE then INSERT) could be a single statement with a CTE or a transaction

### Code review instructions

**Start at:** `pkg/annotate/store.go`

Key symbols to review:
- `Open()` — confirms dir creation, WAL mode, migration order
- `migrate()` — confirms schema correctness (column names, indexes)
- `AddAnnotation()` — confirms JSON encoding of tags/taxonomy
- `GetAnnotationsForSession()` — confirms `scanAnnotations` maps all fields correctly
- `Update()` — confirms dynamic SET clause generation and ErrNotFound handling
- `markUnsynced()` — confirms UPDATE-then-INSERT pattern for SQLite compatibility

**Validate with:**
```bash
go test -v ./pkg/annotate/
golangci-lint run ./pkg/annotate/...
```
