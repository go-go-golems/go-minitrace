---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: cmd/go-minitrace/cmds/query/duckdb.go
      Note: Existing DuckDB command - follow same Glazed pattern
    - Path: cmd/go-minitrace/main.go
      Note: Register new serve command here
    - Path: pkg/minitrace/schema.go
      Note: All Go types for Session
    - Path: pkg/query/assets.go
      Note: Built-in preset SQL files and ResolvePresetSQL
    - Path: pkg/query/engine.go
      Note: Existing DuckDB query engine - reuse OpenConnection
    - Path: ttmp/2026/04/01/WESEN-OS-001--inspect-wesen-os-deployment-via-go-minitrace-codex-session-analysis/design-doc/03-minitrace-transcript-explorer-ui.md
      Note: UI design doc - the API contract this backend implements
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Backend Implementation Guide: `go-minitrace serve`

## For the backend developer

This document gives you everything you need to implement the `go-minitrace serve` command that powers the Transcript Explorer UI. The frontend is already built in `web/` (React + MUI + RTK Query). Your job is the Go HTTP server that loads a minitrace archive into DuckDB, serves the REST API, and embeds the built frontend.

Read these before starting:
- **UI design doc:** `design-doc/03-minitrace-transcript-explorer-ui.md` — the full screen designs and API contract
- **Existing query engine:** `pkg/query/engine.go` — you will reuse `OpenConnection`, `LoadArchive`, and `RunIntoProcessor`
- **Existing DuckDB command:** `cmd/go-minitrace/cmds/query/duckdb.go` — the pattern you'll follow for CLI wiring
- **Schema types:** `pkg/minitrace/schema.go` — the Go types for Session, Turn, ToolCall, etc.

---

## 1. What already exists (do not rebuild)

### `pkg/query/engine.go`

The DuckDB query engine is already implemented. Key functions:

```go
// Opens a DuckDB connection (use ":memory:" for in-process)
func OpenConnection(ctx context.Context, dbPath string) (*sql.DB, *sql.Conn, error)

// Loads .minitrace.json files into a table via read_json()
func LoadArchive(ctx context.Context, conn *sql.Conn, opts LoadOptions) error

// Runs SQL and pipes rows into a Glazed processor
func RunIntoProcessor(ctx context.Context, conn *sql.Conn, sqlText string, gp middlewares.Processor) error
```

