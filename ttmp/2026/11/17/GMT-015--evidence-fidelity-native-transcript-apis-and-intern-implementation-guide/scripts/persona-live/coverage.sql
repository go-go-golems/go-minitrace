SELECT session_id, tool_name, COUNT(*) AS calls,
       COUNT(*) - COUNT(success) AS unknown_success,
       COUNT(*) - COUNT(exit_code) AS missing_exit,
       COUNT(CASE WHEN truncated = 1 THEN 1 END) AS marked_truncated
FROM tool_calls
GROUP BY session_id, tool_name
ORDER BY session_id, tool_name;
