SELECT
  s.session_id,
  s.agent_framework,
  s.turn_count AS stored_turn_metric,
  s.tool_call_count AS stored_tool_metric,
  (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS actual_turns,
  (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS actual_tools
FROM sessions s
ORDER BY s.session_id;
