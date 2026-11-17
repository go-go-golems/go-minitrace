-- Instruction-file-read candidate search: filter read-like tool calls whose
-- file_path or arguments mention AGENTS.md / CLAUDE.md. This is a *candidate*
-- search (mention-level), not yet proof that the file content entered context.
SELECT
  tc.session_id,
  s.agent_framework AS framework,
  tc.tool_call_id,
  tc.tool_name,
  tc.file_path,
  tc.success,
  tc.outcome_status,
  tc.output_status,
  tc.full_bytes,
  substr(COALESCE(tc.arguments_json,''),1,160) AS args_head
FROM tool_calls tc
JOIN sessions s ON s.session_id = tc.session_id
WHERE (tc.tool_name LIKE '%read%' OR tc.tool_name LIKE '%view%' OR tc.tool_name LIKE '%cat%' OR tc.tool_name LIKE '%open%')
  AND (tc.file_path LIKE '%AGENTS.md%' OR tc.file_path LIKE '%CLAUDE.md%'
       OR tc.arguments_json LIKE '%AGENTS.md%' OR tc.arguments_json LIKE '%CLAUDE.md%')
ORDER BY s.agent_framework, tc.session_id, tc.source_ordinal
