/* sqleton
name: read-ratio-distribution
short: Inspect read-before-write behavior across sessions
flags:
  - name: framework
    type: stringList
    help: Restrict to selected frameworks
  - name: limit
    type: int
    default: 100
    help: Limit the number of rows returned
*/
SELECT
  s.agent_framework AS framework,
  s.session_id AS id,
  s.tool_call_count AS tools,
  s.read_count AS reads,
  ROUND(m.read_ratio, 2) AS read_ratio,
  s.title
FROM sessions s
LEFT JOIN metrics m ON m.session_id = s.session_id
WHERE 1=1
{{ if .framework -}}
AND s.agent_framework IN ({{ .framework | sqlStringIn }})
{{ end -}}
ORDER BY read_ratio ASC
LIMIT {{ .limit }};
