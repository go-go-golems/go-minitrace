WITH ranked AS (
  SELECT session_id, tool_name,
         COUNT(*) AS n,
         ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY COUNT(*) DESC, tool_name) AS rn
  FROM tool_calls
  GROUP BY session_id, tool_name
)
SELECT session_id, tool_name, n FROM ranked WHERE rn <= 5
ORDER BY session_id, rn;
