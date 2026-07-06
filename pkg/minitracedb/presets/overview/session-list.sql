SELECT
  s.session_id AS id,
  s.agent_framework AS framework,
  s.model,
  s.title,
  s.turn_count AS turns,
  s.tool_call_count AS tools,
  ROUND(s.duration_seconds, 1) AS duration_s,
  ROUND(m.read_ratio, 2) AS read_ratio,
  s.started_at,
  s.source_format
FROM sessions s
LEFT JOIN metrics m ON m.session_id = s.session_id
ORDER BY s.started_at;
