# go-minitrace

A transcript analysis toolkit that converts AI agent sessions into queryable archives and lets you build reusable analysis tooling with SQL and JavaScript.

**The core problem it solves:** when something breaks in a codebase you've worked on for months, the question is rarely "what changed?" — it's "what did this look like when it worked, and which sessions show the transition?" go-minitrace treats transcript analysis as a **reduction pipeline**, not a reading task.

---

## What it does

go-minitrace converts raw session stores into structured `.minitrace.json` archives, then provides DuckDB-backed querying, reusable structured commands, and a web UI to explore the results.

Supported sources: Claude Code, Codex, Pi, claude.ai, ChatGPT, Geppetto/Pinocchio.

## Installation

```bash
# Homebrew (recommended — includes the embedded web UI)
brew tap go-go-golems/go-go-go
brew install go-minitrace

# Go install (NOTE: the embedded web UI will be missing;
# use Homebrew or build from source for the full experience)
go install github.com/go-go-golems/go-minitrace/cmd/go-minitrace@latest

# From source (includes the embedded web UI)
git clone https://github.com/go-go-golems/go-minitrace.git
cd go-minitrace
go generate ./...   # builds the frontend with Dagger (requires Docker)
go build ./...
```

## Quick start

```bash
# Convert sessions into queryable archives
go-minitrace convert claude-code --output-dir ./output
go-minitrace convert pi --output-dir ./output

# Query with built-in presets (no SQL required)
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset session-list

# Run a structured command
go-minitrace query commands overview session-list \
  --archive-glob './output/active/*/*.minitrace.json'

# Start the web UI
go-minitrace serve --archive-glob './output/active/*/*.minitrace.json'
```

---

## Why this exists

A common mistake when debugging with historical sessions is treating every transcript as equally important. That leads to two failures:

1. **Token waste** — too much raw material, too little structure.
2. **Reasoning waste** — the investigator spends time rediscovering the same filters, archive globs, and path patterns instead of asking progressively better questions.

go-minitrace is most effective when treated as a **small analysis framework**, not a one-off SQL runner. The recommended workflow is a three-layer reduction funnel:

```
Native session stores    →    .minitrace.json archives    →    DuckDB table
                                (queryable)                     (structured)
                                                                 │
                              ┌─────────────────┐                │
                              │  SQL leaves     │◀───────────────┘
                              │  (filter/search)│
                              └────────┬────────┘
                                       │
                              ┌────────▼────────┐
                              │  JS summarizers │
                              │  (explain to    │
                              │   humans)       │
                              └────────┬────────┘
                                       │
                              ┌────────▼────────┐
                              │  Compact        │
                              │  evidence /     │
                              │  next hypothesis│
                              └─────────────────┘
```

1. **Session inventory** — find the 4 sessions that matter, not the 400 that don't.
2. **Evidence extraction** — pull only the tool calls that carry actual signal (bash output, file touches, git history).
3. **Summarization** — turn raw rows into a chronological narrative a human can act on.

Every stage reduces entropy. The end result is not a dump — it's a report.

---

## The three query layers

### 1. Built-in presets (no SQL required)

Six presets ship with go-minitrace for common questions:

| Preset | Description |
|--------|-------------|
| `session-list` | One row per session: framework, model, turns, tools, duration |
| `framework-summary` | Aggregate stats grouped by framework |
| `tool-operation-breakdown` | Tool call counts by framework and operation type |
| `timing-analysis` | Duration, active time, TTFA, and idle ratio by framework |
| `read-ratio-distribution` | Read/write/execute breakdown per session |
| `annotations` | All annotations across sessions |

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset framework-summary
```

### 2. Custom SQL

When a preset doesn't fit, write SQL directly. The table schema exposes the full session structure:

```sql
-- Sessions with highest tool-call density (tools per turn)
SELECT id, title,
       CAST(metrics->>'tool_call_count' AS INT) AS tools,
       CAST(metrics->>'turn_count' AS INT) AS turns,
       CAST(metrics->>'tool_call_count' AS DOUBLE) /
         CAST(metrics->>'turn_count' AS DOUBLE) AS density
