SELECT session_id, COUNT(*) AS failed, SUM(CASE WHEN success THEN 0 ELSE 1 END) AS not_success
FROM tool_calls WHERE success = 0 GROUP BY session_id ORDER BY failed DESC;
