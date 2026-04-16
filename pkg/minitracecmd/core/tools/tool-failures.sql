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
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  json_extract(tc, '$.operation_type') AS operation,
  COALESCE(
    CAST(json_extract(tc, '$.input.file_path') AS VARCHAR),
    LEFT(CAST(json_extract(tc, '$.input.command') AS VARCHAR), 200)
  ) AS target,
  LEFT(CAST(json_extract(tc, '$.output.error') AS VARCHAR), 300) AS error,
  json_extract(tc, '$.timestamp') AS timestamp
FROM {{TABLE_NAME}},
  UNNEST(tool_calls) AS t(tc)
WHERE
  json_extract(tc, '$.output.success') = false
{{ if .tool_filter -}}
  AND REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') IN ({{ .tool_filter | sqlStringIn }})
{{ end -}}
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT)
LIMIT {{ .limit }};
