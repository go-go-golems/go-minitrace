SELECT
  s.session_id AS session_id,
  s.agent_framework AS framework,
  (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_rows,
  (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_rows,
  s.turn_count AS sessions_rollup_turns,
  s.tool_call_count AS sessions_rollup_tools
FROM sessions s
ORDER BY s.session_id;
