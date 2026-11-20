SELECT s.session_id, s.agent_framework,
 (SELECT COUNT(*) FROM turns t WHERE t.session_id=s.session_id) AS turn_count,
 (SELECT COUNT(*) FROM tool_calls t WHERE t.session_id=s.session_id) AS tool_count,
 (SELECT COUNT(*) FROM tool_calls t WHERE t.session_id=s.session_id AND t.success IS NULL) AS unknown_tool_success_count,
 (SELECT MIN(source_ordinal) FROM tool_calls t WHERE t.session_id=s.session_id) AS min_source_ordinal,
 (SELECT MAX(source_ordinal) FROM tool_calls t WHERE t.session_id=s.session_id) AS max_source_ordinal
FROM sessions s ORDER BY s.agent_framework, s.session_id;
