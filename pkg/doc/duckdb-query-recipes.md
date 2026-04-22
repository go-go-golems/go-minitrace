---
Title: DuckDB Query Recipes
Slug: duckdb-query-recipes
Short: Ready-to-use SQL queries for common minitrace analysis questions
Topics:
- minitrace
- duckdb
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: Example
---

This page collects ready-to-use SQL queries for common analysis questions. Each recipe can be used with `--sql` inline or saved to a file and run with `--sql-file`.

All recipes assume the table is called `sessions_base` (the default).

## Session overview

### How many sessions do I have?

```sql
SELECT COUNT(*) AS total_sessions FROM sessions_base;
```

### Sessions by quality tier

```sql
SELECT quality, COUNT(*) AS sessions
FROM sessions_base
GROUP BY quality
ORDER BY sessions DESC;
```

### Sessions by source format

```sql
SELECT
  provenance->>'source_format' AS source_format,
  COUNT(*) AS sessions
FROM sessions_base
GROUP BY source_format
ORDER BY sessions DESC;
```

## Model analysis

### Model usage by framework

```sql
SELECT
  environment->>'agent_framework' AS framework,
  environment->>'model' AS model,
  COUNT(*) AS sessions,
  ROUND(AVG(CAST(metrics->>'tool_call_count' AS INT)), 1) AS avg_tools
FROM sessions_base
GROUP BY framework, model
ORDER BY sessions DESC;
```

### Sessions using multiple models

```sql
SELECT
  id, title,
  CAST(metrics->>'unique_models' AS INT) AS models,
  CAST(metrics->>'model_switches' AS INT) AS switches
FROM sessions_base
WHERE CAST(metrics->>'unique_models' AS INT) > 1
ORDER BY models DESC;
```

## Token analysis

### Token totals by framework

```sql
SELECT
  environment->>'agent_framework' AS framework,
  ROUND(SUM(CAST(metrics->>'total_input_tokens' AS BIGINT)) / 1e6, 2) AS input_M,
  ROUND(SUM(CAST(metrics->>'total_output_tokens' AS BIGINT)) / 1e6, 2) AS output_M,
  ROUND(SUM(CAST(metrics->>'total_cache_read_tokens' AS BIGINT)) / 1e6, 2) AS cache_read_M,
  ROUND(SUM(CAST(metrics->>'total_cache_creation_tokens' AS BIGINT)) / 1e6, 2) AS cache_create_M
FROM sessions_base
GROUP BY framework;
```

### Average tokens per session

```sql
SELECT
  environment->>'agent_framework' AS framework,
  ROUND(AVG(CAST(metrics->>'total_input_tokens' AS BIGINT)), 0) AS avg_input,
  ROUND(AVG(CAST(metrics->>'total_output_tokens' AS BIGINT)), 0) AS avg_output,
  ROUND(AVG(CAST(metrics->>'median_response_tokens' AS INT)), 0) AS avg_median_response,
  ROUND(AVG(CAST(metrics->>'max_response_tokens' AS INT)), 0) AS avg_max_response
FROM sessions_base
GROUP BY framework;
```

### Cache efficiency

```sql
SELECT
  environment->>'agent_framework' AS framework,
  ROUND(
    SUM(CAST(metrics->>'total_cache_read_tokens' AS BIGINT)) * 100.0 /
    NULLIF(SUM(CAST(metrics->>'total_cache_read_tokens' AS BIGINT)) +
           SUM(CAST(metrics->>'total_input_tokens' AS BIGINT)), 0),
    1
  ) AS cache_hit_pct
FROM sessions_base
GROUP BY framework;
```

## Tool analysis

### Most used tools

```sql
SELECT
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  COUNT(*) AS uses
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
GROUP BY tool
ORDER BY uses DESC
LIMIT 20;
```

### Tool usage by framework

```sql
SELECT
  environment->>'agent_framework' AS framework,
  REPLACE(CAST(json_extract(tc, '$.operation_type') AS VARCHAR), '"', '') AS operation,
  COUNT(*) AS uses
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
GROUP BY framework, operation
ORDER BY framework, uses DESC;
```

### Tool success rate

```sql
SELECT
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  COUNT(*) AS total,
  SUM(CASE WHEN CAST(json_extract(tc, '$.output.success') AS BOOL) THEN 1 ELSE 0 END) AS succeeded,
  ROUND(
    SUM(CASE WHEN CAST(json_extract(tc, '$.output.success') AS BOOL) THEN 1 ELSE 0 END) * 100.0 / COUNT(*),
    1
  ) AS success_pct
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
GROUP BY tool
HAVING COUNT(*) > 10
ORDER BY success_pct ASC;
```

