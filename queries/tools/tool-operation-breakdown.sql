-- tool-operation-breakdown.sql
-- Count tool calls by framework and operation type.
-- Usage:
--   duckdb analysis.duckdb -init queries/load.sql -f queries/tool-operation-breakdown.sql

SELECT
  environment->>'agent_framework' AS framework,
  REPLACE(CAST(json_extract(tc, '$.operation_type') AS VARCHAR), '"', '') AS operation,
  COUNT(*) AS count
FROM sessions_base,
UNNEST(tool_calls) AS t(tc)
GROUP BY framework, operation
ORDER BY framework, count DESC;
