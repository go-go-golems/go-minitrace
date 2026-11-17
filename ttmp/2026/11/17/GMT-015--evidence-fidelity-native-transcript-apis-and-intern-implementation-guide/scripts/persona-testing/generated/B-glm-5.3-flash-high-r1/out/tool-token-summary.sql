-- Tool-call outcome summary plus per-turn usage coverage (normalized SQLite v3).
SELECT agent_framework,
       COUNT(*) AS tool_calls,
       SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) AS success_true,
       SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) AS success_false,
       SUM(CASE WHEN success IS NULL THEN 1 ELSE 0 END) AS success_unknown
FROM tool_calls t JOIN sessions s USING (session_id)
GROUP BY agent_framework;
