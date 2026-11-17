/* sqleton
name: session-counts
short: Count turns per session across all archives
params:
output: json
*/
SELECT s.session_id, s.agent_framework, COUNT(t.id) AS turn_count
FROM sessions s LEFT JOIN turns t ON t.session_id = s.session_id
GROUP BY s.session_id, s.agent_framework
ORDER BY s.session_id;
