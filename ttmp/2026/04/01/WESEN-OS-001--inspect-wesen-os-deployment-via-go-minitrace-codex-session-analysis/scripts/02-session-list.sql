-- 02-session-list.sql
-- Full session list for the last 2 weeks with key metrics
SELECT
  id,
  timing->>'started_at'                                          AS started_at,
  title,
  CAST(metrics->>'turn_count'      AS INT)                       AS turns,
  CAST(metrics->>'tool_call_count' AS INT)                       AS tools,
  ROUND(CAST(timing->>'duration_seconds' AS DOUBLE) / 3600, 2)  AS hours,
  classification,
  operational_context->>'working_directory'                      AS workdir
FROM sessions_base
ORDER BY timing->>'started_at';
