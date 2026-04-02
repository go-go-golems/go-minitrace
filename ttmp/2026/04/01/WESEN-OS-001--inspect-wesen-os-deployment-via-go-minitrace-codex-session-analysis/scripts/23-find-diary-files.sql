-- 23-find-diary-files.sql
-- Extract all distinct ttmp/ paths referenced in diary-related tool calls.
-- These are the actual diary files we can read on disk for content analysis.

SELECT DISTINCT
  regexp_extract(
    CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR),
    '((?:/home/manuel/[^ ]+|(?:openai-app-server/)?ttmp/)[^ ]*diary[^ ]*\.md)',
    1
  ) AS diary_path
FROM sessions_base s
CROSS JOIN UNNEST(tool_calls) AS t(tc)
WHERE s.id IN (
  '019d174c-fc68-7c00-8f1b-7fcc067c1fd6',
  '019d376d-0103-7dc3-a96d-650c7c2e1cf7',
  '019d4a35-9c8d-7f10-8fef-ef0650432725'
)
AND CAST(tc->>'tool_name' AS VARCHAR) = 'exec_command'
AND LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%diary%'
AND regexp_extract(
    CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR),
    '((?:/home/manuel/[^ ]+|(?:openai-app-server/)?ttmp/)[^ ]*diary[^ ]*\.md)',
    1
  ) != ''
ORDER BY diary_path;
