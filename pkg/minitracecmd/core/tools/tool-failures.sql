/* sqleton
name: tool-failures
short: List all failed tool calls with error details
flags:
  - name: tool_filter
    type: stringList
    help: Filter by tool name (e.g. bash, write, edit)
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
  substr(tc.error, 1, 300) AS error,
  tc.timestamp
FROM tool_calls tc
WHERE tc.success = 0
{{ if .tool_filter -}}
  AND tc.tool_name IN ({{ .tool_filter | sqlStringIn }})
{{ end -}}
ORDER BY tc.emitting_turn_index
LIMIT {{ .limit }};
