SELECT session_id, tool_name, success, exit_code, duration_ms,
       length(error) AS err_len, error, substr(result,1,300) AS result_head,
       framework_metadata_json
FROM tool_calls
WHERE session_id='01a0a5c3-342f-7447-be74-b6c35f542ff7' AND success=0
ORDER BY timestamp LIMIT 1;
