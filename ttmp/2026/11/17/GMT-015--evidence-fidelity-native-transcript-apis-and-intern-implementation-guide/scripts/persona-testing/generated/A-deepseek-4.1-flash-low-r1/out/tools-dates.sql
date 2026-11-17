SELECT
  s.session_id,
  s.agent_framework AS framework,
  (SELECT tc.tool_name FROM tool_calls tc
     WHERE tc.session_id = s.session_id
     GROUP BY tc.tool_name
     ORDER BY COUNT(*) DESC, tc.tool_name ASC LIMIT 1) AS top_tool,
  (SELECT COUNT(*) FROM tool_calls tc
     WHERE tc.session_id = s.session_id
       AND tc.tool_name = (SELECT tc2.tool_name FROM tool_calls tc2
            WHERE tc2.session_id = s.session_id
            GROUP BY tc2.tool_name ORDER BY COUNT(*) DESC, tc2.tool_name ASC LIMIT 1)) AS top_tool_count,
  (SELECT MIN(ts) FROM (SELECT timestamp AS ts FROM turns WHERE session_id=s.session_id AND timestamp IS NOT NULL AND timestamp<>'' UNION ALL SELECT timestamp FROM tool_calls WHERE session_id=s.session_id AND timestamp IS NOT NULL AND timestamp<>'')) AS first_activity,
  (SELECT MAX(ts) FROM (SELECT timestamp AS ts FROM turns WHERE session_id=s.session_id AND timestamp IS NOT NULL AND timestamp<>'' UNION ALL SELECT timestamp FROM tool_calls WHERE session_id=s.session_id AND timestamp IS NOT NULL AND timestamp<>'')) AS last_activity
FROM sessions s
ORDER BY s.agent_framework, s.session_id;
