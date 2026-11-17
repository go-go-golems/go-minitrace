SELECT s.session_id,s.agent_framework,s.started_at,s.ended_at,
 (SELECT COUNT(*) FROM turns t WHERE t.session_id=s.session_id) AS normalized_turns,
 (SELECT COUNT(*) FROM tool_calls t WHERE t.session_id=s.session_id) AS normalized_tools,
 (SELECT COUNT(*) FROM tool_calls t WHERE t.session_id=s.session_id AND t.success=0) AS normalized_failed_tools,
 (SELECT COUNT(*) FROM tool_calls t WHERE t.session_id=s.session_id AND t.success IS NULL) AS normalized_unknown_tools,
 (SELECT SUM(input_tokens) FROM turns t WHERE t.session_id=s.session_id) AS recorded_input,
 (SELECT SUM(output_tokens) FROM turns t WHERE t.session_id=s.session_id) AS recorded_output,
 (SELECT SUM(cache_read_tokens) FROM turns t WHERE t.session_id=s.session_id) AS recorded_cache_read,
 (SELECT SUM(cache_creation_tokens) FROM turns t WHERE t.session_id=s.session_id) AS recorded_cache_write,
 (SELECT SUM(reasoning_tokens) FROM turns t WHERE t.session_id=s.session_id) AS recorded_reasoning
FROM sessions s ORDER BY s.session_id;
