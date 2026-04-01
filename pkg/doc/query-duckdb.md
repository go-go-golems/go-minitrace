---
Title: Query DuckDB Examples
Slug: query-duckdb
Short: Worked examples of querying minitrace archives with the DuckDB backend
Topics:
- minitrace
- duckdb
- glazed
Commands:
- query duckdb
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: Example
---

This page shows concrete query examples against converted minitrace archives. For the full flag reference, see `go-minitrace help query-commands`. For guidance on writing your own queries, see `go-minitrace help writing-duckdb-queries`.

All examples assume an archive at `./output/active/*/*.minitrace.json`. Adjust the `--archive-glob` for your setup.

## Using built-in presets

List all sessions sorted by start time:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset session-list
```

Compare frameworks side by side:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset framework-summary \
  --output json
```

See how tools are used across frameworks:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset tool-operation-breakdown
```

## Custom SQL examples

### Count sessions by model

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "
    SELECT
      environment->>'model' AS model,
      COUNT(*) AS sessions,
      ROUND(AVG(CAST(metrics->>'tool_call_count' AS INT)), 1) AS avg_tools
    FROM sessions_base
    GROUP BY model
    ORDER BY sessions DESC
  "
```

### Token usage by framework

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "
    SELECT
      environment->>'agent_framework' AS framework,
      ROUND(SUM(CAST(metrics->>'total_input_tokens' AS BIGINT)) / 1e6, 1) AS input_M,
      ROUND(SUM(CAST(metrics->>'total_output_tokens' AS BIGINT)) / 1e6, 1) AS output_M,
      ROUND(SUM(CAST(metrics->>'total_cache_read_tokens' AS BIGINT)) / 1e6, 1) AS cache_M
    FROM sessions_base
    GROUP BY framework
  "
```

### Most active tools

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "
    SELECT
      REPLACE(CAST(json_extract(tc, '\$.tool_name') AS VARCHAR), '\"', '') AS tool,
      COUNT(*) AS invocations
    FROM sessions_base, UNNEST(tool_calls) AS t(tc)
    GROUP BY tool
    ORDER BY invocations DESC
    LIMIT 20
  "
```

### Sessions by hour of day

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "
    SELECT
      CAST(timing->>'hour_of_day' AS INT) AS hour,
      COUNT(*) AS sessions
    FROM sessions_base
    WHERE timing->>'hour_of_day' IS NOT NULL
    GROUP BY hour
    ORDER BY hour
  "
```

### Find long-running sessions

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "
    SELECT
      id, title,
      ROUND(CAST(timing->>'duration_seconds' AS DOUBLE) / 3600, 1) AS hours,
      CAST(metrics->>'turn_count' AS INT) AS turns,
      CAST(metrics->>'tool_call_count' AS INT) AS tools
    FROM sessions_base
    WHERE provenance->>'source_format' NOT LIKE '%subagent%'
    ORDER BY hours DESC
    LIMIT 10
  "
```

### Filter to non-subagent sessions only

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "
    SELECT COUNT(*) AS main_sessions
    FROM sessions_base
    WHERE provenance->>'source_format' NOT LIKE '%subagent%'
  "
```

## Using a SQL file

Save a query to a file:

```sql
-- my-query.sql
SELECT
  environment->>'agent_framework' AS framework,
  environment->>'model' AS model,
  COUNT(*) AS sessions
FROM sessions_base
GROUP BY framework, model
ORDER BY sessions DESC;
```

Run it:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql-file ./my-query.sql
```

## Combining with external tools

Pipe JSON output to jq for further filtering:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset session-list --output json \
  | jq '[.[] | select(.turns > 50)]'
```

Export to CSV for spreadsheets:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset framework-summary --output csv > summary.csv
```

## See also

- `go-minitrace help query-commands` — full flag reference
- `go-minitrace help writing-duckdb-queries` — DuckDB JSON syntax and query patterns
- `go-minitrace help duckdb-query-recipes` — more query recipes
