-- tool-failures: All failed tool calls with error details
-- Essential for debugging session issues
-- Usage:
--   duckdb analysis.duckdb -init queries/load.sql -f queries/tools/tool-failures.sql

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
FROM sessions_base,
  UNNEST(tool_calls) AS t(tc)
WHERE
  json_extract(tc, '$.output.success') = false
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT);
