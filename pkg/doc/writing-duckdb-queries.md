---
Title: Writing DuckDB Queries
Slug: writing-duckdb-queries
Short: Learn DuckDB JSON operators and query patterns for the minitrace schema
Topics:
- minitrace
- duckdb
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial teaches you how to write DuckDB SQL queries against minitrace archives. It covers the JSON access syntax, type casting, array operations, and common patterns you need for analysis.

All examples use the `query duckdb` command with `--sql`. The loaded table is called `sessions_base` by default.

## The sessions_base table

When go-minitrace loads an archive, it creates a table with these columns:

- **Top-level strings**: `id`, `title`, `summary`, `classification`, `profile` — directly queryable
- **JSON objects**: `provenance`, `flags`, `environment`, `operational_context`, `timing`, `metrics` — access fields with `->>`
- **JSON arrays**: `turns`, `tool_calls`, `annotations` — iterate with `UNNEST`

## Accessing JSON fields

DuckDB uses the `->>` operator to extract a string value from a JSON column:

```sql
-- Extract the model from the environment JSON
SELECT environment->>'model' AS model FROM sessions_base;

-- Extract nested fields
SELECT environment->>'agent_framework' AS framework FROM sessions_base;
SELECT provenance->>'source_format' AS source FROM sessions_base;
SELECT timing->>'started_at' AS started FROM sessions_base;
SELECT metrics->>'turn_count' AS turns FROM sessions_base;
```

The `->` operator (single arrow) returns JSON; `->>` (double arrow) returns a string. For most queries you want `->>` because it gives you a plain value you can GROUP BY, filter, or display.

## Type casting

JSON field extraction with `->>` always returns a string. To do math, you must CAST:

```sql
-- Integer fields
CAST(metrics->>'turn_count' AS INT)
CAST(metrics->>'tool_call_count' AS INT)

-- Float fields
CAST(metrics->>'read_ratio' AS DOUBLE)
CAST(timing->>'duration_seconds' AS DOUBLE)

-- Large numbers (token counts can be very large)
CAST(metrics->>'total_input_tokens' AS BIGINT)
CAST(metrics->>'total_output_tokens' AS BIGINT)
```

Common pattern for aggregation:

```sql
SELECT
  environment->>'agent_framework' AS framework,
  COUNT(*) AS sessions,
  ROUND(AVG(CAST(metrics->>'tool_call_count' AS INT)), 1) AS avg_tools,
  ROUND(AVG(CAST(timing->>'duration_seconds' AS DOUBLE)), 0) AS avg_duration_s
FROM sessions_base
GROUP BY framework;
```

## Working with arrays: UNNEST

The `turns`, `tool_calls`, and `annotations` columns are JSON arrays. To query individual elements, use `UNNEST`:

```sql
-- Count tool calls by name across all sessions
SELECT
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool_name,
  COUNT(*) AS invocations
FROM sessions_base,
UNNEST(tool_calls) AS t(tc)
GROUP BY tool_name
ORDER BY invocations DESC;
```

The `UNNEST(tool_calls) AS t(tc)` clause expands each tool call array element into a row. The variable `tc` holds one JSON element that you can query with `json_extract()`.

### Extracting fields from array elements

Inside an unnested element, use `json_extract(element, '$.field')`:

```sql
-- Tool call details
json_extract(tc, '$.tool_name')      -- tool name
json_extract(tc, '$.operation_type') -- READ, MODIFY, etc.
json_extract(tc, '$.output.success') -- whether it succeeded
json_extract(tc, '$.timestamp')      -- when it ran
```

For string extraction from within `json_extract`, wrap in `CAST(... AS VARCHAR)` and strip quotes:

```sql
REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool_name
```

### Querying annotations

Annotations follow the same `UNNEST` pattern, but there is one workflow-specific nuance: `go-minitrace query duckdb` reads the `.minitrace.json` archive files it loads. If you created or edited annotations through `go-minitrace annotate ...`, sync them first:

```bash
go-minitrace annotate sync --output-dir ./output
```

Then query them like any other JSON array:

```sql
SELECT
  id AS session_id,
  REPLACE(CAST(json_extract(ann, '$.scope.type') AS VARCHAR), '"', '') AS scope_type,
  REPLACE(CAST(json_extract(ann, '$.scope.target_id') AS VARCHAR), '"', '') AS target_id,
  REPLACE(CAST(json_extract(ann, '$.content.category') AS VARCHAR), '"', '') AS category,
  REPLACE(CAST(json_extract(ann, '$.content.title') AS VARCHAR), '"', '') AS title
FROM sessions_base,
     UNNEST(annotations) AS a(ann);
```

Common annotation paths:

- `$.scope.type`
- `$.scope.target_id`
- `$.content.category`
- `$.content.title`
- `$.content.detail`
- `$.taxonomy_mappings.minitrace`
- `$.classification`

Examples:

