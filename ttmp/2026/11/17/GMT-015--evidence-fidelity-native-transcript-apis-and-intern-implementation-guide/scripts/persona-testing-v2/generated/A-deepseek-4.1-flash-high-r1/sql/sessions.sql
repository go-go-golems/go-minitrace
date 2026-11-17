SELECT s.session_id,
       (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_count,
       (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_count
FROM sessions s
ORDER BY s.session_id;
