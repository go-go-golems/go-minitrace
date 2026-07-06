---
Title: Query Commands
Slug: query-commands
Short: Query converted minitrace archives with the normalized SQLite engine using presets, custom SQL, or structured commands
Topics:
- minitrace
- sqlite
- glazed
- sqleton
Commands:
- query
- query run
- query commands
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `query` group loads converted minitrace archives into a normalized SQLite database and runs queries against it. There are two user-facing workflows:

- `query run` for built-in presets, inline SQL, and SQL files against the normalized schema
- `query commands` for repository-backed sqleton-style structured commands, including SQL templates and JavaScript handlers

Both workflows share the same engine: a sandboxed, read-only SQLite query runner over the normalized tables. SQL structured commands render read-only SQL against that runner; JavaScript structured commands call `require("minitrace").db().RuntimeArchives().QueryCommandDefaults().Build()` and can run several queries and combine or score the results in JavaScript.

## Query workflow choices

Choose `query run` when you are exploring, prototyping, or already have raw SQL in hand. It is the shortest path from idea to result.

Choose `query commands` when the query should become a named, reusable analysis tool. Structured commands give you typed parameters, aliases, CLI discoverability, and a matching form in the `/query` web UI.

For the full structured-command authoring and repository-loading guide, see `go-minitrace help structured-query-commands`.

## How `query run` works

When you run `query run`, go-minitrace:

1. Expands the archive globs and fingerprints the matching `.minitrace.json` files
2. Builds (or reuses from cache) a normalized SQLite database with one table per entity: `sessions`, `turns`, `tool_calls`, `turn_tool_calls`, `files`, `annotations`, `handovers`, `metrics`, `attachments`, `events`
3. Runs either a named preset, inline SQL, or a SQL file through the sandboxed read-only query runner
4. Streams results through Glazed for output formatting

The runner enforces an allowlist of the normalized tables plus the `sessions_base` compatibility view; DDL, writes, and access to `sqlite_master` are rejected.

## query run

```bash
go-minitrace query run [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--archive-glob` | `./output/active/*/*.minitrace.json` | Repeatable glob flag matching minitrace session files |
| `--preset` | | Named built-in query to run |
| `--sql` | | Inline SQL to run against the normalized database |
| `--sql-file` | | Path to a SQL file to execute |
| `--max-rows` | `1000` | Maximum number of rows to return |
| `--max-cell-chars` | `4000` | Maximum number of characters per result cell |
| `--timeout-ms` | `5000` | Query timeout in milliseconds |

Exactly one of `--preset`, `--sql`, or `--sql-file` must be specified. They are mutually exclusive.

### Query modes

**Preset mode** runs one of the built-in queries:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset session-list
```

**Inline SQL mode** runs arbitrary SQL against the normalized database:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "SELECT COUNT(*) AS total FROM sessions"
```

**SQL file mode** reads a query from a file. This is useful for saved query libraries:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql-file ./my-query.sql
```

### Built-in presets

Presets are embedded under `pkg/minitracedb/presets/` and addressed by bare name (or `folder/name` when ambiguous):

| Preset | Description |
|--------|-------------|
| `overview/session-list` | One row per session: id, framework, model, title, turns, tools, duration, read ratio, start time, source format |
| `overview/framework-summary` | Aggregate stats by agent framework: session count, average tools/turns/read ratio/duration/TTFA |
| `overview/annotations` | All annotations joined with sessions: session ID, framework, annotator, category, title, scope |
| `timing/timing-analysis` | Duration, active time, TTFA, idle ratio, min/max duration by framework |
| `tools/tool-operation-breakdown` | Tool call counts grouped by framework and operation type (read, modify, create, execute, delegate) |
| `tools/tool-failures` | Failed tool calls with turn, tool, target, and error detail |
| `tools/read-ratio-distribution` | Per-session breakdown of reads, modifies, creates, executes with read ratio |
| `files/file-operations` | Every file touch in session order with success/failure status |
| `files/file-timeline` | Chronological operations on files, with result content classified into short labels |

### Output formatting

Query results flow through Glazed, so all standard output options work:

```bash
# Default table output
go-minitrace query run --archive-glob '...' --preset session-list

# JSON for piping
go-minitrace query run --archive-glob '...' --preset session-list --output json

# CSV for spreadsheets
go-minitrace query run --archive-glob '...' --preset session-list --output csv

