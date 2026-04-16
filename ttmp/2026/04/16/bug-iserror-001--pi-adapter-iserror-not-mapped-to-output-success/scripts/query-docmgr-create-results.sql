-- Query 9: Get the full result of the first docmgr ticket create to see
-- the actual working directory used
SELECT
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  CAST(json_extract(tc, '$.input.command') AS VARCHAR) AS command,
  CAST(json_extract(tc, '$.output.result') AS VARCHAR) AS full_result
FROM sessions_base,
  UNNEST(tool_calls) AS t(tc)
WHERE
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') = 'bash'
  AND CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%docmgr ticket create%'
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT);
