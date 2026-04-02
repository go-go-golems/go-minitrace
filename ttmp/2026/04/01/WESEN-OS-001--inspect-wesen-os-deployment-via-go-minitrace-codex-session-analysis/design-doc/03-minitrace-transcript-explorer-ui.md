---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: ttmp/2026/04/01/WESEN-OS-001--inspect-wesen-os-deployment-via-go-minitrace-codex-session-analysis/scripts/10-human-blocks.sql
      Note: Block decomposition query - the core data model for the Transcript Viewer
    - Path: ttmp/2026/04/01/WESEN-OS-001--inspect-wesen-os-deployment-via-go-minitrace-codex-session-analysis/scripts/21-doc-creation-timeline.sql
      Note: Ticket/doc creation events - feeds artifact badges
    - Path: ttmp/2026/04/01/WESEN-OS-001--inspect-wesen-os-deployment-via-go-minitrace-codex-session-analysis/scripts/24-git-commits.sql
      Note: Commit extraction - feeds artifact badges in blocks
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Minitrace Transcript Explorer UI

## Executive Summary

A browser-based UI for interactively exploring converted minitrace session archives. The user can browse sessions, read full transcripts with collapsible tool-call detail, write and run ad-hoc DuckDB SQL queries, and manage a library of saved/preset queries — all against the same archive that `go-minitrace query duckdb` uses from the CLI.

The UI is served by the existing Go binary (`go-minitrace serve`) with an embedded SPA frontend. No external database — DuckDB runs in-process, loaded from the same `--archive-glob` the CLI uses.

---

## Problem Statement

The CLI pipeline (`discover → convert → query duckdb --sql`) is powerful for scripted analysis, but the human loop of "write query → scan results → adjust → drill into a session → read turns → go back to query" is painfully slow in a terminal. Specific friction points from the WESEN-OS-001 investigation:

1. **No way to read a transcript.** To see what happened in a session, you write a `CROSS JOIN UNNEST(turns)` query, pipe to JSON, pipe to Python, and scroll. There is no "just show me the conversation."
2. **No way to cross-check.** When a query surfaces an interesting session, you want to click into it and read the context. That requires a new query with the session ID hardcoded.
3. **Query iteration is slow.** Each query re-invocation loads the archive from disk. In the UI, the archive stays loaded in a single DuckDB connection.
4. **Saved queries are ad-hoc files.** We put them in `scripts/01-xxx.sql` by convention, but there's no way to browse, run, or organize them interactively.

---

## Proposed Solution

### Architecture

```
┌─────────────────────────────────────────────────────┐
│  Browser (React SPA)                                │
│                                                     │
│  ┌──────────┐ ┌──────────────┐ ┌────────────────┐  │
│  │ Session  │ │  Transcript  │ │  Query Editor  │  │
│  │ Browser  │ │  Viewer      │ │  + Results     │  │
│  └────┬─────┘ └──────┬───────┘ └───────┬────────┘  │
│       │              │                 │            │
│       └──────────────┴─────────────────┘            │
│                      │ REST / WebSocket             │
└──────────────────────┼──────────────────────────────┘
                       │
┌──────────────────────┼──────────────────────────────┐
│  go-minitrace serve  │                              │
│                      ▼                              │
│  ┌─────────────────────────────────┐                │
│  │  DuckDB (in-process, :memory:) │                │
│  │  loaded from --archive-glob    │                │
│  └─────────────────────────────────┘                │
│                                                     │
│  ┌──────────────────┐  ┌────────────────────────┐   │
│  │ Preset dir       │  │ User query dir         │   │
│  │ (--preset-dir)   │  │ (--query-dir)          │   │
│  │ read-only .sql   │  │ read-write .sql files  │   │
│  └──────────────────┘  └────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

### Invocation

```bash
go-minitrace serve \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset-dir ./presets \
  --query-dir ./my-queries \
  --port 8080
