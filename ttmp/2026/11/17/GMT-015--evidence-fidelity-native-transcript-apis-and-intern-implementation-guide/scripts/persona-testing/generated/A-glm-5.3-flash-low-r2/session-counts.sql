-- Per-session ACTUAL row counts from turns and tool_calls (vs metadata aggregates)
SELECT
  s.session_id,
  s.agent_framework AS framework,
  (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id)     AS turn_count_actual,
  (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_count_actual,
  s.turn_count AS turn_count_meta,
  s.tool_call_count AS tool_count_meta
FROM sessions s ORDER BY s.started_at;
