-- Per-session actual counts (turn rows / tool_call rows, not aggregate metadata),
-- top tool per session, plus session date range.
SELECT
  s.session_id,
  s.agent_framework,
  s.started_at,
  s.ended_at,
  (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_count,
  (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_count,
  (SELECT tool_name FROM tool_calls tc WHERE tc.session_id = s.session_id
   GROUP BY tool_name ORDER BY COUNT(*) DESC LIMIT 1) AS top_tool
FROM sessions s
ORDER BY s.started_at;