```

- `--archive-glob`: same glob as `query duckdb`, loaded once at startup.
- `--preset-dir`: directory of read-only `.sql` files shipped with go-minitrace or a project.
- `--query-dir`: directory where user-created queries are persisted. The UI writes here.
- `--port`: HTTP server port.

---

## Screen Designs

### Screen 1: Session Browser

The landing page. Shows all sessions in a filterable, sortable table. Clicking a row opens the Transcript Viewer.

```
┌─────────────────────────────────────────────────────────────────────┐
│  minitrace explorer                          [Sessions] [Query]    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Filter: [wesen-os_____________]  Framework: [all ▾]  Quality: [▾] │
│  Date range: [2026-03-18] → [2026-04-01]          Sort: [date ▾]  │
│                                                                     │
│  ┌───┬────────────┬────────┬──────────────────────┬─────┬─────┬───┐│
│  │   │ Date       │ Hours  │ Title                │Turns│Tools│ Q ││
│  ├───┼────────────┼────────┼──────────────────────┼─────┼─────┼───┤│
│  │ ● │ 03-22 20:47│  24.7h │ Read the docs in     │ 315 │1015 │ A ││
│  │   │            │  (3.1) │ geppetto and create   │     │     │   ││
│  │   │            │        │ a new ticket to bring  │     │     │   ││
│  │   │            │        │ wesen-os/ up to...     │     │     │   ││
│  ├───┼────────────┼────────┼──────────────────────┼─────┼─────┼───┤│
│  │ ● │ 03-28 22:29│  87.5h │ Let's work on ticket │1467 │3307 │ A ││
│  │   │            │ (12.3) │ NPM-PUBLISH-001.     │     │     │   ││
│  │   │            │        │ Read the guide and    │     │     │   ││
│  │   │            │        │ create detailed...    │     │     │   ││
│  ├───┼────────────┼────────┼──────────────────────┼─────┼─────┼───┤│
│  │ ● │ 04-01 14:02│   1.3h │ Work on SQLITE-FED-  │ 121 │ 300 │ A ││
│  │   │            │  (0.8) │ 001 docmgr ticket.   │     │     │   ││
│  └───┴────────────┴────────┴──────────────────────┴─────┴─────┴───┘│
│                                                                     │
│  86 sessions │ 943h wall / 151h active │ 16,566 turns              │
└─────────────────────────────────────────────────────────────────────┘
```

**Design notes:**
- Hours column shows `wall (active)` so you immediately see idle ratio.
- The filter box does full-text search across title AND first-turn content AND workdir, matching the multi-vector approach from the CLI analysis.
- The `●` indicator is color-coded by active%: green (>50%), yellow (10-50%), red (<10%).
- Click any row → opens Transcript Viewer for that session.
- Right-click → "Copy session ID", "Open in Query Editor with WHERE id = '...'".

### Screen 2: Transcript Viewer

The core reading experience. Shows the session as a conversation with collapsible tool-call blocks.

```
┌─────────────────────────────────────────────────────────────────────┐
│  ← Sessions    019d174c  Profile Migration          [Query] [JSON] │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─ Session Info ───────────────────────────────────────────────┐   │
│  │ Started: 2026-03-22 20:47  Duration: 24.7h (3.1h active)   │   │
│  │ Workdir: ~/workspaces/2026-03-02/os-openai-app-server       │   │
│  │ Turns: 315  Tools: 1015  Model: gpt-5.4                    │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌─ Block 1 ── 2026-03-22 20:47 ── 23 agent turns, 130 tools ──┐  │
│  │                                                               │  │
│  │  👤 USER                                                      │  │
│  │  Read the docs in geppetto and create a new ticket to bring   │  │
│  │  wesen-os/ up to the new profiles and profile registry        │  │
│  │  settings, and have it load the pinocchio config and profile  │  │
│  │  files (not wesen-os ones).                                   │  │
│  │                                                               │  │
│  │  Create the docmgr ticket, keep a diary. Then create a        │  │
│  │  detailed analysis / design / implementation guide...         │  │
│  │                                                               │  │
│  │  🤖 ASSISTANT                                                 │  │
│  │  Using ticket-research-docmgr-remarkable with diary support   │  │
│  │  because this is a ticketed research/doc deliverable...       │  │
│  │                                                               │  │
│  │  ▶ 🔧 exec_command: pwd                         [0.3s] ✓     │  │
│  │  ▶ 🔧 exec_command: ls -la                      [0.2s] ✓     │  │
│  │  ▶ 🔧 exec_command: cat geppetto/pkg/doc/...    [0.1s] ✓     │  │
│  │    ... (127 more tool calls)                     [expand all] │  │
│  │                                                               │  │
│  │  🤖 ASSISTANT                                                 │  │
│  │  I've confirmed the repo contains sibling geppetto,           │  │
│  │  pinocchio, and wesen-os trees. The next pass is...           │  │
│  │                                                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─ Block 2 ── 2026-03-23 02:36 ── 5.8h gap ── 2 turns, 4 ─────┐ │
│  │                                                               │  │
│  │  👤 USER                                                      │  │
│  │  how does this all impact go-go-os-chat?                      │  │
│  │                                                               │  │
│  │  🤖 ASSISTANT                                                 │  │
│  │  The impact is significant. go-go-os-chat still depends on    │  │
│  │  the legacy geppetto/pkg/profiles package...                  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  Blocks: [1] [2] [3] [4] [5] ... [34]     Jump to block: [__]     │
└─────────────────────────────────────────────────────────────────────┘
```

**Design notes:**
- The transcript is grouped into **blocks** (same as the human-blocks query), not flat turns. Each block starts with the user input and contains all agent turns until the next user input.
- Block headers show: timestamp, gap since previous block, agent turn count, tool call count.
- Tool calls are **collapsed by default** with a one-line summary: tool name, first argument, duration, success/fail. Click `▶` to expand and see full arguments + output.
- Long assistant responses are truncated with a "show more" toggle.
- **Artifact badges** inline: when a tool call is a `git commit`, `docmgr ticket create`, `docmgr doc add`, or a diary write, it gets a colored badge so you can scan for landmarks.
- Block navigation bar at the bottom for jumping. Also keyboard: `j`/`k` for next/prev block, `t` for tool call toggle.

### Expanded Tool Call

```
│  ▼ 🔧 exec_command                                  [2.1s] ✓     │
│  ┌──────────────────────────────────────────────────────────┐     │
│  │ cmd: git commit -m "refactor(profilechat): resolve       │     │
│  │      engine profiles with pinocchio runtime"             │     │
│  │ workdir: ~/workspaces/.../wesen-os/workspace-links/      │     │
│  │          go-go-os-frontend                               │     │
│  │                                                          │     │
│  │ output:                                                  │     │
│  │ [main 4a7b2c1] refactor(profilechat): resolve engine     │     │
│  │  profiles with pinocchio runtime                         │     │
│  │  3 files changed, 47 insertions(+), 89 deletions(-)     │     │
│  │                                                          │     │
│  │              [Copy cmd] [Copy output] [→ Query Editor]   │     │
│  └──────────────────────────────────────────────────────────┘     │
```

### Screen 3: Query Editor

A split-pane SQL editor (top) and results table (bottom). The left sidebar shows preset and saved queries.

```
┌─────────────────────────────────────────────────────────────────────┐
│  minitrace explorer                          [Sessions] [Query]    │
├──────────────┬──────────────────────────────────────────────────────┤
│              │                                                      │
│  PRESETS     │  ┌─ Query ──────────────────────────────────────┐   │
│  ──────────  │  │                                              │   │
│  📁 core     │  │  SELECT                                      │   │
│   session-   │  │    id,                                       │   │
│    list      │  │    timing->>'started_at' AS started,         │   │
│   framework- │  │    LEFT(title, 60) AS title,                 │   │
│    summary   │  │    CAST(metrics->>'turn_count' AS INT)       │   │
│   tool-op-   │  │      AS turns                                │   │
│    breakdown │  │  FROM sessions_base                           │   │
│              │  │  WHERE LOWER(title) LIKE '%wesen%'           │   │
│  📁 analysis │  │  ORDER BY timing->>'started_at'              │   │
│   human-     │  │                                              │   │
│    blocks    │  │                              [▶ Run] [Save]  │   │
│   autopilot  │  └──────────────────────────────────────────────┘   │
│   diary-     │                                                      │
│    writes    │  ┌─ Results (3 rows, 12ms) ─────────────────────┐   │
│   git-       │  │                                              │   │
│    commits   │  │  id         │started    │title        │turns │   │
│   docmgr-    │  │  ───────────┼───────────┼─────────────┼──────│   │
│    ops       │  │  019d174c.. │03-22 20:47│Read the docs│  315 │   │
│              │  │  019d376d.. │03-28 22:29│Let's work on│ 1467 │   │
│  SAVED       │  │  019d4a35.. │04-01 14:02│Work on SQLI │  121 │   │
│  ──────────  │  │                                              │   │
│  ★ my wesen  │  │  [Export CSV] [Export JSON] [→ Session]      │   │
│    filter    │  └──────────────────────────────────────────────┘   │
│  ★ ci debug  │                                                      │
│    pattern   │                                                      │
│              │                                                      │
│  [+ New]     │                                                      │
└──────────────┴──────────────────────────────────────────────────────┘
```

**Design notes:**

**Left sidebar — query library:**
- **Presets** (from `--preset-dir`): read-only `.sql` files organized in subdirectories. These ship with go-minitrace or are project-specific. Clicking loads the SQL into the editor.
- **Saved** (from `--query-dir`): user-created queries. The "Save" button writes the current editor content to a new `.sql` file in this directory. User names the file.
- Files are sorted alphabetically. Subdirectories become collapsible folders.
- Search/filter box at the top of the sidebar.

**Editor pane:**
- Syntax-highlighted SQL editor (CodeMirror or Monaco).
- `Ctrl+Enter` runs the query. `Ctrl+S` saves.
- **Template variables:** the editor supports `{{session_id}}` placeholders. When you navigate from Session Browser → Query Editor with a session, the variable is pre-filled.
- **Query history:** the last 50 executed queries are kept in browser local storage, accessible via a dropdown.

**Results pane:**
- Rendered as a sortable table.
- Columns with `id` values are clickable links → opens Transcript Viewer.
- `[Export CSV]` / `[Export JSON]` downloads the result set.
- `[→ Session]` appears when exactly one session ID is in the results — jumps to its transcript.
- For queries that return long text columns (e.g., turn content), the cell shows a truncated preview with a "show full" popover.

### Save Query Dialog

```
┌───────────────────────────────────────────┐
│  Save Query                               │
│                                           │
│  Name: [wesen-os-deploy-filter___________]│
│                                           │
│  Folder: [analysis ▾]                     │
│                                           │
│  Description (optional):                  │
│  [Find sessions referencing wesen-os in   ]│
│  [title, workdir, or first-turn content   ]│
│                                           │
│  Preview path:                            │
│  my-queries/analysis/wesen-os-deploy-     │
│  filter.sql                               │
│                                           │
│           [Cancel]  [Save]                │
└───────────────────────────────────────────┘
```

The saved file includes the description as a SQL comment header:

```sql
-- wesen-os-deploy-filter.sql
-- Find sessions referencing wesen-os in title, workdir,
-- or first-turn content
SELECT ...
```

### Screen 4: Cross-referencing — Query → Transcript → Query

The core workflow loop. This is the screen flow, not a separate screen:

```
  Session Browser                    Query Editor
  ┌──────────────┐                  ┌──────────────┐
  │ click row    │──────────────────│ click [→     │
  │              │                  │  Session]    │
  └──────┬───────┘                  └──────┬───────┘
         │                                 │
         ▼                                 ▼
  Transcript Viewer ◄──────────────────────┘
  ┌──────────────┐
  │ reading...   │
  │ see commit   │
  │ click [→     │──── opens Query Editor with
  │  Query       │     WHERE id = 'this session'
  │  Editor]     │     pre-filled
  └──────────────┘
