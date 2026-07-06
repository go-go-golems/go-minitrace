---
name: go-minitrace-transcript-analysis
description: Use when analyzing previous Pi or Codex coding-agent transcripts with go-minitrace, especially to find sessions by repo/date, convert targeted subsets into minitrace archives, run SQL queries on the normalized SQLite engine, and summarize findings with concrete evidence and caveats.
---

# go-minitrace Transcript Analysis

## Overview

Use this skill when the user wants to inspect prior coding-agent sessions, compare Pi and Codex behavior, or build summaries from transcript archives instead of reading raw JSONL manually.

Keep the workflow evidence-first:

1. Discover candidate native sessions (filter by repo and time window with `--cwd-contains` / `--since`).
2. Convert only the subset you need (`--source-session` / `--source-list`).
3. Query the resulting `.minitrace.json` files with `go-minitrace query run` (normalized SQLite).
4. Summarize findings with explicit caveats.

Read `references/queries.md` before writing custom SQL.

Before inventing a query or command from scratch, use `go-minitrace help` to discover what is already embedded. The help tree includes both ad hoc SQL guidance and reusable structured query commands, so prefer the built-ins first and only fall back to custom SQL when necessary.

## Native stores

- Codex sessions usually live under `~/.codex/sessions/YYYY/MM/DD/*.jsonl`.
- Pi sessions usually live under `~/.pi/agent/sessions/--slugged-cwd--/*.jsonl`.

## Workflow

### 0. Use the built-in help tree first

The fastest way to avoid re-deriving existing analysis paths is to inspect the embedded help pages:

- `go-minitrace help query-commands` — choose between ad hoc SQL (`query run`) and reusable structured commands (`query commands`); flag reference and preset list
- `go-minitrace help writing-queries` — normalized tables, joins, `json_extract` patterns, filtering
- `go-minitrace help query-recipes` — ready-made SQL examples for common questions
- `go-minitrace help structured-query-commands` — how repository-backed query commands are discovered, named, and run
- `go-minitrace help query-duckdb` — migration table if you find old DuckDB-era SQL lying around
- `go-minitrace query commands --help` — list the embedded command groups currently available
- `go-minitrace query commands overview session-list --help` — inspect one embedded command's flags and usage

The embedded catalog currently includes examples such as:

- `go-minitrace query commands overview session-list`
- `go-minitrace query commands overview framework-summary`
- `go-minitrace query commands timing timing-analysis`
- `go-minitrace query commands overview aliases codex-framework-summary`

Use `go-minitrace query run` for quick ad hoc analysis, and `go-minitrace query commands` when the analysis should become a named, reusable command with typed flags and web-UI support.

### 1. Discover sessions for one repository and time window

`discover` supports native filtering — no grep staging needed. Every discover command (pi, codex, claude-code, copilot) accepts:

- `--cwd-contains <substring>` — case-sensitive match on the session working directory
- `--since <RFC3339 or YYYY-MM-DD>` — sessions started at or after this time

Rows include `id`, `format_hint`, `source_path`, `cwd`, and `started_at`:

```bash
go-minitrace discover codex --cwd-contains my-repo --since 2026-06-01 --output json
go-minitrace discover pi --cwd-contains my-repo --output json | jq length
```

### 2. Convert exactly the sessions you found

`convert pi`, `convert codex`, and `convert claude-code` accept a repeatable `--source-session` flag (explicit files) and `--source-list` (file with newline-separated paths; `#` comments and blank lines ignored). No staging directories, no full-store scans:

```bash
# pipe discover straight into a targeted conversion
go-minitrace discover codex --cwd-contains my-repo --since 2026-06-01 \
  --output json | jq -r '.[].source_path' > /tmp/sessions.txt

go-minitrace convert codex --source-list /tmp/sessions.txt --output-dir ./analysis/codex
go-minitrace convert pi --source-session /path/to/one/session.jsonl --output-dir ./analysis/pi
```

Manifests are maintained read-merge-write (each conversion rescans and merges), so repeated targeted conversions into the same output directory keep the manifest complete. Sessions that fail to convert are skipped and reported as `status: failed` rows; the command exits 0 as long as at least one session converted.

### 3. Query the archive

