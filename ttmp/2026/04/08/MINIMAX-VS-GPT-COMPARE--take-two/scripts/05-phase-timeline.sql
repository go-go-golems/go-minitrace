-- Error and failure analysis
SELECT
  json_extract(tc, '$.tool_name') AS tool_name,
  CAST(json_extract(tc, '$.input.command') AS VARCHAR) AS command,
  COUNT(*) AS count
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE json_extract(tc, '$.tool_name') = '"bash"'
  AND (
    CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%error%'
    OR CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%fail%'
    OR CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%cannot%'
  )
GROUP BY tool_name, command
ORDER BY count DESC
LIMIT 20;