```

Every entity is linked:
- Session IDs in query results → Transcript Viewer.
- "Open in Query Editor" from Transcript Viewer → Editor with session pre-filled.
- Tool call in transcript with `git commit` → badge is clickable, opens query showing all commits in this session.
- Tool call with `docmgr ticket create` → badge links to the ticket path.

---

## API Design

The Go backend exposes a small REST API:

### Endpoints

```
GET  /api/sessions
     Returns session list (same as session-list preset).
     Query params: ?q=wesen-os&from=2026-03-18&to=2026-04-01

GET  /api/sessions/:id
     Returns full session JSON (all turns, tool calls).

GET  /api/sessions/:id/blocks
     Returns the human-block decomposition (same as
     10-human-blocks.sql but computed server-side).

POST /api/query
     Body: { "sql": "SELECT ...", "params": {} }
     Runs arbitrary SQL against sessions_base.
     Returns: { "columns": [...], "rows": [...], "duration_ms": 12 }

GET  /api/presets
     Lists all .sql files from --preset-dir.
     Returns: [{ "name": "human-blocks", "folder": "analysis",
                 "path": "analysis/human-blocks.sql",
                 "description": "...", "sql": "..." }]

GET  /api/queries
     Lists all .sql files from --query-dir.
     Same shape as presets.

