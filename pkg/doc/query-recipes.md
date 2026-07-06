---
Title: Query Recipes
Slug: query-recipes
Short: Ready-to-use SQL queries for common minitrace analysis questions
Topics:
- minitrace
- sqlite
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: Example
---

This page collects ready-to-use SQL queries for common analysis questions. Each recipe can be used with `--sql` inline or saved to a file and run with `--sql-file`:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql-file ./my-recipe.sql
```

All recipes target the normalized SQLite schema: `sessions`, `turns`, `tool_calls`, `files`, `annotations`, `metrics`, `events`, `attachments`, `handovers`. Per-session rollups live both as columns on `sessions` (counts, durations) and in the wider `metrics` table (tokens, ratios, subagents).

## Session overview

### How many sessions do I have?

```sql
SELECT COUNT(*) AS total_sessions FROM sessions;
```

### Sessions by quality tier

```sql
SELECT quality, COUNT(*) AS sessions
FROM sessions
GROUP BY quality
ORDER BY sessions DESC;
```

### Sessions by source format

```sql
SELECT source_format, COUNT(*) AS sessions
FROM sessions
GROUP BY source_format
ORDER BY sessions DESC;
```

## Model analysis

### Model usage by framework

```sql
SELECT agent_framework AS framework, model,
       COUNT(*) AS sessions,
       ROUND(AVG(tool_call_count), 1) AS avg_tools
FROM sessions
GROUP BY framework, model
ORDER BY sessions DESC;
```

### Sessions using multiple models

```sql
SELECT s.session_id, s.title,
       m.unique_models AS models,
       m.model_switches AS switches
FROM sessions s
JOIN metrics m USING (session_id)
WHERE m.unique_models > 1
ORDER BY models DESC;
```

## Token analysis

### Token totals by framework

```sql
SELECT s.agent_framework AS framework,
       ROUND(SUM(m.total_input_tokens) / 1e6, 2) AS input_M,
       ROUND(SUM(m.total_output_tokens) / 1e6, 2) AS output_M,
       ROUND(SUM(m.total_cache_read_tokens) / 1e6, 2) AS cache_read_M,
       ROUND(SUM(m.total_cache_creation_tokens) / 1e6, 2) AS cache_create_M
FROM sessions s
JOIN metrics m USING (session_id)
GROUP BY framework;
```

### Average tokens per session

```sql
SELECT s.agent_framework AS framework,
       ROUND(AVG(m.total_input_tokens), 0) AS avg_input,
       ROUND(AVG(m.total_output_tokens), 0) AS avg_output,
       ROUND(AVG(m.median_response_tokens), 0) AS avg_median_response,
       ROUND(AVG(m.max_response_tokens), 0) AS avg_max_response
FROM sessions s
JOIN metrics m USING (session_id)
GROUP BY framework;
```

### Cache efficiency

```sql
SELECT s.agent_framework AS framework,
       ROUND(SUM(m.total_cache_read_tokens) * 100.0 /
             NULLIF(SUM(m.total_cache_read_tokens) + SUM(m.total_input_tokens), 0),
             1) AS cache_hit_pct
FROM sessions s
JOIN metrics m USING (session_id)
GROUP BY framework;
```

## Tool analysis

### Most used tools

```sql
SELECT tool_name AS tool, COUNT(*) AS uses
FROM tool_calls
GROUP BY tool
ORDER BY uses DESC
LIMIT 20;
```

### Tool usage by framework

```sql
SELECT s.agent_framework AS framework,
       tc.operation_type AS operation,
       COUNT(*) AS uses
FROM tool_calls tc
JOIN sessions s USING (session_id)
GROUP BY framework, operation
ORDER BY framework, uses DESC;
```

### Tool success rate

```sql
SELECT tool_name AS tool,
       COUNT(*) AS total,
       SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) AS succeeded,
       ROUND(SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) * 100.0 / COUNT(*), 1) AS success_pct
