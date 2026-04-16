-- Query 1: Find all tool calls targeting jellyfin-001 or diary files
-- Tracks every read/write/edit attempt on the docmgr ticket files
SELECT
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  json_extract(tc, '$.input.file_path') AS file_path,
  json_extract(tc, '$.output.success') AS success,
  LEFT(CAST(json_extract(tc, '$.output.error') AS VARCHAR), 300) AS error,
  CASE
    WHEN CAST(json_extract(tc, '$.output.result') AS VARCHAR) LIKE '%Successfully%' THEN true
    ELSE false
  END AS wrote_ok,
  CASE
    WHEN CAST(json_extract(tc, '$.output.result') AS VARCHAR) LIKE '%File not found%' THEN true
    ELSE false
  END AS file_missing
FROM sessions_base,
  UNNEST(tool_calls) AS t(tc)
WHERE
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') IN ('write', 'edit', 'read')
  AND (
    CAST(json_extract(tc, '$.input.file_path') AS VARCHAR) LIKE '%jellyfin-001%'
    OR CAST(json_extract(tc, '$.input.file_path') AS VARCHAR) LIKE '%diary%'
  )
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT);