FROM sessions_base
WHERE CAST(metrics->>'turn_count' AS INT) > 0
ORDER BY density DESC
LIMIT 10;

-- Tool failures across all sessions
SELECT id, title,
       tc->>'tool_name' AS tool,
       tc->'output'->>'error' AS error
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE (COALESCE(tc->'output'->>'success', 'true')) = 'false';

-- Bash output containing a specific keyword (e.g. a runtime error)
SELECT id AS session_id, title, timing->>'started_at' AS started_at,
       tc->>'id' AS call_id,
       json_extract_string(tc, '$.input.command') AS bash_command,
       json_extract_string(tc, '$.output.result') AS bash_output
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE (tc->>'tool_name') = 'bash'
  AND json_extract_string(tc, '$.output.result') LIKE '%Connect successful%'
ORDER BY started_at, session_id
LIMIT 50;
```

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "SELECT id, title,
               CAST(metrics->>'tool_call_count' AS INT) AS tools
        FROM sessions_base
        WHERE (environment->>'agent_framework') = 'pi'
        ORDER BY tools DESC
        LIMIT 20"
```

Output formats: `--output table` (default), `--output json`, `--output csv`.

### 3. Structured query commands (reusable tooling)

When an analysis task becomes part of your repeatable workflow, promote it to a structured command. Structured commands give you named, typed parameters, alias support, and work identically in CLI, web UI, and API.

A command file is a `.sql` or `.js` file with metadata:

```sql
/* sqleton
name: session-list
short: List minitrace sessions
flags:
  - name: framework
    type: stringList
    help: Filter by agent framework
  - name: limit
    type: int
    default: 100
    help: Limit the number of rows returned
*/
SELECT
  id,
  environment->>'agent_framework' AS framework,
  title,
  CAST(metrics->>'turn_count' AS INT) AS turns,
  CAST(metrics->>'tool_call_count' AS INT) AS tools
FROM {{TABLE_NAME}}
WHERE 1=1
{{ if .framework -}}
AND (environment->>'agent_framework') IN ({{ .framework | sqlStringIn }})
{{ end -}}
ORDER BY timing->>'started_at' DESC
LIMIT {{ .limit }};
```

Repository layout maps directly to CLI structure:

```
query-commands/
└── overview/
    ├── session-list.sql           → go-minitrace query commands overview session-list
    └── framework-summary.sql     → go-minitrace query commands overview framework-summary
```

Run with typed parameters:

```bash
go-minitrace query commands overview session-list \
  --archive-glob './output/active/*/*.minitrace.json' \
  --framework codex,pi \
  --limit 50
```

Load external repositories for team or investigation-specific commands:

```bash
go-minitrace query commands overview session-list \
  --query-repository ./my-commands \
  --archive-glob './output/active/*/*.minitrace.json'
```

Configure repositories via `GO_MINITRACE_QUERY_REPOSITORIES` or in the app config:

```yaml
queryRepositories:
  - ./query-commands/team
  - ~/.config/go-minitrace/query-commands
```

---

## JavaScript command handlers

For analysis tasks that go beyond what SQL can express conveniently — multi-query joins, JS-side scoring, classification logic, async control — write a JS handler.

JS handlers run in a Goja-powered runtime with access to:

- `require("minitrace")` — DuckDB queries (`mt.query()`, `mt.queryOne()`)
- `require("timer")` — `setTimeout`, `setInterval` for async commands
- `require("fs")`, `require("exec")`, `require("path")` — file system and shell access
- `require("console")` — debug output to server logs

