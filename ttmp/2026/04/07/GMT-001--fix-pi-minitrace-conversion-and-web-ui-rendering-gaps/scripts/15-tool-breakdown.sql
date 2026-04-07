SELECT 
  json_extract_string(tc, '$.tool_name') AS tool,
  COUNT(*) AS uses
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
GROUP BY tool
ORDER BY uses DESC
