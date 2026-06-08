---
Title: Analysis Guide
Slug: analysis-guide
Short: End-to-end workflow for analyzing AI agent sessions with go-minitrace — from raw transcripts to structured insights
Topics:
- minitrace
- analysis
- duckdb
- query
- annotation
- workflow
Commands:
- discover
- convert
- query
- query duckdb
- query commands
- annotate
Flags:
- archive-glob
- db-path
- source-dir
- output-dir
- preset
- sql
- sql-file
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: Tutorial
---

This guide walks through the complete analysis workflow: discovering sessions, converting them into the minitrace format, running queries to extract insights, and optionally enriching them with annotations for later review.

It assumes go-minitrace is installed and `go-minitrace --version` prints a version. If not, start with `go-minitrace help getting-started`.

## Start with the embedded help tree

Before writing any custom SQL, inspect what is already built in:

```bash
go-minitrace help                           # top-level help
go-minitrace help query-commands           # query modes and flags
go-minitrace help query-duckdb             # presets, SQL-file mode, loading flags
go-minitrace help structured-query-commands # reusable command authoring
go-minitrace help duckdb-query-recipes      # ready-to-use query patterns
go-minitrace help writing-duckdb-queries    # JSON access, UNNEST, casting
```

Run a command with `--help` to see its full flag surface:

```bash
go-minitrace query commands --help
go-minitrace query commands overview session-list --help
```

The embedded catalog already includes commands for session listing, framework summaries, timing analysis, and nightly reviews. Use the built-ins first; only fall back to custom SQL when you need something new.

## Discover: know what you have before converting

Use `discover` to scan a native session store and report what sessions exist — without writing anything.

```bash
go-minitrace discover claude-code
go-minitrace discover codex --source-dir ~/.codex
go-minitrace discover pi
```

Each output shows session ID, format hint, and source path. Get a raw count:

```bash
go-minitrace discover pi --output json | jq length
```

If a source does not have a discover command (claude.ai, ChatGPT, turnsdb), check the source paths in `go-minitrace help convert-commands` and verify they contain readable session files.

## Convert: turn native sessions into minitrace archives

Pick the right subcommand and point it at the session store:

```bash
go-minitrace convert claude-code --output-dir ./output
go-minitrace convert codex --source-dir ~/.codex --output-dir ./output
go-minitrace convert pi --output-dir ./output
```

Each session becomes one `.minitrace.json` file in `output/active/YYYY-MM/<id>.minitrace.json`. A `manifest.json` tracks all sessions with metadata and statistics.

To preview what would be converted without writing files:

```bash
go-minitrace convert claude-code --dry-run
```

Convert multiple source formats into the same output directory — sessions from different frameworks are fully interoperable in the minitrace schema:

```bash
go-minitrace convert pi --output-dir ./output
go-minitrace convert codex --source-dir ~/.codex --output-dir ./output
```

### Filtering a subset before converting

For Codex, `discover` does not expose the working directory. To convert only sessions for one repository, stage matching JSONL files into a temporary tree first:

```bash
# find sessions for a specific cwd
rg -l '"cwd":"/path/to/repo"' ~/.codex/sessions/ \
  | xargs -I{} cp {} /tmp/staged-codex/sessions/

# convert the staged subset
go-minitrace convert codex --source-dir /tmp/staged-codex --output-dir ./output
```

For Pi, the `--source-session` flag converts one file at a time:

```bash
go-minitrace convert pi \
  --source-session ~/.pi/agent/sessions/--slugged-cwd--/<session-id>.jsonl \
  --output-dir ./output
```

### Conversion output structure

```
output/
├── manifest.json                  # all sessions, counts, quality grades
└── active/
    └── 2026-04/
        ├── <session-id>.minitrace.json
        └── <session-id>.minitrace.json
```

## Query: analyze the converted archive

### Built-in presets

