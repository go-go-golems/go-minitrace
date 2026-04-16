-- Query 6: Check what's happening between turn 14 (last successful edit) 
-- and turn 40 (first file not found). What operations ran that might 
-- have deleted the ttmp directory?
SELECT
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  json_extract(tc, '$.operation_type') AS op_type,
  LEFT(CAST(json_extract(tc, '$.input.file_path') AS VARCHAR), 120) AS file_path,
  LEFT(CAST(json_extract(tc, '$.input.command') AS VARCHAR), 200) AS command,
  json_extract(tc, '$.output.success') AS success,
  LEFT(CAST(json_extract(tc, '$.output.result') AS VARCHAR), 120) AS result_preview
FROM sessions_base,
  UNNEST(tool_calls) AS t(tc)
WHERE
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) BETWEEN 14 AND 48
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT);
