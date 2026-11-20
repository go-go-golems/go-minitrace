-- per-session actual counts from normalized child rows vs metadata aggregates
SELECT
  s.session_id,
  s.agent_framework,
  (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_rows,
  (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_rows,
  s.turn_count AS meta_turn_count,
  s.tool_call_count AS meta_tool_count,
  s.started_at,
  s.ended_at
FROM sessions s
ORDER BY s.agent_framework, s.started_at;

-- top tool names per session (top 5)
WITH ranked AS (
  SELECT session_id, tool_name,
         COUNT(*) AS n,
         ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY COUNT(*) DESC, tool_name) AS rn
  FROM tool_calls
  GROUP BY session_id, tool_name
)
SELECT session_id, tool_name, n FROM ranked WHERE rn <= 5
ORDER BY session_id, rn;

-- date range
SELECT MIN(started_at) AS min_started, MAX(COALESCE(ended_at, started_at)) AS max_ended FROM sessions;
