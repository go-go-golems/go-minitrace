-- Candidate instruction-file mentions/searches/reads in normalized archives
SELECT session_id, tool_name, operation_type, file_path,
       substr(command,1,240) AS cmd, success
FROM tool_calls
WHERE lower(COALESCE(file_path,'')) LIKE '%agent.md%'
   OR lower(COALESCE(file_path,'')) LIKE '%claude.md%'
   OR lower(COALESCE(command,'')) LIKE '%agent.md%'
   OR lower(COALESCE(command,'')) LIKE '%claude.md%'
   OR lower(COALESCE(file_path,'')) LIKE '%readme.md%'
   OR lower(COALESCE(command,'')) LIKE '%readme.md%'
ORDER BY session_id LIMIT 100;
