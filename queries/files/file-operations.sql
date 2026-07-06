-- file-operations: Track all file read/write/edit operations in session order
-- Shows every file touch with success/failure status
-- Usage:
--   go-minitrace query run --archive-glob './output/active/*/*.minitrace.json' --sql-file queries/files/file-operations.sql

SELECT
  tc.session_id,
  tc.emitting_turn_index AS turn,
  tc.tool_name AS tool,
  tc.operation_type AS operation,
  COALESCE(tc.file_path, substr(tc.command, 1, 200)) AS target,
  tc.success,
  substr(tc.error, 1, 200) AS error,
  tc.timestamp
FROM tool_calls tc
WHERE tc.tool_name IN ('write', 'edit', 'read')
ORDER BY tc.session_id, tc.emitting_turn_index;
