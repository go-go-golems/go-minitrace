---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: cmd/go-minitrace/cmds/common/build.go
      Note: BuildCobraCommand pattern used by every new subcommand
    - Path: cmd/go-minitrace/cmds/serve/server.go
      Note: Server struct
    - Path: cmd/go-minitrace/main.go
      Note: Command registration point — where annotate.NewCommand() is wired in
    - Path: pkg/minitrace/builders.go
      Note: BuildAnnotation builder function used as pattern for the store
    - Path: pkg/minitrace/schema.go
      Note: Reference for Annotation
    - Path: pkg/minitrace/util.go
      Note: FormatTimestamp
    - Path: pkg/query/engine.go
      Note: DuckDB connection
    - Path: ttmp/2026/04/04/ANNOTATE-CLI--go-minitrace-annotation-cli-and-storage-backend-design/scripts/07-sqlite-attach-working-join.sql
      Note: Working end-to-end test confirming sqlite_attach works correctly
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---



# go-minitrace Annotation CLI — Implementation Guide

**Ticket:** `ANNOTATE-CLI`
**Date:** 2026-04-04
**Approach:** Parallel SQLite annotation overlay + JSON writeback CLI

---

## Executive Summary

This guide walks through implementing a complete annotation system for go-minitrace — the Go port of the minitrace session trace format. The system allows users to annotate sessions, turns, tool calls, and handover documents after importing trace data. Annotations are stored in a SQLite database (the working store) and can be optionally synced back into the canonical `.minitrace.json` files.

The core insight driving this design: **DuckDB is the right tool for reading and analyzing annotations; SQLite is the right tool for writing and persisting them.** We use each engine for what it does best.

---

## Part 1 — Understanding the Existing System

Before writing any code, you need to understand the four parts of go-minitrace that this implementation will touch.

### 1.1 The Package Structure

go-minitrace lives in `~/code/wesen/corporate-headquarters/go-minitrace/`. The relevant directories:

```
go-minitrace/
├── pkg/
│   ├── minitrace/
│   │   ├── schema.go      # All Go struct definitions for minitrace types
│   │   ├── builders.go    # Builder helpers (BuildSessionSkeleton, BuildTurn, etc.)
│   │   ├── util.go        # Utilities (FormatTimestamp, ParseTimestamp, TruncateContent)
│   │   └── metrics.go     # Metrics computation
│   └── query/
│       └── engine.go      # DuckDB query engine (load, run, normalize)
├── cmd/
│   └── go-minitrace/
│       ├── main.go        # Root command, wires all subcommands via Cobra
│       └── cmds/
│           ├── common/build.go    # Shared Cobra command builder helper
│           ├── serve/            # Web server (handlers, server, blocks, badges)
│           ├── query/            # DuckDB query CLI (root + duckdb subcommand)
│           ├── validate/         # JSON validator
│           ├── convert/          # Session converters (claude_code, codex, pi, etc.)
│           └── discover/         # Format discovery commands
└── queries/
    ├── load.sql                   # DuckDB table loader
    ├── annotations.sql             # Unnest annotations from JSON
    ├── framework-summary.sql       # Framework stats
    ├── timing-analysis.sql
    └── tool-operation-breakdown.sql
```

### 1.2 The Schema (pkg/minitrace/schema.go)

This file defines every type in the minitrace format as Go structs with JSON struct tags. The key types for annotations:

```go
// pkg/minitrace/schema.go (lines 1-22)

type Session struct {
    ID             string
    SchemaVersion  string
    Profile        string
    ScenarioID     *string
    Quality        *string
    Title          *string
    // ... 16 more fields ...
    Annotations    []Annotation   // <-- this is where we write annotations
    Metrics        Metrics
}

type Annotation struct {
    ID               string
    Timestamp        string              // ISO 8601 UTC
    Annotator        string              // "user" | "model" | "automated"
    Scope            AnnotationScope
    Content          AnnotationContent
    TaxonomyMappings TaxonomyMappings
    Classification   *string             // override session classification
}

type AnnotationScope struct {
    Type     string  // "session" | "turn" | "tool_call" | "handover"
    TargetID string  // the ID of the thing being annotated
}

type AnnotationContent struct {
    Category string   // "observation" | "pattern" | "ai-failure" | "recommendation"
    Tags     []string // arbitrary taxonomy labels
    Title    string
    Detail   string
}

type TaxonomyMappings struct {
    Minitrace []string  // e.g. ["F-AUT", "C-SEQ"]
    Mast      []string  // e.g. ["FC2", "2.2"]
    Toolemu   []string  // e.g. ["T2"]
}
```

The full struct definitions run from line 1 through approximately line 220. Every field is accounted for.

### 1.3 The Builder Helpers (pkg/minitrace/builders.go)

Rather than constructing structs directly, the codebase uses builder functions. There's already a `BuildAnnotation` helper:

```go
// pkg/minitrace/builders.go (near line 130)

func BuildAnnotation(
    annotationID string,
    annotator string,
    scopeType string,
    targetID string,
    category string,
    title string,
    detail string,
    tags []string,
    taxonomyMappings *TaxonomyMappings,
) Annotation {
    // ... builds and returns Annotation struct
}
```

This pattern is used throughout the codebase — for `BuildSessionSkeleton`, `BuildTurn`, `BuildToolCall`. You'll follow the same pattern for the annotation store.

### 1.4 The DuckDB Query Engine (pkg/query/engine.go)

The query engine handles all DuckDB interactions:

```go
// pkg/query/engine.go — key functions

// OpenConnection creates a DuckDB connection (single-connection by design)
func OpenConnection(ctx context.Context, dbPath string) (*sql.DB, *sql.Conn, error) {
    db, err := sql.Open("duckdb", dbPath)
    db.SetMaxOpenConns(1)   // <-- single writer
    db.SetMaxIdleConns(1)
    db.SetConnMaxLifetime(0)
    // ...
}

// LoadArchive loads JSON files into a DuckDB table
func LoadArchive(ctx context.Context, conn *sql.Conn, opts LoadOptions) error {
    // Runs: CREATE TEMP TABLE sessions_base AS SELECT * FROM read_json(...)
    // Columns match the minitrace schema:
    //   id, title, summary, classification, profile, provenance (JSON),
    //   flags (JSON), environment (JSON), operational_context (JSON),
    //   timing (JSON), turns (JSON[]), tool_calls (JSON[]),
    //   annotations (JSON[]), metrics (JSON)
    // ...
}

// RunIntoProcessor executes a query and streams rows into a Glazed processor
func RunIntoProcessor(ctx context.Context, conn *sql.Conn, sqlText string, gp Processor) error {
    // Used by the query command to pipe results into table/formats
}

// ResolvePresetSQL substitutes the table name into a named query
func ResolvePresetSQL(presetName string, tableName string) (string, error)
```

The `load.sql` at the root creates this table from JSON files:

```sql
-- queries/load.sql

CREATE OR REPLACE TEMP TABLE sessions_base AS
SELECT *
FROM read_json(
  './output/active/*/*.minitrace.json',
  columns = {
    id: 'VARCHAR',
    annotations: 'JSON[]',   -- <-- annotations are loaded as JSON array
    // ... other columns ...
  },
  ignore_errors = true
);
```

And `annotations.sql` unnests them for querying:

```sql
-- queries/annotations.sql

SELECT
  id AS session_id,
  environment->>'agent_framework' AS framework,
  json_extract(ann, '$.annotator') AS annotator,
  json_extract(ann, '$.content.category') AS category,
  json_extract(ann, '$.content.title') AS title,
  json_extract(ann, '$.scope.type') AS scope_type
FROM sessions_base,
UNNEST(annotations) AS a(ann)
ORDER BY session_id;
```

### 1.5 The Serve Command (cmd/go-minitrace/cmds/serve/)

The `serve` command is a long-running HTTP server. Key architecture:

```go
// cmd/go-minitrace/cmds/serve/server.go

type Server struct {
    conn         *sql.Conn      // DuckDB connection (read-only)
    tableName    string         // "sessions_base"
    presetDirs   []string       // read-only query preset directories
    queryDirs    []string       // user-saved query directories
    sessionIndex map[string]string  // session_id -> file path
    devMode      bool
    mux          *http.ServeMux
}

func NewServer(conn *sql.Conn, settings *ServeSettings, sessionIndex map[string]string) *Server {
    s := &Server{/* ... */}
    s.mux = http.NewServeMux()
    s.routes()  // registers all HTTP handlers
    return s
}

func (s *Server) routes() {
    s.mux.HandleFunc("GET /api/sessions",          s.handleGetSessions)
    s.mux.HandleFunc("GET /api/sessions/{id}",     s.handleGetSession)
    s.mux.HandleFunc("GET /api/sessions/{id}/blocks", s.handleGetSessionBlocks)
    s.mux.HandleFunc("POST /api/query",            s.handleExecuteQuery)
    // ... query save/load/delete endpoints ...
}
```

Notice that `serve` holds the DuckDB connection open for the lifetime of the process. It is a read-only connection. Writes to DuckDB would require closing and reopening — which is precisely why we don't use DuckDB for writes.

### 1.6 The Command Wiring (cmd/go-minitrace/main.go)

New subcommands are registered in `main.go`:

```go
// cmd/go-minitrace/main.go

func main() {
    rootCmd := &cobra.Command{/* ... */}

    discoverCmd, err := discover.NewCommand()
    convertCmd, err := convert.NewCommand()
    queryCmd, err := query.NewCommand()
    serveCmd, err := serve.NewCommand()
    validateCommand, err := validatecmd.NewCommand()

    rootCmd.AddCommand(
        discoverCmd,
        convertCmd,
        queryCmd,
        serveCmd,
        validateCommand,
        // ← new annotate command goes here
    )
}
```

Each command follows the same pattern: a `NewCommand()` function returns a `*cobra.Command`, which is added to the root.

### 1.7 The Command Building Pattern (cmd/go-minitrace/cmds/common/build.go)

Every subcommand uses `common.BuildCobraCommand` to wire a Glazed command description into a Cobra command:

