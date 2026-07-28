-- 04-bash-edits-surf-cli.sql
-- Find bash/exec tool calls that may have modified surf-cli files
-- via sed, cat >, heredocs, echo >>, git apply, etc.
-- These changes are NOT captured by the write/edit tool call queries.
WITH exec_calls AS (
  SELECT
    session_id,
    emitting_turn_index AS turn_index,
    tool_call_id,
    tool_name,
    success,
    exit_code,
    coalesce(nullif(command, ''),
             json_extract(arguments_json, '$.command'),
             json_extract(arguments_json, '$.cmd'),
             json_extract(arguments_json, '$.input'),
             arguments_json) AS command_text,
    substr(result, 1, 500) AS result_head
  FROM tool_calls
  WHERE tool_name IN ('bash', 'exec', 'exec_command', 'shell')
    AND session_id = '019fa02e-27e0-73ef-9e91-013a075adb67'
)
SELECT *
FROM exec_calls
WHERE (
  -- sed commands that modify files
  command_text LIKE '%sed -i%'
  -- cat heredocs that write files
  OR command_text LIKE '%cat >%'
  OR command_text LIKE '%cat <<%'
  OR command_text LIKE '%EOF%'
  -- echo append/write
  OR (command_text LIKE '%echo%' AND (command_text LIKE '%>%' OR command_text LIKE '%>>%'))
  -- git apply / patch
  OR command_text LIKE '%git apply%'
  OR command_text LIKE '%patch %'
  -- tee
  OR command_text LIKE '%tee %'
  -- python script that writes files
  OR (command_text LIKE '%python%' AND command_text LIKE '%open(%' AND command_text LIKE '%.write%')
  -- direct file redirect
  OR (command_text LIKE '%>%' AND command_text LIKE '%linkedin%')
  OR (command_text LIKE '%>%' AND command_text LIKE '%generate-resume%')
  OR (command_text LIKE '%>%' AND command_text LIKE '%main.go%')
)
ORDER BY turn_index;
