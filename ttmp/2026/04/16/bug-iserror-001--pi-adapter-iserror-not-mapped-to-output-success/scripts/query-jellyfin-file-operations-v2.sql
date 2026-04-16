-- Check what file_path looks like for write/edit/read tools
SELECT
  CAST(tc.tool_name AS VARCHAR) AS tool,
  tc.emitting_turn_index AS turn,
  tc.input.file_path AS file_path,
  tc.input.arguments.path AS arg_path
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE
  CAST(tc.tool_name AS VARCHAR) IN ('write', 'edit', 'read')
  AND (
    tc.input.file_path LIKE '%jellyfin%'
    OR tc.input.file_path LIKE '%diary%'
    OR tc.input.arguments.path LIKE '%jellyfin%'
    OR tc.input.arguments.path LIKE '%diary%'
  )
ORDER BY tc.emitting_turn_index;