FROM tool_calls
GROUP BY tool
HAVING COUNT(*) > 10
ORDER BY success_pct ASC;
```

### Slowest tool calls

Pi and claude-code derive `duration_ms` from the emit/result timestamps; codex parses it from the tool output metadata:

```sql
SELECT session_id, tool_name,
       substr(COALESCE(file_path, command, ''), 1, 80) AS target,
       duration_ms
FROM tool_calls
WHERE duration_ms IS NOT NULL
ORDER BY duration_ms DESC
LIMIT 20;
```

### Nonzero exit codes

Exit codes are native for codex and parsed from the `"Error: Exit code N"` result string for claude-code:

```sql
SELECT session_id, tool_name,
       substr(command, 1, 80) AS command,
       exit_code
FROM tool_calls
WHERE exit_code IS NOT NULL AND exit_code != 0
ORDER BY session_id, emitting_turn_index;
```

### Truncated tool outputs

`full_bytes`/`full_hash` describe the full pre-truncation payload:

```sql
SELECT tool_name AS tool,
       COUNT(*) AS truncated_outputs,
       ROUND(AVG(full_bytes), 0) AS avg_full_bytes
FROM tool_calls
WHERE truncated = 1
GROUP BY tool
ORDER BY truncated_outputs DESC;
```

## Timing analysis

### Sessions by duration bucket

```sql
SELECT CASE
         WHEN duration_seconds < 60 THEN '< 1 min'
         WHEN duration_seconds < 600 THEN '1-10 min'
         WHEN duration_seconds < 3600 THEN '10-60 min'
         ELSE '> 1 hour'
       END AS duration_bucket,
       COUNT(*) AS sessions
FROM sessions
WHERE duration_seconds IS NOT NULL
GROUP BY duration_bucket
ORDER BY MIN(duration_seconds);
```

### Activity by hour of day

```sql
SELECT hour_of_day AS hour,
       COUNT(*) AS sessions,
       ROUND(AVG(tool_call_count), 1) AS avg_tools
FROM sessions
WHERE hour_of_day IS NOT NULL
GROUP BY hour
ORDER BY hour;
```

### Activity by day of week

```sql
SELECT CASE day_of_week
         WHEN 0 THEN 'Monday'
         WHEN 1 THEN 'Tuesday'
         WHEN 2 THEN 'Wednesday'
         WHEN 3 THEN 'Thursday'
         WHEN 4 THEN 'Friday'
         WHEN 5 THEN 'Saturday'
         WHEN 6 THEN 'Sunday'
       END AS day,
       COUNT(*) AS sessions
FROM sessions
WHERE day_of_week IS NOT NULL
GROUP BY day, day_of_week
ORDER BY day_of_week;
```

### Daily session volume

```sql
SELECT date(started_at) AS day,
       COUNT(*) AS sessions,
       SUM(tool_call_count) AS total_tools
FROM sessions
WHERE started_at IS NOT NULL
GROUP BY day
ORDER BY day;
```

## Subagent analysis

### Main vs. subagent sessions

```sql
SELECT CASE
         WHEN source_format LIKE '%subagent%' THEN 'subagent'
         ELSE 'main'
       END AS session_type,
       COUNT(*) AS sessions,
       ROUND(AVG(tool_call_count), 1) AS avg_tools,
       ROUND(AVG(turn_count), 1) AS avg_turns
FROM sessions
GROUP BY session_type;
```

### Sessions that spawn the most subagents

```sql
SELECT s.session_id, s.title,
       m.subagent_count AS subagents,
       s.tool_call_count AS tools
FROM sessions s
JOIN metrics m USING (session_id)
WHERE m.subagent_count > 0
ORDER BY subagents DESC
LIMIT 10;
```

### What did spawned agents do?

```sql
SELECT session_id, spawned_agent_type,
       substr(spawned_agent_task_scope, 1, 100) AS task,
       spawned_agent_sub_session_id