```sql
-- Count annotations by category
SELECT
  REPLACE(CAST(json_extract(ann, '$.content.category') AS VARCHAR), '"', '') AS category,
  COUNT(*) AS n
FROM sessions_base,
     UNNEST(annotations) AS a(ann)
GROUP BY category
ORDER BY n DESC;
```

```sql
-- Filter to tool-call-level annotations only
SELECT
  id AS session_id,
  REPLACE(CAST(json_extract(ann, '$.scope.target_id') AS VARCHAR), '"', '') AS tool_call_id,
  REPLACE(CAST(json_extract(ann, '$.content.title') AS VARCHAR), '"', '') AS title
FROM sessions_base,
     UNNEST(annotations) AS a(ann)
WHERE REPLACE(CAST(json_extract(ann, '$.scope.type') AS VARCHAR), '"', '') = 'tool_call';
```

### Multiple UNNEST

You can unnest multiple arrays in the same query, but each creates a cross-product. Usually you want to unnest one array per query:

```sql
-- Per-turn analysis
SELECT
  id,
  REPLACE(CAST(json_extract(turn, '$.role') AS VARCHAR), '"', '') AS role,
  CAST(json_extract(turn, '$.index') AS INT) AS turn_index
FROM sessions_base,
UNNEST(turns) AS t(turn)
WHERE id = 'some-session-id';
```

## Filtering patterns

### By framework

```sql
WHERE environment->>'agent_framework' = 'claude-code'
WHERE environment->>'agent_framework' IN ('claude-code', 'pi')
```

### By source format (exclude subagents)

```sql
WHERE provenance->>'source_format' NOT LIKE '%subagent%'
```

### By quality tier

```sql
WHERE quality = 'A'
```

### By date range

```sql
WHERE timing->>'started_at' >= '2026-03-01'
  AND timing->>'started_at' < '2026-04-01'
```

### By session size

```sql
WHERE CAST(metrics->>'tool_call_count' AS INT) > 10
  AND CAST(metrics->>'turn_count' AS INT) > 5
```

## Common query patterns

### Group-by with aggregation

```sql
SELECT
  environment->>'model' AS model,
  COUNT(*) AS sessions,
  ROUND(AVG(CAST(metrics->>'tool_call_count' AS INT)), 1) AS avg_tools
FROM sessions_base
GROUP BY model
ORDER BY sessions DESC;
```

### Top-N

```sql
SELECT id, title,
  CAST(metrics->>'tool_call_count' AS INT) AS tools
FROM sessions_base
ORDER BY tools DESC
LIMIT 10;
```

### Conditional aggregation

```sql
SELECT
  environment->>'agent_framework' AS framework,
  COUNT(*) FILTER (WHERE quality = 'A') AS quality_a,
  COUNT(*) FILTER (WHERE quality = 'B') AS quality_b,
  COUNT(*) FILTER (WHERE quality = 'C') AS quality_c
FROM sessions_base
GROUP BY framework;
```

### Temporal bucketing

```sql
SELECT
  CAST(timing->>'started_at' AS DATE) AS day,
  COUNT(*) AS sessions
FROM sessions_base
WHERE timing->>'started_at' IS NOT NULL
GROUP BY day
ORDER BY day;
```

## Using --sql-file

For queries you run repeatedly, save them to `.sql` files:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql-file ./my-analysis.sql
```

The repo ships query recipes in the `queries/` directory. You can use these with the external DuckDB CLI too:

```bash
duckdb analysis.duckdb -init queries/load.sql -f queries/framework-summary.sql
```

## Performance tips

- The `ignore_errors = true` in the load step means malformed files are silently skipped. If you get unexpected row counts, validate first.
- For large archives, use `--db-path` with a file to avoid reloading on every query.
- Add `LIMIT` during exploration to avoid printing thousands of rows.
- DuckDB is columnar, so queries that access a few fields from many rows are fast even on large archives.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| `Invalid Input Error: ...` | Trying math on a string extracted by `->>` | Wrap in `CAST(... AS INT)` or `CAST(... AS DOUBLE)` |
| `Binder Error: column not found` | Typo in column name or missing JSON field | Check spelling against `go-minitrace help minitrace-schema` |
| NULL values in aggregation | Some sessions don't have the field | Use `WHERE field IS NOT NULL` or `COALESCE` |
| Empty results from UNNEST | The array column is empty for all matched sessions | Check that sessions have tool_calls or turns data |

## See also

- `go-minitrace help js-api-reference` — use the same SQL patterns inside `mt.query()` and `mt.queryOne()` in JS command handlers
- `go-minitrace help analysis-guide` — where SQL queries fit in the end-to-end analysis workflow
- `go-minitrace help structured-query-commands` — promote useful SQL into a named, reusable structured command
- `go-minitrace help duckdb-query-recipes` — ready-to-use SQL examples for common analysis patterns
- `go-minitrace help annotation-playbook` — operator workflow for creating, syncing, and validating annotations
- `go-minitrace help query-commands` — query command flags and modes
- `go-minitrace help minitrace-schema` — complete field reference
