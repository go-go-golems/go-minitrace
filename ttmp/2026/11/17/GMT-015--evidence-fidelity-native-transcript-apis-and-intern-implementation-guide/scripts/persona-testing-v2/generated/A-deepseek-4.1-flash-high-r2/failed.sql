SELECT session_id, tool_name, operation_type, success, exit_code,
       substr(command,1,110) AS cmd_head, substr(error,1,140) AS err_head, timestamp
FROM tool_calls
WHERE success = 0
ORDER BY timestamp
LIMIT 15;
