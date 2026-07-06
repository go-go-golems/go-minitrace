---
Title: Writing Queries
Slug: writing-queries
Short: Learn SQLite query patterns and JSON operators for the normalized minitrace schema
Topics:
- minitrace
- sqlite
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial teaches you how to write SQL queries against converted minitrace archives. It covers the normalized table layout, the JSON access syntax for long-tail fields, and common patterns you need for analysis.

All examples use the `query run` command with `--sql`:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql 'SELECT COUNT(*) AS sessions FROM sessions'
```

The engine is SQLite (3.38+ JSON operators available), running through a sandboxed read-only query runner. The same SQL works in `query run`, `.sql` structured commands, JS handlers via `mt.db()`, and the `serve` web Query Editor.

## The normalized tables

Converted archives are loaded into one table per entity, keyed by `session_id`:

- **`sessions`** — one row per session. Provenance, environment, timing, operational context, outcome, and rollup counts are real columns: `agent_framework`, `model`, `working_directory`, `git_branch`, `quality`, `source_format`, `started_at`, `duration_seconds`, `turn_count`, `tool_call_count`, and more.
- **`turns`** — one row per conversational turn: `turn_index`, `role`, `content`, `model`, `thinking`, token columns.
- **`tool_calls`** — one row per tool call: `tool_name`, `operation_type`, `file_path`, `command`, `success`, `error`, `exit_code`, `duration_ms`.
- **`turn_tool_calls`** — join table linking turns to their tool calls with an ordinal.
- **`files`** — one row per file path touched by a tool call.
- **`annotations`**, **`handovers`**, **`metrics`**, **`attachments`**, **`events`** — one row per annotation/handover/session-metrics/attachment/event.

Because the interesting fields are real columns, most queries need no JSON extraction and no casting:

```sql
SELECT agent_framework, model, turn_count, tool_call_count
FROM sessions
ORDER BY started_at;
```

## Joining child tables

What used to be nested JSON arrays are now plain rows. Join them to sessions with `USING (session_id)`:

```sql
-- Count tool calls by name across all sessions
SELECT tool_name, COUNT(*) AS invocations
FROM tool_calls
GROUP BY tool_name
ORDER BY invocations DESC;
```

```sql
-- Tool calls with their session's framework
SELECT s.agent_framework, tc.tool_name, tc.operation_type, tc.success
FROM tool_calls tc
JOIN sessions s USING (session_id)
LIMIT 20;
```

```sql
-- Events: compactions, mode changes, rate-limit snapshots
SELECT session_id, kind, title, summary
FROM events
ORDER BY session_id, timestamp;
```

```sql
-- Attachments: images, uploaded files
SELECT session_id, kind, media_type, path, tool_call_id
FROM attachments
WHERE kind = 'image';
```

To reconstruct turn ordering for tool calls, use `emitting_turn_index` on `tool_calls` or the `turn_tool_calls` join table when you need the exact per-turn ordinal.

## Accessing JSON fields

Two kinds of JSON columns remain:

1. **`raw_json`** on every table — the original record, for long-tail fields that never got a column.
2. **`framework_metadata_json`** on turns, tool calls, attachments, and events — adapter-specific extras such as `stop_reason` (pi), `tool_use_result` (claude-code), or `parsed_cmd`/`stderr` (codex).

Use `json_extract()` or the `->>` operator (both return SQL values):

```sql
SELECT session_id,
       json_extract(framework_metadata_json, '$.stop_reason') AS stop_reason
FROM turns
WHERE role = 'assistant';
```

```sql
-- Same thing with the arrow operator
SELECT session_id,
       framework_metadata_json->>'stop_reason' AS stop_reason
FROM turns
WHERE role = 'assistant';
```

```sql
-- Long-tail session fields from raw_json
SELECT session_id,
       json_extract(raw_json, '$.provenance.converter_version') AS converter
FROM sessions;
```

Unlike the old DuckDB engine, SQLite's `->>` already returns an unquoted SQL value — no `REPLACE(CAST(...))` quote-stripping is needed. Numeric JSON values compare and aggregate correctly after `CAST(... AS INTEGER)`/`CAST(... AS REAL)` when precision matters.

## Filtering patterns

### By framework

```sql
WHERE agent_framework = 'claude-code'
WHERE agent_framework IN ('claude-code', 'pi')
```

### By source format (exclude subagents)

```sql
WHERE source_format NOT LIKE '%subagent%'
```

### By quality tier

```sql
WHERE quality = 'A'
```

### By working directory

```sql
WHERE working_directory LIKE '%go-minitrace%'
```

### By date range

```sql
WHERE started_at >= '2026-03-01'
  AND started_at < '2026-04-01'
