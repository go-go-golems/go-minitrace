/* sqleton
name: file-operations
short: Track all file read/write/edit operations in turn order
long: |
  Filters on the normalized operation_type, not the raw tool_name. Tool names
  differ across adapters (claude-code Write/Edit/Read, pi write/edit/read,
  codex exec), so a tool_name filter silently returned one framework's rows and
  dropped the rest. The tool_filter values (write/edit/read) are mapped to
  operation_type (NEW/MODIFY/READ), which is uppercase and consistent across
  every adapter. Note that codex records most exec-based file work as
  operation_type OTHER, so its file operations remain under-represented here —
  that is an adapter-level classification gap, not a filter bug.
flags:
  - name: tool_filter
    type: stringList
    default:
      - write
      - edit
      - read
    help: Filter by operation class (write=NEW, edit=MODIFY, read=READ)
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
WHERE CASE tc.operation_type
        WHEN 'READ' THEN 'read'
        WHEN 'NEW' THEN 'write'
        WHEN 'MODIFY' THEN 'edit'
        ELSE tc.operation_type
      END IN ({{ .tool_filter | sqlStringIn }})
ORDER BY tc.emitting_turn_index
LIMIT {{ .limit }};
