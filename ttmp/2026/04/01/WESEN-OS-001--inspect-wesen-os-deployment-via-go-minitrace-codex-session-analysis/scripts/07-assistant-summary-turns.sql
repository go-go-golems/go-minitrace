-- 07-assistant-summary-turns.sql
-- Extract assistant turns (non-tool) for a session to see what was concluded
-- Shows the last N assistant turns as a session summary
SELECT
  t.idx                                           AS turn_num,
  CAST(t.turn->>'timestamp' AS VARCHAR)           AS ts,
  LEFT(CAST(t.turn->>'content' AS VARCHAR), 1200) AS assistant_text
FROM sessions_base
CROSS JOIN UNNEST(turns) WITH ORDINALITY AS t(turn, idx)
WHERE id = 'SESSION_ID'
  AND CAST(t.turn->>'role'   AS VARCHAR) = 'assistant'
  AND CAST(t.turn->>'source' AS VARCHAR) != 'tool'
  AND LENGTH(CAST(t.turn->>'content' AS VARCHAR)) > 100
ORDER BY t.idx
LIMIT 25;
