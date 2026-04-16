SELECT
  environment->>'agent_framework' AS framework,
  REPLACE(CAST(json_extract(tc, '$.operation_type') AS VARCHAR), '"', '') AS operation,
  COUNT(*) AS count
FROM {{TABLE_NAME}},
UNNEST(tool_calls) AS t(tc)
GROUP BY framework, operation
ORDER BY framework, count DESC;
