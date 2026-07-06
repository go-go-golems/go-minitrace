---
Title: Analysis Guide
Slug: analysis-guide
Short: End-to-end workflow for analyzing AI agent sessions with go-minitrace — from raw transcripts to structured insights
Topics:
- minitrace
- analysis
- sqlite
- query
- annotation
- workflow
Commands:
- discover
- convert
- query
- query run
- query commands
- annotate
Flags:
- archive-glob
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
go-minitrace help query-commands           # query modes, presets, and flags
go-minitrace help structured-query-commands # reusable command authoring
go-minitrace help query-recipes             # ready-to-use query patterns
go-minitrace help writing-queries           # normalized tables, JSON access, joins
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

Each output shows session ID, format hint, source path, working directory, and start time. Get a raw count:

```bash
go-minitrace discover pi --output json | jq length
```

Narrow to one project or time window with the shared filters:

```bash
go-minitrace discover codex --cwd-contains my-repo --since 2026-06-01
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

Use the discover filters to find the sessions you care about, then convert exactly those with the repeatable `--source-session` flag (pi, codex, claude-code) — no staging directory needed:

```bash
# find sessions for one repository
go-minitrace discover codex --cwd-contains my-repo --since 2026-06-01 \
  --output json | jq -r '.[].source_path' > /tmp/sessions.txt

# convert exactly that list
go-minitrace convert codex --source-list /tmp/sessions.txt --output-dir ./output
```

Or pass individual files directly:

```bash
go-minitrace convert pi \
  --source-session ~/.pi/agent/sessions/--slugged-cwd--/<session-id>.jsonl \
  --output-dir ./output
```

Sessions that fail to convert are skipped and reported as `status: failed` rows; the rest of the batch converts normally.

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
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset session-list
```

Nine presets ship with go-minitrace:

| Preset | Shows |
|--------|-------|
| `session-list` | One row per session: framework, model, turns, tools, duration |
| `framework-summary` | Aggregate stats grouped by framework |
| `tool-operation-breakdown` | Tool call counts by framework and operation type |
| `tool-failures` | Failed tool calls with target and error detail |
| `timing-analysis` | Duration, active time, TTFA, and idle ratio by framework |
| `read-ratio-distribution` | Read/write/execute breakdown per session |
| `file-operations` | Every file touch in session order |
| `file-timeline` | Chronological operations on files |
| `annotations` | All annotations across sessions |

### Custom SQL

Start with a preset, then extend it with custom SQL for specific questions:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "SELECT s.session_id, s.title,
               s.tool_call_count AS tools,
               s.turn_count AS turns,
               m.total_input_tokens AS input_tokens
        FROM sessions s
        JOIN metrics m USING (session_id)
        WHERE s.agent_framework = 'pi'
        ORDER BY tools DESC
        LIMIT 20"
```

For queries you reuse, save them as `.sql` files and run them with `--sql-file`:

```bash
go-minitrace query run \
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
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "SELECT agent_framework AS framework,
               COUNT(*) AS sessions,
               AVG(duration_seconds) AS avg_duration_s,
               AVG(tool_call_count) AS avg_tools
        FROM sessions
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
go-minitrace query run \
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
  --scope turn \
  --target-id 5 \
  --category question \
  --title "Why was this tool chosen over alternatives?"
```

### Query annotations

Annotations must be synced into the archive first (`go-minitrace annotate sync --output-dir ./output`), then:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset annotations
```

Or with custom SQL:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "SELECT s.session_id, s.title, a.category, a.title AS annotation_title
        FROM sessions s
        JOIN annotations a USING (session_id)
        WHERE a.category = 'observation'
        ORDER BY s.started_at DESC"
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
  substr(started_at, 1, 7) AS month,
  agent_framework AS framework,
  COUNT(*) AS sessions
FROM sessions
GROUP BY month, framework
ORDER BY month, framework;
```

### Tool use frequency

```sql
SELECT
  tool_name AS tool,
  operation_type AS operation,
  COUNT(*) AS calls,
  COUNT(DISTINCT session_id) AS sessions
FROM tool_calls
GROUP BY tool, operation
ORDER BY calls DESC, tool ASC
LIMIT 20;
```

File-oriented tool calls carry the normalized `file_path` column; fall back to the raw command when it is empty:

```sql
COALESCE(file_path, substr(command, 1, 120))
```

See `go-minitrace help writing-queries` for join patterns and additional nested-field examples.

### Sessions with highest tool-call density (tools per turn)

```sql
SELECT
  session_id,
  title,
  tool_call_count AS tools,
  turn_count AS turns,
  CAST(tool_call_count AS REAL) / turn_count AS density
FROM sessions
WHERE turn_count > 0
ORDER BY density DESC
LIMIT 10;
```

### Failed tool calls

```sql
SELECT
  tc.session_id,
  s.title,
  tc.tool_name AS tool,
  tc.operation_type AS operation,
  tc.error
FROM tool_calls tc
JOIN sessions s USING (session_id)
WHERE tc.success = 0
ORDER BY s.started_at DESC;
```

See `go-minitrace help query-recipes` for more ready-to-use patterns and `go-minitrace help writing-queries` for the join and JSON access mechanics in detail.

## See also

- `go-minitrace help getting-started` — shorter step-by-step tutorial from install through first query
- `go-minitrace help overview` — architecture, format, and supported sources
- `go-minitrace help js-api-reference` — complete JS runtime API for `.js` command handlers
- `go-minitrace help structured-query-commands` — authoring guide for `.sql` and `.js` command files, repository layout, aliases
- `go-minitrace help writing-queries` — SQL patterns: normalized tables, JSON access, annotation joins
- `go-minitrace help query-commands` — preset list, `--sql-file`, and query flags
- `go-minitrace help query-recipes` — ready-to-use SQL examples for common analysis patterns
- `go-minitrace help annotation-playbook` — annotation workflow in depth
- `go-minitrace help minitrace-schema` — every field in a minitrace session document
- `go-minitrace help convert-commands` — detailed reference for each conversion subcommand
