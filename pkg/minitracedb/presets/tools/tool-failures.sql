-- tool-failures: All failed tool calls with error details
-- Essential for debugging session issues
SELECT
  tc.emitting_turn_index AS turn,
  tc.tool_name AS tool,
  tc.operation_type AS operation,
  COALESCE(tc.file_path, substr(tc.command, 1, 200)) AS target,
  substr(tc.error, 1, 300) AS error,
  tc.timestamp
FROM tool_calls tc
WHERE tc.success = 0
ORDER BY tc.emitting_turn_index;
