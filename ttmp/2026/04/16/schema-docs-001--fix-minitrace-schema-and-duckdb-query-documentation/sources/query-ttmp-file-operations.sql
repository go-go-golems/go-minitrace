-- Query 2: Find all file operations (write/edit) in the session, ordered by turn
-- to see the full picture of what files were created/modified
SELECT
  turn_index,
  tool_name,
  CAST(json_extract(tool_args, '$.path') AS VARCHAR) AS file_path,
  tool_result_success,
  LEFT(tool_result_error, 200) AS error_preview
FROM (
  SELECT
    UNNEST(tool_calls) AS tc
  FROM sessions_base
), UNNEST(tc) AS t(tool_call_id, tool_name, tool_args, turn_index, tool_result_success, tool_result_error)
WHERE
  tool_name IN ('write', 'edit')
  AND CAST(json_extract(tool_args, '$.path') AS VARCHAR) LIKE '%ttmp%'
ORDER BY turn_index;
