---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: cmd/go-minitrace/cmds/serve/handlers_sessions.go
      Note: Existing session handlers where annotation CRUD endpoints will be added
    - Path: cmd/go-minitrace/cmds/serve/serve.go
      Note: Serve command where SQLite annotation store will be initialized
    - Path: pkg/minitrace/schema.go
      Note: Go Annotation struct that SQLite schema must match
    - Path: pkg/query/engine.go
      Note: DuckDB engine that needs annotations_flat table integration
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Annotation Storage Backend and CLI Design Decision

## Executive Summary

**Recommendation: Parallel SQLite annotation overlay + JSON writeback CLI**

Store annotations in a dedicated SQLite database alongside the minitrace JSON files. The SQLite DB acts as a fast, queryable, writable overlay. A CLI command merges annotations back into the JSON files on demand. DuckDB reads the merged view for analysis queries.

---

## Problem Statement

We need to add, edit, delete, and query annotations on minitrace sessions after import. Currently:
- Annotations exist as an `annotations[]` array inside each `.minitrace.json` file
- No CLI or UI exists for creating annotations
- DuckDB loads JSON files into an in-memory temp table (read-only)
- The web viewer and CLI have no annotation CRUD capabilities

The core tension: **DuckDB is excellent at reading JSON but poor at writing it back**. We need both query performance and write capability.

---

## Alternatives Considered

### Option A: Edit JSON files directly (Go JSON read-modify-write)

**How it works:** The CLI reads a `.minitrace.json`, parses it into Go structs, appends/modifies the `annotations[]` array, marshals back to JSON, and overwrites the file.

**Pros:**
- Single source of truth — annotations live in the spec-defined location
- No extra storage format to manage
- Files remain self-contained and portable
- Validation is straightforward (just run the existing validator)

**Cons:**
- Slow for bulk operations (parse/serialize multi-MB JSON files per annotation)
- No transactional safety — crash mid-write corrupts the file
- Concurrent writes require file locking
- No indexing — "find all sessions with F-AUT annotations" requires scanning every file
- DuckDB must reload to see new annotations (it loads into a TEMP TABLE at startup)
- Web UI would need to go through the filesystem layer for every write

**Verdict:** Good for the write-back step, but too slow and fragile as the primary annotation store.

### Option B: Store annotations in DuckDB, export to JSON on demand

**How it works:** DuckDB can create persistent tables (`CREATE TABLE` not `CREATE TEMP TABLE`). We could store annotations in a DuckDB table and use `COPY TO` to write JSON back.

**DuckDB write capabilities tested:**

| Operation | Works? | Notes |
|-----------|--------|-------|
| `CREATE TABLE` (persistent) | ✅ | `.duckdb` file on disk |
| `INSERT/UPDATE/DELETE` | ✅ | Full DML support |
| `COPY TO ... FORMAT JSON` | ✅ | Writes entire result set as JSON |
| `COPY TO ... FORMAT JSON, ARRAY false` | ✅ | NDJSON (one JSON object per line) |
| Update JSON column with `list_append` | ✅ | `list_append(col::JSON[], '...'::JSON)::JSON` |
| Write individual files per row | ❌ | No built-in row-level file export |
| `json_insert` / `json_modify` | ❌ | Not available in DuckDB v1.5 |
| `json_merge_patch` | ✅ | But only does object merge, not array append |

**Pros:**
- Queries can join annotations with session data in one engine
- DuckDB is already integrated
- Good bulk analysis performance

**Cons:**
- DuckDB is not designed for transactional OLTP workloads (single-row inserts are slow)
- DuckDB's concurrency model is single-writer — the serve command holds an open connection, blocking writes from other processes
- `duckdb-go` has `SetMaxOpenConns(1)` in the current codebase — explicitly single-connection
- No way to write individual `.minitrace.json` files from DuckDB — you'd need Go code anyway
- Persistent DuckDB files are not human-readable or easily debuggable
- If the DuckDB file corrupts, you lose all annotations

**Verdict:** DuckDB is the wrong engine for annotation writes. It's an analytical engine, not a transactional store.

### Option C: Parallel SQLite annotation database (RECOMMENDED)

