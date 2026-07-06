-- justifications: Tool calls that include an explicit justification
-- Useful for auditing tool-use rationale where the source transcript provided one
-- Usage:
--   go-minitrace query run --archive-glob './output/active/*/*.minitrace.json' --sql-file queries/tools/justifications.sql

SELECT
  tc.session_id,
  tc.emitting_turn_index AS turn,
  tc.tool_name AS tool,
  tc.operation_type AS operation,
  tc.justification,
  COALESCE(tc.command, tc.file_path) AS target,
  tc.timestamp
FROM tool_calls tc
WHERE tc.justification IS NOT NULL AND tc.justification != ''
ORDER BY tc.session_id, tc.emitting_turn_index, tc.timestamp;