```go
// cmd/go-minitrace/cmds/common/build.go

func BuildCobraCommand(command cmds.Command) (*cobra.Command, error) {
    return cli.BuildCobraCommandFromCommand(command,
        cli.WithParserConfig(cli.CobraParserConfig{
            ShortHelpSections: []string{schema.DefaultSlug},
            MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
        }),
    )
}
```

The pattern for a new subcommand:
1. Define a struct with settings fields (each field becomes a CLI flag)
2. Implement `cmds.CommandDescription` (or `BareCommand` for long-running commands)
3. Implement `RunIntoGlazeProcessor` (or `Run` for bare commands)
4. Return from `NewCommand()`
5. Wire into `main.go`

---

## Part 2 — The Annotation Store (pkg/annotate/store.go)

This is the core of the implementation. The annotation store manages a SQLite database that lives alongside the minitrace output directory.

### 2.1 Where the SQLite File Lives

```
output/
├── annotations.db          ← SQLite annotation store (created on first use)
├── active/
│   ├── 2026-03/
│   │   └── *.minitrace.json
│   └── 2026-04/
│       └── *.minitrace.json
└── archive/
    └── ...
```

The `annotations.db` is colocated with the output root. The output root is determined from the archive glob pattern passed to commands. For the `serve` command, this comes from `--archive-glob`. For CLI commands, it will come from `--output-dir` or be inferred from the archive glob.

### 2.2 SQLite Schema

```sql
-- pkg/annotate/schema.sql (embedded via go:embed)

CREATE TABLE IF NOT EXISTS annotations (
    id              TEXT PRIMARY KEY,
    session_id      TEXT NOT NULL,
    annotator       TEXT NOT NULL,
    scope_type      TEXT NOT NULL,
    target_id       TEXT NOT NULL,
    category        TEXT NOT NULL,
    title           TEXT NOT NULL,
    detail          TEXT NOT NULL DEFAULT '',
    tags            TEXT NOT NULL DEFAULT '[]',      -- JSON array
    taxonomy_m      TEXT NOT NULL DEFAULT '[]',     -- JSON array (minitrace codes)
    taxonomy_mast   TEXT NOT NULL DEFAULT '[]',     -- JSON array
    taxonomy_tm     TEXT NOT NULL DEFAULT '[]',      -- JSON array (toolemu)
    classification  TEXT,                            -- NULL = no override
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_anno_session  ON annotations(session_id);
CREATE INDEX IF NOT EXISTS idx_anno_scope   ON annotations(scope_type, target_id);
CREATE INDEX IF NOT EXISTS idx_anno_category ON annotations(category);
CREATE INDEX IF NOT EXISTS idx_anno_annot   ON annotations(annotator);

CREATE TABLE IF NOT EXISTS sync_state (
    session_id       TEXT PRIMARY KEY,
    last_synced_at   TEXT,
    annotation_count INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sessions (
    session_id    TEXT PRIMARY KEY,
    file_path     TEXT NOT NULL,
    title         TEXT,
    framework     TEXT,
    model         TEXT,
    loaded_at     TEXT
);
```

**Design rationale for column naming:**
- `tags`, `taxonomy_m`, `taxonomy_mast`, `taxonomy_tm` are stored as JSON strings. SQLite has no native JSON type — storing as TEXT with JSON content is the standard approach. When reading, you parse the JSON. When writing, you serialize to JSON.
- `classification` is TEXT nullable — NULL means "use the session's own classification."
- `sync_state` tracks which sessions have been synced to JSON, so the sync command can do incremental updates.

### 2.3 The Store Type

```go
// pkg/annotate/store.go

package annotate

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"

    _ "github.com/mattn/go-sqlite3"  // SQLite driver
    "github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

type Store struct {
    db     *sql.DB
    dbPath string  // absolute path to annotations.db
}

// Open opens (or creates) a SQLite annotation store.
// If the database does not exist, it creates it and runs the schema.
func Open(ctx context.Context, outputDir string) (*Store, error) {
    absDir, err := filepath.Abs(outputDir)
    if err != nil {
        return nil, fmt.Errorf("resolving output dir: %w", err)
    }

    // Ensure the directory exists
    if err := os.MkdirAll(absDir, 0755); err != nil {
        return nil, fmt.Errorf("creating output dir: %w", err)
    }

    dbPath := filepath.Join(absDir, "annotations.db")

    db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
    if err != nil {
        return nil, fmt.Errorf("opening sqlite: %w", err)
    }

    // Enable WAL mode for concurrent readers + single writer
    // busy_timeout=5000 prevents "database is locked" on contention

    store := &Store{db: db, dbPath: dbPath}

    if err := store.migrate(ctx); err != nil {
        db.Close()
        return nil, fmt.Errorf("running migrations: %w", err)
    }

    return store, nil
}

// migrate runs schema migrations.
// In v1, this just creates tables if they don't exist.
// Future versions can add ALTER TABLE statements here.
func (s *Store) migrate(ctx context.Context) error {
    schema := `
    CREATE TABLE IF NOT EXISTS annotations (
        id TEXT PRIMARY KEY,
        session_id TEXT NOT NULL,
        annotator TEXT NOT NULL,
        scope_type TEXT NOT NULL,
        target_id TEXT NOT NULL,
        category TEXT NOT NULL,
        title TEXT NOT NULL,
        detail TEXT NOT NULL DEFAULT '',
        tags TEXT NOT NULL DEFAULT '[]',
        taxonomy_m TEXT NOT NULL DEFAULT '[]',
        taxonomy_mast TEXT NOT NULL DEFAULT '[]',
        taxonomy_tm TEXT NOT NULL DEFAULT '[]',
        classification TEXT,
        created_at TEXT NOT NULL,
        updated_at TEXT NOT NULL
    );
    CREATE INDEX IF NOT EXISTS idx_anno_session ON annotations(session_id);
    CREATE INDEX IF NOT EXISTS idx_anno_scope ON annotations(scope_type, target_id);
    CREATE INDEX IF NOT EXISTS idx_anno_category ON annotations(category);
    CREATE INDEX IF NOT EXISTS idx_anno_annot ON annotations(annotator);
    CREATE TABLE IF NOT EXISTS sync_state (
        session_id TEXT PRIMARY KEY,
        last_synced_at TEXT,
        annotation_count INTEGER DEFAULT 0
    );
    CREATE TABLE IF NOT EXISTS sessions (
        session_id TEXT PRIMARY KEY,
        file_path TEXT NOT NULL,
        title TEXT,
        framework TEXT,
        model TEXT,
        loaded_at TEXT
    );
    `
    _, err := s.db.ExecContext(ctx, schema)
    return err
}

func (s *Store) Close() error {
    return s.db.Close()
}
```

### 2.4 CRUD Operations

#### Adding an Annotation

```go
// AddAnnotation inserts a new annotation and returns its ID.
func (s *Store) AddAnnotation(ctx context.Context, ann minitrace.Annotation, sessionID string) error {
    tagsJSON, err := json.Marshal(ann.Content.Tags)
    if err != nil {
        return fmt.Errorf("marshaling tags: %w", err)
    }
    taxM, _ := json.Marshal(ann.TaxonomyMappings.Minitrace)
    taxMast, _ := json.Marshal(ann.TaxonomyMappings.Mast)
    taxTm, _ := json.Marshal(ann.TaxonomyMappings.Toolemu)

    now := minitrace.NowUTC().Format(time.RFC3339)

    query := `
    INSERT INTO annotations (
        id, session_id, annotator, scope_type, target_id,
        category, title, detail,
        tags, taxonomy_m, taxonomy_mast, taxonomy_tm,
        classification,
        created_at, updated_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
    _, err = s.db.ExecContext(ctx, query,
        ann.ID,
        sessionID,
        ann.Annotator,
        ann.Scope.Type,
        ann.Scope.TargetID,
        ann.Content.Category,
        ann.Content.Title,
        ann.Content.Detail,
        string(tagsJSON),
        string(taxM),
        string(taxMast),
        string(taxTm),
        ann.Classification,
        ann.Timestamp,  // already formatted
        now,
    )
    if err != nil {
        return fmt.Errorf("inserting annotation: %w", err)
    }

    // Invalidate sync state for this session
    return s.markUnsynced(ctx, sessionID)
}
```

**Why return the ID?** The caller generates the UUID before calling `AddAnnotation`, so it can display it to the user. This follows the same pattern as the minitrace spec (IDs are "opaque strings, UUIDs recommended").

#### Reading Annotations for a Session

```go
// GetAnnotationsForSession returns all annotations for a given session ID.
func (s *Store) GetAnnotationsForSession(ctx context.Context, sessionID string) ([]minitrace.Annotation, error) {
    query := `
    SELECT id, annotator, scope_type, target_id, category, title, detail,
           tags, taxonomy_m, taxonomy_mast, taxonomy_tm,
           classification, created_at, updated_at
    FROM annotations
    WHERE session_id = ?
    ORDER BY created_at ASC
    `
    rows, err := s.db.QueryContext(ctx, query, sessionID)
    if err != nil {
        return nil, fmt.Errorf("querying annotations: %w", err)
    }
    defer rows.Close()

    var annotations []minitrace.Annotation
    for rows.Next() {
        ann, err := scanAnnotation(rows)
        if err != nil {
            return nil, err
        }
        annotations = append(annotations, ann)
    }
    return annotations, rows.Err()
}

