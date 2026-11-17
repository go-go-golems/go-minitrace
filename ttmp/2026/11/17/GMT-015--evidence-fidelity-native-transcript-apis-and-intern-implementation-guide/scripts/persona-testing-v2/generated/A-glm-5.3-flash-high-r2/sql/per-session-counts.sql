-- Task A: per-session actual turn and tool_call row counts across all nine real normalized archives
SELECT s.session_id,
       s.agent_framework,
       (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_rows,
       (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_rows,
       s.turn_count AS metric_turn_count,
       s.tool_call_count AS metric_tool_call_count
FROM sessions s
ORDER BY s.agent_framework, s.session_id;
