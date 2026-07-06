/* sqleton
name: file-timeline
short: Chronological operations on files matching a path pattern
flags:
  - name: path_like
    type: string
    default: ""
    help: LIKE pattern to filter file paths (e.g. %diary%, %.go). Empty matches all.
  - name: limit
    type: int
    default: 500
    help: Limit the number of rows returned
*/
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
WHERE COALESCE(tc.file_path, tc.command, '') LIKE CASE
    WHEN {{ .path_like | sqlString }} = '' THEN '%'
    ELSE {{ .path_like | sqlLike }}
  END
ORDER BY tc.emitting_turn_index
LIMIT {{ .limit }};
