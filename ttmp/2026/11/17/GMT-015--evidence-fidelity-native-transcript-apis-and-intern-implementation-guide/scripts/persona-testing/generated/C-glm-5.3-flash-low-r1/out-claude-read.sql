-- Instruction-file-read candidate search (mention vs evidence of read)
SELECT s.agent_framework, t.session_id, t.tool_call_id, t.tool_name, t.file_path,
       t.success, t.outcome_status, length(t.result) AS result_bytes, t.truncated, t.full_hash
FROM tool_calls t JOIN sessions s USING(session_id)
WHERE t.tool_name='read'
  AND (t.file_path LIKE '%CLAUDE.md' OR t.file_path LIKE '%AGENTS.md' OR t.file_path LIKE '%SKILL.md');
