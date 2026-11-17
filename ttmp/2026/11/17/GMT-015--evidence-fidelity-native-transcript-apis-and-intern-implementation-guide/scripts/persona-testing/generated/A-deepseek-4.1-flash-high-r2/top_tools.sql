-- Per-session top tool names by row count (ties broken by name).
WITH ranked AS (
  SELECT s.session_id, COALESCE(NULLIF(c.tool_name,''), '<unnamed>') AS tool_name,
         COUNT(*) AS n,
         ROW_NUMBER() OVER (PARTITION BY s.session_id ORDER BY COUNT(*) DESC, COALESCE(NULLIF(c.tool_name,''),'<unnamed>') ASC) AS rn
  FROM tool_calls c JOIN sessions s ON s.session_id = c.session_id
  GROUP BY 1,2
)
SELECT session_id, tool_name, n, rn FROM ranked WHERE rn <= 3 ORDER BY session_id, rn;
