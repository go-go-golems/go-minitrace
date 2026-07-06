/* sqleton
name: timing-analysis
short: Compare timing metrics by framework
flags:
  - name: framework
    type: stringList
    help: Restrict the analysis to selected frameworks
*/
SELECT
  s.agent_framework AS framework,
  COUNT(*) AS sessions,
  ROUND(AVG(s.duration_seconds), 1) AS avg_duration_s,
  ROUND(AVG(s.active_duration_seconds), 1) AS avg_active_s,
  ROUND(AVG(m.time_to_first_action), 1) AS avg_ttfa_s,
  ROUND(AVG(m.idle_ratio), 2) AS avg_idle_ratio,
  ROUND(MIN(s.duration_seconds), 1) AS min_duration_s,
  ROUND(MAX(s.duration_seconds), 1) AS max_duration_s
FROM sessions s
LEFT JOIN metrics m ON m.session_id = s.session_id
WHERE s.duration_seconds IS NOT NULL
{{ if .framework -}}
AND s.agent_framework IN ({{ .framework | sqlStringIn }})
{{ end -}}
GROUP BY framework
ORDER BY avg_duration_s DESC;