For the HTTP server you will NOT use `RunIntoProcessor` (that's the Glazed pipeline). Instead, call `conn.QueryContext(ctx, sql)` directly and marshal the rows to JSON yourself.

### `pkg/query/assets.go`

Built-in presets are embedded via `//go:embed presets/*.sql`. The `ResolvePresetSQL(name, tableName)` function returns the SQL for a preset name. You'll expose these through `GET /api/presets`.

### `pkg/minitrace/schema.go`

All Go types for Session, Turn, ToolCall, etc. are defined here. You'll use these for the `/api/sessions/:id` endpoint where you load and return a full session from the archive JSON file directly (not from DuckDB).

---

## 2. New code to write

### File layout

```
cmd/go-minitrace/cmds/serve/
  serve.go              # Cobra command + Glazed wiring
  server.go             # HTTP server, routes, handlers
  handlers_sessions.go  # /api/sessions, /api/sessions/:id, /api/sessions/:id/blocks
  handlers_query.go     # /api/query
  handlers_presets.go   # /api/presets, /api/queries (CRUD)
  blocks.go             # Block decomposition logic
  badges.go             # Artifact/badge detection for tool calls
  embed.go              # go:embed for frontend assets

pkg/serve/
  (nothing new — keep handlers in the cmd layer for now;
   extract to pkg/ later if they grow)
```

### 2.1 Cobra command: `serve.go`

Create a new Glazed command registered under the root. Follow the exact same pattern as `cmd/go-minitrace/cmds/query/duckdb.go`:

```go
type ServeSettings struct {
    ArchiveGlob string `glazed:"archive-glob"`
    PresetDir   string `glazed:"preset-dir"`
    QueryDir    string `glazed:"query-dir"`
    Port        int    `glazed:"port"`
    DBPath      string `glazed:"db-path"`
    TableName   string `glazed:"table-name"`
    DevMode     bool   `glazed:"dev"`
}
```

Flags:

| Flag | Type | Default | Description |
|---|---|---|---|
| `--archive-glob` | string | `./output/active/*/*.minitrace.json` | Same as `query duckdb` |
| `--preset-dir` | string | `""` | Directory of read-only `.sql` files (in addition to built-in presets) |
| `--query-dir` | string | `./queries` | Directory for user-saved `.sql` files (read-write) |
| `--port` | int | `8080` | HTTP listen port |
| `--db-path` | string | `:memory:` | DuckDB database path |
| `--table-name` | string | `sessions_base` | Table name |
| `--dev` | bool | `false` | Proxy to Vite dev server at :5173 instead of serving embedded assets |

The command should:
1. Open DuckDB connection
2. Load archive
3. Start HTTP server
4. Block until signal

```go
func (c *ServeCommand) Run(ctx context.Context, vals *values.Values) error {
    settings := &ServeSettings{}
    // decode...

    db, conn, err := queryengine.OpenConnection(ctx, settings.DBPath)
    // load archive...

    srv := NewServer(conn, settings)
    return srv.ListenAndServe(ctx, settings.Port)
}
```

Register in `cmd/go-minitrace/main.go` alongside the existing `query`, `convert`, `discover` commands.

### 2.2 HTTP server: `server.go`

Use the standard library `net/http` with a router. No framework needed — the API surface is small.

```go
type Server struct {
    conn      *sql.Conn
    tableName string
    presetDir string
    queryDir  string
    devMode   bool
    mux       *http.ServeMux
}

func NewServer(conn *sql.Conn, settings *ServeSettings) *Server {
    s := &Server{
        conn:      conn,
        tableName: settings.TableName,
        presetDir: settings.PresetDir,
        queryDir:  settings.QueryDir,
        devMode:   settings.DevMode,
    }
    s.mux = http.NewServeMux()
    s.routes()
    return s
}

func (s *Server) routes() {
    s.mux.HandleFunc("GET /api/sessions", s.handleGetSessions)
    s.mux.HandleFunc("GET /api/sessions/{id}", s.handleGetSession)
    s.mux.HandleFunc("GET /api/sessions/{id}/blocks", s.handleGetSessionBlocks)
    s.mux.HandleFunc("POST /api/query", s.handleExecuteQuery)
    s.mux.HandleFunc("GET /api/presets", s.handleGetPresets)
    s.mux.HandleFunc("GET /api/queries", s.handleGetQueries)
    s.mux.HandleFunc("POST /api/queries", s.handleSaveQuery)
    s.mux.HandleFunc("PUT /api/queries/{path...}", s.handleUpdateQuery)
    s.mux.HandleFunc("DELETE /api/queries/{path...}", s.handleDeleteQuery)

    // Frontend: serve embedded SPA or proxy to Vite dev server
    if s.devMode {
        s.mux.Handle("/", httputil.NewSingleHostReverseProxy(
            &url.URL{Scheme: "http", Host: "localhost:5173"},
        ))
    } else {
        s.mux.Handle("/", http.FileServer(http.FS(frontendFS)))
    }
}
```

### 2.3 Frontend embedding: `embed.go`

```go
package serve

import (
    "embed"
    "io/fs"
)

//go:embed all:frontend
var frontendEmbedFS embed.FS

var frontendFS, _ = fs.Sub(frontendEmbedFS, "frontend")
```

The Makefile / `go generate` step should build the frontend and copy it:

```makefile
.PHONY: frontend
frontend:
	cd web && npm ci && npm run build
	rm -rf cmd/go-minitrace/cmds/serve/frontend
	cp -r web/dist cmd/go-minitrace/cmds/serve/frontend
```

For the SPA fallback (all non-API, non-file routes → `index.html`), wrap the file server:

```go
func spaHandler(fsys fs.FS) http.Handler {
    fileServer := http.FileServer(http.FS(fsys))
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Try the file first
        f, err := fsys.Open(strings.TrimPrefix(r.URL.Path, "/"))
        if err != nil {
            // Fall back to index.html for SPA routing
            r.URL.Path = "/"
        } else {
            f.Close()
        }
        fileServer.ServeHTTP(w, r)
    })
}
```

---

## 3. API endpoint specifications

### `GET /api/sessions`

Returns the session list. Runs a SQL query internally (the `session-list` preset or equivalent).

**Response:** `[]SessionSummary`

```go
func (s *Server) handleGetSessions(w http.ResponseWriter, r *http.Request) {
    // Use the session-list preset SQL
    sqlText, _ := queryengine.ResolvePresetSQL("session-list", s.tableName)
    rows, err := s.conn.QueryContext(r.Context(), sqlText)
    // ... marshal rows to JSON
    // The frontend expects the shape defined in web/src/types/session.ts SessionSummary
}
```

However, the `session-list` preset returns flat columns. The frontend expects nested JSON (`timing: {...}`, `metrics: {...}`). You have two options:

**Option A (recommended):** Write a custom SQL query that returns the full JSON blobs:

```sql
SELECT
  id, title, summary, classification,
  timing, metrics, environment, operational_context
FROM sessions_base
ORDER BY timing->>'started_at'
```

Then in Go, unmarshal the JSON columns (`timing`, `metrics`, etc.) into the nested response struct. DuckDB returns JSON columns as strings.

**Option B:** Load all `.minitrace.json` files from disk and return them as summaries. This is more reliable but slower.

**Query parameter support** (nice-to-have, not required for v1):
- `?q=wesen-os` — full-text filter
- `?from=2026-03-18&to=2026-04-01` — date range

### `GET /api/sessions/{id}`

Returns the full session detail including all turns and tool calls. This should **read the original `.minitrace.json` file from disk**, not query DuckDB, because DuckDB's JSON column storage truncates/normalizes content.

```go
func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")

    // Step 1: find the file path from DuckDB
    row := s.conn.QueryRowContext(r.Context(),
        "SELECT provenance->>'source_path' FROM "+s.tableName+" WHERE id = $1",
        sessionID)
    var sourcePath string
    row.Scan(&sourcePath)

    // Step 2: read the .minitrace.json file
    data, _ := os.ReadFile(sourcePath)
    var session minitrace.Session
    json.Unmarshal(data, &session)

    // Step 3: compute blocks and return
    blocks := ComputeBlocks(session)
    // ... marshal response
}
```

Wait — the `source_path` in provenance points to the **original JSONL** (e.g., `~/.codex/sessions/...`), not the converted `.minitrace.json`. You need the **converted file path**. Two options:

1. Query the archive glob for the matching session ID file.
2. Store the output path in the session or derive it from the archive glob + session ID.

The simplest approach: the `manifest.json` written by `convert` or just scan the archive glob for a file containing the session ID:

```go
// During server init, build an index: session ID → minitrace JSON file path
func buildSessionIndex(archiveGlob string) (map[string]string, error) {
    files, _ := filepath.Glob(archiveGlob)
    index := make(map[string]string)
    for _, f := range files {
        // filename is like "019d174c-fc68-7c00-8f1b-7fcc067c1fd6.minitrace.json"
        base := filepath.Base(f)
        id := strings.TrimSuffix(base, ".minitrace.json")
        index[id] = f
    }
    return index, nil
}
```

### `GET /api/sessions/{id}/blocks`

Returns the block decomposition for a session. This is the most complex endpoint.

**Response:** `[]SessionBlock` (see `web/src/types/session.ts`)

The block computation logic:

```go
// blocks.go
func ComputeBlocks(session minitrace.Session) []SessionBlock {
    // 1. Walk turns, grouping by user input boundaries
    // 2. For each block, count agent turns and tool calls
    // 3. Compute gap_minutes from previous block's user timestamp
    // 4. Detect artifacts (commits, tickets, diary writes)

    blocks := []SessionBlock{}
    var currentBlock *SessionBlock

    // Build a tool call lookup: id → ToolCall
    tcByID := map[string]minitrace.ToolCall{}
    for _, tc := range session.ToolCalls {
        tcByID[tc.ID] = tc
    }

    for _, turn := range session.Turns {
        if turn.Role == "user" {
            // Close previous block
            if currentBlock != nil {
                blocks = append(blocks, *currentBlock)
            }
            // Start new block
            currentBlock = &SessionBlock{
                BlockNum:    len(blocks) + 1,
                UserTurnIdx: turn.Index,
                UserTs:      deref(turn.Timestamp),
                UserContent: turn.Content,
                // gap computed after
            }
        }
        if currentBlock == nil {
            continue
        }
        // Add turn to current block
        blockTurn := convertTurn(turn, tcByID)
        currentBlock.Turns = append(currentBlock.Turns, blockTurn)

        if turn.Role != "user" {
            currentBlock.AgentTurns++
        }
        currentBlock.ToolCalls += len(turn.ToolCallsInTurn)
    }
    if currentBlock != nil {
        blocks = append(blocks, *currentBlock)
    }

    // Compute gaps
    for i := 1; i < len(blocks); i++ {
        prev := parseTime(blocks[i-1].UserTs)
        curr := parseTime(blocks[i].UserTs)
        gap := curr.Sub(prev).Minutes()
        blocks[i].GapMinutes = &gap
    }

    // Detect artifacts per block
    for i := range blocks {
        blocks[i].Artifacts = detectArtifacts(blocks[i], tcByID)
    }

    return blocks
}
```

### `POST /api/query`

Runs arbitrary SQL against the loaded DuckDB table. This is the query editor backend.

**Request:** `{ "sql": "SELECT ..." }`

**Response:** `{ "columns": [...], "rows": [...], "duration_ms": 12, "row_count": 6 }`

```go
func (s *Server) handleExecuteQuery(w http.ResponseWriter, r *http.Request) {
    var req struct{ SQL string `json:"sql"` }
    json.NewDecoder(r.Body).Decode(&req)

    start := time.Now()
    rows, err := s.conn.QueryContext(r.Context(), req.SQL)
    if err != nil {
        // Return error in a structured way the frontend expects
        json.NewEncoder(w).Encode(map[string]any{
            "columns": []string{}, "rows": []any{},
            "duration_ms": time.Since(start).Milliseconds(),
            "row_count": 0,
            "error": map[string]string{"message": err.Error()},
        })
        return
    }
    defer rows.Close()

    columns, _ := rows.Columns()
    var resultRows []map[string]any

    for rows.Next() {
        values := make([]any, len(columns))
        scanArgs := make([]any, len(columns))
        for i := range scanArgs { scanArgs[i] = &values[i] }
        rows.Scan(scanArgs...)

        row := make(map[string]any)
        for i, col := range columns {
            row[col] = queryengine.NormalizeValue(values[i])
        }
        resultRows = append(resultRows, row)
    }

    json.NewEncoder(w).Encode(map[string]any{
        "columns":     columns,
        "rows":        resultRows,
        "duration_ms": time.Since(start).Milliseconds(),
        "row_count":   len(resultRows),
    })
}
```

**Security note:** This runs arbitrary SQL. In v1 this is fine (local tool, single user). If you ever expose this over a network, add query timeout, row limits, and read-only mode.

### `GET /api/presets`

Returns all available preset queries: the built-in ones from `pkg/query/presets/*.sql` plus any files in `--preset-dir`.

**Response:** `[]SavedQuery`

```go
func (s *Server) handleGetPresets(w http.ResponseWriter, r *http.Request) {
    var presets []SavedQuery

    // 1. Built-in presets from pkg/query/assets.go
    for _, name := range queryengine.ListPresets() {
        sql, _ := queryengine.ResolvePresetSQL(name, s.tableName)
        presets = append(presets, SavedQuery{
            Name:     name,
            Folder:   "built-in",
            Path:     "built-in/" + name + ".sql",
            SQL:      sql,
            Readonly: true,
            Description: extractSQLComment(sql), // first line -- comment
        })
    }

    // 2. External presets from --preset-dir
    if s.presetDir != "" {
        presets = append(presets, loadSQLDir(s.presetDir, true)...)
    }

    json.NewEncoder(w).Encode(presets)
}
```

### `GET /api/queries` + `POST /api/queries` + `PUT` + `DELETE`

CRUD for the `--query-dir`. Files are plain `.sql` on disk. The description is extracted from the first `-- comment` line.

```go
func loadSQLDir(dir string, readonly bool) []SavedQuery {
    var queries []SavedQuery
    filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
        if d.IsDir() || !strings.HasSuffix(path, ".sql") { return nil }
        content, _ := os.ReadFile(path)
        rel, _ := filepath.Rel(dir, path)
        folder := filepath.Dir(rel)
        if folder == "." { folder = "" }
        name := strings.TrimSuffix(filepath.Base(path), ".sql")
        queries = append(queries, SavedQuery{
            Name:        name,
            Folder:      folder,
            Path:        rel,
            SQL:         string(content),
            Readonly:    readonly,
            Description: extractSQLComment(string(content)),
        })
        return nil
    })
    return queries
}

func extractSQLComment(sql string) string {
    // Extract first -- comment line as description
    for _, line := range strings.Split(sql, "\n") {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "-- ") {
            return strings.TrimPrefix(line, "-- ")
        }
        if line != "" && !strings.HasPrefix(line, "--") {
            break
        }
    }
    return ""
}
```

For `POST /api/queries`, write a new `.sql` file:

```go
func (s *Server) handleSaveQuery(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Name        string `json:"name"`
        Folder      string `json:"folder"`
        Description string `json:"description"`
        SQL         string `json:"sql"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    // Sanitize name (alphanumeric + hyphens only)
    safeName := sanitizeFilename(req.Name)
    dir := filepath.Join(s.queryDir, req.Folder)
    os.MkdirAll(dir, 0755)
    path := filepath.Join(dir, safeName+".sql")

    content := "-- " + req.Description + "\n" + req.SQL
    os.WriteFile(path, []byte(content), 0644)

    rel, _ := filepath.Rel(s.queryDir, path)
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(SavedQuery{
        Name:        safeName,
        Folder:      req.Folder,
        Path:        rel,
        SQL:         req.SQL,
        Readonly:    false,
        Description: req.Description,
    })
}
```

---

## 4. Badge / artifact detection: `badges.go`

This is the logic that tags tool calls with landmark badges. The frontend renders these as colored chips.

```go
type BadgeType string

