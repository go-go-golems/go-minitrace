SELECT
  CASE id
    WHEN '2d525241-fe32-417b-8576-b29ce3b3e47c' THEN 'minimax'
    WHEN '7f61f412-40f0-417f-ab85-4dffdb9927e5' THEN 'gpt-5.4'
    ELSE 'other'
  END AS run,
  id,
  title,
  environment->>'model' AS model,
  environment->>'agent_framework' AS framework,
  CAST(metrics->>'turn_count' AS INT) AS turns,
  CAST(metrics->>'tool_call_count' AS INT) AS tool_calls,
  ROUND(CAST(timing->>'duration_seconds' AS DOUBLE) / 60.0, 1) AS minutes
FROM sessions_base
WHERE id IN (
  '2d525241-fe32-417b-8576-b29ce3b3e47c',
  '7f61f412-40f0-417f-ab85-4dffdb9927e5'
)
ORDER BY run;
