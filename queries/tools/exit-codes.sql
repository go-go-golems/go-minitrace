-- exit-codes: Tool calls with explicit process exit codes
-- Useful for auditing command outcomes beyond boolean success
-- Usage:
--   duckdb analysis.duckdb -init queries/load.sql -f queries/tools/exit-codes.sql

SELECT
  sb.id AS session_id,
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  json_extract(tc, '$.operation_type') AS operation,
  COALESCE(
    CAST(json_extract(tc, '$.input.command') AS VARCHAR),
    CAST(json_extract(tc, '$.input.file_path') AS VARCHAR)
  ) AS target,
  CAST(json_extract(tc, '$.output.exit_code') AS INT) AS exit_code,
  json_extract(tc, '$.output.success') AS success,
  json_extract(tc, '$.timestamp') AS timestamp
FROM sessions_base AS sb,
  UNNEST(tool_calls) AS t(tc)
WHERE
  json_extract(tc, '$.output.exit_code') IS NOT NULL
ORDER BY
  sb.id,
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT),
  json_extract(tc, '$.timestamp');
