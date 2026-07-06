-- tool-operation-breakdown: Count tool calls by framework and operation type
-- Usage:
--   go-minitrace query run --archive-glob './output/active/*/*.minitrace.json' --sql-file queries/tools/tool-operation-breakdown.sql

SELECT
  s.agent_framework AS framework,
  tc.operation_type AS operation,
  COUNT(*) AS count
FROM tool_calls tc
JOIN sessions s ON s.session_id = tc.session_id
GROUP BY framework, operation
ORDER BY framework, count DESC;
