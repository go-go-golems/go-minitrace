/* sqleton
name: annotations
short: List annotations across sessions
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
  a.session_id,
  s.agent_framework AS framework,
  a.annotator,
  a.category,
  a.title,
  a.scope_type
FROM annotations a
JOIN sessions s ON s.session_id = a.session_id
WHERE 1=1
{{ if .framework -}}
AND s.agent_framework IN ({{ .framework | sqlStringIn }})
{{ end -}}
ORDER BY a.session_id
LIMIT {{ .limit }};
