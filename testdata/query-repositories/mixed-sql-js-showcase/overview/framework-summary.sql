/* sqleton
name: framework-summary
short: Summarize sessions by framework
flags:
  - name: framework
    type: stringList
    help: Restrict to specific agent frameworks
  - name: limit
    type: int
    default: 10
    help: Maximum number of rows to return
*/
SELECT
  environment->>'agent_framework' AS framework,
  COUNT(*) AS session_count,
  AVG(CAST(metrics->>'tool_call_count' AS DOUBLE)) AS avg_tool_calls
FROM {{TABLE_NAME}}
WHERE 1=1
{{ if .framework -}}
AND environment->>'agent_framework' IN ({{ .framework | sqlStringIn }})
{{ end -}}
GROUP BY 1
ORDER BY session_count DESC, framework ASC
LIMIT {{ .limit }};
