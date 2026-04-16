-- Query 5: Find all git operations in the session that might affect ttmp files
SELECT
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  LEFT(CAST(json_extract(tc, '$.input.command') AS VARCHAR), 300) AS command,
  json_extract(tc, '$.output.success') AS success,
  LEFT(CAST(json_extract(tc, '$.output.result') AS VARCHAR), 200) AS result_preview
FROM sessions_base,
  UNNEST(tool_calls) AS t(tc)
WHERE
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') = 'bash'
  AND CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%git%'
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT);