const (
    BadgeCommit       BadgeType = "commit"
    BadgeTicketCreate BadgeType = "ticket-create"
    BadgeDocAdd       BadgeType = "doc-add"
    BadgeDiaryWrite   BadgeType = "diary-write"
    BadgeError        BadgeType = "error"
)

func DetectBadges(tc minitrace.ToolCall) []BadgeType {
    var badges []BadgeType

    cmd := ""
    if tc.Input.Command != nil {
        cmd = *tc.Input.Command
    }
    // Also check arguments.cmd (Codex puts it there)
    if cmd == "" && tc.Input.Arguments != nil {
        if argMap, ok := tc.Input.Arguments.(map[string]any); ok {
            if c, ok := argMap["cmd"].(string); ok {
                cmd = c
            }
        }
    }

    cmdLower := strings.ToLower(cmd)

    if !tc.Output.Success {
        badges = append(badges, BadgeError)
    }
    if strings.Contains(cmd, "git commit") {
        badges = append(badges, BadgeCommit)
    }
    if strings.Contains(cmdLower, "docmgr ticket create") {
        badges = append(badges, BadgeTicketCreate)
    }
    if strings.Contains(cmdLower, "docmgr doc add") {
        badges = append(badges, BadgeDocAdd)
    }
    if (strings.Contains(cmdLower, "diary") || strings.Contains(cmdLower, "diary.md")) &&
       (strings.Contains(cmdLower, "apply_patch") ||
        strings.Contains(cmdLower, "cat >") ||
        strings.Contains(cmdLower, "tee ")) {
        badges = append(badges, BadgeDiaryWrite)
    }

    return badges
}