# Select specific fields
go-minitrace query run --archive-glob '...' --preset session-list --fields id,framework,turns,tools
```

## The normalized schema

The database has one table per minitrace entity:

| Table | Contents |
|-------|----------|
| `sessions` | One row per session: provenance, environment, timing, operational context, outcome, and rollup counts as real columns |
| `turns` | One row per conversational turn: role, content, model, thinking, token usage |
| `tool_calls` | One row per tool call: tool name, operation type, file path, command, success, error, exit code, duration |
| `turn_tool_calls` | Join table preserving tool-call membership and ordinal per turn |
| `files` | One row per file path touched by a tool call |
| `annotations` | One row per annotation with scope, content, and taxonomy mappings |
| `handovers` | Received/produced handover documents per session |
| `metrics` | Wide per-session metrics: tokens, timing, subagents, model switches |
| `attachments` | Artifact references (images, files) linked to turns, tool calls, or events |
| `events` | Timeline events such as compactions, mode changes, or rate-limit snapshots |

Fields that used to require `UNNEST` on JSON arrays are now plain rows in child tables keyed by `session_id`:

```sql
SELECT tc.tool_name, tc.operation_type, tc.success
FROM tool_calls tc
JOIN sessions s USING (session_id)
LIMIT 20;
```

Long-tail fields that never got a real column are still reachable through JSON: every table carries a `raw_json` column with the original record, and turns/tool calls carry `framework_metadata_json`:

```sql
SELECT session_id,
       json_extract(framework_metadata_json, '$.stop_reason') AS stop_reason
FROM turns
WHERE framework_metadata_json IS NOT NULL
LIMIT 10;
```

### The sessions_base compatibility view

Session-level SQL written for the removed DuckDB engine keeps working against the `sessions_base` view, which reconstructs the old shape (scalar columns `id`, `title`, `summary`, `classification`, `profile` plus JSON blob columns `provenance`, `flags`, `environment`, `operational_context`, `timing`, `turns`, `tool_calls`, `annotations`, `metrics`) from the `sessions` table. SQLite >= 3.38 supports `->`/`->>` on those JSON text columns:

```sql
SELECT id, environment->>'model' AS model
FROM sessions_base
LIMIT 5;
```

Per-tool-call and per-turn SQL should move to the `tool_calls`/`turns` tables — SQLite has no `UNNEST`. See `go-minitrace help query-duckdb` for the full migration table.

## Querying annotations

Annotations synced into the archives appear in the `annotations` table:

```sql
SELECT a.session_id, a.scope_type, a.category, a.title
FROM annotations a
ORDER BY a.session_id;
```

One subtle but important rule: `query run` reads the `.minitrace.json` archive files it loads. If you created or edited annotations through `go-minitrace annotate ...`, run `go-minitrace annotate sync --output-dir ...` first so the archive contains those changes. (The web UI is different: `serve` ATTACHes the live annotation store as schema `anno`, so `anno.annotations` is always current there.)

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| Query returns 0 rows | Archive glob doesn't match any files | Check the glob path with `ls` first |
| `one of preset, sql, or sql-file must be specified` | No query mode given | Add `--preset`, `--sql`, or `--sql-file` |
| `preset, sql, and sql-file are mutually exclusive` | More than one query mode specified | Use exactly one |
| `access to table "..." is not allowed` | Query references a table outside the allowlist (including `sqlite_master`) | Stick to the normalized tables; use `db.schema()` from JS or `go-minitrace help minitrace-schema` for introspection |
| `no such function: UNNEST` or similar | DuckDB-era SQL | Rewrite against the child tables; see `go-minitrace help query-duckdb` |
| Query cancelled | Timeout exceeded | Raise `--timeout-ms` |

## Structured query commands

The `query commands` subgroup loads repository-backed sqleton-style command files. Repository subdirectories become nested CLI groups, so `pkg/minitracecmd/core/overview/session-list.sql` is exposed as `go-minitrace query commands overview session-list`.

Those files define:

- a command name and help text
- typed Glazed fields
- optional aliases with prefilled defaults
- a SQL template (rendered against the normalized schema; the `{{TABLE_NAME}}` placeholder substitutes to the `sessions_base` compatibility view) or a JavaScript handler

A simple example looks like this:

```bash
go-minitrace query commands overview session-list \
  --archive-glob './output/active/*/*.minitrace.json' \
  --framework codex,pi
```

The legacy `--db-path`, `--table-name`, and `--persist-loaded` flags are deprecated: SQL commands ignore them (with a warning) and always run against the normalized database built from `--archive-glob`. JS commands still see the values on `mt.runtime` but should use `mt.db()` instead.

The repository-backed flow is the right choice when a query should be reusable by other people, promoted into the web UI, or shared through config/env/flag-discovered command repositories.

See `go-minitrace help structured-query-commands` for:

- how repository loading works
- how to configure `queryRepositories`, `GO_MINITRACE_QUERY_REPOSITORIES`, and `--query-repository`
- how to write `/* sqleton ... */` command files
- how to write `.alias.yaml` shortcut files

## See also

- `go-minitrace help structured-query-commands` — run and author sqleton-style structured query commands
- `go-minitrace help annotation-playbook` — operator workflow for creating, syncing, and validating annotations
- `go-minitrace help writing-queries` — how to write custom SQL against the normalized schema
- `go-minitrace help query-recipes` — ready-to-use query examples
- `go-minitrace help query-duckdb` — migrating saved DuckDB SQL to the normalized engine
- `go-minitrace help output-formats-and-pipelines` — detailed Glazed output formatting guide
- `go-minitrace help minitrace-schema` — field reference for the archive JSON and normalized tables
