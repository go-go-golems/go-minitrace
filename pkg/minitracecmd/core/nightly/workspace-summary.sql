/* sqleton
name: workspace-summary
short: Summarize nightly-review work by workspace
long: |
  Group the prior day's transcripts by working directory so the nightly review
  can be written as a sequence of workspace stories instead of a single flat
  list.
flags:
  - name: day
    type: date
    help: Filter sessions to one calendar day based on started_at
  - name: framework
    type: stringList
    help: Optional agent-framework filter
*/
SELECT
  s.working_directory,
  COUNT(*) AS sessions,
  ROUND(SUM(s.duration_seconds) / 3600, 1) AS hours,
  SUM(s.tool_call_count) AS tools,
  SUM(s.turn_count) AS turns,
  ROUND(AVG(m.read_ratio), 2) AS avg_read_ratio,
  MIN(s.started_at) AS first_started_at,
  MAX(s.started_at) AS last_started_at
FROM sessions s
LEFT JOIN metrics m ON m.session_id = s.session_id
WHERE 1=1
{{ if .day -}}
  AND date(s.started_at) = date({{ .day | sqlDate }})
{{ end -}}
{{ if .framework -}}
  AND s.agent_framework IN ({{ .framework | sqlStringIn }})
{{ end -}}
GROUP BY s.working_directory
ORDER BY hours DESC, tools DESC, s.working_directory ASC;
