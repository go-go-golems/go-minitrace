-- Query 4: Timeline of jellyfin-001 diary operations with success/failure
-- Includes the full result snippet for context
SELECT
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  LEFT(CAST(json_extract(tc, '$.input.file_path') AS VARCHAR), 120) AS file_path,
  LEFT(CAST(json_extract(tc, '$.input.command') AS VARCHAR), 120) AS command,
  json_extract(tc, '$.output.success') AS success,
  CASE
    WHEN CAST(json_extract(tc, '$.output.result') AS VARCHAR) LIKE '%File not found%' THEN 'FILE_NOT_FOUND'
    WHEN CAST(json_extract(tc, '$.output.result') AS VARCHAR) LIKE '%Successfully%' THEN 'WROTE_OK'
    WHEN CAST(json_extract(tc, '$.output.result') AS VARCHAR) LIKE '%Ticket Workspace Created%' THEN 'TICKET_CREATED'
    WHEN CAST(json_extract(tc, '$.output.result') AS VARCHAR) LIKE '%No such file%' THEN 'NO_SUCH_FILE'
    WHEN CAST(json_extract(tc, '$.output.result') AS VARCHAR) LIKE '%Document Created%' THEN 'DOC_CREATED'
    ELSE LEFT(CAST(json_extract(tc, '$.output.result') AS VARCHAR), 80)
  END AS result_summary,
  json_extract(tc, '$.timestamp') AS ts
FROM sessions_base,
  UNNEST(tool_calls) AS t(tc)
WHERE
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') IN ('write', 'edit', 'bash')
  AND (
    CAST(json_extract(tc, '$.input.file_path') AS VARCHAR) LIKE '%jellyfin-001%'
    OR CAST(json_extract(tc, '$.input.file_path') AS VARCHAR) LIKE '%diary%'
    OR CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%jellyfin-001%'
    OR CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%docmgr%'
  )
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT);
