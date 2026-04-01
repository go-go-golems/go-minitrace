-- 24-git-commits.sql
-- All git commit commands made during the 3 wesen-os sessions.
-- Shows what was actually committed and when.

SELECT
  s.id AS session_id,
  CAST(tc->>'timestamp' AS VARCHAR) AS ts,
  CAST(tc->'output'->>'success' AS VARCHAR) AS success,
  LEFT(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR), 300) AS cmd,
  LEFT(CAST(tc->'output'->>'result' AS VARCHAR), 200) AS result
FROM sessions_base s
CROSS JOIN UNNEST(tool_calls) AS t(tc)
WHERE s.id IN (
  '019d174c-fc68-7c00-8f1b-7fcc067c1fd6',
  '019d376d-0103-7dc3-a96d-650c7c2e1cf7',
  '019d4a35-9c8d-7f10-8fef-ef0650432725'
)
AND CAST(tc->>'tool_name' AS VARCHAR) = 'exec_command'
AND CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR) LIKE '%git commit%'
AND CAST(tc->'output'->>'success' AS BOOLEAN) = true
ORDER BY s.id, CAST(tc->>'timestamp' AS VARCHAR);
