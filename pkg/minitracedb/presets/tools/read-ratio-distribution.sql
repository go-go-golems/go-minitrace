SELECT
  s.agent_framework AS framework,
  s.session_id AS id,
  s.tool_call_count AS tools,
  s.read_count AS reads,
  s.modify_count AS modifies,
  s.create_count AS creates,
  s.execute_count AS executes,
  ROUND(m.read_ratio, 2) AS read_ratio
FROM sessions s
LEFT JOIN metrics m ON m.session_id = s.session_id
WHERE s.tool_call_count > 0
ORDER BY read_ratio DESC, tools DESC, id;
