/* sqleton
name: tool-operation-breakdown
short: Count tool calls by operation type and framework
flags:
  - name: framework
    type: stringList
    help: Restrict to selected frameworks
*/
SELECT
  environment->>'agent_framework' AS framework,
  REPLACE(CAST(json_extract(tc, '$.operation_type') AS VARCHAR), '"', '') AS operation,
  COUNT(*) AS count
FROM {{TABLE_NAME}},
  UNNEST(tool_calls) AS t(tc)
WHERE 1=1
{{ if .framework -}}
AND (environment->>'agent_framework') IN ({{ .framework | sqlStringIn }})
{{ end -}}
GROUP BY framework, operation
ORDER BY count DESC;