`go-minitrace query run` builds (and caches) a normalized SQLite database from the archive glob and runs sandboxed read-only SQL. The tables are `sessions`, `turns`, `tool_calls`, `turn_tool_calls`, `files`, `annotations`, `handovers`, `metrics`, `attachments`, `events`, plus a `sessions_base` compatibility view for old blob-style session SQL.

Start with the built-in presets:

```bash
go-minitrace query run \
  --archive-glob './analysis/*/active/*/*.minitrace.json' \
  --preset framework-summary
```

The nine presets: `session-list`, `framework-summary`, `annotations`, `timing-analysis`, `tool-operation-breakdown`, `tool-failures`, `read-ratio-distribution`, `file-operations`, `file-timeline`.

Then switch to custom SQL for repo-specific questions. Save every SQL file you write inside the working folder before running it, so the full analysis path is reproducible:

```bash
go-minitrace query run \
  --archive-glob './analysis/*/active/*/*.minitrace.json' \
  --sql-file ./queries/tool-frequency.sql
```

Limits: `--max-rows` (default 1000), `--max-cell-chars` (default 4000), `--timeout-ms` (default 5000). The sandbox rejects writes, DDL, and `sqlite_master`; use `go-minitrace help minitrace-schema` (or `db.schema()` from JS) for introspection.

### 4. Write and run JS command handlers

SQL is sufficient for most analysis tasks, but JavaScript command handlers let you go further: scoring, multi-query joins in JS, async logic, relative helper modules, and richer row shapes. Use JS when the analysis logic is complex enough that SQL becomes unwieldy or when you need to reuse shared helper code.

Start by reading the two embedded help pages:

```bash
go-minitrace help js-api-reference
go-minitrace help structured-query-commands
```

The database API is the **builder**: `mt.db()` composes sources and limits, `Build()` returns a handle. In query commands, always start from the runtime archives:

```js
const mt = require("minitrace");
const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
try {
  const rows = db.query("SELECT session_id, title FROM sessions ORDER BY started_at DESC LIMIT 10");
  // db.queryOne(sql) -> first row; db.queryResult(sql) -> {columns, rows, count, truncated, error}
  // db.schema() / db.tables() -> introspect the normalized schema (sqlite_master is blocked)
} finally {
  db.close();
}
```

Do NOT use `mt.query(sql)` to execute SQL (it builds named recipe objects now), and do not reference `mt.tableName` or `sessions_base` from JS — `mt.runtime.tableName`/`dbPath`/`persistLoaded` are vestigial echoes of deprecated flags. `mt.sql.string()`, `mt.sql.stringIn()`, and `mt.sql.like()` remain the escaping helpers.

A minimal JS command that wraps a single SQL query:

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

If that lives in `my-commands/overview/session-tools.js`, the CLI path becomes:

```bash
go-minitrace query commands overview session-tools session-list \
  --query-repository ./my-commands \
  --archive-glob './analysis/*/active/*/*.minitrace.json' \
  --framework codex
```

Path rule: multi-verb or differently named JS files keep the file-stem level (`overview session-tools session-list`), but a self-named single-verb file collapses the redundant level — `hardware-research/research-summary.js` containing only a `research-summary` verb runs as `query commands hardware-research research-summary`.

Error shape: JS failures render as a compact one-line error plus `file:line` location; with `--output json`, a parseable envelope `{"error": ..., "location": ..., "command": ...}` is printed on stdout. Automation should treat the presence of an `error` key as failure.

For repeated project work, avoid passing `--query-repository` manually by configuring discovery once. CLI/env take precedence over config, and the embedded catalog comes last:

```bash
# Linux/macOS use ':' as the path-list separator; Windows uses ';'.
export GO_MINITRACE_QUERY_REPOSITORIES="$PWD/query-commands:$HOME/shared-minitrace-queries"
```

```yaml
# ~/.config/go-minitrace/config.yaml, <git-root>/.go-minitrace.yml,
# <cwd>/.go-minitrace.yml, or the .override.yml variants
queryRepositories:
  - ./query-commands
```

Relative `queryRepositories` entries resolve relative to the config file directory. If a higher-layer config file contains `queryRepositories`, it replaces lower-layer config-derived repositories; explicit `--query-repository` and `GO_MINITRACE_QUERY_REPOSITORIES` are still prepended.

**When to reach for JS instead of SQL:**

