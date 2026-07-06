/* sqleton
name: session-list
short: List minitrace sessions
flags:
  - name: framework
    type: stringList
    help: Filter by agent framework
  - name: title_like
    type: string
    help: Filter titles with LIKE
  - name: limit
    type: int
    default: 100
    help: Limit the number of rows returned
*/
SELECT
  s.session_id AS id,
  s.agent_framework AS framework,
  s.model,
  s.title,
  s.turn_count AS turns,
  s.tool_call_count AS tools,
  ROUND(s.duration_seconds, 1) AS duration_s,
  ROUND(m.read_ratio, 2) AS read_ratio,
  s.started_at,
  s.source_format
FROM sessions s
LEFT JOIN metrics m ON m.session_id = s.session_id
WHERE 1=1
{{ if .framework -}}
AND s.agent_framework IN ({{ .framework | sqlStringIn }})
{{ end -}}
{{ if .title_like -}}
AND s.title LIKE {{ .title_like | sqlLike }}
{{ end -}}
ORDER BY s.started_at DESC
LIMIT {{ .limit }};
