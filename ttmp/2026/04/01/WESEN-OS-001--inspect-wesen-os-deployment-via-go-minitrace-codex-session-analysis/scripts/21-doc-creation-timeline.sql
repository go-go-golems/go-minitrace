-- 21-doc-creation-timeline.sql
-- All docmgr doc add / ticket create / doc relate calls with full arguments.
-- This tells us which tickets and documents were created, when, and in what order.

SELECT
  s.id AS session_id,
  CAST(tc->>'timestamp' AS VARCHAR) AS ts,
  CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR) AS cmd
FROM sessions_base s
CROSS JOIN UNNEST(tool_calls) AS t(tc)
WHERE s.id IN (
  '019d174c-fc68-7c00-8f1b-7fcc067c1fd6',
  '019d376d-0103-7dc3-a96d-650c7c2e1cf7',
  '019d4a35-9c8d-7f10-8fef-ef0650432725'
)
AND CAST(tc->>'tool_name' AS VARCHAR) = 'exec_command'
AND (
  CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR) LIKE '%docmgr doc add%'
  OR CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR) LIKE '%docmgr ticket create%'
)
ORDER BY CAST(tc->>'timestamp' AS VARCHAR);