The fastest analysis path — no SQL required:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset session-list
```

Six presets ship with go-minitrace:

| Preset | Shows |
|--------|-------|
| `session-list` | One row per session: framework, model, turns, tools, duration |
| `framework-summary` | Aggregate stats grouped by framework |
| `tool-operation-breakdown` | Tool call counts by framework and operation type |
| `timing-analysis` | Duration, active time, TTFA, and idle ratio by framework |
| `read-ratio-distribution` | Read/write/execute breakdown per session |
| `annotations` | All annotations across sessions |

### Custom SQL

Start with a preset, then extend it with custom SQL for specific questions:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "SELECT id, title,
               CAST(metrics->>'tool_call_count' AS INT) AS tools,
               CAST(metrics->>'turn_count' AS INT) AS turns,
               CAST(metrics->>'total_input_tokens' AS INT) AS input_tokens
        FROM sessions_base
        WHERE (environment->>'agent_framework') = 'pi'
        ORDER BY tools DESC
        LIMIT 20"
```

For queries you reuse, save them as `.sql` files and run them with `--sql-file`:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql-file ./queries/top-sessions.sql
```

### Structured query commands

When an analysis task becomes part of your repeatable workflow, promote it to a structured query command. Structured commands give you named, typed parameters and work identically in the CLI, the web UI, and the API.

```bash
go-minitrace query commands overview session-list \
  --archive-glob './output/active/*/*.minitrace.json' \
  --framework codex,pi \
  --limit 50
```

See `go-minitrace help structured-query-commands` for how to author your own `.sql` and `.js` command files, and `go-minitrace help js-api-reference` for the full JavaScript runtime API.

### JavaScript command handlers

JavaScript command handlers run in a Goja-powered JS runtime and have access to `require("minitrace")` for builder-composed normalized SQLite databases through `mt.db()`, plus the full Goja NodeJS stdlib (`timer`, `fs`, `exec`, `path`, `console`, etc.). Use JS when SQL alone is not enough.

**When to reach for JS instead of SQL:**

- Multi-query joins: run several SQL queries and combine or post-process their results in JS
- JS-side scoring and classification (e.g. computing a `focus_score` from ratios, categorizing sessions as `tool-orchestrator` vs `balanced-builder`)
- Async logic with `require("timer")`
- Reusable helper modules shared across multiple commands
- Output shapes richer than flat SQL rows (cards, summaries, synthesized rows)

**A minimal JS command file** (`my-commands/overview/session-tools.js`):

```js
__section__("filters", {
  fields: {
    framework: { type: "stringList", help: "Filter by framework" },
    limit:     { type: "int",        default: 25, help: "Row limit" },
  },
});

function sessionList(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    return db.query(`
      SELECT session_id, title, agent_framework AS framework
      FROM sessions
      WHERE 1=1
      ${filters.framework?.length
        ? `AND agent_framework IN (${mt.sql.stringIn(filters.framework)})`
        : ""}
      ORDER BY started_at DESC
      LIMIT ${filters.limit}
    `);
  } finally {
    db.close();
  }
}

__verb__("sessionList", {
  name:  "session-list",
  short: "List minitrace sessions",
  fields: { filters: { bind: "filters" } },
});
```

The filename stem (`overview/session-tools`) usually becomes a CLI group, and each `__verb__` name (`session-list`) becomes a leaf command:

```bash
go-minitrace query commands overview session-tools session-list \
  --query-repository ./my-commands \
  --archive-glob './output/active/*/*.minitrace.json' \
  --framework codex
