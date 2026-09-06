SELECT s.session_id, s.turn_count, s.tool_call_count,
 (SELECT count(*) FROM tool_calls c WHERE c.session_id=s.session_id AND c.tool_name='exec' AND c.operation_type='OTHER') AS exec_other,
 (SELECT count(*) FROM tool_calls c WHERE c.session_id=s.session_id AND c.command IS NOT NULL AND c.command!='') AS commands,
 (SELECT count(*) FROM tool_calls c WHERE c.session_id=s.session_id AND c.exit_code IS NOT NULL) AS exit_codes,
 (SELECT count(*) FROM tool_calls c WHERE c.session_id=s.session_id AND c.result LIKE '%map[text:%') AS map_outputs,
 (SELECT count(*) FROM tool_calls c LEFT JOIN turns t ON t.session_id=c.session_id AND t.turn_index=c.emitting_turn_index WHERE c.session_id=s.session_id AND c.emitting_turn_index IS NOT NULL AND t.turn_index IS NULL) AS orphan_links
FROM sessions s ORDER BY s.session_id;
