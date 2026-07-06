-- framework-summary: Aggregate statistics per agent framework
-- Usage:
--   go-minitrace query run --archive-glob './output/active/*/*.minitrace.json' --sql-file queries/overview/framework-summary.sql

SELECT
  s.agent_framework AS framework,
  COUNT(*) AS sessions,
  ROUND(AVG(s.tool_call_count), 1) AS avg_tools,
  ROUND(AVG(s.turn_count), 1) AS avg_turns,
  ROUND(AVG(m.read_ratio), 2) AS avg_read_ratio,
  ROUND(AVG(s.duration_seconds), 1) AS avg_duration_s,
  ROUND(AVG(m.time_to_first_action), 1) AS avg_ttfa_s
FROM sessions s
LEFT JOIN metrics m ON m.session_id = s.session_id
GROUP BY framework
ORDER BY avg_tools DESC;
