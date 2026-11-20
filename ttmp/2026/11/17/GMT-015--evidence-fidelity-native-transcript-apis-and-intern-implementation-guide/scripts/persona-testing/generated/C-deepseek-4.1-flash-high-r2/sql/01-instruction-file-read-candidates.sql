-- Instruction-file READ candidates from normalized archives.
-- Separates a real read (operation_type=READ + file_path + captured result)
-- from shell searches/mentions (EXECUTE + command text only).
SELECT s.agent_framework AS fw,
       substr(s.session_id,1,8) AS sid,
       tc.emitting_turn_index AS turn,
       tc.tool_name AS tool,
       tc.operation_type AS op,
       tc.file_path,
       length(COALESCE(tc.result,'')) AS result_len,
       substr(COALESCE(tc.result,''),1,60) AS result_head
FROM tool_calls tc JOIN sessions s USING(session_id)
WHERE COALESCE(tc.file_path,'') LIKE '%AGENTS.md%'
   OR COALESCE(tc.file_path,'') LIKE '%CLAUDE.md%'
   OR COALESCE(tc.file_path,'') LIKE '%SKILL.md%'
   OR COALESCE(tc.command,'')  LIKE '%AGENTS.md%'
   OR COALESCE(tc.command,'')  LIKE '%CLAUDE.md%'
   OR COALESCE(tc.command,'')  LIKE '%SKILL.md%'
ORDER BY fw, sid, turn;
