-- 10-human-blocks.sql
-- For each user turn, show: timestamp, content preview, then count
-- how many assistant turns and tool calls follow before the next user turn.
-- This gives a "block" view: human intervention → autonomous agent run.
--
-- Replace SESSION_ID before running.

WITH numbered AS (
  SELECT
    t.idx,
    CAST(t.turn->>'role' AS VARCHAR)      AS role,
    CAST(t.turn->>'content' AS VARCHAR)   AS content,
    CAST(t.turn->>'timestamp' AS VARCHAR) AS ts,
    json_array_length(COALESCE(t.turn->'tool_calls_in_turn', '[]'::JSON)) AS tc_count
  FROM sessions_base
  CROSS JOIN UNNEST(turns) WITH ORDINALITY AS t(turn, idx)
  WHERE id = 'SESSION_ID'
),
blocks AS (
  SELECT *,
    SUM(CASE WHEN role = 'user' THEN 1 ELSE 0 END) OVER (ORDER BY idx) AS block_num
  FROM numbered
)
SELECT
  block_num                                        AS blk,
  MIN(CASE WHEN role = 'user' THEN idx END)        AS turn,
  MIN(CASE WHEN role = 'user' THEN ts END)         AS user_ts,
  LEFT(MIN(CASE WHEN role = 'user' THEN content END), 120) AS user_prompt,
  COUNT(*) FILTER (WHERE role != 'user')           AS agent_turns,
  SUM(tc_count)                                    AS tool_calls
FROM blocks
GROUP BY block_num
ORDER BY block_num;
