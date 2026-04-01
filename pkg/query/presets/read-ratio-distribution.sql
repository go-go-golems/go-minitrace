SELECT
  environment->>'agent_framework' AS framework,
  id,
  CAST(metrics->>'tool_call_count' AS INT) AS tools,
  CAST(metrics->>'read_count' AS INT) AS reads,
  CAST(metrics->>'modify_count' AS INT) AS modifies,
  CAST(metrics->>'create_count' AS INT) AS creates,
  CAST(metrics->>'execute_count' AS INT) AS executes,
  ROUND(CAST(metrics->>'read_ratio' AS DOUBLE), 2) AS read_ratio
FROM {{TABLE_NAME}}
WHERE CAST(metrics->>'tool_call_count' AS INT) > 0
ORDER BY read_ratio DESC, tools DESC, id;
