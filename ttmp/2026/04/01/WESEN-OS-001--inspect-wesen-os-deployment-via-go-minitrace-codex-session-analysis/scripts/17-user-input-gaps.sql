-- 17-user-input-gaps.sql
-- Time gaps between consecutive user inputs in the 3 wesen-os sessions.
-- Only shows gaps > 30 minutes (periods where human was away).

WITH user_turns AS (
  SELECT
    s.id AS session_id,
    t.idx AS turn_idx,
    CAST(t.turn->>'timestamp' AS TIMESTAMP) AS ts,
    LEFT(CAST(t.turn->>'content' AS VARCHAR), 120) AS content
  FROM sessions_base s
  CROSS JOIN UNNEST(turns) WITH ORDINALITY AS t(turn, idx)
  WHERE s.id IN (
    '019d174c-fc68-7c00-8f1b-7fcc067c1fd6',
    '019d376d-0103-7dc3-a96d-650c7c2e1cf7',
    '019d4a35-9c8d-7f10-8fef-ef0650432725'
  )
  AND CAST(t.turn->>'role' AS VARCHAR) = 'user'
),
gaps AS (
  SELECT
    *,
    ts - LAG(ts) OVER (PARTITION BY session_id ORDER BY turn_idx) AS gap
  FROM user_turns
)
SELECT
  session_id,
  turn_idx,
  ts,
  ROUND(EXTRACT(EPOCH FROM gap) / 60, 1) AS gap_minutes,
  content
FROM gaps
WHERE EXTRACT(EPOCH FROM gap) > 1800  -- gaps > 30 minutes
ORDER BY session_id, ts;