FROM tool_calls
WHERE spawned_agent_type IS NOT NULL
ORDER BY session_id, emitting_turn_index;
```

## Annotation analysis

These recipes assume the archive already contains the annotations you care about. If you created or edited annotations through `go-minitrace annotate ...`, run `go-minitrace annotate sync --output-dir ...` first. (In the web UI, `serve` exposes the live annotation store as `anno.annotations` instead.)

### Count annotations by category

```sql
SELECT category, COUNT(*) AS annotations
FROM annotations
GROUP BY category
ORDER BY annotations DESC;
```

### Annotation volume by framework

```sql
SELECT s.agent_framework AS framework, COUNT(*) AS annotations
FROM annotations a
JOIN sessions s USING (session_id)
GROUP BY framework
ORDER BY annotations DESC;
```

### Turn-level vs tool-call-level vs session-level labels

```sql
SELECT scope_type, COUNT(*) AS annotations
FROM annotations
GROUP BY scope_type
ORDER BY annotations DESC;
```

### AI failures by framework

```sql
SELECT s.agent_framework AS framework, COUNT(*) AS ai_failures
FROM annotations a
JOIN sessions s USING (session_id)
WHERE a.category = 'ai-failure'
GROUP BY framework
ORDER BY ai_failures DESC;
```

### Tool-call annotations with target IDs

```sql
SELECT session_id, target_id AS tool_call_id, category, title
FROM annotations
WHERE scope_type = 'tool_call'
ORDER BY session_id;
```

### Minitrace taxonomy codes

Taxonomy mappings are stored as JSON arrays. The sandbox does not allow the `json_each` table-valued function, so match codes with `json_extract` or `LIKE`:

```sql
SELECT session_id, title,
       json_extract(minitrace_taxonomy_json, '$[0]') AS first_code
FROM annotations
WHERE minitrace_taxonomy_json IS NOT NULL;
```

```sql
-- All annotations carrying a specific code
SELECT session_id, category, title
FROM annotations
WHERE minitrace_taxonomy_json LIKE '%"wrong-conclusion"%';
```

## Advanced patterns

### Read ratio distribution by framework

```sql
SELECT s.agent_framework AS framework,
       ROUND(m.read_ratio, 1) AS read_ratio_bucket,
       COUNT(*) AS sessions
FROM sessions s
JOIN metrics m USING (session_id)
WHERE m.read_ratio IS NOT NULL
GROUP BY framework, read_ratio_bucket
ORDER BY framework, read_ratio_bucket;
```

### Sessions with high idle ratio

```sql
SELECT s.session_id, s.title,
       ROUND(m.idle_ratio, 2) AS idle_ratio,
       ROUND(s.duration_seconds / 60, 1) AS total_min,
       ROUND(s.active_duration_seconds / 60, 1) AS active_min
FROM sessions s
JOIN metrics m USING (session_id)
WHERE m.idle_ratio > 0.5
ORDER BY idle_ratio DESC
LIMIT 20;
```

### Files touched most often

```sql
SELECT path, COUNT(*) AS touches,
       SUM(CASE WHEN operation_type != 'READ' THEN 1 ELSE 0 END) AS writes
FROM files
GROUP BY path
ORDER BY touches DESC
LIMIT 20;
```

## Using the queries/ directory

The repository ships the same recipes (and more) as saved SQL files in `queries/`, organized by topic (`overview/`, `tools/`, `timing/`, `files/`, `framework-metadata/`). They run directly:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql-file queries/overview/framework-summary.sql
```

The same directory is `serve`'s default `--query-dir`, so every file also appears as a saved query in the web UI. There is no separate loading step: the normalized database is built (and cached) from the archives automatically.

## See also

- `go-minitrace help writing-queries` — JSON access, join patterns, and filtering in depth
- `go-minitrace help analysis-guide` — end-to-end workflow where these recipes fit into the full analysis loop
- `go-minitrace help js-api-reference` — translate useful recipes into `mt.db()` JS handlers when you need reusable multi-query logic
- `go-minitrace help structured-query-commands` — promote a useful recipe into a named, reusable structured command
- `go-minitrace help annotation-playbook` — operator workflow for creating, syncing, and validating annotations
- `go-minitrace help query-commands` — full query flag reference
- `go-minitrace help minitrace-schema` — field reference for all queryable fields
