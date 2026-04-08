SELECT 
  json_extract_string(tc, '$.input.arguments.command') AS command,
  COUNT(*) AS uses
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE json_extract_string(tc, '$.tool_name') = 'bash'
  AND json_extract_string(tc, '$.input.arguments.command') IS NOT NULL
GROUP BY command
ORDER BY uses DESC
LIMIT 30
