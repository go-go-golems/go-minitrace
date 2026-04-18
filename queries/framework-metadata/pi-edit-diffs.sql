-- pi-edit-diffs: Inspect preserved Pi edit diffs from tool result metadata
-- Useful for forensic review of Pi edit operations
-- Usage:
--   duckdb analysis.duckdb -init queries/load.sql -f queries/framework-metadata/pi-edit-diffs.sql

SELECT
  sb.id AS session_id,
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  CAST(json_extract(tc, '$.input.file_path') AS VARCHAR) AS file_path,
  CAST(json_extract(tc, '$.framework_metadata.first_changed_line') AS INT) AS first_changed_line,
  CAST(json_extract(tc, '$.framework_metadata.diff') AS VARCHAR) AS diff,
  json_extract(tc, '$.timestamp') AS timestamp
FROM sessions_base AS sb,
  UNNEST(tool_calls) AS t(tc)
WHERE
  CAST(json_extract(sb.environment, '$.agent_framework') AS VARCHAR) = '"pi"'
  AND json_extract(tc, '$.framework_metadata.diff') IS NOT NULL
ORDER BY
  sb.id,
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT),
  json_extract(tc, '$.timestamp');