**How it works:** A separate SQLite database (e.g., `annotations.db`) stores all annotations in a relational schema. The minitrace JSON files are never modified during normal annotation workflows. A separate `sync`/`export` command merges annotations back into the JSON files when needed.

**Pros:**
- SQLite is the right tool for OLTP: fast single-row inserts, ACID transactions, concurrent readers
- Annotations are queryable alongside session data (DuckDB can attach SQLite or load from both)
- The `.minitrace.json` files remain immutable authoritative sources; annotations are an overlay
- Multiple processes can read the SQLite DB simultaneously
- Crash-safe (SQLite's WAL mode)
- Human-inspectable with `sqlite3` CLI
- The `go-minitrace serve` web server can write to SQLite without conflicting with DuckDB reads
- Annotations can exist before, during, or after analysis sessions
- Easy to implement annotation versioning / audit trail in SQLite

**Cons:**
- Two storage formats to manage (SQLite + JSON)
- Need a sync step to make annotations visible in the JSON files
- Slightly more complex CLI surface
- DuckDB queries need to be aware of both sources

**Verdict:** Best balance of write performance, query capability, and safety. The sync step is a feature, not a bug — it gives explicit control over when the canonical JSON files are modified.

### Option D: Sidecar JSON files (one `.annotations.json` per session)

**How it works:** Each session gets a companion file: `019d4a02-6bb9-7921-8a2e-224b071acd0f.annotations.json` alongside the `.minitrace.json`.

**Pros:**
- Human-readable
- No database dependency
- Easy to version control
- Can be loaded alongside the session

**Cons:**
- No indexing — cross-session queries require scanning all sidecar files
- File management overhead (hundreds of tiny files)
- No transactional safety across files
- Harder to implement in the web UI (need file I/O per annotation operation)
- Duplicates the session ID in both the annotation file and the session file

**Verdict:** Decent for small collections, but doesn't scale to hundreds of sessions. Doesn't solve the queryability problem.

---

## Detailed Design: Option C (Parallel SQLite)

### SQLite schema

```sql
CREATE TABLE IF NOT EXISTS annotations (
    id TEXT PRIMARY KEY,                 -- UUID
    session_id TEXT NOT NULL,            -- FK to minitrace session
    annotator TEXT NOT NULL,             -- 'user', 'model', 'automated'
    
    -- Scope
    scope_type TEXT NOT NULL,            -- 'session', 'turn', 'tool_call', 'handover'
    target_id TEXT NOT NULL,             -- what it refers to
    
    -- Content
    category TEXT NOT NULL,              -- 'observation', 'pattern', 'ai-failure', 'recommendation'
    title TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    tags TEXT NOT NULL DEFAULT '[]',     -- JSON array of strings
    
    -- Taxonomy
    taxonomy_minitrace TEXT DEFAULT '[]', -- JSON array
    taxonomy_mast TEXT DEFAULT '[]',      -- JSON array  
    taxonomy_toolemu TEXT DEFAULT '[]',   -- JSON array
    
    -- Classification override
    classification TEXT,                 -- NULL = no override
    
    -- Metadata
    created_at TEXT NOT NULL,            -- ISO 8601 UTC
    updated_at TEXT NOT NULL,
    
    FOREIGN KEY (session_id) REFERENCES sessions(session_id)
);

CREATE INDEX idx_annotations_session ON annotations(session_id);
CREATE INDEX idx_annotations_scope ON annotations(scope_type, target_id);
CREATE INDEX idx_annotations_category ON annotations(category);
CREATE INDEX idx_annotations_annotator ON annotations(annotator);

-- Track which sessions have been synced
CREATE TABLE IF NOT EXISTS sync_state (
    session_id TEXT PRIMARY KEY,
    last_synced_at TEXT,
    annotation_count INTEGER DEFAULT 0
);

-- Session registry (populated from manifest/JSON scan)
CREATE TABLE IF NOT EXISTS sessions (
    session_id TEXT PRIMARY KEY,
    file_path TEXT NOT NULL,             -- absolute path to .minitrace.json
    title TEXT,
    framework TEXT,
    model TEXT,
    loaded_at TEXT
);
```

### CLI commands

```
# Add an annotation
go-minitrace annotate add \
  --session 019d4a02-6bb9-7921-8a2e-224b071acd0f \
  --scope session \
  --category ai-failure \
  --title "Over-autonomy at turn 15" \
  --detail "The agent edited files without being asked" \
  --tags "F-AUT,C-PHA" \
  --taxonomy-minitrace "F-AUT" \
  --annotator user

# Add annotation targeting a specific turn
go-minitrace annotate add \
  --session 019d4a02-... \
  --scope turn \
  --target-id "15" \
  --category observation \
  --title "Agent started editing unprompted"

# Add annotation targeting a specific tool call
go-minitrace annotate add \
  --session 019d4a02-... \
  --scope tool_call \
  --target-id "tc_abc123" \
  --category ai-failure \
  --title "Write without read" \
  --tags "F-ASM,C-SEQ"

# List annotations for a session
go-minitrace annotate list --session 019d4a02-...

# List all F-AUT failures across sessions
go-minitrace annotate list --taxonomy F-AUT

# Delete an annotation
go-minitrace annotate delete --id ann-uuid

# Edit an annotation
go-minitrace annotate edit --id ann-uuid --title "Updated title" --detail "Updated detail"

# Interactive annotation (TUI / prompts)
go-minitrace annotate interactive --session 019d4a02-...

# Sync annotations back to JSON files
go-minitrace annotate sync [--session SESSION_ID] [--dry-run]

# Import annotations from a JSON file (batch)
go-minitrace annotate import --file annotations.json

# Export annotations (standalone, without session data)
go-minitrace annotate export [--format json|csv] [--session SESSION_ID]
```

### DuckDB integration: merged view

The serve command and query engine load annotations from SQLite into DuckDB, creating a unified query surface:

```sql
-- In the DuckDB session, after loading sessions_base:
-- Install and load the SQLite extension (or use read_json on exported annotations)
-- Option 1: Attach SQLite directly
INSTALL sqlite; LOAD sqlite;
CALL sqlite_attach('annotations.db', 'anno', READ_ONLY);

-- Option 2: Load annotations as a JSON temp table (simpler, no SQLite extension needed)
CREATE OR REPLACE TEMP TABLE annotations_flat AS
SELECT * FROM read_json('annotations-export.json');

-- Option 3 (recommended): Go code loads from SQLite, inserts into DuckDB
-- This avoids the SQLite extension dependency
```

**Recommended approach for go-minitrace:** Go code reads annotations from SQLite at server startup (and on refresh), and creates a `annotations_flat` temp table in DuckDB alongside `sessions_base`. The existing `annotations.sql` query then works unchanged with a minor modification to join against this table instead of unnesting from the JSON column.

### Sync mechanism (write-back to JSON)

The `annotate sync` command:
1. Reads annotations from SQLite for each session
2. Loads the `.minitrace.json` file into Go structs
3. Replaces the `annotations[]` array with the current SQLite state
4. Writes the file back (atomic: write to `.tmp`, then rename)
5. Updates `sync_state` in SQLite

This is an explicit, user-controlled operation. The JSON files remain the portable interchange format; SQLite is the working store.

### Web UI integration

The serve command already has an HTTP API. Add these endpoints:

```
GET    /api/sessions/:id/annotations          — List annotations for a session
POST   /api/sessions/:id/annotations          — Create annotation
PUT    /api/annotations/:annId                — Update annotation
DELETE /api/annotations/:annId                — Delete annotation
GET    /api/annotations?taxonomy=F-AUT        — Cross-session annotation search
POST   /api/annotations/sync                  — Trigger sync to JSON
```

The web UI TranscriptViewer component gets an annotation panel:
- Click on a turn or tool call → "Add annotation" button
- Sidebar shows existing annotations with taxonomy badges
- Filter by category/failure code across sessions

### File placement

```
output/
├── annotations.db                         ← SQLite annotation store (new)
├── active/
│   └── 2026-03/
│       ├── manifest.json
│       └── *.minitrace.json               ← Unchanged until sync
└── archive/
    └── ...
```

The `annotations.db` lives at the output root. It's created on first `annotate add` if it doesn't exist.

---

## Why Not DuckDB Writes?

To be explicit about the DuckDB write question:

1. **DuckDB can write to persistent tables** — yes, it supports `CREATE TABLE` and `INSERT/UPDATE/DELETE` on disk-backed `.duckdb` files.

2. **DuckDB can export JSON** — `COPY (SELECT ...) TO 'file.json' (FORMAT JSON)` works.

3. **But it can't write individual minitrace files** — DuckDB writes one file per `COPY` command. There's no way to say "update the annotations array in `/path/to/session-uuid.minitrace.json`". You'd have to:
   - Load all sessions into a DuckDB table
   - Modify the annotations column
   - Export the entire table back to NDJSON
   - Split the NDJSON back into individual files
   This is fragile, loses formatting, and is architecturally wrong.

4. **Concurrency** — DuckDB is single-writer by design. The `serve` command already holds a connection. You'd need to close/reopen the DB for every write, or use a persistent file with WAL mode — but that still blocks concurrent writers.

5. **JSON manipulation is limited** — DuckDB has no `json_insert` or array-append function for JSON columns. `list_append` works on DuckDB-native lists but is awkward for nested JSON. The annotation schema has nested objects (`scope`, `content`, `taxonomy_mappings`) that are painful to construct in SQL.

**Bottom line:** DuckDB is for reading and analyzing. SQLite is for writing and tracking. Go code is for manipulating the JSON files.

---

## Implementation Plan

### Phase 1: SQLite annotation store + CLI (core)

1. Create `pkg/annotate/store.go` — SQLite-backed annotation store
   - `Open(dbPath string) (*Store, error)`
   - `AddAnnotation(ctx, sessionID, annotation) error`
   - `GetAnnotations(ctx, sessionID) ([]Annotation, error)`
   - `ListAnnotations(ctx, filter) ([]Annotation, error)`
   - `UpdateAnnotation(ctx, annID, patch) error`
   - `DeleteAnnotation(ctx, annID) error`
   - `GetUnsynced(ctx) ([]SessionAnnotations, error)`
   - `MarkSynced(ctx, sessionID) error`

2. Create `cmd/go-minitrace/cmds/annotate/` — CLI commands
   - `add.go`, `list.go`, `edit.go`, `delete.go`, `sync.go`
   - Wire into main.go

3. Create `pkg/annotate/sync.go` — JSON write-back
   - Read `.minitrace.json`, replace `annotations[]`, write atomically

### Phase 2: DuckDB integration

4. Update `pkg/query/engine.go` — load annotations into DuckDB
   - Read from SQLite, create `annotations_flat` temp table
   - Update `annotations.sql` to join against it

5. Update serve command — annotation API endpoints
   - CRUD handlers backed by SQLite store
   - Refresh DuckDB annotations table on mutation

### Phase 3: Web UI

6. Add annotation panel to TranscriptViewer
7. Add cross-session annotation search to QueryEditor
8. Add annotation badges/indicators to SessionBrowser

### Phase 4: Advanced features

9. Interactive annotation mode (TUI prompts for taxonomy selection)
10. Batch import/export (JSON, CSV)
11. Annotation templates (predefined failure patterns)
12. Annotation conflict detection (multiple annotators on same scope)

---

## Open Questions

1. **Should annotations.db be colocated with the output dir or in a user config dir?** Colocated is simpler and keeps annotations with the data. But multiple output dirs need multiple annotation DBs or a central one. Recommend: colocated, one per output root.

2. **Should sync be automatic or manual?** Manual (explicit command). Auto-sync adds complexity and risks unintended file modifications. The SQLite DB is the working truth; JSON files are the interchange format.

3. **How to handle annotation IDs?** UUIDs, generated by the CLI. The minitrace spec says "opaque strings, UUIDs recommended."

4. **Should the web UI write directly to SQLite during serve?** Yes. The serve command already has the SQLite path available. Annotations written via the API are immediately queryable (refresh the DuckDB temp table).

5. **What about annotations that reference sessions not yet loaded?** The `sessions` table in SQLite is populated lazily. You can annotate any session ID; it doesn't need to exist in DuckDB's current view.

6. **Migration from existing annotations in JSON?** An `annotate import` command that reads the `annotations[]` array from JSON files and inserts them into SQLite. This allows round-tripping.
