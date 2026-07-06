-- exit-codes: Tool calls with explicit process exit codes
-- Useful for auditing command outcomes beyond boolean success
-- Usage:
--   go-minitrace query run --archive-glob './output/active/*/*.minitrace.json' --sql-file queries/tools/exit-codes.sql

SELECT
  tc.session_id,
  tc.emitting_turn_index AS turn,
  tc.tool_name AS tool,
  tc.operation_type AS operation,
  COALESCE(tc.command, tc.file_path) AS target,
  tc.exit_code,
  tc.success,
  tc.timestamp
FROM tool_calls tc
WHERE tc.exit_code IS NOT NULL
ORDER BY tc.session_id, tc.emitting_turn_index, tc.timestamp;
