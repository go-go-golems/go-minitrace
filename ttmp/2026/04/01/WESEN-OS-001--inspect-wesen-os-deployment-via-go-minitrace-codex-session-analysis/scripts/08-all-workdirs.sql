-- 08-all-workdirs.sql
-- All distinct working directories seen in the last 2 weeks
SELECT
  operational_context->>'working_directory'            AS workdir,
  COUNT(*)                                             AS sessions,
  SUM(CAST(metrics->>'turn_count' AS INT))             AS total_turns,
  SUM(CAST(metrics->>'tool_call_count' AS INT))        AS total_tools,
  ROUND(SUM(CAST(timing->>'duration_seconds' AS DOUBLE)) / 3600, 1) AS total_hours
FROM sessions_base
GROUP BY workdir
ORDER BY total_turns DESC;
