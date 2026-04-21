-- Inspect a single tool_call entry structure
SELECT json_structure(tool_calls[1]) AS tc_structure
FROM sessions_base
WHERE tool_calls IS NOT NULL
LIMIT 1;
