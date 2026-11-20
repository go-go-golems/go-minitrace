-- Actual counts come from child tables, not sessions.turn_count/tool_call_count
SELECT
  s.session_id,
  s.agent_framework,
  (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS actual_turn_count,
  s.turn_count AS metadata_turn_count,
  (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS actual_tool_count,
  s.tool_call_count AS metadata_tool_count,
  s.started_at,
  s.ended_at
FROM sessions s
ORDER BY s.agent_framework, s.started_at;