```

**Key `require("minitrace")` API:**

| Export | Description |
|--------|-------------|
| `mt.db()` | Create a fluent normalized SQLite builder. In query commands, start with `mt.db().RuntimeArchives().QueryCommandDefaults().Build()`. |
| `db.query(sql)` | Execute read-only SQL against normalized tables such as `sessions`, `turns`, `tool_calls`, `files`, `metrics`, and `events`; return `Array<Record>`. |
| `db.queryOne(sql)` | Same but return only the first row or `null`/`undefined`. |
| `db.queryResult(sql)` | Return `{ columns, rows, count, truncated, error }` instead of throwing on query validation errors. |
| `db.schema()` / `db.tables()` | Discover the normalized SQLite schema from inside JS. |
| `db.close()` | Release or close the DB handle; call it in `finally`. |
| `mt.sql.string(val)` | Single-quoted, escaped SQL string literal. |
| `mt.sql.stringIn(arr)` | Comma-separated quoted list for `IN (...)` clauses. |
| `mt.sql.like(val)` | `LIKE` pattern with `%` wildcards on both sides. |
| `mt.runtime` | Read-only context object (`archiveGlob`, `commandName`, plus compatibility fields). |

See `go-minitrace help js-api-reference` for the complete API, all built-in modules, scanner markers, and the supported field type set.

**Start from working examples.** The testdata showcase directories demonstrate every practical pattern:

- `testdata/query-repositories/js-showcase/` — pure JS commands: multi-verb files, aliases targeting JS, relative helpers, async via `require("timer")`, `mt.db().RuntimeArchives().QueryCommandDefaults().Build()`, `db.queryOne()`, multi-query joins in JS, JS-side scoring, tool co-occurrence
- `testdata/query-repositories/mixed-sql-js-showcase/` — the same commands side-by-side as `.sql` and `.js` equivalents so you can compare SQL vs JS approaches directly

Smoke the showcases against your own local archive:

```bash
go-minitrace query commands \
  --query-repository ./testdata/query-repositories/js-showcase \
  analysis workspace-lab workspace-scoreboard \
  --archive-glob './output/active/*/*.minitrace.json' \
  --output json
```

**Authoring workflow:**

1. Create a repository directory: `mkdir -p ./my-commands/overview`
2. Write the `.js` file (copy from the showcase and adapt)
3. Verify the command surface: `go-minitrace query commands <group...> <leaf> --help`
4. Run it: `go-minitrace query commands <group...> <leaf> --archive-glob ...`
5. Open `go-minitrace serve --archive-glob ...` and use the Commands sidebar and debug panels in `/query`
6. Add an alias (`.alias.yaml`) only after the base command is solid

For JS files, remember the path rule precisely:

- multi-verb or differently named JS files keep the extra file-stem level, e.g. `overview session-tools session-list`
- self-named single-verb JS files collapse the redundant extra level, e.g. `hardware-research/research-summary.js` with a single `research-summary` verb runs as `hardware-research research-summary`

### Multi-framework analysis

Load all frameworks at once for cross-framework comparisons:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "SELECT environment->>'agent_framework' AS framework,
               COUNT(*) AS sessions,
               AVG(CAST(metrics->>'total_duration_ms' AS DOUBLE) / 1000) AS avg_duration_s,
               AVG(CAST(metrics->>'tool_call_count' AS INT)) AS avg_tools
        FROM sessions_base
        GROUP BY framework
        ORDER BY sessions DESC"
```

### Output formats

All query results flow through Glazed's processor pipeline, so you get multiple output formats for free:

```bash
--output table    # default, human-readable terminal table
--output json     # newline-delimited JSON
--output csv      # comma-separated values
--fields col1,col2,col3  # select specific columns
```

Pipe JSON output to `jq` for further shell-based analysis:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset framework-summary \
  --output json | jq '.[] | select(.framework == "pi")'
```

## Annotate: add human review notes

After querying, mark sessions or individual turns for follow-up. Annotations are stored in `output/annotations.db` and synced back into `.minitrace.json` when you call `annotate sync`.

### Add a session-level annotation

```bash
go-minitrace annotate add \
  --output-dir ./output \
  --session <SESSION_ID> \
  --category observation \
  --title "Interesting parallel tool use pattern"
```

### Add a turn-level annotation

```bash
go-minitrace annotate add \
  --output-dir ./output \
  --session <SESSION_ID> \
  --turn 5 \
  --category question \
  --title "Why was this tool chosen over alternatives?"
```

### Query annotations

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset annotations
```

