SELECT
  environment->>'agent_framework' AS framework,
  COUNT(*) AS sessions,
  ROUND(AVG(CAST(metrics->>'tool_call_count' AS INT)), 1) AS avg_tools,
  ROUND(AVG(CAST(metrics->>'turn_count' AS INT)), 1) AS avg_turns,
  ROUND(AVG(CAST(metrics->>'read_ratio' AS DOUBLE)), 2) AS avg_read_ratio,
  ROUND(AVG(CAST(timing->>'duration_seconds' AS DOUBLE)), 1) AS avg_duration_s,
  ROUND(AVG(CAST(metrics->>'time_to_first_action' AS DOUBLE)), 1) AS avg_ttfa_s
FROM {{TABLE_NAME}}
GROUP BY framework
ORDER BY avg_tools DESC;
