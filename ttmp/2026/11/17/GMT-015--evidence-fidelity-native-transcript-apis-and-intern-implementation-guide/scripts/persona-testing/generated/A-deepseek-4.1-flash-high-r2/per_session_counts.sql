-- Per-session actual row counts vs metadata aggregates, across all 9 normalized archives.
SELECT
  s.session_id,
  s.agent_framework,
  (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id)      AS turn_rows,
  (SELECT COUNT(*) FROM tool_calls c WHERE c.session_id = s.session_id) AS tool_rows,
  COALESCE(s.turn_count,0)      AS meta_turn_count,
  COALESCE(s.tool_call_count,0) AS meta_tool_count,
  (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) - COALESCE(s.turn_count,0)           AS turn_delta,
  (SELECT COUNT(*) FROM tool_calls c WHERE c.session_id = s.session_id) - COALESCE(s.tool_call_count,0) AS tool_delta,
  (SELECT MIN(t.timestamp) FROM turns t WHERE t.session_id = s.session_id) AS first_turn_ts,
  (SELECT MAX(t.timestamp) FROM turns t WHERE t.session_id = s.session_id) AS last_turn_ts
FROM sessions s
ORDER BY s.session_id;
