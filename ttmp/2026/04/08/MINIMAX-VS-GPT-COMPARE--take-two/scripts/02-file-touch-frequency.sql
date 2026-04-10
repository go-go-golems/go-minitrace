-- File touch frequency: which files were read/written/edited most
SELECT
  json_extract(tc, '$.input.file_path') AS file_path,
  json_extract(tc, '$.tool_name') AS tool,
  COUNT(*) AS count
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE json_extract(tc, '$.tool_name') IN ('"read"', '"write"', '"edit"')
  AND json_extract(tc, '$.input.file_path') IS NOT NULL
GROUP BY tool, file_path
ORDER BY count DESC
LIMIT 40;
