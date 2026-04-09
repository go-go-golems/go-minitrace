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
  id,
  environment->>'agent_framework' AS framework,
  environment->>'model' AS model,
  title,
  CAST(metrics->>'turn_count' AS INT) AS turns,
  CAST(metrics->>'tool_call_count' AS INT) AS tools,
  ROUND(CAST(timing->>'duration_seconds' AS DOUBLE), 1) AS duration_s,
  ROUND(CAST(metrics->>'read_ratio' AS DOUBLE), 2) AS read_ratio,
  timing->>'started_at' AS started_at,
  provenance->>'source_format' AS source_format
FROM {{TABLE_NAME}}
WHERE 1=1
{{ if .framework -}}
AND (environment->>'agent_framework') IN ({{ .framework | sqlStringIn }})
{{ end -}}
{{ if .title_like -}}
AND title LIKE {{ .title_like | sqlLike }}
{{ end -}}
ORDER BY timing->>'started_at' DESC
LIMIT {{ .limit }};