func scanAnnotation(rows *sql.Rows) (minitrace.Annotation, error) {
    var (
        id, sessionID, annotator, scopeType, targetID string
        category, title, detail string
        tags, taxM, taxMast, taxTm string
        classification, createdAt, updatedAt *string
    )
    err := rows.Scan(
        &id, &annotator, &scopeType, &targetID,
        &category, &title, &detail,
        &tags, &taxM, &taxMast, &taxTm,
        &classification, &createdAt, &updatedAt,
    )
    if err != nil {
        return minitrace.Annotation{}, fmt.Errorf("scanning row: %w", err)
    }

    var tagsSlice []string
    var taxMSlice, taxMastSlice, taxTmSlice []string
    json.Unmarshal([]byte(tags), &tagsSlice)
    json.Unmarshal([]byte(taxM), &taxMSlice)
    json.Unmarshal([]byte(taxMast), &taxMastSlice)
    json.Unmarshal([]byte(taxTm), &taxTmSlice)

    var cls *string
    if classification != nil {
        cls = classification
    }

    return minitrace.Annotation{
        ID:        id,
        Timestamp: *createdAt,  // use created_at as timestamp
        Annotator: annotator,
        Scope: minitrace.AnnotationScope{
            Type:     scopeType,
            TargetID: targetID,
        },
        Content: minitrace.AnnotationContent{
            Category: category,
            Tags:     tagsSlice,
            Title:    title,
            Detail:   detail,
        },
        TaxonomyMappings: minitrace.TaxonomyMappings{
            Minitrace: taxMSlice,
            Mast:      taxMastSlice,
            Toolemu:   taxTmSlice,
        },
        Classification: cls,
    }, nil
}
```

#### Listing with Filters

```go
// ListOptions defines filters for listing annotations.
type ListOptions struct {
    SessionID   string
    ScopeType   string
    Category    string
    Annotator   string
    Taxonomy    string   // filter by taxonomy code in any taxonomy field
    Limit       int
    Offset      int
}

// List returns annotations matching the given filters.
// Empty filters return all annotations (up to Limit).
func (s *Store) List(ctx context.Context, opts ListOptions) ([]AnnotationRow, error) {
    query := `
    SELECT id, session_id, annotator, scope_type, target_id,
           category, title, detail,
           tags, taxonomy_m, taxonomy_mast, taxonomy_tm,
           classification, created_at, updated_at
    FROM annotations
    WHERE 1=1
    `
    var args []any
    var cond []string

    if opts.SessionID != "" {
        cond = append(cond, "session_id = ?")
        args = append(args, opts.SessionID)
    }
    if opts.ScopeType != "" {
        cond = append(cond, "scope_type = ?")
        args = append(args, opts.ScopeType)
    }
    if opts.Category != "" {
        cond = append(cond, "category = ?")
        args = append(args, opts.Category)
    }
    if opts.Annotator != "" {
        cond = append(cond, "annotator = ?")
        args = append(args, opts.Annotator)
    }
    if opts.Taxonomy != "" {
        // Match in any taxonomy column
        cond = append(cond, "(taxonomy_m LIKE ? OR taxonomy_mast LIKE ? OR taxonomy_tm LIKE ?)")
        like := "%" + opts.Taxonomy + "%"
        args = append(args, like, like, like)
    }

    for _, c := range cond {
        query += " AND " + c
    }

    query += " ORDER BY created_at DESC"

    if opts.Limit > 0 {
        query += fmt.Sprintf(" LIMIT %d", opts.Limit)
    }
    if opts.Offset > 0 {
        query += fmt.Sprintf(" OFFSET %d", opts.Offset)
    }

    rows, err := s.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []AnnotationRow
    for rows.Next() {
        row, err := scanAnnotationRow(rows)
        if err != nil {
            return nil, err
        }
        results = append(results, row)
    }
    return results, rows.Err()
}

// AnnotationRow is the flat row type returned by List.
type AnnotationRow struct {
    ID            string
    SessionID     string
    Annotator     string
    ScopeType     string
    TargetID      string
    Category      string
    Title         string
    Detail        string
    Tags          []string
    TaxonomyM     []string
    TaxonomyMast  []string
    TaxonomyTm    []string
    Classification *string
    CreatedAt     string
    UpdatedAt     string
}
```

#### Updating an Annotation

```go
// Update patches specific fields of an existing annotation.
func (s *Store) Update(ctx context.Context, id string, patch AnnotationPatch) error {
    var sets []string
    var args []any

    if patch.Title != nil {
        sets = append(sets, "title = ?")
        args = append(args, *patch.Title)
    }
    if patch.Detail != nil {
        sets = append(sets, "detail = ?")
        args = append(args, *patch.Detail)
    }
    if patch.Category != nil {
        sets = append(sets, "category = ?")
        args = append(args, *patch.Category)
    }
    if patch.Tags != nil {
        tagsJSON, _ := json.Marshal(*patch.Tags)
        sets = append(sets, "tags = ?")
        args = append(args, string(tagsJSON))
    }
    if patch.TaxonomyM != nil {
        taxJSON, _ := json.Marshal(*patch.TaxonomyM)
        sets = append(sets, "taxonomy_m = ?")
        args = append(args, string(taxJSON))
    }
    if patch.Classification != nil {
        sets = append(sets, "classification = ?")
        args = append(args, *patch.Classification)
    }

    if len(sets) == 0 {
        return nil  // nothing to update
    }

    sets = append(sets, "updated_at = ?")
    args = append(args, time.Now().UTC().Format(time.RFC3339))
    args = append(args, id)

    query := "UPDATE annotations SET " + strings.Join(sets, ", ") + " WHERE id = ?"
    result, err := s.db.ExecContext(ctx, query, args...)
    if err != nil {
        return fmt.Errorf("updating annotation: %w", err)
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        return ErrNotFound
    }

    // Get session_id for this annotation and mark unsynced
    var sessionID string
    _ = s.db.QueryRowContext(ctx, "SELECT session_id FROM annotations WHERE id = ?", id).Scan(&sessionID)
    _ = s.markUnsynced(ctx, sessionID)

    return nil
}

type AnnotationPatch struct {
    Title          *string
    Detail         *string
    Category       *string
    Tags           *[]string
    TaxonomyM      *[]string
    Classification *string
}

var ErrNotFound = errors.New("annotation not found")
```

#### Deleting an Annotation

```go
// Delete removes an annotation by ID.
func (s *Store) Delete(ctx context.Context, id string) error {
    // Get session_id first (to invalidate sync state)
    var sessionID string
    err := s.db.QueryRowContext(ctx, "SELECT session_id FROM annotations WHERE id = ?", id).Scan(&sessionID)
    if err == sql.ErrNoRows {
        return ErrNotFound
    }
    if err != nil {
        return fmt.Errorf("looking up annotation: %w", err)
    }

    _, err = s.db.ExecContext(ctx, "DELETE FROM annotations WHERE id = ?", id)
    if err != nil {
        return fmt.Errorf("deleting annotation: %w", err)
    }

    return s.markUnsynced(ctx, sessionID)
}
```

### 2.5 Sync State Tracking

```go
func (s *Store) markUnsynced(ctx context.Context, sessionID string) error {
    _, err := s.db.ExecContext(ctx,
        "INSERT INTO sync_state (session_id, annotation_count) VALUES (?, 0) "+
            "ON CONFLICT(session_id) DO UPDATE SET annotation_count = 0, last_synced_at = NULL",
        sessionID,
    )
    return err
}

func (s *Store) markSynced(ctx context.Context, sessionID string, count int) error {
    now := time.Now().UTC().Format(time.RFC3339)
    _, err := s.db.ExecContext(ctx,
        "UPDATE sync_state SET last_synced_at = ?, annotation_count = ? WHERE session_id = ?",
        now, count, sessionID,
    )
    return err
}

