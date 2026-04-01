-- 22-diary-writes.sql
-- All tool calls that wrote to diary files (apply_patch, tee, cat >, sed)
-- in ttmp directories. Shows when diary entries were actually committed to disk.

SELECT
  s.id AS session_id,
  CAST(tc->>'timestamp' AS VARCHAR) AS ts,
  CAST(tc->'output'->>'success' AS VARCHAR) AS success,
  LEFT(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR), 300) AS cmd
FROM sessions_base s
CROSS JOIN UNNEST(tool_calls) AS t(tc)
WHERE s.id IN (
  '019d174c-fc68-7c00-8f1b-7fcc067c1fd6',
  '019d376d-0103-7dc3-a96d-650c7c2e1cf7',
  '019d4a35-9c8d-7f10-8fef-ef0650432725'
)
AND CAST(tc->>'tool_name' AS VARCHAR) = 'exec_command'
AND LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%diary%'
AND (
  LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%apply_patch%'
  OR LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%tee %'
  OR LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%cat >%'
  OR LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%cat <<_%'
  OR LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%sed %'
  OR LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%printf%'
)
ORDER BY s.id, CAST(tc->>'timestamp' AS VARCHAR);
