-- file-timeline: Chronological operations on files matching a path pattern
-- Result content is classified into short labels for quick scanning
SELECT
  tc.emitting_turn_index AS turn,
  tc.tool_name AS tool,
  tc.operation_type AS operation,
  COALESCE(tc.file_path, substr(tc.command, 1, 120)) AS target,
  tc.success,
  CASE
    WHEN tc.result LIKE '%File not found%' THEN 'FILE_NOT_FOUND'
    WHEN tc.result LIKE '%Successfully%' THEN 'OK'
    WHEN tc.result LIKE '%No such file%' THEN 'NO_SUCH_FILE'
    WHEN tc.error IS NOT NULL AND tc.error != '' THEN 'ERROR'
    ELSE substr(tc.result, 1, 80)
  END AS result_summary,
  substr(tc.error, 1, 200) AS error,
  tc.timestamp
FROM tool_calls tc
WHERE COALESCE(tc.file_path, tc.command, '') LIKE '%'
ORDER BY tc.emitting_turn_index;
