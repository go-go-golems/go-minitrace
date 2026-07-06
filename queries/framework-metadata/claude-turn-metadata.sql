-- claude-turn-metadata: Inspect preserved Claude Code turn metadata
-- Shows entrypoint, thread context, stop reason, and cache bucket detail
-- Usage:
--   go-minitrace query run --archive-glob './output/active/*/*.minitrace.json' --sql-file queries/framework-metadata/claude-turn-metadata.sql

SELECT
  t.session_id,
  t.turn_index AS turn,
  t.role,
  json_extract(t.framework_metadata_json, '$.entrypoint') AS entrypoint,
  json_extract(t.framework_metadata_json, '$.slug') AS slug,
  json_extract(t.framework_metadata_json, '$.parent_uuid') AS parent_uuid,
  json_extract(t.framework_metadata_json, '$.is_sidechain') AS is_sidechain,
  json_extract(t.framework_metadata_json, '$.stop_reason') AS stop_reason,
  json_extract(t.framework_metadata_json, '$.cache_creation') AS cache_creation,
  t.timestamp
FROM turns t
JOIN sessions s ON s.session_id = t.session_id
WHERE s.agent_framework = 'claude-code'
  AND t.framework_metadata_json IS NOT NULL
ORDER BY t.session_id, t.turn_index;
