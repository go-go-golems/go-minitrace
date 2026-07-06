/* sqleton
name: session-inventory
short: List nightly-review sessions for a day
long: |
  List the sessions that belong in a nightly transcript review. The command is
  intentionally reusable because daily review work often needs to be resumed in
  later windows with the same filters.
flags:
  - name: day
    type: date
    help: Filter sessions to one calendar day based on started_at
  - name: framework
    type: stringList
    help: Optional agent-framework filter
  - name: title_like
    type: string
    help: Optional LIKE filter on the session title
  - name: limit
    type: int
    default: 100
    help: Limit the result set
*/
SELECT
  s.session_id AS id,
  s.agent_framework AS framework,
  s.model,
  s.working_directory,
  s.title,
  s.turn_count AS turns,
  s.tool_call_count AS tools,
  ROUND(s.duration_seconds / 3600, 1) AS hours,
  ROUND(m.read_ratio, 2) AS read_ratio,
  s.started_at,
  s.ended_at,
  s.source_format
FROM sessions s
LEFT JOIN metrics m ON m.session_id = s.session_id
WHERE 1=1
{{ if .day -}}
  AND date(s.started_at) = date({{ .day | sqlDate }})
{{ end -}}
{{ if .framework -}}
  AND s.agent_framework IN ({{ .framework | sqlStringIn }})
{{ end -}}
{{ if .title_like -}}
  AND s.title LIKE {{ .title_like | sqlLike }}
{{ end -}}
ORDER BY s.started_at ASC
LIMIT {{ .limit }};
