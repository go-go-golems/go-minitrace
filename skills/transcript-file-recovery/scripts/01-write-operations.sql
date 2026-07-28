-- 01-write-operations-surf-cli.sql
-- Find all NEW/MODIFY write operations targeting surf-cli files.
-- This is the decisive query for identifying what code each session wrote.
-- We need to recover:
--   go/internal/cli/commands/linkedin_connections.go     (NEW)
--   go/internal/cli/commands/scripts/linkedin_connections.js (NEW)
--   go/internal/cli/commands/linkedin_jobs.go            (MODIFIED)
--   go/internal/cli/commands/linkedin_profile.go         (MODIFIED)
--   go/internal/cli/commands/scripts/linkedin_profile.js (MODIFIED)
--   go/cmd/surf-go/main.go                               (MODIFIED)
--   scripts/generate-resume.py                           (NEW)
--   manuel-odendahl-resume.html                          (NEW)
WITH calls AS (
  SELECT
    session_id,
    emitting_turn_index AS turn_index,
    tool_call_id,
    tool_name,
    operation_type,
    success,
    file_path,
    coalesce(nullif(command, ''),
             json_extract(arguments_json, '$.command'),
             json_extract(arguments_json, '$.input'),
             arguments_json) AS command_text,
    substr(arguments_json, 1, 200) AS args_head,
    substr(result, 1, 200) AS result_head,
    arguments_json,
    result
  FROM tool_calls
)
SELECT
  session_id,
  turn_index,
  tool_name,
  operation_type,
  file_path,
  substr(file_path, -60) AS file_short,
  success,
  result_head
FROM calls
WHERE operation_type IN ('NEW', 'MODIFY')
  AND (
    lower(coalesce(file_path, '')) LIKE '%surf-cli%'
    OR lower(coalesce(file_path, '')) LIKE '%linkedin%'
    OR lower(coalesce(command_text, '')) LIKE '%linkedin_connections%'
    OR lower(coalesce(command_text, '')) LIKE '%linkedin_profile%'
    OR lower(coalesce(command_text, '')) LIKE '%linkedin_jobs%'
    OR lower(coalesce(command_text, '')) LIKE '%generate-resume%'
    OR lower(coalesce(command_text, '')) LIKE '%manuel-odendahl-resume%'
  )
ORDER BY session_id, turn_index;
