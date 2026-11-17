SELECT
  s.agent_framework AS framework,
  tc.session_id,
  tc.emitting_turn_index AS turn,
  tc.tool_call_id,
  tc.tool_name AS tool,
  tc.operation_type AS operation,
  tc.success,
  tc.exit_code,
  tc.command,
  tc.error,
  tc.timestamp
FROM tool_calls tc
JOIN sessions s USING (session_id)
WHERE tc.success = 0
ORDER BY s.agent_framework, tc.session_id, tc.emitting_turn_index
LIMIT 50;
