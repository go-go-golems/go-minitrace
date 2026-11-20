-- Instruction-file evidence probe over normalized archives.
-- A hit here is only a *candidate*: it shows a tool call whose name/args mention
-- an instruction file. Read evidence additionally requires operation_type=READ
-- plus a non-empty output.result that actually carries instruction text.

-- 1. All mentions of instruction files, with weak vs strong evidence markers.
SELECT
  tc.session_id,
  tc.emitting_turn_index,
  tc.tool_name,
  tc.operation_type,
  COALESCE(tc.file_path, json_extract(tc.arguments_json, '$.path'), json_extract(tc.arguments_json, '$.file_path')) AS candidate_path,
  substr(tc.command, 1, 160) AS command_preview,
  CASE WHEN tc.operation_type = 'READ' THEN 'strong: READ-op'
       WHEN tc.tool_name IN ('grep','Grep','web_search') THEN 'weak: search only'
       ELSE 'weak: mention only' END AS evidence_strength,
  coalesce(length(tc.result), 0) AS output_chars,
  substr(tc.result, 1, 200) AS output_preview
FROM tool_calls tc
WHERE lower(coalesce(tc.file_path,'')) LIKE '%agents.md%'
   OR lower(coalesce(tc.file_path,'')) LIKE '%claude.md%'
   OR lower(coalesce(tc.arguments_json,'')) LIKE '%agents.md%'
   OR lower(coalesce(tc.arguments_json,'')) LIKE '%claude.md%'
   OR lower(coalesce(tc.command,'')) LIKE '%agents.md%'
   OR lower(coalesce(tc.command,'')) LIKE '%claude.md%'
ORDER BY evidence_strength, tc.session_id, tc.emitting_turn_index
LIMIT 40;
