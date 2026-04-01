SELECT
  environment->>'agent_framework' AS framework,
  COUNT(*) AS sessions,
  ROUND(AVG(CAST(timing->>'duration_seconds' AS DOUBLE)), 1) AS avg_duration_s,
  ROUND(AVG(CAST(timing->>'active_duration_seconds' AS DOUBLE)), 1) AS avg_active_s,
  ROUND(AVG(CAST(metrics->>'time_to_first_action' AS DOUBLE)), 1) AS avg_ttfa_s,
  ROUND(AVG(CAST(metrics->>'idle_ratio' AS DOUBLE)), 2) AS avg_idle_ratio,
  ROUND(MIN(CAST(timing->>'duration_seconds' AS DOUBLE)), 1) AS min_duration_s,
  ROUND(MAX(CAST(timing->>'duration_seconds' AS DOUBLE)), 1) AS max_duration_s
FROM {{TABLE_NAME}}
WHERE timing->>'duration_seconds' IS NOT NULL
GROUP BY framework
ORDER BY avg_duration_s DESC;
