SELECT session_id, tool_name, calls, rn AS rank
FROM (
  SELECT session_id, tool_name, COUNT(*) AS calls,
         ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY COUNT(*) DESC, tool_name ASC) AS rn
  FROM tool_calls
  GROUP BY session_id, tool_name
)
WHERE rn <= 3
ORDER BY session_id, rn;
