-- file-operations: All file-target evidence with independent target outcomes.
SELECT
  f.session_id,
  tc.emitting_turn_index AS turn,
  tc.tool_name AS tool,
  f.operation_type AS operation,
  f.path AS target,
  f.success,
  f.evidence_kind,
  f.evidence_status,
  f.resolved,
  tc.timestamp
FROM files f
JOIN tool_calls tc ON tc.session_id=f.session_id AND tc.tool_call_id=f.tool_call_id
JOIN sessions s ON s.session_id=f.session_id
WHERE COALESCE(s.agent_framework,'')!='codex' OR f.evidence_kind!='legacy_scalar'
ORDER BY tc.timestamp, tc.tool_call_id, f.target_ordinal;
