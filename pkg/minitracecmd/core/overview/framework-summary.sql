/* sqleton
name: framework-summary
short: Summarize sessions by framework
flags:
  - name: framework
    type: stringList
    help: Restrict the summary to selected frameworks
*/
SELECT
  s.agent_framework AS framework,
  COUNT(*) AS sessions,
  ROUND(AVG(s.tool_call_count), 1) AS avg_tools,
  ROUND(AVG(s.turn_count), 1) AS avg_turns,
  ROUND(AVG(m.read_ratio), 2) AS avg_read_ratio,
  ROUND(AVG(s.duration_seconds), 1) AS avg_duration_s,
  ROUND(AVG(m.time_to_first_action), 1) AS avg_ttfa_s
FROM sessions s
LEFT JOIN metrics m ON m.session_id = s.session_id
WHERE 1=1
{{ if .framework -}}
AND s.agent_framework IN ({{ .framework | sqlStringIn }})
{{ end -}}
GROUP BY framework
ORDER BY sessions DESC;
