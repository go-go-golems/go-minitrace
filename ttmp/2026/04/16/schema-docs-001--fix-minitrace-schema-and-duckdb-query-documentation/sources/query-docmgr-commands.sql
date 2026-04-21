-- Query 2: All docmgr/ttmp-related bash commands
SELECT
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  LEFT(CAST(json_extract(tc, '$.input.command') AS VARCHAR), 400) AS command,
  json_extract(tc, '$.output.success') AS success,
  CASE
    WHEN CAST(json_extract(tc, '$.output.result') AS VARCHAR) LIKE '%Ticket Workspace Created%' THEN true
    ELSE false
  END AS created_ticket,
  CASE
    WHEN CAST(json_extract(tc, '$.output.result') AS VARCHAR) LIKE '%No such file%' THEN true
    ELSE false
  END AS no_such_file,
  LEFT(CAST(json_extract(tc, '$.output.error') AS VARCHAR), 200) AS error
FROM sessions_base,
  UNNEST(tool_calls) AS t(tc)
WHERE
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') = 'bash'
  AND (
    CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%docmgr%'
    OR CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%jellyfin-001%'
  )
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT);
