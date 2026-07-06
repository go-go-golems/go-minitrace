SELECT
  a.session_id,
  s.agent_framework AS framework,
  a.annotator,
  a.category,
  a.title,
  a.scope_type
FROM annotations a
JOIN sessions s ON s.session_id = a.session_id
ORDER BY a.session_id;
