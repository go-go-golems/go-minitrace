/* sqleton
name: tool-counts
short: Count tool_calls rows per session
output: json
*/
SELECT tc.session_id, COUNT(*) AS tool_count
FROM tool_calls tc
GROUP BY tc.session_id
ORDER BY tc.session_id;