// GetUnsyncedSessions returns sessions that have annotations but haven't been synced.
func (s *Store) GetUnsyncedSessions(ctx context.Context) ([]SyncState, error) {
    rows, err := s.db.QueryContext(ctx, `
    SELECT ss.session_id, ss.last_synced_at, ss.annotation_count, a.count
    FROM sync_state ss
    JOIN (
        SELECT session_id, COUNT(*) as count FROM annotations GROUP BY session_id
    ) a ON a.session_id = ss.session_id
    WHERE ss.last_synced_at IS NULL OR ss.annotation_count != a.count
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var states []SyncState
    for rows.Next() {
        var st SyncState
        rows.Scan(&st.SessionID, &st.LastSyncedAt, &st.AnnotationCount, &st.CurrentCount)
        states = append(states, st)
    }
    return states, rows.Err()
}

type SyncState struct {
    SessionID       string
    LastSyncedAt    *string
    AnnotationCount int   // count at last sync (may be stale)
    CurrentCount    int   // actual current count
}
```

---

## Part 3 — The Sync Module (pkg/annotate/sync.go)

The sync module handles writing annotations back to `.minitrace.json` files.

### 3.1 The Algorithm

```
For each session that needs syncing:
    1. Load the .minitrace.json file into a minitrace.Session struct
    2. Query SQLite for all annotations for this session_id
    3. Convert SQLite annotation rows back to minitrace.Annotation structs
    4. Replace session.Annotations with the fresh list from SQLite
    5. Serialize to JSON (preserving formatting if possible)
    6. Write atomically: write to {file}.tmp, then os.Rename to {file}
    7. Update sync_state in SQLite
```

### 3.2 Atomic Write Pattern

```go
// pkg/annotate/sync.go

package annotate

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"

    "github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

// SyncOptions controls the sync behavior.
type SyncOptions struct {
    DryRun    bool       // don't write files, just report what would happen
    SessionID string     // if non-empty, sync only this session
}

// SyncSession syncs annotations from SQLite back to a single .minitrace.json file.
func (s *Store) SyncSession(ctx context.Context, filePath string, annotations []minitrace.Annotation, dryRun bool) error {
    // Step 1: Read the existing file
    payload, err := os.ReadFile(filePath)
    if err != nil {
        return fmt.Errorf("reading session file: %w", err)
    }

    // Step 2: Unmarshal into Session struct
    var session minitrace.Session
    if err := json.Unmarshal(payload, &session); err != nil {
        return fmt.Errorf("unmarshaling session JSON: %w", err)
    }

    // Step 3: Replace annotations
    session.Annotations = annotations

    // Step 4: Marshal back to JSON
    // Use json.MarshalIndent to produce readable output (2-space indent)
    output, err := json.MarshalIndent(session, "", "  ")
    if err != nil {
        return fmt.Errorf("marshaling session: %w", err)
    }

    if dryRun {
        fmt.Printf("[dry-run] would update %s with %d annotations\n", filePath, len(annotations))
        return nil
    }

    // Step 5: Atomic write
    tmpPath := filePath + ".tmp"
    if err := os.WriteFile(tmpPath, output, 0644); err != nil {
        return fmt.Errorf("writing temp file: %w", err)
    }
    if err := os.Rename(tmpPath, filePath); err != nil {
        os.Remove(tmpPath)  // clean up on failure
        return fmt.Errorf("renaming temp file: %w", err)
    }

    return nil
}
```

**Why the tmp-then-rename pattern?** `os.Rename` is atomic on POSIX systems when both paths are on the same filesystem. If the process crashes between `WriteFile` and `Rename`, the original file is untouched. The `.tmp` file is cleaned up on error.

**About JSON formatting:** `json.MarshalIndent` reformats the JSON with 2-space indentation. This changes the file's formatting, which is a minor downside. For production, you might want to use a library like `github.com/tidwall/pretty` that preserves the original formatting where possible. For v1, the simple approach is fine.

### 3.3 Bulk Sync Command

```go
// SyncAll syncs all sessions that have unsynced annotations.
func (s *Store) SyncAll(ctx context.Context, sessionIndex map[string]string, opts SyncOptions) (*SyncReport, error) {
    unsynced, err := s.GetUnsyncedSessions(ctx)
    if err != nil {
        return nil, fmt.Errorf("getting unsynced sessions: %w", err)
    }

    report := &SyncReport{}

    for _, st := range unsynced {
        // Skip if filtering to a specific session
        if opts.SessionID != "" && st.SessionID != opts.SessionID {
            continue
        }

        filePath, ok := sessionIndex[st.SessionID]
        if !ok {
            report.Skipped = append(report.Skipped, st.SessionID)
            continue  // session file not found, skip
        }

        // Get fresh annotation list from SQLite
        annotations, err := s.GetAnnotationsForSession(ctx, st.SessionID)
        if err != nil {
            report.Errors = append(report.Errors, SyncError{
                SessionID: st.SessionID,
                Error:     err.Error(),
            })
            continue
        }

        err = s.SyncSession(ctx, filePath, annotations, opts.DryRun)
        if err != nil {
            report.Errors = append(report.Errors, SyncError{
                SessionID: st.SessionID,
                Error:     err.Error(),
            })
            continue
        }

        if !opts.DryRun {
            if err := s.markSynced(ctx, st.SessionID, len(annotations)); err != nil {
                // Non-fatal: log but continue
                report.Errors = append(report.Errors, SyncError{
                    SessionID: st.SessionID,
                    Error:     fmt.Sprintf("synced but failed to mark synced: %s", err),
                })
            }
        }

        report.Synced = append(report.Synced, st.SessionID)
    }

    return report, nil
}

type SyncReport struct {
    Synced  []string
    Skipped []string  // session ID known but file not in index
    Errors  []SyncError
}

type SyncError struct {
    SessionID string
    Error     string
}
```

---

## Part 4 — The DuckDB Integration (pkg/annotate/duckdb.go)

DuckDB reads annotations from SQLite and makes them queryable alongside session data.

### 4.1 The Approach: DuckDB's sqlite_scanner Extension

DuckDB ships with a built-in `sqlite_scanner` extension. It is **NOT an external dependency** — it ships with the DuckDB binary you're already using. It registers itself when loaded with `LOAD sqlite_scanner` and provides three functions:

| Function | Type | Purpose |
|----------|------|---------|
| `sqlite_attach(path, overwrite => bool)` | Table-producing | Attach a SQLite `.db` file as schema objects |
| `sqlite_query(db, sql)` | Table-producing | Run a SQLite query against an attached DB |
| `sqlite_scan(db, table)` | Table-producing | Scan a table from an attached SQLite DB |

**Key findings from testing (see scripts/07-sqlite-attach-working-join.sql):**

1. `INSTALL sqlite_scanner` and `LOAD sqlite_scanner` are required — the extension is available but not auto-loaded
2. `sqlite_attach` uses **named parameters**: `CALL sqlite_attach('/path/to/db', overwrite => true)` — NOT positional arguments
3. Attached SQLite tables land directly in the `main` schema — query as `SELECT * FROM annotations`, no schema prefix needed
4. Annotations are live — DuckDB queries the SQLite file directly, no refresh needed after writes
5. The Go integration is a single `INSTALL/LOAD sqlite_scanner` call before loading sessions

### 4.2 Go Integration: Loading the Extension

The DuckDB query engine initializes a connection in `OpenConnection`. At startup (before loading sessions), we install and load the extension, then attach the annotations database:

```go
// pkg/annotate/duckdb.go

package annotate

import (
    "context"
    "database/sql"
    "fmt"
    "path/filepath"
)

// AttachAnnotationsToDuckDB installs sqlite_scanner, loads it,
// and attaches the annotations SQLite database to the DuckDB connection.
// This makes annotations queryable as regular tables alongside sessions_base.
func AttachAnnotationsToDuckDB(ctx context.Context, conn *sql.Conn, outputDir string) error {
    // Install and load the sqlite_scanner extension (built into DuckDB)
    _, err := conn.ExecContext(ctx, "INSTALL sqlite_scanner")
    if err != nil {
        return fmt.Errorf("installing sqlite_scanner: %w", err)
    }

    _, err = conn.ExecContext(ctx, "LOAD sqlite_scanner")
    if err != nil {
        return fmt.Errorf("loading sqlite_scanner: %w", err)
    }

    // Build path to annotations.db
    absDir, err := filepath.Abs(outputDir)
    if err != nil {
        return fmt.Errorf("resolving output dir: %w", err)
    }
    dbPath := filepath.Join(absDir, "annotations.db")

    // Attach the SQLite DB to DuckDB.
    // overwrite=true allows re-attaching if the same DB is already open.
    // SQLite tables land in the main schema alongside DuckDB tables.
    _, err = conn.ExecContext(ctx,
        "CALL sqlite_attach($1, overwrite => true)",
        dbPath,
    )
    if err != nil {
        // If annotations.db doesn't exist yet, that's OK — no annotations to query
        // The SQLite DB is created on first write, not on first read
        // But sqlite_attach fails if the file doesn't exist at all
        // So we need to create it first, or skip if not present
        if !annotationDBExists(dbPath) {
            return nil  // No annotations DB yet — skip attachment
        }
        return fmt.Errorf("attaching annotations.db: %w", err)
    }

    return nil
}

// annotationDBExists checks if the annotations database file exists.
func annotationDBExists(dbPath string) bool {
    // Use a lightweight check without importing os
    _, err := os.Stat(dbPath)
    return err == nil
}
```

### 4.3 Refreshing Annotations (No Refresh Needed!)

The beautiful thing about `sqlite_attach` is that **annotations are live**. When you add an annotation via the CLI or web UI, the SQLite file is updated. The next DuckDB query automatically sees the new data. No temp table refresh needed.

This is in contrast to the annotations_flat approach (export SQLite → JSON → load into DuckDB temp table), which would require re-running the export on every mutation.

### 4.4 Annotation Queries (queries/annotations.sql)

The existing `annotations.sql` query needs to be updated to read from SQLite directly:

```sql
-- queries/annotations.sql
-- Annotations are stored in the attached SQLite annotations.db.
-- SQLite tables land in the main schema — no schema prefix needed.
-- sessions_base is loaded from .minitrace.json files (existing load.sql).

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

### 4.5 Column Name Mapping

The SQLite schema uses abbreviated column names for brevity:

| SQLite column | Query alias | Maps to |
|---|---|---|
| `session_id` | `session_id` | minitrace field |
| `scope_type` | `scope_type` | `AnnotationScope.Type` |
| `target_id` | `target_id` | `AnnotationScope.TargetID` |
| `category` | `category` | `AnnotationContent.Category` |
| `title` | `title` | `AnnotationContent.Title` |
| `detail` | `detail` | `AnnotationContent.Detail` |
| `tags` | `tags` | `AnnotationContent.Tags` (JSON array string) |
| `taxonomy_m` | `taxonomy_minitrace` | `TaxonomyMappings.Minitrace` |
| `taxonomy_mast` | `taxonomy_mast` | `TaxonomyMappings.Mast` |
| `taxonomy_tm` | `taxonomy_toolemu` | `TaxonomyMappings.Toolemu` |
| `classification` | `classification` | `Annotation.Classification` |
| `created_at` | `created_at` | `Annotation.Timestamp` |
| `updated_at` | `updated_at` | metadata |

### 4.6 Serve Command Startup

In the `serve` command, the DuckDB connection is opened before loading sessions:

```go
// cmd/go-minitrace/cmds/serve/serve.go

// After opening the DuckDB connection...
db, conn, err := queryengine.OpenConnection(signalCtx, settings.DBPath)
// ...

// Attach annotations from SQLite (after connection, before loading sessions)
if err := annotate.AttachAnnotationsToDuckDB(signalCtx, conn, outputDir); err != nil {
    log.Warn().Err(err).Msg("could not attach annotations database")
}

// Now load sessions (annotations are already attached and live)
if err := queryengine.LoadArchive(signalCtx, conn, queryengine.LoadOptions{
    ArchiveGlobs: settings.ArchiveGlob,
    TableName:    settings.TableName,
}); err != nil {
    return err
}
```

Since annotations are live, the `refreshAnnotationsFlat` function is no longer needed. The serve command never needs to refresh — SQLite is the source of truth, and DuckDB queries it on every request.

---

## Part 5 — The CLI Commands (cmd/go-minitrace/cmds/annotate/)

The CLI follows the standard go-minitrace command pattern.

### 5.1 Command Structure

```
go-minitrace annotate      ← root (groups all subcommands)
├── go-minitrace annotate add        ← add a new annotation
├── go-minitrace annotate list       ← list annotations (with filters)
├── go-minitrace annotate edit       ← edit an existing annotation
├── go-minitrace annotate delete     ← delete an annotation
├── go-minitrace annotate sync       ← sync annotations to JSON files
└── go-minitrace annotate import     ← import annotations from JSON into SQLite
```

### 5.2 The Root Command

```go
// cmd/go-minitrace/cmds/annotate/root.go

package annotate

import (
    "github.com/spf13/cobra"
)

func NewCommand() (*cobra.Command, error) {
    root := &cobra.Command{
        Use:   "annotate",
        Short: "Manage annotations on minitrace sessions",
        Long: `Store, query, and sync annotations on converted minitrace sessions.

Annotations are stored in a SQLite database (annotations.db) alongside the
output directory. Use "sync" to write annotations back into the .minitrace.json
files.

Examples:
  go-minitrace annotate add --session SESSION_ID --category observation --title "Found bug"
  go-minitrace annotate list --category ai-failure --taxonomy F-AUT
  go-minitrace annotate sync --dry-run
`,
    }

    root.AddCommand(newAddCommand())
    root.AddCommand(newListCommand())
    root.AddCommand(newEditCommand())
    root.AddCommand(newDeleteCommand())
    root.AddCommand(newSyncCommand())
    root.AddCommand(newImportCommand())

    return root, nil
}
```

### 5.3 The Add Command

```go
// cmd/go-minitrace/cmds/annotate/add.go

package annotate

import (
    "context"
    "fmt"
    "strings"

    "github.com/go-go-golems/go-minitrace/pkg/annotate"
    "github.com/go-go-golems/go-minitrace/pkg/minitrace"
    "github.com/google/uuid"
    "github.com/spf13/cobra"
    "github.com/spf13/pflag"
)

func newAddCommand() *cobra.Command {
    var (
        outputDir    string
        sessionID    string
        scopeType    string  // "session" | "turn" | "tool_call" | "handover"
        targetID     string
        annotator    string  // "user" | "model" | "automated"
        category     string  // "observation" | "pattern" | "ai-failure" | "recommendation"
        title        string
        detail       string
        tags         string  // comma-separated
        taxonomyM    string  // comma-separated minitrace codes
        taxonomyMast  string  // comma-separated MAST codes
        taxonomyTm   string  // comma-separated ToolEmu codes
        classification string
    )

    cmd := &cobra.Command{
        Use:   "add",
        Short: "Add an annotation to a session",
        Example: `
  go-minitrace annotate add \
    --output-dir ./output \
    --session 019d4a02-6bb9-7921-8a2e-224b071acd0f \
    --scope session \
    --category ai-failure \
    --title "Over-autonomy at turn 15" \
    --detail "Agent edited files without being asked" \
    --tags F-AUT,C-SEQ \
    --annotator user
`,
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := cmd.Context()

            // Open the annotation store
            store, err := annotate.Open(ctx, outputDir)
            if err != nil {
                return fmt.Errorf("opening annotation store: %w", err)
            }
            defer store.Close()

            // Build taxonomy mappings
            taxM := parseCommaList(taxonomyM)
            taxMast := parseCommaList(taxonomyMast)
            taxTm := parseCommaList(taxonomyTm)

            // Default scope_type to "session" if not specified
            if scopeType == "" {
                scopeType = "session"
            }
            // Default target_id to sessionID if scope is "session"
            if targetID == "" {
                targetID = sessionID
            }

            // Build the annotation
            ann := minitrace.BuildAnnotation(
                uuid.New().String(),
                annotator,
                scopeType,
                targetID,
                category,
                title,
                detail,
                parseCommaList(tags),
                &minitrace.TaxonomyMappings{
                    Minitrace: taxM,
                    Mast:      taxMast,
                    Toolemu:   taxTm,
                },
            )

            if classification != "" {
                ann.Classification = &classification
            }

            // Validate
            if err := validateAnnotation(ann); err != nil {
                return fmt.Errorf("validation failed: %w", err)
            }

            // Insert
            if err := store.AddAnnotation(ctx, ann, sessionID); err != nil {
                return fmt.Errorf("adding annotation: %w", err)
            }

            cmd.Println("Annotation added:")
            cmd.Printf("  ID:         %s\n", ann.ID)
            cmd.Printf("  Session:    %s\n", sessionID)
            cmd.Printf("  Scope:     %s / %s\n", ann.Scope.Type, ann.Scope.TargetID)
            cmd.Printf("  Category:  %s\n", ann.Content.Category)
            cmd.Printf("  Title:     %s\n", ann.Content.Title)

            return nil
        },
    }

    flags := cmd.Flags()
    flags.StringVar(&outputDir, "output-dir", "./output", "Output directory (contains annotations.db)")
    flags.StringVar(&sessionID, "session", "", "Session ID (required)")
    flags.StringVar(&scopeType, "scope", "", "Scope: session, turn, tool_call, handover")
    flags.StringVar(&targetID, "target-id", "", "Target ID (defaults to session ID for session scope)")
    flags.StringVar(&annotator, "annotator", "user", "Annotator: user, model, automated")
    flags.StringVar(&category, "category", "", "Category: observation, pattern, ai-failure, recommendation")
    flags.StringVar(&title, "title", "", "Annotation title (required)")
    flags.StringVar(&detail, "detail", "", "Annotation detail text")
    flags.StringVar(&tags, "tags", "", "Comma-separated tags")
    flags.StringVar(&taxonomyM, "taxonomy-minitrace", "", "Comma-separated minitrace failure codes (F-AUT, C-SEQ, etc.)")
    flags.StringVar(&taxonomyMast, "taxonomy-mast", "", "Comma-separated MAST codes")
    flags.StringVar(&taxonomyTm, "taxonomy-toolemu", "", "Comma-separated ToolEmu codes")
    flags.StringVar(&classification, "classification", "", "Override session classification")

    cmd.MarkFlagRequired("session")
    cmd.MarkFlagRequired("title")
    cmd.MarkFlagRequired("category")

    return cmd
}

func parseCommaList(s string) []string {
    if strings.TrimSpace(s) == "" {
        return []string{}
    }
    parts := strings.Split(s, ",")
    result := make([]string, 0, len(parts))
    for _, p := range parts {
        p = strings.TrimSpace(p)
        if p != "" {
            result = append(result, p)
        }
    }
    return result
}

func validateAnnotation(ann minitrace.Annotation) error {
    validCategories := map[string]bool{
        "observation": true, "pattern": true, "ai-failure": true, "recommendation": true,
    }
    validScopes := map[string]bool{
        "session": true, "turn": true, "tool_call": true, "handover": true,
    }
    validAnnotators := map[string]bool{
        "user": true, "model": true, "automated": true,
    }

    if !validCategories[ann.Content.Category] {
        return fmt.Errorf("invalid category: %s", ann.Content.Category)
    }
    if !validScopes[ann.Scope.Type] {
        return fmt.Errorf("invalid scope_type: %s", ann.Scope.Type)
    }
    if !validAnnotators[ann.Annotator] {
        return fmt.Errorf("invalid annotator: %s", ann.Annotator)
    }
    return nil
}
```

### 5.4 The List Command

```go
// cmd/go-minitrace/cmds/annotate/list.go

func newListCommand() *cobra.Command {
    var (
        outputDir  string
        sessionID  string
        scopeType  string
        category   string
        annotator  string
        taxonomy   string
        limit      int
        format     string  // "table" (default) | "json"
    )

    cmd := &cobra.Command{
        Use:   "list",
        Short: "List annotations (with optional filters)",
        Example: `
  # List all annotations for a session
  go-minitrace annotate list --session SESSION_ID

  # List all F-AUT failures across all sessions
  go-minitrace annotate list --taxonomy F-AUT --limit 100

  # List all ai-failure annotations as JSON
  go-minitrace annotate list --category ai-failure --format json
`,
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := cmd.Context()

            store, err := annotate.Open(ctx, outputDir)
            if err != nil {
                return fmt.Errorf("opening annotation store: %w", err)
            }
            defer store.Close()

            if limit == 0 {
                limit = 50
            }

            rows, err := store.List(ctx, annotate.ListOptions{
                SessionID: sessionID,
                ScopeType: scopeType,
                Category:  category,
                Annotator: annotator,
                Taxonomy:  taxonomy,
                Limit:     limit,
            })
            if err != nil {
                return fmt.Errorf("listing annotations: %w", err)
            }

            if len(rows) == 0 {
                cmd.Println("No annotations found.")
                return nil
            }

            if format == "json" {
                return printJSON(cmd, rows)
            }

            // Table format
            printHeader(cmd, "ID", "SESSION", "SCOPE", "TARGET", "CATEGORY", "TITLE", "ANNOTATOR", "CREATED")
            for _, row := range rows {
                printRow(cmd,
                    truncate(row.ID, 8),
                    truncate(row.SessionID, 12),
                    row.ScopeType,
                    truncate(row.TargetID, 12),
                    row.Category,
                    truncate(row.Title, 30),
                    row.Annotator,
                    truncate(row.CreatedAt, 10),
                )
            }
            cmd.Printf("\n%d annotation(s)\n", len(rows))

            return nil
        },
    }

    flags := cmd.Flags()
    flags.StringVar(&outputDir, "output-dir", "./output", "Output directory")
    flags.StringVar(&sessionID, "session", "", "Filter by session ID")
    flags.StringVar(&scopeType, "scope", "", "Filter by scope type")
    flags.StringVar(&category, "category", "", "Filter by category")
    flags.StringVar(&annotator, "annotator", "", "Filter by annotator")
    flags.StringVar(&taxonomy, "taxonomy", "", "Filter by taxonomy code (matches any taxonomy field)")
    flags.IntVar(&limit, "limit", 50, "Maximum number of results")
    flags.StringVar(&format, "format", "table", "Output format: table, json")

    return cmd
}
```

### 5.5 The Sync Command

```go
// cmd/go-minitrace/cmds/annotate/sync.go