```js
// overview/session-tools.js
__section__("filters", {
  title: "Filters",
  fields: {
    framework: {
      type: "stringList",
      help: "Filter by agent framework",
    },
    limit: {
      type: "int",
      default: 25,
      help: "Maximum number of rows",
    },
  },
});

function sessionList(filters) {
  const mt = require("minitrace");

  // Run multiple queries and combine in JS
  const rows = mt.query(`
    SELECT
      id,
      title,
      environment->>'agent_framework' AS framework,
      CAST(metrics->>'tool_call_count' AS INT) AS tools,
      CAST(metrics->>'turn_count' AS INT) AS turns
    FROM ${mt.tableName}
    WHERE 1=1
    ${filters.framework?.length
      ? `AND (environment->>'agent_framework') IN (${mt.sql.stringIn(filters.framework)})`
      : ""}
    ORDER BY timing->>'started_at' DESC
    LIMIT ${filters.limit}
  `);

  // Post-process in JS: compute density, classify session type
  return rows.map(row => ({
    ...row,
    density: row.turns > 0 ? (row.tools / row.turns).toFixed(2) : 0,
    type: row.tools / row.turns > 2 ? "tool-heavy" : "balanced",
  }));
}

__verb__("sessionList", {
  name: "session-list",
  short: "List minitrace sessions",
  fields: { filters: { bind: "filters" } },
});
```

This becomes:

```bash
go-minitrace query commands overview session-tools session-list \
  --archive-glob './output/active/*/*.minitrace.json' \
  --framework codex \
  --limit 25 \
  --output json
```

**When to use JS instead of SQL:**

| SQL is the right tool | JS is the right tool |
|-----------------------|----------------------|
| Single-query filtering and grouping | Multi-query joins: run several queries and combine in JS |
| Aggregations (`COUNT`, `AVG`, `GROUP BY`) | JS-side scoring and classification |
| Simple predicates | Async logic with `require("timer")` |
| Row-oriented questions | Reusable helper modules shared across commands |
| Single-column transformations | Output shapes richer than flat SQL rows |

See `go-minitrace help js-api-reference` for the complete `require("minitrace")` API.

---

## The web UI

`go-minitrace serve` starts an HTTP server that loads the archive into an in-memory DuckDB table and exposes:

- **Sessions** (`/sessions`) — sortable session list with framework, model, turns, tools, duration, active %, and annotation badges
- **Query** (`/query`) — interactive query editor with structured command forms, SQL rendering, and result tables
- **REST API** — structured command execution under `/api/v2/query-commands`

![Sessions list](screenshot-1.png)

![Session detail](screenshot-2.png)

![Query editor with results](screenshot-3-query.png)

---

## Annotations

Add human review notes to sessions and turns. Annotations live in `output/annotations.db` and sync back into `.minitrace.json` files.

```bash
go-minitrace annotate add \
  --output-dir ./output \
  --session <session-id> \
  --category observation \
  --title "Session worth revisiting for context"

go-minitrace annotate sync --output-dir ./output
```

Categories: `observation`, `ai-failure`, `user-error`, `environment-issue`, `success`, `question`, `to-discuss`, `to-improve`

Full workflow: `go-minitrace help annotation-playbook`.

---

## Supported sources

| Source | Format | Command |
|--------|--------|---------|
| Claude Code | JSON (dir-v1, subagent transcripts) | `convert claude-code` |
| Codex | JSONL (sessions + exec) | `convert codex` |
| Pi | JSONL | `convert pi` |
| claude.ai | ZIP export | `convert claude-ai` |
| ChatGPT | ZIP export / JSON | `convert chatgpt`, `convert chatgpt-json` |
| Geppetto / Pinocchio | `turns.db` | `convert turnsdb` |

## Development

```bash
make lint
make test
make build
```

Pre-commit hooks:

```bash
lefthook install
```

Release:

```bash
make goreleaser
```

## Security

CLI output, logs, and exported data may contain sensitive information from your session history. Review tool output before sharing it.

## License

MIT