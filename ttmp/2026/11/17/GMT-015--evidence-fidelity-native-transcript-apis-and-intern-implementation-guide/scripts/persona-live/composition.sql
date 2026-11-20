SELECT s.session_id, COUNT(t.tool_call_id) AS calls,
       COUNT(CASE WHEN t.success = 1 THEN 1 END) AS successful,
       COUNT(CASE WHEN t.success = 0 THEN 1 END) AS failed
FROM sessions s LEFT JOIN tool_calls t ON t.session_id = s.session_id
GROUP BY s.session_id ORDER BY s.session_id;