func newSyncCommand() *cobra.Command {
    var (
        outputDir    string
        archiveGlob  string  // same as serve --archive-glob, used to build session index
        sessionID    string  // if set, sync only this session
        dryRun       bool
    }

    cmd := &cobra.Command{
        Use:   "sync",
        Short: "Sync annotations from SQLite back to .minitrace.json files",
        Example: `
  # Dry-run: see what would be synced
  go-minitrace annotate sync \
    --output-dir ./output \
    --archive-glob './output/active/*/*.minitrace.json' \
    --dry-run

  # Real sync
  go-minitrace annotate sync \
    --output-dir ./output \
    --archive-glob './output/active/*/*.minitrace.json'

  # Sync a specific session
  go-minitrace annotate sync --session SESSION_ID
`,
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := cmd.Context()

            // Open the annotation store
            store, err := annotate.Open(ctx, outputDir)
            if err != nil {
                return fmt.Errorf("opening annotation store: %w", err)
            }
            defer store.Close()

            // Build session index from glob
            sessionIndex, err := buildSessionIndex(archiveGlob)
            if err != nil {
                return fmt.Errorf("building session index: %w", err)
            }

            cmd.Printf("Indexing %d sessions...\n", len(sessionIndex))

            report, err := store.SyncAll(ctx, sessionIndex, annotate.SyncOptions{
                DryRun:    dryRun,
                SessionID: sessionID,
            })
            if err != nil {
                return fmt.Errorf("running sync: %w", err)
            }

            // Print report
            if dryRun {
                cmd.Println("\n[dry-run] — no files were modified")
            }
            if len(report.Synced) > 0 {
                cmd.Printf("\nSynced: %d session(s)\n", len(report.Synced))
                for _, id := range report.Synced {
                    cmd.Printf("  ✓ %s\n", id)
                }
            }
            if len(report.Skipped) > 0 {
                cmd.Printf("\nSkipped (file not found): %d\n", len(report.Skipped))
                for _, id := range report.Skipped {
                    cmd.Printf("  - %s\n", id)
                }
            }
            if len(report.Errors) > 0 {
                cmd.Printf("\nErrors: %d\n", len(report.Errors))
                for _, e := range report.Errors {
                    cmd.Printf("  ✗ %s: %s\n", e.SessionID, e.Error)
                }
                return fmt.Errorf("sync completed with errors")
            }

            return nil
        },
    }

    flags := cmd.Flags()
    flags.StringVar(&outputDir, "output-dir", "./output", "Output directory (contains annotations.db)")
    flags.StringVar(&archiveGlob, "archive-glob", "./output/active/*/*.minitrace.json",
        "Glob pattern for .minitrace.json files (used to build session index)")
    flags.StringVar(&sessionID, "session", "", "Sync only this session ID")
    flags.BoolVar(&dryRun, "dry-run", false, "Show what would be synced without modifying files")

    return cmd
}
```

### 5.6 Wiring Into main.go

After all command files are created, update `main.go`:

```go
// cmd/go-minitrace/main.go