// Aggregate artifact summary for a block
func DetectBlockArtifacts(block SessionBlock, tcByID map[string]minitrace.ToolCall) BlockArtifacts {
    var arts BlockArtifacts
    for _, turn := range block.Turns {
        for _, tcID := range turn.ToolCallsInTurn {
            tc, ok := tcByID[tcID]
            if !ok { continue }
            badges := DetectBadges(tc)
            for _, b := range badges {
                switch b {
                case BadgeCommit:
                    msg := extractCommitMessage(tc)
                    if msg != "" { arts.Commits = append(arts.Commits, msg) }
                case BadgeTicketCreate:
                    ticket := extractTicketID(tc)
                    if ticket != "" { arts.TicketsCreated = append(arts.TicketsCreated, ticket) }
                case BadgeDocAdd:
                    doc := extractDocTitle(tc)
                    if doc != "" { arts.DocsAdded = append(arts.DocsAdded, doc) }
                case BadgeDiaryWrite:
                    arts.DiaryWrites++
                }
            }
        }
    }
    return arts
}
```

---

## 5. Dev mode: two-process workflow

When `--dev` is passed, the Go server proxies all non-`/api` requests to the Vite dev server:

```
Terminal 1:  cd web && npm run dev          # Vite at :5173 with HMR
Terminal 2:  go run ./cmd/go-minitrace serve --dev --archive-glob '...'  # API at :8080
```

The frontend's `vite.config.ts` should proxy `/api` to `:8080`:

```ts
// web/vite.config.ts
export default defineConfig({
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
});
```

Or alternatively, the Go server at `:8080` proxies non-`/api` to `:5173`. Either direction works. The Go-side proxy is already shown above. Pick whichever is simpler to wire — the Vite proxy approach is probably easier since you just add it to `vite.config.ts` and run the Go server normally.

---

## 6. Response types (Go structs)

Match these to `web/src/types/session.ts` and `web/src/types/query.ts`:

```go
// handlers_sessions.go

