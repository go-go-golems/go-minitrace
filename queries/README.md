# Saved SQL Queries for go-minitrace

A library of saved SQL files written against the normalized SQLite schema
(`normalized-sqlite-v3`). These files are plain, single-statement `SELECT`/`WITH`
queries. They power two things:

1. **CLI usage** via `go-minitrace query run --sql-file <path>`.
2. **The web UI**: this directory is `serve`'s default `--query-dir`, so every
   file here shows up as a saved query (the leading `-- comment` line is used
   as the description).

## No loading step required

There is no separate "load" step anymore (the old DuckDB `load.sql` bootstrap
has been removed). Both `go-minitrace query run --archive-glob ...` and
`go-minitrace serve` build the normalized SQLite database from your
`*.minitrace.json` archives automatically (and cache it), so you just point
them at the archives and run queries:

```bash
# Run a saved query file
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql-file queries/overview/session-list.sql

# Run a built-in preset
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset framework-summary

# Ad hoc SQL
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql 'SELECT COUNT(*) AS sessions FROM sessions'
```

## Schema

Queries run through a sandboxed, read-only runner. The sandbox allows a single
`SELECT` (or `WITH ... SELECT`) statement over the normalized schema tables:

- `sessions` — one row per session with flattened columns such as
  `agent_framework`, `model`, `working_directory`, `started_at`,
  `duration_seconds`, `turn_count`, `tool_call_count`
- `turns` — one row per conversational turn (`turn_index`, `role`, `content`,
  token counts, `framework_metadata_json`)
- `tool_calls` — one row per tool call (`tool_name`, `operation_type`,
  `success`, `error`, `exit_code`, `duration_ms`, `command`, `file_path`,
  `framework_metadata_json`)
- `turn_tool_calls`, `files`, `annotations`, `handovers`, `metrics`,
  `attachments`, `events`

Every table also carries a `raw_json` column with the original record, and
framework-specific detail lives in `framework_metadata_json` columns — use
SQLite's `json_extract(...)` on those. In addition to the schema tables, the
sandbox allows the `sessions_base` compatibility view, which reconstructs the
legacy blob-shaped table for old session-level queries.

## Available files

### `overview/` — session-level summaries

| File | Purpose |
|------|---------|
| `session-list.sql` | List sessions with key metadata |
| `framework-summary.sql` | Aggregate stats by framework |
| `annotations.sql` | List annotations joined with their sessions |

### `tools/` — tool call analysis

| File | Purpose |
|------|---------|
| `tool-operation-breakdown.sql` | Count tool calls by framework and operation type |
| `tool-failures.sql` | All failed tool calls with error details |
| `read-ratio-distribution.sql` | Inspect read-before-write style behavior |
| `exit-codes.sql` | Tool calls with explicit process exit codes |
| `justifications.sql` | Tool calls that include an explicit justification |

### `timing/` — temporal analysis

| File | Purpose |
|------|---------|
| `timing-analysis.sql` | Compare duration, TTFA, and idle ratio |

### `files/` — file-level operation tracking

| File | Purpose |
|------|---------|
| `file-operations.sql` | Every read/write/edit on files in turn order |
| `file-timeline.sql` | Chronological ops on files matching a path pattern (edit the `LIKE '%'` filter at the bottom) |

### `framework-metadata/` — framework-specific metadata examples

| File | Purpose |
|------|---------|
| `codex-exec-metadata.sql` | Inspect preserved Codex execution metadata such as source, parsed command info, stdout/stderr, and exit code |
| `claude-turn-metadata.sql` | Inspect preserved Claude Code turn metadata such as entrypoint, thread context, stop reason, and cache buckets |
| `pi-edit-diffs.sql` | Inspect preserved Pi edit diffs and first changed line metadata |

### Removed files

- `load.sql` — the DuckDB `read_json(...)` bootstrap is gone; loading is no
  longer needed because the normalized SQLite DB is built automatically.

## Custom queries

Drop your own queries in `my-queries/`. They won't be tracked by git (it's in
`.gitignore`). Queries saved from the web UI are also created there.

## Preset names for CLI usage

The built-in preset set is available via `go-minitrace query run --preset <name>`.
The saved queries in `queries/` are plain SQL files on disk and are **not**
compiled into the binary.

Current built-in presets:

```
annotations, framework-summary, session-list,
tool-operation-breakdown, tool-failures, read-ratio-distribution,
timing-analysis,
file-operations, file-timeline
```

## Why this shape

Everything runs on one engine: the same normalized SQLite database backs the
CLI (`query run`), the SQL command runtime, and the web UI. Saved queries are
single-statement files so they behave identically in all three places, and the
sandbox keeps them read-only.