import (
    // ... existing imports ...
    "github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/annotate"
)

func main() {
    rootCmd := &cobra.Command{/* ... */}
    // ... existing commands ...

    annotateCmd, err := annotate.NewCommand()
    cobra.CheckErr(err)

    rootCmd.AddCommand(
        discoverCmd,
        convertCmd,
        queryCmd,
        serveCmd,
        validateCommand,
        annotateCmd,  // ← add this
    )
}
```

---

## Part 6 — HTTP API Endpoints (cmd/go-minitrace/cmds/serve/)

The `serve` command gets new endpoints for annotation CRUD, backed by the SQLite store.

### 6.1 Architecture Change

The `Server` struct gains an annotation store and a session index (for sync):

```go
// cmd/go-minitrace/cmds/serve/server.go — additions

type Server struct {
    conn           *sql.Conn  // DuckDB (read-only)
    tableName      string
    presetDirs     []string
    queryDirs      []string
    sessionIndex   map[string]string  // session_id -> file path
    devMode        bool
    mux            *http.ServeMux

    // New fields
    annoStore      *annotate.Store
    annoIndex      map[string]string  // same as sessionIndex
}

func NewServer(/* existing args */, annoStore *annotate.Store, annoIndex map[string]string) *Server {
    s := &Server{
        // ... existing fields ...
        annoStore:    annoStore,
        annoIndex:    annoIndex,
    }
    s.mux = http.NewServeMux()
    s.routes()
    return s
}
```

The `ServeSettings` struct gets a new field:

```go
// cmd/go-minitrace/cmds/serve/serve.go

type ServeSettings struct {
    // ... existing fields ...
    AnnotationsDBPath string  `glazed:"annotations-db-path"`
}
```

### 6.2 New Routes

```go
func (s *Server) routes() {
    // Existing routes ...
    s.mux.HandleFunc("GET /api/sessions",              s.handleGetSessions)
    s.mux.HandleFunc("GET /api/sessions/{id}",         s.handleGetSession)
    s.mux.HandleFunc("GET /api/sessions/{id}/blocks",  s.handleGetSessionBlocks)

    // New annotation routes
    s.mux.HandleFunc("GET /api/sessions/{id}/annotations", s.handleGetSessionAnnotations)
    s.mux.HandleFunc("POST /api/sessions/{id}/annotations", s.handleCreateAnnotation)

    s.mux.HandleFunc("GET /api/annotations",              s.handleListAnnotations)
    s.mux.HandleFunc("PUT /api/annotations/{annId}",      s.handleUpdateAnnotation)
    s.mux.HandleFunc("DELETE /api/annotations/{annId}",   s.handleDeleteAnnotation)

    s.mux.HandleFunc("POST /api/annotations/sync",         s.handleSyncAnnotations)

    // Query routes (existing) ...
    s.mux.HandleFunc("POST /api/query",                  s.handleExecuteQuery)
}
```

### 6.3 Handler Implementations

#### GET /api/sessions/:id/annotations

```go
func (s *Server) handleGetSessionAnnotations(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")
    ctx := r.Context()

    annotations, err := s.annoStore.GetAnnotationsForSession(ctx, sessionID)
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{
            "error": fmt.Sprintf("loading annotations: %s", err),
        })
        return
    }

    writeJSON(w, http.StatusOK, map[string]any{
        "session_id":  sessionID,
        "count":       len(annotations),
        "annotations": annotations,
    })
}
```

#### POST /api/sessions/:id/annotations

```go
type CreateAnnotationRequest struct {
    Annotator     string   `json:"annotator"`
    ScopeType     string   `json:"scope_type"`
    TargetID      string   `json:"target_id"`
    Category      string   `json:"category"`
    Title         string   `json:"title"`
    Detail        string   `json:"detail"`
    Tags          []string `json:"tags"`
    TaxonomyM     []string `json:"taxonomy_minitrace"`
    TaxonomyMast  []string `json:"taxonomy_mast"`
    TaxonomyTm    []string `json:"taxonomy_toolemu"`
    Classification *string  `json:"classification"`
}

func (s *Server) handleCreateAnnotation(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")
    ctx := r.Context()

    var req CreateAnnotationRequest
    if err := decodeRequest(r, &req); err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
        return
    }

    // Validate inputs
    if req.Title == "" {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
        return
    }
    if req.Category == "" {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "category is required"})
        return
    }

    // Default target_id
    targetID := req.TargetID
    if targetID == "" {
        targetID = sessionID
    }
    if req.ScopeType == "" {
        req.ScopeType = "session"
    }
    if req.Annotator == "" {
        req.Annotator = "user"
    }

    taxonomyMappings := &minitrace.TaxonomyMappings{
        Minitrace: req.TaxonomyM,
        Mast:      req.TaxonomyMast,
        Toolemu:   req.TaxonomyTm,
    }

    ann := minitrace.BuildAnnotation(
        uuid.New().String(),
        req.Annotator,
        req.ScopeType,
        targetID,
        req.Category,
        req.Title,
        req.Detail,
        req.Tags,
        taxonomyMappings,
    )
    ann.Classification = req.Classification

    if err := s.annoStore.AddAnnotation(ctx, ann, sessionID); err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{
            "error": fmt.Sprintf("creating annotation: %s", err),
        })
        return
    }

    // Annotations are live via sqlite_scanner — no refresh needed.
    // DuckDB queries the SQLite file directly on every request.

    writeJSON(w, http.StatusCreated, ann)
}
```

#### POST /api/annotations/sync

```go
type SyncRequest struct {
    SessionID string `json:"session_id"`  // optional, sync all if empty
    DryRun    bool   `json:"dry_run"`
}

type SyncResponse struct {
    Synced  []string       `json:"synced"`
    Skipped []string       `json:"skipped"`
    Errors  []SyncError    `json:"errors"`
}

func (s *Server) handleSyncAnnotations(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    var req SyncRequest
    if err := decodeRequest(r, &req); err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
        return
    }

    report, err := s.annoStore.SyncAll(ctx, s.annoIndex, annotate.SyncOptions{
        DryRun:    req.DryRun,
        SessionID: req.SessionID,
    })
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{
            "error": fmt.Sprintf("sync failed: %s", err),
        })
        return
    }

    status := http.StatusOK
    if len(report.Errors) > 0 {
        status = http.StatusPartialContent
    }

    writeJSON(w, status, SyncResponse{
        Synced:  report.Synced,
        Skipped: report.Skipped,
        Errors:  report.Errors,
    })
}
```

### 6.4 Refreshing the DuckDB Annotations Table

Since annotations are live via sqlite_scanner (DuckDB queries SQLite directly), no refresh is needed after mutations. The annotations appear in DuckDB queries immediately.

---

## Part 7 — Web UI (web/src/)

The React frontend gets annotation support in three places.

### 7.1 New API Client Functions (web/src/api/minitrace.ts)

```typescript
// web/src/api/minitrace.ts — add these functions

export async function getSessionAnnotations(sessionId: string): Promise<Annotation[]> {
  const res = await fetch(`/api/sessions/${sessionId}/annotations`);
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  const data = await res.json();
  return data.annotations as Annotation[];
}

