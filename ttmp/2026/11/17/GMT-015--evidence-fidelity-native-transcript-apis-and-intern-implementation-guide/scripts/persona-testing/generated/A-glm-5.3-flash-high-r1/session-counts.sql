-- Per-session turn and tool-call counts, counted from actual normalized rows
-- (not the sessions.turn_count / tool_call_count metadata columns).
SELECT
  s.session_id,
  s.agent_framework AS framework,
  (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_count,
  (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_count
FROM sessions s
ORDER BY s.started_at;