### Truncated tool outputs

```sql
SELECT
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  COUNT(*) AS truncated_outputs,
  ROUND(AVG(CAST(json_extract(tc, '$.output.full_bytes') AS INT)), 0) AS avg_full_bytes
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE CAST(json_extract(tc, '$.output.truncated') AS BOOL) = true
GROUP BY tool
ORDER BY truncated_outputs DESC;
```

## Timing analysis

### Sessions by duration bucket

```sql
SELECT
  CASE
    WHEN CAST(timing->>'duration_seconds' AS DOUBLE) < 60 THEN '< 1 min'
    WHEN CAST(timing->>'duration_seconds' AS DOUBLE) < 600 THEN '1-10 min'
    WHEN CAST(timing->>'duration_seconds' AS DOUBLE) < 3600 THEN '10-60 min'
    ELSE '> 1 hour'
  END AS duration_bucket,
  COUNT(*) AS sessions
FROM sessions_base
WHERE timing->>'duration_seconds' IS NOT NULL
GROUP BY duration_bucket
ORDER BY MIN(CAST(timing->>'duration_seconds' AS DOUBLE));
```

### Activity by hour of day

```sql
SELECT
  CAST(timing->>'hour_of_day' AS INT) AS hour,
  COUNT(*) AS sessions,
  ROUND(AVG(CAST(metrics->>'tool_call_count' AS INT)), 1) AS avg_tools
FROM sessions_base
WHERE timing->>'hour_of_day' IS NOT NULL
GROUP BY hour
ORDER BY hour;
```

### Activity by day of week

```sql
SELECT
  CASE CAST(timing->>'day_of_week' AS INT)
    WHEN 0 THEN 'Monday'
    WHEN 1 THEN 'Tuesday'
    WHEN 2 THEN 'Wednesday'
    WHEN 3 THEN 'Thursday'
    WHEN 4 THEN 'Friday'
    WHEN 5 THEN 'Saturday'
    WHEN 6 THEN 'Sunday'
  END AS day,
  COUNT(*) AS sessions
FROM sessions_base
WHERE timing->>'day_of_week' IS NOT NULL
GROUP BY day, CAST(timing->>'day_of_week' AS INT)
ORDER BY CAST(timing->>'day_of_week' AS INT);
```

### Daily session volume

```sql
SELECT
  CAST(timing->>'started_at' AS DATE) AS day,
  COUNT(*) AS sessions,
  SUM(CAST(metrics->>'tool_call_count' AS INT)) AS total_tools
FROM sessions_base
WHERE timing->>'started_at' IS NOT NULL
GROUP BY day
ORDER BY day;
```

## Subagent analysis

### Main vs. subagent sessions

```sql
SELECT
  CASE
    WHEN provenance->>'source_format' LIKE '%subagent%' THEN 'subagent'
    ELSE 'main'
  END AS session_type,
  COUNT(*) AS sessions,
  ROUND(AVG(CAST(metrics->>'tool_call_count' AS INT)), 1) AS avg_tools,
  ROUND(AVG(CAST(metrics->>'turn_count' AS INT)), 1) AS avg_turns
FROM sessions_base
GROUP BY session_type;
```

### Sessions that spawn the most subagents

```sql
SELECT
  id, title,
  CAST(metrics->>'subagent_count' AS INT) AS subagents,
  CAST(metrics->>'tool_call_count' AS INT) AS tools
FROM sessions_base
WHERE CAST(metrics->>'subagent_count' AS INT) > 0
ORDER BY subagents DESC
LIMIT 10;
```

## Annotation analysis

These recipes assume the archive already contains the annotations you care about. If you created or edited annotations through `go-minitrace annotate ...`, run `go-minitrace annotate sync --output-dir ...` first.

### Count annotations by category

```sql
SELECT
  REPLACE(CAST(json_extract(ann, '$.content.category') AS VARCHAR), '"', '') AS category,
  COUNT(*) AS annotations
FROM sessions_base,
     UNNEST(annotations) AS a(ann)
GROUP BY category
ORDER BY annotations DESC;
```

### Annotation volume by framework