export async function createAnnotation(
  sessionId: string,
  payload: CreateAnnotationPayload
): Promise<Annotation> {
  const res = await fetch(`/api/sessions/${sessionId}/annotations`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.json();
}

export async function updateAnnotation(
  annId: string,
  payload: Partial<CreateAnnotationPayload>
): Promise<Annotation> {
  const res = await fetch(`/api/annotations/${annId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.json();
}

export async function deleteAnnotation(annId: string): Promise<void> {
  const res = await fetch(`/api/annotations/${annId}`, { method: 'DELETE' });
  if (!res.ok) throw new Error(`API error: ${res.status}`);
}

export async function syncAnnotations(
  sessionId?: string,
  dryRun = false
): Promise<SyncResponse> {
  const res = await fetch(`/api/annotations/sync`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ session_id: sessionId ?? null, dry_run: dryRun }),
  });
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.json();
}

// Types
export interface Annotation {
  id: string;
  timestamp: string;
  annotator: string;
  scope: { type: string; target_id: string };
  content: {
    category: string;
    tags: string[];
    title: string;
    detail: string;
  };
  taxonomy_mappings: {
    minitrace: string[];
    mast: string[];
    toolemu: string[];
  };
  classification?: string;
}

export interface CreateAnnotationPayload {
  annotator: string;
  scope_type: string;
  target_id: string;
  category: string;
  title: string;
  detail: string;
  tags: string[];
  taxonomy_minitrace: string[];
  taxonomy_mast: string[];
  taxonomy_toolemu: string[];
  classification?: string;
}

export interface SyncResponse {
  synced: string[];
  skipped: string[];
  errors: { session_id: string; error: string }[];
}
```

### 7.2 Annotation Panel in TranscriptViewer

```typescript
// web/src/components/TranscriptViewer/AnnotationPanel.tsx

import { useState, useEffect } from 'react';
import {
  getSessionAnnotations,
  createAnnotation,
  deleteAnnotation,
  type Annotation,
  type CreateAnnotationPayload,
} from '../../api/minitrace';

interface AnnotationPanelProps {
  sessionId: string;
  selectedScope?: { type: string; targetId: string };
  onClose: () => void;
}

const CATEGORY_COLORS: Record<string, string> = {
  observation: '#4a90d9',
  pattern: '#9b59b6',
  'ai-failure': '#e74c3c',
  recommendation: '#27ae60',
};

export function AnnotationPanel({ sessionId, selectedScope, onClose }: AnnotationPanelProps) {
  const [annotations, setAnnotations] = useState<Annotation[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);

  // Form state
  const [category, setCategory] = useState('observation');
  const [title, setTitle] = useState('');
  const [detail, setDetail] = useState('');
  const [tags, setTags] = useState('');
  const [taxonomyM, setTaxonomyM] = useState('');

  useEffect(() => {
    loadAnnotations();
  }, [sessionId]);

  async function loadAnnotations() {
    setLoading(true);
    try {
      const anns = await getSessionAnnotations(sessionId);
      setAnnotations(anns);
    } catch (e) {
      console.error('Failed to load annotations:', e);
    } finally {
      setLoading(false);
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const scopeType = selectedScope?.type ?? 'session';
    const targetId = selectedScope?.targetId ?? sessionId;

    const payload: CreateAnnotationPayload = {
      annotator: 'user',
      scope_type: scopeType,
      target_id: targetId,
      category,
      title,
      detail,
      tags: tags.split(',').map(t => t.trim()).filter(Boolean),
      taxonomy_minitrace: taxonomyM.split(',').map(t => t.trim()).filter(Boolean),
      taxonomy_mast: [],
      taxonomy_toolemu: [],
    };

    try {
      await createAnnotation(sessionId, payload);
      setShowForm(false);
      setTitle('');
      setDetail('');
      setTags('');
      setTaxonomyM('');
      await loadAnnotations();
    } catch (e) {
      console.error('Failed to create annotation:', e);
    }
  }

  async function handleDelete(annId: string) {
    if (!confirm('Delete this annotation?')) return;
    try {
      await deleteAnnotation(annId);
      await loadAnnotations();
    } catch (e) {
      console.error('Failed to delete annotation:', e);
    }
  }

  return (
    <div className="annotation-panel">
      <div className="panel-header">
        <h3>Annotations</h3>
        <button onClick={onClose}>×</button>
      </div>

      <button
        className="add-btn"
        onClick={() => setShowForm(!showForm)}
      >
        {showForm ? 'Cancel' : '+ Add Annotation'}
      </button>

      {showForm && (
        <form onSubmit={handleSubmit} className="annotation-form">
          <select value={category} onChange={e => setCategory(e.target.value)}>
            <option value="observation">Observation</option>
            <option value="pattern">Pattern</option>
            <option value="ai-failure">AI Failure</option>
            <option value="recommendation">Recommendation</option>
          </select>

          <input
            type="text"
            placeholder="Title"
            value={title}
            onChange={e => setTitle(e.target.value)}
            required
          />

          <textarea
            placeholder="Detail"
            value={detail}
            onChange={e => setDetail(e.target.value)}
          />

          <input
            type="text"
            placeholder="Tags (comma-separated, e.g. F-AUT, C-SEQ)"
            value={tags}
            onChange={e => setTags(e.target.value)}
          />

          <input
            type="text"
            placeholder="Taxonomy codes (e.g. F-AUT, F-INS)"
            value={taxonomyM}
            onChange={e => setTaxonomyM(e.target.value)}
          />

          {selectedScope && (
            <div className="scope-hint">
              Targeting: {selectedScope.type} / {selectedScope.targetId}
            </div>
          )}

          <button type="submit">Save Annotation</button>
        </form>
      )}

      <div className="annotation-list">
        {loading && <p>Loading...</p>}
        {!loading && annotations.length === 0 && <p>No annotations yet.</p>}
        {annotations.map(ann => (
          <div key={ann.id} className="annotation-item" style={{ borderLeftColor: CATEGORY_COLORS[ann.content.category] }}>
            <div className="ann-header">
              <span className="ann-category">{ann.content.category}</span>
              <span className="ann-scope">{ann.scope.type}</span>
              <button className="delete-btn" onClick={() => handleDelete(ann.id)}>×</button>
            </div>
            <div className="ann-title">{ann.content.title}</div>
            {ann.content.detail && <div className="ann-detail">{ann.content.detail}</div>}
            {ann.content.tags.length > 0 && (
              <div className="ann-tags">
                {ann.content.tags.map(t => (
                  <span key={t} className="tag">{t}</span>
                ))}
              </div>
            )}
            {ann.taxonomy_mappings.minitrace.length > 0 && (
              <div className="ann-taxonomy">
                {ann.taxonomy_mappings.minitrace.map(t => (
                  <span key={t} className="taxonomy-badge">{t}</span>
                ))}
              </div>
            )}
            <div className="ann-meta">
              {ann.annotator} · {new Date(ann.timestamp).toLocaleString()}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
```

### 7.3 Annotation Badges in Session Browser

In the session list (browser), add a badge showing annotation counts per category:

```typescript
// web/src/components/SessionBrowser/SessionBrowser.tsx — additions

// Fetch annotation counts alongside session data
async function loadSessions(): Promise<SessionSummary[]> {
  // ... existing load logic ...
}

// After loading sessions, fetch annotation counts
// Add to SessionSummary:
interface SessionSummary {
  id: string;
  // ... existing fields ...
  annotation_counts?: {
    total: number;
    ai_failure: number;
    observation: number;
  };
}

// Display in session row:
<div className="session-row">
  {/* ... existing fields ... */}
  {session.annotation_counts && (
    <div className="annotation-badges">
      {session.annotation_counts.ai_failure > 0 && (
        <span className="badge badge-failure">
          {session.annotation_counts.ai_failure} failure
        </span>
      )}
      {session.annotation_counts.observation > 0 && (
        <span className="badge badge-observation">
          {session.annotation_counts.observation} obs
        </span>
      )}
    </div>
  )}
</div>
```

---

## Part 8 — Directory Structure After Implementation

```
go-minitrace/
├── pkg/
│   ├── minitrace/                 # existing
│   ├── query/                    # existing
│   └── annotate/                  # NEW
│       ├── store.go               # SQLite CRUD store
│       ├── schema.go              # (optional) embed schema.sql here
│       ├── sync.go                # JSON write-back logic
│       └── duckdb.go              # DuckDB sqlite_scanner attachment
├── cmd/
│   └── go-minitrace/
│       ├── main.go                # register annotate command
│       └── cmds/
│           ├── common/            # existing
│           ├── serve/             # extend with annotation endpoints
│           ├── annotate/          # NEW
│           │   ├── root.go
│           │   ├── add.go
│           │   ├── list.go
│           │   ├── edit.go
│           │   ├── delete.go
│           │   ├── sync.go
│           │   └── import.go
│           ├── query/
│           ├── convert/
│           ├── validate/
│           └── discover/
├── queries/
│   ├── load.sql                   # existing
│   ├── annotations.sql            # existing
│   └── annotations.sql            # UPDATE (uses attached SQLite tables)
└── web/
    └── src/
        ├── api/
        │   └── minitrace.ts       # extend with annotation API calls
        └── components/
            ├── TranscriptViewer/
            │   └── AnnotationPanel.tsx  # NEW
            └── SessionBrowser/
                └── SessionBrowser.tsx   # extend with annotation badges
```

---

## Part 9 — Implementation Phases

### Phase 1: Core Store + CLI (1-2 days)

1. Create `pkg/annotate/` package with `store.go` — SQLite CRUD
2. Create `cmd/go-minitrace/cmds/annotate/` — `add`, `list`, `delete` commands
3. Wire into `main.go`
4. Manual testing: add/list/delete annotations from existing sessions
5. Verify with `sqlite3 output/annotations.db "SELECT * FROM annotations"`

**Deliverable:** Working CLI for creating and listing annotations, persisted in SQLite.

### Phase 2: Edit + Sync (1 day)

1. Add `edit` command
2. Create `sync.go` — atomic JSON writeback
3. Add `sync` command with `--dry-run`
4. Verify synced JSON validates with `go-minitrace validate`

**Deliverable:** Annotations can be written back to `.minitrace.json` files.

### Phase 3: DuckDB Integration (0.5 day)

1. Create `duckdb.go` — `AttachAnnotationsToDuckDB` (calls `INSTALL/LOAD sqlite_scanner` + `sqlite_attach`)
2. Wire into serve startup (after DuckDB connection, before sessions load)
3. Update `queries/annotations.sql` to join SQLite `annotations` table with `sessions_base`
4. Verify cross-session queries work (e.g., "all F-AUT failures across all sessions")

**Deliverable:** Annotations live in DuckDB via sqlite_scanner — no temp table refresh needed.

### Phase 4: HTTP API (0.5 day)

1. Add `annotate.Store` to `Server` struct
2. Add annotation CRUD handlers to serve
3. Test with `curl`:
   ```
   curl -X POST http://localhost:8080/api/sessions/SESSION_ID/annotations \
     -H 'Content-Type: application/json' \
     -d '{"category":"ai-failure","title":"Test","scope_type":"session","target_id":"SESSION_ID"}'
   ```

**Deliverable:** Annotations accessible via HTTP API.

### Phase 5: Web UI (1-2 days)

1. Add TypeScript types and API client functions
2. Create `AnnotationPanel` component
3. Integrate into `TranscriptViewerPage`
4. Add annotation count badges to `SessionBrowser`
5. Add cross-session annotation search to `QueryEditorPage`

**Deliverable:** Full annotation workflow in the browser.

### Phase 6: Polish (0.5 day)

1. Add `import` command for batch importing annotations from JSON
2. Add interactive annotation mode (TUI prompts for taxonomy selection)
3. Update `go-minitrace validate` to validate annotation structure
4. Write tests for `pkg/annotate/store.go`
5. Update README and documentation

---

## Part 10 — Testing Strategy

### Unit Tests

```go
// pkg/annotate/store_test.go

func TestStore_CRUD(t *testing.T) {
    tmpDir := t.TempDir()
    store, err := Open(context.Background(), tmpDir)
    if err != nil {
        t.Fatalf("Open: %v", err)
    }
    defer store.Close()

    // Add
    ann := minitrace.BuildAnnotation(
        "test-uuid",
        "user",
        "session",
        "sess-001",
        "observation",
        "Test title",
        "Test detail",
        []string{"test-tag"},
        nil,
    )
    if err := store.AddAnnotation(context.Background(), ann, "sess-001"); err != nil {
        t.Fatalf("AddAnnotation: %v", err)
    }

    // List
    rows, err := store.List(context.Background(), annotate.ListOptions{SessionID: "sess-001"})
    if err != nil {
        t.Fatalf("List: %v", err)
    }
    if len(rows) != 1 {
        t.Errorf("expected 1 annotation, got %d", len(rows))
    }

    // Delete
    if err := store.Delete(context.Background(), "test-uuid"); err != nil {
        t.Fatalf("Delete: %v", err)
    }
    rows, _ = store.List(context.Background(), annotate.ListOptions{})
    if len(rows) != 0 {
        t.Errorf("expected 0 annotations after delete, got %d", len(rows))
    }
}

func TestSync_Atomic(t *testing.T) {
    tmpDir := t.TempDir()
    sessionFile := filepath.Join(tmpDir, "test.minitrace.json")

    // Write a session file
    session := minitrace.Session{
        ID:             "sess-001",
        SchemaVersion:  minitrace.SchemaVersion,
        Profile:        "organic",
        Classification: "internal",
        Annotations:    []minitrace.Annotation{},
        // ... required fields ...
    }
    data, _ := json.Marshal(session)
    os.WriteFile(sessionFile, data, 0644)

    // Sync with annotations
    store, _ := Open(context.Background(), tmpDir)
    defer store.Close()

    ann := minitrace.BuildAnnotation(
        "ann-001", "user", "session", "sess-001",
        "observation", "Test", "", nil, nil,
    )

    index := map[string]string{"sess-001": sessionFile}
    report, err := store.SyncAll(context.Background(), index, annotate.SyncOptions{})
    if err != nil {
        t.Fatalf("SyncAll: %v", err)
    }
    if len(report.Errors) > 0 {
        t.Errorf("sync errors: %v", report.Errors)
    }

    // Verify file was updated
    updated, _ := os.ReadFile(sessionFile)
    var updatedSession minitrace.Session
    json.Unmarshal(updated, &updatedSession)

    if len(updatedSession.Annotations) != 1 {
        t.Errorf("expected 1 annotation in file, got %d", len(updatedSession.Annotations))
    }
}
```

### Integration Tests

```bash
# End-to-end test script

# 1. Create a test session file
TEST_DIR=$(mktemp -d)
cat > "$TEST_DIR/sess-001.minitrace.json" <<'EOF'
{
  "id": "sess-001",
  "schema_version": "minitrace-v0.2.0",
  "profile": "organic",
  "classification": "internal",
  "provenance": {"source_format": "test", "converted_at": "2026-04-04T00:00:00Z", "converter_version": "test"},
  "flags": {"for_research": false, "needs_cleaning": false, "contains_error": false, "contains_pii": false, "category": []},
  "environment": {"model": "test", "tools_enabled": [], "agent_framework": "test"},
  "timing": {"privacy_level": "full", "duration_seconds": 1.0},
  "turns": [{"index": 0, "role": "user", "source": null, "content": "Hello"}],
  "tool_calls": [],
  "annotations": [],
  "metrics": {"turn_count": 1, "tool_call_count": 0}
}
EOF

# 2. Add an annotation
go-minitrace annotate add \
  --output-dir "$TEST_DIR" \
  --session sess-001 \
  --category ai-failure \
  --title "Test failure" \
  --taxonomy-minitrace F-AUT \
  --annotator automated

# 3. Verify in SQLite
sqlite3 "$TEST_DIR/annotations.db" "SELECT id, category, title FROM annotations"

# 4. Sync to JSON
go-minitrace annotate sync \
  --output-dir "$TEST_DIR" \
  --archive-glob "$TEST_DIR/*.minitrace.json"

# 5. Verify JSON was updated
cat "$TEST_DIR/sess-001.minitrace.json" | python3 -m json.tool | grep -A5 annotations

# 6. Validate the updated JSON
go-minitrace validate --path "$TEST_DIR/sess-001.minitrace.json"
```

---

## Part 11 — API Reference Summary

### CLI Commands

| Command | Flags | Description |
|---------|-------|-------------|
| `annotate add` | `--session`, `--category`, `--title`, `--scope`, `--target-id`, `--detail`, `--tags`, `--taxonomy-minitrace`, `--annotator` | Create a new annotation |
| `annotate list` | `--session`, `--category`, `--annotator`, `--taxonomy`, `--limit`, `--format` | List annotations with filters |
| `annotate edit` | `--id`, `--title`, `--detail`, `--category`, `--tags`, `--taxonomy-minitrace` | Update an existing annotation |
| `annotate delete` | `--id` | Delete an annotation |
| `annotate sync` | `--archive-glob`, `--session`, `--dry-run` | Write annotations back to JSON files |
| `annotate import` | `--file` | Batch import annotations from JSON |

### HTTP Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/sessions/:id/annotations` | List all annotations for a session |
| `POST` | `/api/sessions/:id/annotations` | Create an annotation |
| `GET` | `/api/annotations` | List all annotations (with filters as query params) |
| `PUT` | `/api/annotations/:annId` | Update an annotation |
| `DELETE` | `/api/annotations/:annId` | Delete an annotation |
| `POST` | `/api/annotations/sync` | Sync all (or one) sessions to JSON |

---

## Part 12 — Failure Modes and Edge Cases

### Concurrent Writes

SQLite's WAL mode handles concurrent reads fine. For concurrent writes, SQLite serializes them automatically. If two processes try to write simultaneously, one waits up to 5 seconds (the busy_timeout we set), then fails with a "database is locked" error. For v1, this is acceptable — the annotation use case is single-user. If multi-user becomes important, consider moving to a server-based store (PostgreSQL, SQLite via a web service).

### Sync Race Condition

If an annotation is being edited while a sync is running:
1. Sync reads the annotation from SQLite
2. Another process updates it
3. Sync writes the old state back
4. The update is lost

Mitigation: Lock the session in SQLite during sync, or do sync as a single transaction. For v1, document that sync and concurrent edits are not safe simultaneously.

### Large Sessions

Multi-MB `.minitrace.json` files are loaded into memory during sync. For sessions with 1000+ tool calls, this could use significant memory. This is a known limitation — the JSON format itself doesn't support streaming updates.

### Classification Escalation

The spec says annotations can only escalate classification toward more restrictive. The `Update` method doesn't enforce this. Add validation:

```go
func validateClassificationEscalation(current, override string) error {
    order := map[string]int{
        "public": 0, "internal": 1, "confidential": 2, "customer-confidential": 3,
    }
    if order[override] < order[current] {
        return fmt.Errorf("classification can only escalate, not de-escalate: %s → %s", current, override)
    }
    return nil
}
```

### Empty Annotations Array

The minitrace spec says `annotations` defaults to `[]`. `json.Unmarshal` into a `[]Annotation` field that is `null` in the JSON produces a nil slice, which marshals to `null` in JSON output — not `[]`. Fix in `SyncSession`:

```go
func (s *Store) SyncSession(ctx context.Context, filePath string, annotations []minitrace.Annotation, dryRun bool) error {
    // ...
    session.Annotations = annotations
    // Ensure non-nil empty slice, so JSON output is [] not null
    if session.Annotations == nil {
        session.Annotations = []minitrace.Annotation{}
    }
    // ...
}
```

### Session Not in Index

During sync, the session index is built from the `--archive-glob`. If a session in SQLite has no matching file in the glob (e.g., the session was moved or deleted), it's skipped with a warning. The annotation is NOT deleted from SQLite.

---

## Open Questions

1. **SQLite dependency:** Adding `github.com/mattn/go-sqlite3` means compiling CGO code. Does the project already support CGO builds? If not, consider `modernc.org/sqlite` (pure Go) or `mattn/go-sqlite3` with CGO enabled by default.

2. **Multiple output directories:** If the user has sessions in multiple output dirs, they need multiple `annotations.db` files. Should we support a central annotations store? Or one per output root?

3. **Annotation versioning:** Should edits preserve the history of an annotation? (Add `previous_content` column or use a separate `annotation_history` table.)

4. **Automated annotation detection:** The spec mentions automated annotators. Should we include a built-in analyzer that scans sessions for known patterns (e.g., "write without preceding read" → F-ASM)? This would be a separate subcommand: `go-minitrace annotate analyze --session SESSION_ID`.

5. **Annotation templates:** Predefined annotation snippets for common patterns (e.g., "Over-autonomy", "Completion-bias"). Could be stored in a `~/.minitrace/annotation-templates.json` file.

6. **UUID generation:** Use `github.com/google/uuid`. Already a common dependency in Go projects.

7. **Web UI state:** Since annotations are live via sqlite_scanner (DuckDB queries SQLite directly on every request), no explicit refresh is needed. The web UI React component uses `useEffect` with `sessionId` as a dependency — on navigation, annotations are fetched fresh. No WebSocket or SWR cache invalidation needed for v1.

---

## Key Dependencies to Add

```go
// go.mod additions

require (
    github.com/mattn/go-sqlite3 v1.14.22    // SQLite driver
    github.com/google/uuid v1.6.0              // UUID generation
)
```
