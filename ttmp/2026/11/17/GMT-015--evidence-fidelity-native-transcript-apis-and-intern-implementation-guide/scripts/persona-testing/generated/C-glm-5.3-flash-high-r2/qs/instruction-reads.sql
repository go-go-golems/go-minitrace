-- Instruction-file-read candidates: normalized SQL over archives.
-- Read candidates (evidence of a read tool call, output_status recorded):
SELECT session_id, position_in_session, tool_name, file_path, output_status, outcome_status, length(raw_json) AS raw_len
FROM tool_calls
WHERE (file_path LIKE '%CLAUDE.md%' OR file_path LIKE '%AGENTS.md%'
       OR arguments_json LIKE '%CLAUDE.md%' OR arguments_json LIKE '%AGENTS.md%')
  AND tool_name IN ('Read','read','View','cat')
ORDER BY session_id, position_in_session;

-- Actual output of the selected candidate (session 01a0ee23..., read of CLAUDE.md):
SELECT session_id, tool_name, file_path, substr(raw_json, 1, 100) AS head,
       substr(raw_json, instr(raw_json, '"result"'), 700) AS output_head
FROM tool_calls
WHERE session_id='01a0ee23-bd68-75c1-a81f-2c43771fc216' AND tool_name='read' AND file_path='CLAUDE.md';

-- Mentions only (no read): turns that mention the instruction files, per session
-- (compare with the read candidates above: mention != evidence of a read).
SELECT s.agent_framework, s.session_id, COUNT(*) AS mention_turns
FROM turns t JOIN sessions s ON s.session_id=t.session_id
WHERE t.content LIKE '%AGENTS.md%' OR t.content LIKE '%CLAUDE.md%'
GROUP BY 1,2 ORDER BY 1,2;
