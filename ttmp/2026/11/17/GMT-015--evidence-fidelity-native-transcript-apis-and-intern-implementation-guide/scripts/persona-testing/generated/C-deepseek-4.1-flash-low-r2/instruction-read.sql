-- Instruction-file-read candidate: normalized tool_calls whose target looks like
-- an agent instruction file but whose operation is a real READ.
SELECT session_id, source_ordinal, emitting_turn_index, tool_name,
       operation_type, file_path, length(result) AS result_len,
       substr(result, 1, 400) AS result_head
FROM tool_calls
WHERE file_path LIKE '%obsidian-vault-writing/SKILL.md';

-- Contrast: directory listing that only MENTIONS AGENTS.md (EXECUTE, not READ).
SELECT session_id, source_ordinal, tool_name, operation_type,
       substr(command, 1, 120) AS cmd
FROM tool_calls
WHERE command LIKE 'ls && ls docs%';

-- All read-type calls against instruction-like files.
SELECT session_id, tool_name, operation_type, file_path,
       length(result) AS result_len, substr(result, 1, 160) AS result_head
FROM tool_calls
WHERE file_path IS NOT NULL
  AND (lower(file_path) LIKE '%agent%' OR lower(file_path) LIKE '%claude.md%'
       OR lower(file_path) LIKE '%instructions%' OR lower(file_path) LIKE '%.md')
ORDER BY session_id, source_ordinal;
