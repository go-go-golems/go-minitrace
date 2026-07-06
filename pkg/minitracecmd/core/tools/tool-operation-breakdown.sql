/* sqleton
name: tool-operation-breakdown
short: Count tool calls by operation type and framework
flags:
  - name: framework
    type: stringList
    help: Restrict to selected frameworks
*/
SELECT
  s.agent_framework AS framework,
  tc.operation_type AS operation,
  COUNT(*) AS count
FROM tool_calls tc
JOIN sessions s ON s.session_id = tc.session_id
WHERE 1=1
{{ if .framework -}}
AND s.agent_framework IN ({{ .framework | sqlStringIn }})
{{ end -}}
GROUP BY framework, operation
ORDER BY count DESC;