type SessionSummaryResponse struct {
    ID                 string                    `json:"id"`
    Title              string                    `json:"title"`
    Summary            *string                   `json:"summary"`
    Classification     string                    `json:"classification"`
    Timing             minitrace.Timing          `json:"timing"`
    Metrics            minitrace.Metrics         `json:"metrics"`
    Environment        minitrace.Environment     `json:"environment"`
    OperationalContext minitrace.OperationalContext `json:"operational_context"`
}

type SessionDetailResponse struct {
    SessionSummaryResponse
    Provenance minitrace.Provenance `json:"provenance"`
    Blocks     []SessionBlock       `json:"blocks"`
}

type SessionBlock struct {
    BlockNum    int             `json:"block_num"`
    UserTurnIdx int            `json:"user_turn_idx"`
    UserTs      string          `json:"user_ts"`
    UserContent string          `json:"user_content"`
    AgentTurns  int             `json:"agent_turns"`
    ToolCalls   int             `json:"tool_calls"`
    GapMinutes  *float64        `json:"gap_minutes"`
    Turns       []TurnResponse  `json:"turns"`
    Artifacts   BlockArtifacts  `json:"artifacts"`
}

type TurnResponse struct {
    Idx             int                `json:"idx"`
    Role            string             `json:"role"`
    Source          string             `json:"source"`
    Content         string             `json:"content"`
    Timestamp       string             `json:"timestamp"`
    ToolCallsInTurn []ToolCallResponse `json:"tool_calls_in_turn"`
}

