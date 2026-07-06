-- pi-edit-diffs: Inspect preserved Pi edit diffs from tool result metadata
-- Useful for forensic review of Pi edit operations
-- Usage:
--   go-minitrace query run --archive-glob './output/active/*/*.minitrace.json' --sql-file queries/framework-metadata/pi-edit-diffs.sql

SELECT
  tc.session_id,
  tc.emitting_turn_index AS turn,
  tc.tool_name AS tool,
  tc.file_path,
  json_extract(tc.framework_metadata_json, '$.first_changed_line') AS first_changed_line,
  json_extract(tc.framework_metadata_json, '$.diff') AS diff,
  tc.timestamp
FROM tool_calls tc
JOIN sessions s ON s.session_id = tc.session_id
WHERE s.agent_framework = 'pi'
  AND json_extract(tc.framework_metadata_json, '$.diff') IS NOT NULL
ORDER BY tc.session_id, tc.emitting_turn_index, tc.timestamp;
