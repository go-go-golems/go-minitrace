SELECT session_id, tool_name, substr(COALESCE(command, file_path, ''), 1, 80) AS target,
       exit_code, outcome_status, success, emitting_turn_index
FROM tool_calls
WHERE (exit_code IS NOT NULL AND exit_code != 0) OR success = 0
ORDER BY session_id, emitting_turn_index
LIMIT 20;
