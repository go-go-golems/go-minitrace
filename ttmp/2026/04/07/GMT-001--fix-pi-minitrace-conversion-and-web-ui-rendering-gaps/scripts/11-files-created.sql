SELECT 
  json_extract_string(tc, '$.input.arguments.path') AS path,
  COUNT(*) AS uses
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE json_extract_string(tc, '$.tool_name') = 'write'
GROUP BY path
ORDER BY uses DESC
LIMIT 20
