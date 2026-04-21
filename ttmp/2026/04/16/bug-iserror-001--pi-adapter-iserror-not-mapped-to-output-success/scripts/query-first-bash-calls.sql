-- Query 10: Check what the first docmgr command ran as (pwd check)
-- The turn 2 docmgr command doesn't have a 'cd' prefix, so it ran 
-- in whatever the shell's cwd was at that point
-- Check for any pwd/cwd indicators in the first few bash calls
SELECT
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  LEFT(CAST(json_extract(tc, '$.input.command') AS VARCHAR), 200) AS command,
  LEFT(CAST(json_extract(tc, '$.output.result') AS VARCHAR), 200) AS result_preview
FROM sessions_base,
  UNNEST(tool_calls) AS t(tc)
WHERE
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') = 'bash'
  AND CAST(json_extract(tc, '$.emitting_turn_index') AS INT) <= 5
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT);
