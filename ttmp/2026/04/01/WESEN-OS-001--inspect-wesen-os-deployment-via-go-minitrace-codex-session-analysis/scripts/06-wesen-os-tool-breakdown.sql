-- 06-wesen-os-tool-breakdown.sql
-- Tool call breakdown for the three key wesen-os sessions
SELECT
  id,
  REPLACE(CAST(tc->>'tool_name' AS VARCHAR), '"', '') AS tool,
  COUNT(*) AS calls
FROM sessions_base
CROSS JOIN UNNEST(tool_calls) AS t(tc)
WHERE id IN (
  '019d174c-fc68-7c00-8f1b-7fcc067c1fd6',
  '019d376d-0103-7dc3-a96d-650c7c2e1cf7',
  '019d4a35-9c8d-7f10-8fef-ef0650432725'
)
GROUP BY id, tool
ORDER BY id, calls DESC;
