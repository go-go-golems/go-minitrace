# Query Notes

These queries target the normalized SQLite schema used by `go-minitrace query run`
(tables: `sessions`, `turns`, `tool_calls`, `turn_tool_calls`, `files`,
`annotations`, `handovers`, `metrics`, `attachments`, `events`).

## Stable columns

On `sessions` (one row per session):

- `agent_framework`, `provider_hint`, `model`
- `working_directory`, `git_branch`, `quality`, `source_format`
- `turn_count`, `tool_call_count`, `read_count`, `modify_count`, `create_count`, `execute_count`
- `started_at`, `ended_at`, `duration_seconds`, `active_duration_seconds`

On `metrics` (join `USING (session_id)`): `idle_ratio`, `time_to_first_action`,
`read_ratio`, token totals, `subagent_count`, `session_cost`.

On `tool_calls`: `tool_name`, `operation_type`, `file_path`, `command`,
`success`, `error`, `exit_code`, `duration_ms`, `framework_metadata_json`.

Long-tail fields: `json_extract(raw_json, '$....')` on any table.

## Session counts

```sql
SELECT COUNT(*) AS sessions, COUNT(DISTINCT session_id) AS distinct_ids
FROM sessions;
```

## Framework and model summary

```sql
SELECT
  agent_framework AS framework,
  provider_hint AS provider,
  model,
  COUNT(*) AS sessions,
  SUM(turn_count) AS turns,
  SUM(tool_call_count) AS tool_calls
FROM sessions
GROUP BY framework, provider, model
ORDER BY framework, model;
```

## Tool frequency

```sql
SELECT
  s.agent_framework AS framework,
  tc.tool_name,
  COUNT(*) AS calls
FROM tool_calls tc
JOIN sessions s USING (session_id)
GROUP BY framework, tc.tool_name
ORDER BY framework, calls DESC, tc.tool_name;
```

## Activity profile

```sql
SELECT
  s.session_id,
  s.agent_framework AS framework,
  s.read_count AS reads,
  s.modify_count AS modifies,
  s.create_count AS creates,
  s.execute_count AS executes,
  m.delegate_count AS delegates
FROM sessions s
JOIN metrics m USING (session_id)
ORDER BY framework, s.session_id;
```

## Timing summary

```sql
SELECT
  s.agent_framework AS framework,
  AVG(m.idle_ratio) AS avg_idle_ratio,
  AVG(m.time_to_first_action) AS avg_time_to_first_action,
  MIN(s.started_at) AS earliest,
  MAX(s.ended_at) AS latest
FROM sessions s
JOIN metrics m USING (session_id)
GROUP BY framework
ORDER BY framework;
```

## Failed tool calls with duration

```sql
SELECT
  tc.session_id,
  tc.tool_name,
  substr(COALESCE(tc.file_path, tc.command, ''), 1, 80) AS target,
  tc.exit_code,
  tc.duration_ms,
  substr(tc.error, 1, 120) AS error
FROM tool_calls tc
WHERE tc.success = 0
ORDER BY tc.session_id, tc.emitting_turn_index;
```

## Notes

- `go-minitrace query run --preset framework-summary` is the fastest first pass.
- The database build is cached per archive-glob fingerprint, so repeated queries are fast.
- `.minitrace.json` files are the source of truth; the query engine reads them directly and never consults manifests.
- Old session-level SQL using `->>` on blob columns still runs against the `sessions_base` compatibility view, but prefer the real columns above.
