---
Title: End-to-End Analysis Walkthrough
Slug: end-to-end-analysis
Short: Complete walkthrough analyzing your Claude Code and Pi usage from raw sessions to insights
Topics:
- minitrace
- tutorial
- duckdb
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: Application
---

This walkthrough takes a realistic scenario — analyzing your Claude Code and Pi usage — through the complete go-minitrace pipeline from discovery to actionable insights. Every command is runnable as shown.

## Scenario

You want to understand your AI agent usage over the past month: how many sessions, which models, how tools are used, how long sessions last, and how usage differs between Claude Code and Pi.

## Step 1: Survey your session stores

Start by discovering what data is available:

```bash
go-minitrace discover claude-code --output json | jq length
go-minitrace discover pi --output json | jq length
```

Example output: 2,502 Claude Code sessions, 52 Pi sessions.

Check for subagent sessions in Claude Code (these are nested inside project directories):

```bash
go-minitrace discover claude-code --output json \
  | jq '[.[].format_hint] | group_by(.) | map({key: .[0], count: length}) | from_entries'
```

## Step 2: Convert to minitrace

Convert both frameworks into the same output directory:

```bash
go-minitrace convert claude-code --output-dir ./analysis
go-minitrace convert pi --output-dir ./analysis
```

Verify the conversion produced the expected output:

```bash
ls ./analysis/active/
cat ./analysis/manifest.json | jq '.statistics'
```

The manifest shows total sessions, quality distribution, and date range.

Quick validation:

```bash
go-minitrace validate --path ./analysis --recursive --output json \
  | jq '[.[] | select(.valid_json)] | length'
```

## Step 3: Get the big picture

Start with the framework summary:

```bash
go-minitrace query duckdb \
  --archive-glob './analysis/active/*/*.minitrace.json' \
  --preset framework-summary
```

This shows per-framework averages for tools, turns, read ratio, duration, and time to first action. You will immediately see the difference in scale between Claude Code and Pi sessions.

## Step 4: Model usage analysis

See which models you rely on:

```bash
go-minitrace query duckdb \
  --archive-glob './analysis/active/*/*.minitrace.json' \
  --sql "
    SELECT
      environment->>'agent_framework' AS framework,
      environment->>'model' AS model,
      COUNT(*) AS sessions,
      ROUND(AVG(CAST(metrics->>'tool_call_count' AS INT)), 1) AS avg_tools,
      ROUND(AVG(CAST(metrics->>'turn_count' AS INT)), 1) AS avg_turns
    FROM sessions_base
    GROUP BY framework, model
    ORDER BY sessions DESC
  "
```

## Step 5: Token consumption

Understand your token spend:

```bash
go-minitrace query duckdb \
  --archive-glob './analysis/active/*/*.minitrace.json' \
  --sql "
    SELECT
      environment->>'agent_framework' AS framework,
      ROUND(SUM(CAST(metrics->>'total_input_tokens' AS BIGINT)) / 1e6, 2) AS input_M,
      ROUND(SUM(CAST(metrics->>'total_output_tokens' AS BIGINT)) / 1e6, 2) AS output_M,
      ROUND(SUM(CAST(metrics->>'total_cache_read_tokens' AS BIGINT)) / 1e6, 2) AS cache_read_M
    FROM sessions_base
    GROUP BY framework
  "
```

Cache read tokens are often significantly larger than input tokens because Anthropic's prompt caching serves repeated context from cache. A high cache-to-input ratio means the caching is working well.

## Step 6: Tool usage patterns

See which tools are used most and how the usage pattern differs between frameworks:

```bash
go-minitrace query duckdb \
  --archive-glob './analysis/active/*/*.minitrace.json' \
  --preset tool-operation-breakdown
```

For a more detailed tool-name view:

```bash
go-minitrace query duckdb \
  --archive-glob './analysis/active/*/*.minitrace.json' \
  --sql "
    SELECT
      environment->>'agent_framework' AS framework,
      REPLACE(CAST(json_extract(tc, '\$.tool_name') AS VARCHAR), '\"', '') AS tool,
      COUNT(*) AS uses
    FROM sessions_base, UNNEST(tool_calls) AS t(tc)
    GROUP BY framework, tool
    ORDER BY framework, uses DESC
  " --output json | jq 'group_by(.framework) | .[] | {framework: .[0].framework, tools: [.[:10] | .[] | {tool, uses}]}'
```

## Step 7: Temporal patterns

When do you use AI agents?

```bash
go-minitrace query duckdb \
  --archive-glob './analysis/active/*/*.minitrace.json' \
  --sql "
    SELECT
      CAST(timing->>'hour_of_day' AS INT) AS hour,
      COUNT(*) AS sessions,
      ROUND(AVG(CAST(metrics->>'tool_call_count' AS INT)), 1) AS avg_tools
    FROM sessions_base
    WHERE timing->>'hour_of_day' IS NOT NULL
    GROUP BY hour
    ORDER BY hour
  "
```

## Step 8: Find your most complex sessions

Identify sessions that may be interesting to review:

```bash
go-minitrace query duckdb \
  --archive-glob './analysis/active/*/*.minitrace.json' \
  --sql "
    SELECT
      id, title,
      environment->>'agent_framework' AS framework,
      CAST(metrics->>'turn_count' AS INT) AS turns,
      CAST(metrics->>'tool_call_count' AS INT) AS tools,
      ROUND(CAST(timing->>'duration_seconds' AS DOUBLE) / 60, 1) AS minutes,
      quality
    FROM sessions_base
    WHERE provenance->>'source_format' NOT LIKE '%subagent%'
    ORDER BY tools DESC
    LIMIT 20
  "
```

## Step 9: Subagent analysis

If you use Claude Code with subagent delegation, analyze that separately:

```bash
go-minitrace query duckdb \
  --archive-glob './analysis/active/*/*.minitrace.json' \
  --sql "
    SELECT
      CASE
        WHEN provenance->>'source_format' LIKE '%subagent%' THEN 'subagent'
        ELSE 'main'
      END AS session_type,
      COUNT(*) AS sessions,
      ROUND(AVG(CAST(metrics->>'tool_call_count' AS INT)), 1) AS avg_tools,
      ROUND(AVG(CAST(metrics->>'turn_count' AS INT)), 1) AS avg_turns
    FROM sessions_base
    WHERE environment->>'agent_framework' = 'claude-code'
    GROUP BY session_type
  "
```

## Step 10: Export for further analysis

Save results to CSV for spreadsheet analysis:

```bash
go-minitrace query duckdb \
  --archive-glob './analysis/active/*/*.minitrace.json' \
  --preset session-list --output csv > sessions.csv
```

Or create a persistent DuckDB database for interactive exploration:

```bash
duckdb analysis.duckdb
```

Then inside DuckDB:

```sql
.read queries/load.sql
-- Now run any query interactively
SELECT COUNT(*) FROM sessions_base;
```

## See also

- `go-minitrace help getting-started` — shorter getting-started tutorial
- `go-minitrace help writing-duckdb-queries` — how to write the SQL queries used above
- `go-minitrace help duckdb-query-recipes` — more query recipes to try
