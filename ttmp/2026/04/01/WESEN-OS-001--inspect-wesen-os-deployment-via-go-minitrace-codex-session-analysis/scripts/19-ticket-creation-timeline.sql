-- 19-ticket-creation-timeline.sql
-- Find all ticket-creation and mkdir ttmp events across the 3 wesen-os sessions.
-- This shows when new tickets/workspaces were spawned during the session.

SELECT
  s.id AS session_id,
  CAST(tc->>'timestamp' AS VARCHAR) AS ts,
  LEFT(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR), 200) AS cmd
FROM sessions_base s
CROSS JOIN UNNEST(tool_calls) AS t(tc)
WHERE s.id IN (
  '019d174c-fc68-7c00-8f1b-7fcc067c1fd6',
  '019d376d-0103-7dc3-a96d-650c7c2e1cf7',
  '019d4a35-9c8d-7f10-8fef-ef0650432725'
)
AND CAST(tc->>'tool_name' AS VARCHAR) = 'exec_command'
AND (
  LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%ticket create%'
  OR (
    LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%mkdir%'
    AND LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%ttmp%'
    AND LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%---%'
  )
)
ORDER BY CAST(tc->>'timestamp' AS VARCHAR);
