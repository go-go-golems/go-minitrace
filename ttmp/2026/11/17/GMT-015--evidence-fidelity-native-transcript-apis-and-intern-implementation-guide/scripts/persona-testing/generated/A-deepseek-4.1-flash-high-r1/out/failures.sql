SELECT session_id, tool_name, emitting_turn_index, success, exit_code, substr(command,1,160) AS command, substr(error,1,240) AS error
FROM tool_calls
WHERE success = 0 OR (exit_code IS NOT NULL AND exit_code <> 0)
ORDER BY session_id, emitting_turn_index LIMIT 40;
