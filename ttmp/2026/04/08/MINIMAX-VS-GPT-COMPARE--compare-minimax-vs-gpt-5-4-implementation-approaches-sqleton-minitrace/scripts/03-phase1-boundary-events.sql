WITH relevant_tool_calls AS (
  SELECT
    CASE sb.id
      WHEN '2d525241-fe32-417b-8576-b29ce3b3e47c' THEN 'minimax'
      WHEN '7f61f412-40f0-417f-ab85-4dffdb9927e5' THEN 'gpt-5.4'
      ELSE 'other'
    END AS run,
    sb.id AS session_id,
    REPLACE(CAST(json_extract(tc, '$.id') AS VARCHAR), '"', '') AS tool_call_id,
    REPLACE(CAST(json_extract(tc, '$.timestamp') AS VARCHAR), '"', '') AS event_ts,
    REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool_name,
    REPLACE(CAST(json_extract(tc, '$.input.command') AS VARCHAR), '"', '') AS command
  FROM sessions_base sb,
       UNNEST(sb.tool_calls) AS t(tc)
  WHERE sb.id IN (
      '2d525241-fe32-417b-8576-b29ce3b3e47c',
      '7f61f412-40f0-417f-ab85-4dffdb9927e5'
    )
    AND REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') = 'bash'
)
SELECT
  run,
  session_id,
  tool_call_id,
  event_ts,
  command
FROM relevant_tool_calls
WHERE command LIKE '%git commit%'
   OR command LIKE '%docmgr task check%'
   OR command LIKE '%docmgr changelog update%'
   OR command LIKE '%Add embedded MinitraceCommand assets%'
   OR command LIKE '%Phase 1 — parser, compiler, catalog, tests, and core commands%'
ORDER BY run, event_ts;
