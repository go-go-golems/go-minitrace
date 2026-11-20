SELECT
  s.session_id,
  s.agent_framework,
  (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_rows,
  (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_rows,
  s.turn_count AS meta_turn_count,
  s.tool_call_count AS meta_tool_count,
  s.started_at,
  s.ended_at
FROM sessions s
ORDER BY s.agent_framework, s.started_at;
