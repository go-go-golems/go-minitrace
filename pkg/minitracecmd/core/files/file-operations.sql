/* sqleton
name: file-operations
short: Track all file read/write/edit operations in turn order
flags:
  - name: tool_filter
    type: stringList
    default:
      - write
      - edit
      - read
    help: Filter by tool name (write, edit, read)
  - name: limit
    type: int
    default: 500
    help: Limit the number of rows returned
*/
SELECT
  tc.emitting_turn_index AS turn,
  tc.tool_name AS tool,
  tc.operation_type AS operation,
  COALESCE(tc.file_path, substr(tc.command, 1, 200)) AS target,
  tc.success,
  substr(tc.error, 1, 200) AS error,
  tc.timestamp
FROM tool_calls tc
WHERE tc.tool_name IN ({{ .tool_filter | sqlStringIn }})
ORDER BY tc.emitting_turn_index
LIMIT {{ .limit }};