Or with custom SQL:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "SELECT s.id, s.title, a.category, a.title AS annotation_title
        FROM sessions_base s
        JOIN annotations a ON a.session_id = s.id
        WHERE a.category = 'observation'
        ORDER BY s.timing->>'started_at' DESC"
```

For the full annotation workflow — categories, syncing, browsing, and the web UI — see `go-minitrace help annotation-playbook`.

## Validate: check archive integrity

After conversion or manual edits, verify the JSON files are well-formed:

```bash
go-minitrace validate --path ./output --recursive
```

This reports any parse errors and stops on the first malformed file.

## Serve: web UI and API

Run the HTTP server for interactive querying and an annotation browser:

```bash
go-minitrace serve --archive-glob './output/active/*/*.minitrace.json'
```

The server exposes:

- `/query` — interactive query editor with structured command forms, SQL rendering, and result tables
- `/api/v2/query-commands` — REST endpoint for structured command execution
- `/annotate` — annotation browser and editor

## Common analysis patterns

### Session counts by framework over time

```sql
SELECT
  SUBSTR(timing->>'started_at', 1, 7) AS month,
  environment->>'agent_framework' AS framework,
  COUNT(*) AS sessions
FROM sessions_base
GROUP BY month, framework
ORDER BY month, framework;
```

### Tool use frequency

```sql
SELECT
  tc->>'tool_name' AS tool,
  tc->>'operation_type' AS operation,
  COUNT(*) AS calls,
  COUNT(DISTINCT id) AS sessions
FROM sessions_base,
     UNNEST(tool_calls) AS t(tc)
GROUP BY tool, operation
ORDER BY calls DESC, tool ASC
LIMIT 20;
```

If you want to inspect file-oriented tool calls, prefer a path fallback like:

```sql
COALESCE(tc->'input'->>'file_path', tc->'input'->'arguments'->>'path')
```

See `go-minitrace help writing-duckdb-queries` for the JSON[] caveats, 1-based list indexing, and additional nested-field examples.

### Sessions with highest tool-call density (tools per turn)

```sql
SELECT
  id,
  title,
  CAST(metrics->>'tool_call_count' AS INT) AS tools,
  CAST(metrics->>'turn_count' AS INT) AS turns,
  CAST(metrics->>'tool_call_count' AS DOUBLE) /
    CAST(metrics->>'turn_count' AS DOUBLE) AS density
FROM sessions_base
WHERE CAST(metrics->>'turn_count' AS INT) > 0
ORDER BY density DESC
LIMIT 10;
```

### Failed tool calls

```sql
SELECT
  id,
  title,
  tc->>'tool_name' AS tool,
  tc->>'operation_type' AS operation,
  tc->'output'->>'error' AS error
FROM sessions_base,
     UNNEST(tool_calls) AS t(tc)
WHERE (COALESCE(tc->'output'->>'success', 'true')) = 'false'
ORDER BY timing->>'started_at' DESC;
```

See `go-minitrace help duckdb-query-recipes` for more ready-to-use patterns and `go-minitrace help writing-duckdb-queries` for the JSON access, casting, and UNNEST mechanics in detail.

## See also

- `go-minitrace help getting-started` — shorter step-by-step tutorial from install through first query
- `go-minitrace help overview` — architecture, format, and supported sources
- `go-minitrace help js-api-reference` — complete JS runtime API for `.js` command handlers
- `go-minitrace help structured-query-commands` — authoring guide for `.sql` and `.js` command files, repository layout, aliases
- `go-minitrace help writing-duckdb-queries` — SQL patterns: JSON access, UNNEST, casting, annotation joins
- `go-minitrace help query-duckdb` — preset list, `--sql-file`, and archive loading flags
- `go-minitrace help duckdb-query-recipes` — ready-to-use SQL examples for common analysis patterns
- `go-minitrace help annotation-playbook` — annotation workflow in depth
- `go-minitrace help minitrace-schema` — every field in a minitrace session document
- `go-minitrace help convert-commands` — detailed reference for each conversion subcommand