type ToolCallResponse struct {
    ID            string          `json:"id"`
    ToolName      string          `json:"tool_name"`
    Timestamp     string          `json:"timestamp"`
    OperationType string          `json:"operation_type"`
    Input         ToolCallInput   `json:"input"`
    Output        ToolCallOutput  `json:"output"`
    Badges        []BadgeType     `json:"badges"`
}

type ToolCallInput struct {
    Command   string         `json:"command,omitempty"`
    Arguments map[string]any `json:"arguments,omitempty"`
    FilePath  string         `json:"file_path,omitempty"`
}

type ToolCallOutput struct {
    Success    bool    `json:"success"`
    Result     *string `json:"result"`
    Error      *string `json:"error"`
    DurationMs int     `json:"duration_ms"`
    Truncated  bool    `json:"truncated"`
}

type BlockArtifacts struct {
    Commits        []string `json:"commits"`
    TicketsCreated []string `json:"tickets_created"`
    DocsAdded      []string `json:"docs_added"`
    DiaryWrites    int      `json:"diary_writes"`
}

// handlers_query.go

type QueryRequest struct {
    SQL string `json:"sql"`
}

type QueryResponse struct {
    Columns    []string         `json:"columns"`
    Rows       []map[string]any `json:"rows"`
    DurationMs int64            `json:"duration_ms"`
    RowCount   int              `json:"row_count"`
    Error      *QueryError      `json:"error,omitempty"`
}