POST /api/queries
     Body: { "name": "my-filter", "folder": "analysis",
             "description": "...", "sql": "SELECT ..." }
     Saves a new .sql file to --query-dir.

PUT  /api/queries/:path
     Updates an existing saved query.

DELETE /api/queries/:path
     Deletes a saved query.
```

### Session block computation

The `/api/sessions/:id/blocks` endpoint is crucial. It runs the human-blocks decomposition server-side and returns:

```json
[
  {
    "block_num": 1,
    "user_turn_idx": 1,
    "user_ts": "2026-03-22T20:47:44Z",
    "user_content": "Read the docs in geppetto...",
    "agent_turns": 23,
    "tool_calls": 130,
    "gap_minutes": null,
    "turns": [
      {
        "idx": 1,
        "role": "user",
        "content": "...",
        "tool_calls_in_turn": []
      },
      {
        "idx": 2,
        "role": "assistant",
        "content": "Using ticket-research...",
        "tool_calls_in_turn": [
          {
            "id": "call_zPfJ...",
            "tool_name": "exec_command",
            "cmd": "pwd",
            "success": true,
            "duration_ms": 259,
            "output_preview": "/home/manuel/workspaces/...",
            "badges": []
          },
          ...
        ]
      }
    ],
    "artifacts": {
      "commits": ["refactor(profilechat): resolve..."],
      "tickets_created": ["APP-30-WESEN-OS-..."],
      "docs_added": ["Investigation diary (reference)"],
      "diary_writes": 1
    }
  },
  ...
]
```

The `artifacts` summary is computed by scanning tool calls in the block for known patterns (git commit, docmgr ticket create, docmgr doc add, diary file writes). This pre-computation means the frontend doesn't need to parse command strings.

The `badges` array on each tool call marks landmark operations: `"commit"`, `"ticket-create"`, `"doc-add"`, `"diary-write"`, `"error"` (for failed tool calls). The frontend renders these as colored pill badges.

---

## Design Decisions

### 1. DuckDB in-process, not a separate server

DuckDB is embedded in the Go process. The archive is loaded once at startup into an in-memory database. All queries run against this single connection. This means:
- Zero deployment complexity (no database to manage).
- Fast queries (the data is already in memory).
- The `POST /api/query` endpoint is just `db.Query(sql)` internally.
- Downside: the process memory footprint scales with archive size. For typical usage (hundreds of sessions), this is fine.

### 2. Preset dir is read-only, query dir is read-write

Presets ship with the tool or are committed to a repo. They should not be accidentally modified by the UI. User queries are a separate namespace that the UI can freely create/edit/delete.

Both directories use the same format: plain `.sql` files with optional `-- description` comment headers. No metadata database — the filesystem is the source of truth.

### 3. Block-based transcript, not flat turn list

The flat turn list is unusable for sessions with hundreds of turns. The block decomposition (group by user input) is the natural reading unit. The user scans block headers ("what did I ask? how much work followed?"), then expands interesting blocks.

### 4. Tool calls collapsed by default

A 200-tool-call block would be unreadable expanded. The default is a one-line summary per tool call. The user expands individual calls or clicks "expand all" when they want the full picture.

### 5. Badges for landmarks

Git commits, ticket creations, document additions, and diary writes are visually marked in the transcript. This lets you scan a long session and immediately see where the important state transitions happened, without reading every tool call.

---

## Alternatives Considered

### TUI instead of browser

A terminal UI (like `go-minitrace query duckdb --tui`) would avoid the web stack but:
- Cannot render long transcripts readably (terminal width constraints).
- Cannot do split-pane query editor + results.
- Cannot embed a real code editor for SQL.
- The Go ecosystem has good embedded web server support (embed SPA, serve from binary).

A TUI could still be useful as a lightweight `session-list → pick → show transcript` flow, but it's a different tool from the full query workbench.

### Electron / Tauri desktop app

Overkill. The Go binary already exists; serving a SPA on localhost is simpler and works everywhere.

### Jupyter / Observable notebook

Good for one-off analysis, bad for the "browse → drill → query → browse" loop. Notebooks are write-forward; the explorer needs random access.

---

## Implementation Plan

### Phase 1: Backend API + minimal session browser

1. Add `go-minitrace serve` command with `--archive-glob`, `--preset-dir`, `--query-dir`, `--port`.
2. Implement `/api/sessions`, `/api/sessions/:id`, `/api/query`.
3. Embed a minimal React SPA that renders the session browser table with filtering.
4. Ship presets from the existing `scripts/` directory in this ticket.

### Phase 2: Transcript Viewer

1. Implement `/api/sessions/:id/blocks` with artifact detection.
2. Build the block-based transcript viewer with collapsible tool calls and badges.
3. Add keyboard navigation (`j`/`k`, `t` for tool toggle).

### Phase 3: Query Editor + saved queries

1. Implement `/api/presets`, `/api/queries` (CRUD).
2. Embed CodeMirror with SQL syntax highlighting.
3. Build the split-pane editor + results view.
4. Add save/load/delete for user queries.
5. Cross-link: session ID columns in results → Transcript Viewer.

### Phase 4: Polish

1. Template variables (`{{session_id}}`).
2. Query history in local storage.
3. Export (CSV, JSON) from results pane.
4. URL routing so you can bookmark a session or query.
5. Dark mode (default).
