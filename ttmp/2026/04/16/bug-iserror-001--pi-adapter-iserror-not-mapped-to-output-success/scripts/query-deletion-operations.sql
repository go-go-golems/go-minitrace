-- Query 8: Check if there are operations between turns 14-40 that 
-- touch the ttmp directory or run rm/clean operations
SELECT
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  LEFT(CAST(json_extract(tc, '$.input.command') AS VARCHAR), 300) AS command,
  json_extract(tc, '$.output.success') AS success,
  LEFT(CAST(json_extract(tc, '$.output.result') AS VARCHAR), 150) AS result_preview
FROM sessions_base,
  UNNEST(tool_calls) AS t(tc)
WHERE
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') = 'bash'
  AND CAST(json_extract(tc, '$.emitting_turn_index') AS INT) BETWEEN 15 AND 42
  AND (
    CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%rm%'
    OR CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%clean%'
    OR CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%ttmp%'
    OR CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%git%'
    OR CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%checkout%'
  )
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT);
