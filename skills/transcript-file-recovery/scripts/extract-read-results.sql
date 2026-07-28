-- extract-read-results.sql
-- Extract read tool results as base states for files that were only edited
-- (never created via write, not on git main). The `result` column contains
-- the full file content at the time of the read.
--
-- Usage:
--   go-minitrace query run \
--     --archive-glob 'archives/*/*/*.minitrace.json' \
--     --sql-file scripts/extract-read-results.sql \
--     --output json --max-cell-chars 100000 \
--     > sources/read-results.json
--
-- Replace <SESSION_ID> and <filename> with your values.
SELECT
  emitting_turn_index AS turn_index,
  file_path,
  result
FROM tool_calls
WHERE session_id = '<SESSION_ID>'
  AND tool_name = 'read'
  AND lower(coalesce(file_path, '')) LIKE '%<filename>%'
ORDER BY emitting_turn_index;
