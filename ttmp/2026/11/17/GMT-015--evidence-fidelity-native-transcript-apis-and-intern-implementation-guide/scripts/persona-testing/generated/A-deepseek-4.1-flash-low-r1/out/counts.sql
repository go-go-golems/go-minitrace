SELECT
  s.session_id,
  s.agent_framework AS framework,
  (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS actual_turns,
  (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS actual_tools,
  s.turn_count AS meta_turns,
  s.tool_call_count AS meta_tools,
  s.started_at,
  s.ended_at
FROM sessions s
ORDER BY s.agent_framework, s.session_id;
