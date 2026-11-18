SELECT f.session_id, f.tool_call_id, f.target_ordinal, f.path, f.evidence_status
FROM files f JOIN sessions s ON s.session_id = f.session_id
WHERE LOWER(f.operation_type) IN ('write','edit','create','delete','rename','move','patch','update')
  AND (COALESCE(s.agent_framework,'') != 'codex' OR f.evidence_kind != 'legacy_scalar')
ORDER BY f.session_id, f.tool_call_id, f.target_ordinal, f.path;
