-- 05-deep-read-session.sql
-- Extract full turn content for a specific session id.
-- Replace SESSION_ID with the target session.
-- Usage: run with --sql "$(cat 05-deep-read-session.sql | sed 's/SESSION_ID/the-uuid/')"
SELECT
  t.idx                                        AS turn_num,
  CAST(t.turn->>'role'        AS VARCHAR)      AS role,
  CAST(t.turn->>'source'      AS VARCHAR)      AS source,
  CAST(t.turn->>'timestamp'   AS VARCHAR)      AS ts,
  LEFT(CAST(t.turn->>'content' AS VARCHAR), 2000) AS content_preview
FROM sessions_base
CROSS JOIN UNNEST(turns) WITH ORDINALITY AS t(turn, idx)
WHERE id = 'SESSION_ID'
  AND CAST(t.turn->>'role' AS VARCHAR) = 'user'
ORDER BY t.idx
LIMIT 30;
