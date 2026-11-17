-- (B) Mention/search only: rg -il over **/CLAUDE.md; result is a hit list, not instruction text.
SELECT session_id, emitting_turn_index AS turn, tool_name, operation_type,
       substr(command,1,300) AS command, length(COALESCE(result,'')) AS result_len,
       substr(COALESCE(result,''),1,200) AS result_head
FROM tool_calls
WHERE session_id LIKE '01a0a5c3%' AND emitting_turn_index=4
  AND command LIKE '%CLAUDE.md%';