```sql
SELECT
  environment->>'agent_framework' AS framework,
  COUNT(*) AS annotations
FROM sessions_base,
     UNNEST(annotations) AS a(ann)
GROUP BY framework
ORDER BY annotations DESC;
```

### Turn-level vs tool-call-level vs session-level labels

```sql
SELECT
  REPLACE(CAST(json_extract(ann, '$.scope.type') AS VARCHAR), '"', '') AS scope_type,
  COUNT(*) AS annotations
FROM sessions_base,
     UNNEST(annotations) AS a(ann)
GROUP BY scope_type
ORDER BY annotations DESC;
```

### AI failures by framework

```sql
SELECT
  environment->>'agent_framework' AS framework,
  COUNT(*) AS ai_failures
FROM sessions_base,
     UNNEST(annotations) AS a(ann)
WHERE REPLACE(CAST(json_extract(ann, '$.content.category') AS VARCHAR), '"', '') = 'ai-failure'
GROUP BY framework
ORDER BY ai_failures DESC;
```

### Tool-call annotations with target IDs

```sql
SELECT
  id AS session_id,
  REPLACE(CAST(json_extract(ann, '$.scope.target_id') AS VARCHAR), '"', '') AS tool_call_id,
  REPLACE(CAST(json_extract(ann, '$.content.category') AS VARCHAR), '"', '') AS category,
  REPLACE(CAST(json_extract(ann, '$.content.title') AS VARCHAR), '"', '') AS title
FROM sessions_base,
     UNNEST(annotations) AS a(ann)
WHERE REPLACE(CAST(json_extract(ann, '$.scope.type') AS VARCHAR), '"', '') = 'tool_call'
ORDER BY session_id;
```

### Minitrace taxonomy code usage

```sql
SELECT
  REPLACE(CAST(json_extract(code, '$') AS VARCHAR), '"', '') AS taxonomy_code,
  COUNT(*) AS annotations
FROM sessions_base,
     UNNEST(annotations) AS a(ann),
     UNNEST(CAST(json_extract(ann, '$.taxonomy_mappings.minitrace') AS JSON[])) AS t(code)
GROUP BY taxonomy_code
ORDER BY annotations DESC;
```

## Advanced patterns

### Read ratio distribution by framework

```sql
SELECT
  environment->>'agent_framework' AS framework,
  ROUND(CAST(metrics->>'read_ratio' AS DOUBLE), 1) AS read_ratio_bucket,
  COUNT(*) AS sessions
FROM sessions_base
WHERE metrics->>'read_ratio' IS NOT NULL
GROUP BY framework, read_ratio_bucket
ORDER BY framework, read_ratio_bucket;
```

### Sessions with high idle ratio

```sql
SELECT
  id, title,
  ROUND(CAST(metrics->>'idle_ratio' AS DOUBLE), 2) AS idle_ratio,
  ROUND(CAST(timing->>'duration_seconds' AS DOUBLE) / 60, 1) AS total_min,
  ROUND(CAST(timing->>'active_duration_seconds' AS DOUBLE) / 60, 1) AS active_min
FROM sessions_base
WHERE CAST(metrics->>'idle_ratio' AS DOUBLE) > 0.5
ORDER BY idle_ratio DESC
LIMIT 20;
```

## Using the queries/ directory

The repository ships standalone SQL files in `queries/` for use with the external DuckDB CLI:

```bash
duckdb analysis.duckdb
```

```sql
.read queries/load.sql
.read queries/session-list.sql
.read queries/framework-summary.sql
.read queries/tool-operation-breakdown.sql
.read queries/timing-analysis.sql
.read queries/annotations.sql
```

The `queries/load.sql` file defaults to the glob `./output/active/*/*.minitrace.json`. Edit it if your archive is elsewhere.

## See also

- `go-minitrace help analysis-guide` — end-to-end workflow where these recipes fit into the full analysis loop
- `go-minitrace help js-api-reference` — port any of these SQL patterns into a `mt.query()` JS handler
- `go-minitrace help structured-query-commands` — promote a useful recipe into a named, reusable structured command
- `go-minitrace help query-duckdb` — how to run these recipes with `--preset`, `--sql`, and `--sql-file`
- `go-minitrace help writing-duckdb-queries` — JSON access, UNNEST, casting, and annotation join patterns in depth
- `go-minitrace help annotation-playbook` — operator workflow for creating, syncing, and validating annotations
- `go-minitrace help query-commands` — full query flag reference
- `go-minitrace help minitrace-schema` — field reference for all queryable fields