type QueryError struct {
    Message string `json:"message"`
}

// handlers_presets.go

type SavedQuery struct {
    Name        string `json:"name"`
    Folder      string `json:"folder"`
    Path        string `json:"path"`
    Description string `json:"description"`
    SQL         string `json:"sql"`
    Readonly    bool   `json:"readonly"`
}
```

---

## 7. Implementation order

Work in this order so you can test each piece independently:

### Phase 1: Skeleton + query endpoint (testable immediately)

1. Create `cmd/go-minitrace/cmds/serve/serve.go` with the Cobra command and settings.
2. Create `server.go` with the mux and `POST /api/query` handler.
3. Register the `serve` command in `main.go`.
4. Test: `go run ./cmd/go-minitrace serve --archive-glob '...'` + `curl -X POST localhost:8080/api/query -d '{"sql":"SELECT COUNT(*) FROM sessions_base"}'`

### Phase 2: Session list + detail

1. Implement `GET /api/sessions` — query DuckDB, return nested JSON.
2. Implement the session file index (`buildSessionIndex`).
3. Implement `GET /api/sessions/{id}` — load `.minitrace.json` from disk, return full session.
4. Test: `curl localhost:8080/api/sessions | jq length`

### Phase 3: Blocks

1. Implement `blocks.go` — `ComputeBlocks()` function.
2. Implement `badges.go` — `DetectBadges()` and `DetectBlockArtifacts()`.
3. Implement `GET /api/sessions/{id}/blocks`.
4. Test: `curl localhost:8080/api/sessions/019d174c-.../blocks | jq '.[0].artifacts'`

### Phase 4: Presets + saved queries

1. Implement `GET /api/presets` — merge built-in + preset-dir.
2. Implement `GET /api/queries` + `POST /api/queries` — read/write query-dir.
3. Test: `curl localhost:8080/api/presets | jq '.[].name'`

### Phase 5: Frontend embedding

1. Add `embed.go` with `//go:embed all:frontend`.
2. Add SPA fallback handler.
3. Add Makefile target to build frontend + copy to embed dir.
4. Add `--dev` mode proxy.
5. Test: `make frontend && go run ./cmd/go-minitrace serve --archive-glob '...'` → open browser.

---

## 8. Testing strategy

- **Unit tests for `blocks.go`:** Create a `minitrace.Session` with known turns, verify block boundaries, gap computation, and artifact detection. This is the most complex logic.
- **Unit tests for `badges.go`:** Test each badge pattern against sample tool calls.
- **Integration test for `/api/query`:** Load a small fixture archive, run a query, verify column/row structure.
- **No need to test `/api/sessions` separately** — it's a thin SQL + marshal layer.
- **Manual test:** Run against the real archive from the WESEN-OS-001 investigation (`/tmp/minitrace-output/active/*/*.minitrace.json`) and verify the UI works end-to-end.

---

## 9. Gotchas from the existing codebase

1. **DuckDB connection is single-threaded.** `engine.go` sets `SetMaxOpenConns(1)`. All HTTP handlers share one `*sql.Conn`. For v1 this is fine (single user), but if you need concurrency later, you'll need a connection pool or mutex.

2. **`NormalizeValue` in `engine.go`** converts `[]byte` to string and `time.Time` to RFC3339. Reuse this in your query handler.

3. **The `ToolCallsInTurn` field on `Turn`** is `[]string` (tool call IDs), not the full tool call objects. You need to join with the session's `ToolCalls` array to get the full details. The session file index makes this easy — load the whole session and build a map.

4. **`Input.Arguments` is `any`** (could be `map[string]any` or `nil`). Always type-assert carefully. Codex puts the command in `arguments.cmd`, not in `Input.Command`.

5. **The archive glob** may contain thousands of files. The `buildSessionIndex` function should be called once at startup, not per request.

6. **Session files can be large.** A 1467-turn session with 3307 tool calls is ~5-10 MB as JSON. The `/api/sessions/{id}` and `/api/sessions/{id}/blocks` endpoints should stream or set appropriate Content-Length.
