-- justifications: Tool calls that include an explicit justification
-- Useful for auditing tool-use rationale where the source transcript provided one
-- Usage:
--   duckdb analysis.duckdb -init queries/load.sql -f queries/tools/justifications.sql

SELECT
  sb.id AS session_id,
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  json_extract(tc, '$.operation_type') AS operation,
  CAST(json_extract(tc, '$.input.justification') AS VARCHAR) AS justification,
  COALESCE(
    CAST(json_extract(tc, '$.input.command') AS VARCHAR),
    CAST(json_extract(tc, '$.input.file_path') AS VARCHAR)
  ) AS target,
  json_extract(tc, '$.timestamp') AS timestamp
FROM sessions_base AS sb,
  UNNEST(tool_calls) AS t(tc)
WHERE
  json_extract(tc, '$.input.justification') IS NOT NULL
ORDER BY
  sb.id,
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT),
  json_extract(tc, '$.timestamp');
