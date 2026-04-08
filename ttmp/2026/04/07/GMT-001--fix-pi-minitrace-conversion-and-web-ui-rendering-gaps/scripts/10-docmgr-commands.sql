SELECT 
  json_extract_string(tc, '$.input.arguments.command') AS command,
  tc->>'timestamp' AS ts
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE json_extract_string(tc, '$.tool_name') = 'bash'
  AND (json_extract_string(tc, '$.input.arguments.command') LIKE '%mkdir%ttmp%'
       OR json_extract_string(tc, '$.input.arguments.command') LIKE '%docmgr%ticket%create%'
       OR json_extract_string(tc, '$.input.arguments.command') LIKE '%docmgr%task%')
ORDER BY tc->>'timestamp'
