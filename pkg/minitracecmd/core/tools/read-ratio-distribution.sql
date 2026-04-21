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
  environment->>'agent_framework' AS framework,
  id,
  CAST(metrics->>'tool_call_count' AS INT) AS tools,
  CAST(metrics->>'read_count' AS INT) AS reads,
  ROUND(CAST(metrics->>'read_ratio' AS DOUBLE), 2) AS read_ratio,
  title
FROM {{TABLE_NAME}}
WHERE 1=1
{{ if .framework -}}
AND (environment->>'agent_framework') IN ({{ .framework | sqlStringIn }})
{{ end -}}
ORDER BY read_ratio ASC
LIMIT {{ .limit }};