- The analysis needs multiple SQL queries whose results are joined or post-processed in JS
- You need JS-side scoring or classification logic (e.g. computing a `focus_score` from ratios)
- You need async behavior (e.g. delaying, batching, or rate-limiting)
- Several commands share helper utilities
- The output shape is richer than a flat SQL result set (cards, summaries)

**The showcase repositories are the best starting point.** Copy one and adapt it:

```bash
go-minitrace query commands --query-repository ./testdata/query-repositories/js-showcase --help
```

The `js-showcase` directory demonstrates every pattern: multi-verb files, aliases targeting JS commands, relative helper modules, pure synthetic rows, async commands with `require("timer")`, `db.queryOne()` reshaping, multi-query joins in JS, JS-side scoring, and per-session tool co-occurrence analysis. `mixed-sql-js-showcase/` shows the same commands side-by-side as `.sql` and `.js`.

Run them against real local sessions to see non-synthetic output:

```bash
# convert Pi sessions locally (nothing leaves the machine)
go-minitrace convert pi --source-dir ~/.pi/agent/sessions --output-dir /tmp/pi-mini

# smoke the JS showcases against the local archive
go-minitrace query commands \
  --query-repository ./testdata/query-repositories/js-showcase \
  analysis workspace-lab workspace-scoreboard \
  --archive-glob '/tmp/pi-mini/active/*/*.minitrace.json' \
  --output json
```

Validate the commands before trusting their output — run through the CLI first, then test with `--output json` to confirm the row shape matches your expectation.

## What to extract in summaries

- Session counts by framework and model (`sessions`)
- Turn counts and tool-call counts (`sessions` rollup columns; `metrics` for tokens/ratios)
- Dominant tool families from the `tool_calls` table
- Timing and latency patterns (`duration_seconds`, `metrics.idle_ratio`, `tool_calls.duration_ms`)
- Failed tool calls (`tool_calls.success = 0`, `exit_code`)
- Outlier sessions worth manual reading
- Data-quality caveats: which fields the source adapter actually populates (e.g. pi has no exit codes or git branch; see `go-minitrace help adapter-reference` for the fidelity matrix)

## Scripts

- `scripts/query_minitrace.sh`: run ad hoc SQL against a minitrace archive glob (`query run` wrapper)

(The old discover/stage-by-cwd scripts were removed: `discover --cwd-contains/--since` and `convert --source-session/--source-list` cover those workflows natively, and manifest drift on repeated targeted conversions is fixed.)

## Embedded documentation quick reference

These embedded help pages cover the full surface area:

| Help page | What it covers |
|-----------|----------------|
| `go-minitrace help js-api-reference` | `require("minitrace")` builder API, all built-in modules, scanner markers, field types |
| `go-minitrace help structured-query-commands` | Authoring `.sql` and `.js` files, repository layout, aliases |
| `go-minitrace help query-recipes` | Ready-to-use SQL for common analysis patterns |
| `go-minitrace help writing-queries` | Normalized tables, joins, `json_extract`, annotation queries |
| `go-minitrace help minitrace-schema` | Every field in a minitrace session document and its SQL column |
| `go-minitrace help adapter-reference` | Per-adapter fidelity matrix (durations, exit codes, tokens, thinking) |
| `go-minitrace help annotation-playbook` | Annotation CLI and web UI workflow (`anno.annotations` live ATTACH in serve) |
| `go-minitrace help query-commands` | Presets, `--sql-file`, and query flags |
| `go-minitrace help query-duckdb` | Migration table for legacy DuckDB-era SQL |

## If a local go-minitrace checkout exists

`go-minitrace help` and `go-minitrace help --ui` already expose a substantial set of embedded commands, queries, and tutorials. Check those first before inventing new SQL or shell workflows; many common analysis tasks already have examples or preset queries you can reuse.

Useful implementation entry points:

- `cmd/go-minitrace/main.go` — CLI wiring
- `cmd/go-minitrace/cmds/convert/` — converter flags including `--source-session`/`--source-list`
- `cmd/go-minitrace/cmds/discover/filters.go` — `--cwd-contains`/`--since`
- `pkg/minitrace/archive.go` — archive/manifest read-merge-write
- `pkg/minitracedb/` — normalized schema, sandboxed query runner, embedded presets

Use those files when the user wants implementation-level analysis or when you need to explain why a query or manifest behaves a certain way.
