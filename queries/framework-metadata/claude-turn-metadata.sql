-- claude-turn-metadata: Inspect preserved Claude Code turn metadata
-- Shows entrypoint, thread context, stop reason, and cache bucket detail
-- Usage:
--   duckdb analysis.duckdb -init queries/load.sql -f queries/framework-metadata/claude-turn-metadata.sql

SELECT
  sb.id AS session_id,
  CAST(json_extract(turn_json, '$.index') AS INT) AS turn,
  CAST(json_extract(turn_json, '$.role') AS VARCHAR) AS role,
  CAST(json_extract(turn_json, '$.framework_metadata.entrypoint') AS VARCHAR) AS entrypoint,
  CAST(json_extract(turn_json, '$.framework_metadata.slug') AS VARCHAR) AS slug,
  CAST(json_extract(turn_json, '$.framework_metadata.parent_uuid') AS VARCHAR) AS parent_uuid,
  CAST(json_extract(turn_json, '$.framework_metadata.is_sidechain') AS BOOLEAN) AS is_sidechain,
  CAST(json_extract(turn_json, '$.framework_metadata.stop_reason') AS VARCHAR) AS stop_reason,
  json_extract(turn_json, '$.framework_metadata.cache_creation') AS cache_creation,
  json_extract(turn_json, '$.timestamp') AS timestamp
FROM sessions_base AS sb,
  UNNEST(turns) AS t(turn_json)
WHERE
  CAST(json_extract(sb.environment, '$.agent_framework') AS VARCHAR) = '"claude-code"'
  AND json_extract(turn_json, '$.framework_metadata') IS NOT NULL
ORDER BY
  sb.id,
  CAST(json_extract(turn_json, '$.index') AS INT);
