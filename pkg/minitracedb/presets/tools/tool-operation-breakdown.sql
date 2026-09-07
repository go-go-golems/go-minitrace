SELECT
  s.agent_framework AS framework,
  tc.record_kind,
  tc.operation_type AS operation,
  COUNT(*) AS count
FROM tool_calls tc
JOIN sessions s ON s.session_id = tc.session_id
GROUP BY framework, tc.record_kind, operation
ORDER BY framework, count DESC;
