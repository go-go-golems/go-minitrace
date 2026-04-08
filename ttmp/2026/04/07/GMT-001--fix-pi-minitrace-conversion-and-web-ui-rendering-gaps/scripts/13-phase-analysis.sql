SELECT 
  CASE
    WHEN CAST(tc->>'timestamp' AS TIMESTAMP) < '2026-04-06T23:00:00Z' THEN 'Phase 1: PPPP-001 SDK Build Fix'
    WHEN CAST(tc->>'timestamp' AS TIMESTAMP) < '2026-04-07T00:13:00Z' THEN 'Phase 2: PPPP-001 Cleanup & Obsidian'
    WHEN CAST(tc->>'timestamp' AS TIMESTAMP) < '2026-04-07T01:30:00Z' THEN 'Phase 3: PPPP-002 Planning'
    WHEN CAST(tc->>'timestamp' AS TIMESTAMP) < '2026-04-07T02:17:00Z' THEN 'Phase 4: PPPP-003 DRM/KMS'
    ELSE 'Phase 5: PPPP-004 Ghidra RE'
  END AS phase,
  COUNT(*) AS tool_calls,
  SUM(CASE WHEN json_extract_string(tc, '$.tool_name') = 'bash' THEN 1 ELSE 0 END) AS bash,
  SUM(CASE WHEN json_extract_string(tc, '$.tool_name') = 'read' THEN 1 ELSE 0 END) AS reads,
  SUM(CASE WHEN json_extract_string(tc, '$.tool_name') = 'write' THEN 1 ELSE 0 END) AS writes,
  SUM(CASE WHEN json_extract_string(tc, '$.tool_name') = 'edit' THEN 1 ELSE 0 END) AS edits
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
GROUP BY phase
ORDER BY MIN(CAST(tc->>'timestamp' AS TIMESTAMP))
