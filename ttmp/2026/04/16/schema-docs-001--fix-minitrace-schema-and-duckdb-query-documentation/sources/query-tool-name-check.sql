-- Inspect actual tool_name values
SELECT DISTINCT tc.tool_name, typeof(tc.tool_name) AS tn_type
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
LIMIT 20;