```

### By session size

```sql
WHERE tool_call_count > 10
  AND turn_count > 5
```

## Common query patterns

### Group-by with aggregation

```sql
SELECT model,
       COUNT(*) AS sessions,
       ROUND(AVG(tool_call_count), 1) AS avg_tools
FROM sessions
GROUP BY model
ORDER BY sessions DESC;
```

### Top-N

```sql
SELECT session_id, title, tool_call_count AS tools
FROM sessions
ORDER BY tools DESC
LIMIT 10;
```

### Conditional aggregation

SQLite supports the `FILTER` clause on aggregates:

```sql
SELECT agent_framework,
       COUNT(*) FILTER (WHERE quality = 'A') AS quality_a,
       COUNT(*) FILTER (WHERE quality = 'B') AS quality_b,
       COUNT(*) FILTER (WHERE quality = 'C') AS quality_c
FROM sessions
GROUP BY agent_framework;
```

### Temporal bucketing

```sql
SELECT date(started_at) AS day, COUNT(*) AS sessions
FROM sessions
WHERE started_at IS NOT NULL
GROUP BY day
ORDER BY day;
```

### Failure hunting

```sql
SELECT s.agent_framework, tc.tool_name,
       substr(COALESCE(tc.error, tc.result), 1, 120) AS detail
FROM tool_calls tc
JOIN sessions s USING (session_id)
WHERE tc.success = 0
ORDER BY tc.session_id, tc.emitting_turn_index;
```

## Querying annotations

Annotations synced into the archives live in the `annotations` table:

```sql
SELECT category, COUNT(*) AS n
FROM annotations
GROUP BY category
ORDER BY n DESC;
```

```sql
-- Tool-call-level annotations only
SELECT session_id, target_id AS tool_call_id, title
FROM annotations
WHERE scope_type = 'tool_call';
```

One workflow-specific nuance: `query run` reads the `.minitrace.json` archive files it loads. If you created or edited annotations through `go-minitrace annotate ...`, sync them first:

```bash
go-minitrace annotate sync --output-dir ./output
```

(In the web UI this is unnecessary: `serve` ATTACHes the live annotation store as schema `anno`, so `anno.annotations` is always current.)

## The sessions_base compatibility view

Saved session-level SQL from the removed DuckDB engine still works against the `sessions_base` view, which exposes the old blob columns (`provenance`, `environment`, `timing`, `metrics`, ...) as JSON text:

```sql
SELECT id, environment->>'agent_framework' AS framework
FROM sessions_base;
```

Prefer the real columns on `sessions` for new queries; see `go-minitrace help query-duckdb` for the migration table.

## Using --sql-file

For queries you run repeatedly, save them to `.sql` files:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql-file ./my-analysis.sql
```

The repo ships ready-made recipes in the `queries/` directory, and queries worth keeping can be promoted to structured commands (`go-minitrace help structured-query-commands`).

## Performance tips

- The database build is cached by archive fingerprint, so repeated queries against the same globs skip the conversion step.
- Add `LIMIT` during exploration; `--max-rows` (default 1000) caps output regardless.
- Long cells are truncated at `--max-cell-chars` (default 4000); raise it when you need full diffs or results.
- Raise `--timeout-ms` (default 5000) for heavy aggregations over large archive sets.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| `no such column: ...` | Typo, or field lives in `raw_json` | Check `go-minitrace help minitrace-schema` or `db.schema()` from JS |
| `access to table "..." is not allowed` | Table outside the sandbox allowlist (including `sqlite_master`) | Query only the normalized tables; use `db.schema()` for introspection |
| NULL values in aggregation | Some sessions don't have the field | Use `WHERE field IS NOT NULL` or `COALESCE` |
| `no such function: UNNEST` | DuckDB-era SQL | Query the child tables directly; see `go-minitrace help query-duckdb` |
| Empty child tables | Archive has no tool calls/turns for the matched sessions | Verify with `SELECT COUNT(*) FROM tool_calls` |

## See also

- `go-minitrace help query-recipes` — ready-to-use SQL examples for common analysis patterns
- `go-minitrace help query-duckdb` — migrating saved DuckDB SQL to the normalized engine
- `go-minitrace help js-api-reference` — JavaScript command handlers with `mt.db()` for reusable multi-query logic
- `go-minitrace help analysis-guide` — where SQL queries fit in the end-to-end analysis workflow
- `go-minitrace help structured-query-commands` — promote useful SQL into a named, reusable structured command
- `go-minitrace help annotation-playbook` — operator workflow for creating, syncing, and validating annotations
- `go-minitrace help query-commands` — query command flags and modes
- `go-minitrace help minitrace-schema` — complete field reference
