SELECT session_id, tool_name, COUNT(*) AS n
FROM tool_calls GROUP BY session_id, tool_name
ORDER BY session_id, n DESC, tool_name;
