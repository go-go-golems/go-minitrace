-- codex-exec-metadata: Inspect preserved Codex execution metadata
-- Shows command source, parsed command info, stdout/stderr, and exit codes
-- Usage:
--   go-minitrace query run --archive-glob './output/active/*/*.minitrace.json' --sql-file queries/framework-metadata/codex-exec-metadata.sql

SELECT
  tc.session_id,
  tc.emitting_turn_index AS turn,
  tc.command,
  tc.exit_code,
  json_extract(tc.framework_metadata_json, '$.source') AS source,
  json_extract(tc.framework_metadata_json, '$.parsed_cmd') AS parsed_cmd,
  json_extract(tc.framework_metadata_json, '$.stdout') AS stdout,
  json_extract(tc.framework_metadata_json, '$.stderr') AS stderr,
  tc.timestamp
FROM tool_calls tc
JOIN sessions s ON s.session_id = tc.session_id
WHERE s.agent_framework = 'codex'
  AND json_extract(tc.framework_metadata_json, '$.source') IS NOT NULL
ORDER BY tc.session_id, tc.emitting_turn_index, tc.timestamp;
