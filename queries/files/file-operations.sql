-- file-operations: Track all file read/write/edit operations in session order
-- Shows every file touch with success/failure status
-- Usage:
--   duckdb analysis.duckdb -init queries/load.sql -f queries/files/file-operations.sql

SELECT
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  json_extract(tc, '$.operation_type') AS operation,
  COALESCE(
    CAST(json_extract(tc, '$.input.file_path') AS VARCHAR),
    LEFT(CAST(json_extract(tc, '$.input.command') AS VARCHAR), 200)
  ) AS target,
  json_extract(tc, '$.output.success') AS success,
  LEFT(CAST(json_extract(tc, '$.output.error') AS VARCHAR), 200) AS error,
  json_extract(tc, '$.timestamp') AS timestamp
FROM sessions_base,
  UNNEST(tool_calls) AS t(tc)
WHERE
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') IN ('write', 'edit', 'read')
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT);
