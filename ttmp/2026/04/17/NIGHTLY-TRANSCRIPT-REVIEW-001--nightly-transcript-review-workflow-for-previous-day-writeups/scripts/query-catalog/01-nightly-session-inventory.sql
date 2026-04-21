/* sqleton
name: nightly-session-inventory
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
  id,
  environment->>'agent_framework' AS framework,
  environment->>'model' AS model,
  COALESCE(
    operational_context->>'working_directory',
    environment->>'working_directory',
    provenance->>'cwd'
  ) AS working_directory,
  title,
  CAST(metrics->>'turn_count' AS INT) AS turns,
  CAST(metrics->>'tool_call_count' AS INT) AS tools,
  ROUND(CAST(timing->>'duration_seconds' AS DOUBLE) / 3600, 1) AS hours,
  ROUND(CAST(metrics->>'read_ratio' AS DOUBLE), 2) AS read_ratio,
  timing->>'started_at' AS started_at,
  timing->>'ended_at' AS ended_at,
  provenance->>'source_format' AS source_format
FROM {{TABLE_NAME}}
WHERE 1=1
{{ if .day -}}
  AND CAST(timing->>'started_at' AS DATE) = CAST({{ .day | sqlDate }} AS DATE)
{{ end -}}
{{ if .framework -}}
  AND environment->>'agent_framework' IN ({{ .framework | sqlStringIn }})
{{ end -}}
{{ if .title_like -}}
  AND title LIKE {{ .title_like | sqlLike }}
{{ end -}}
ORDER BY timing->>'started_at' ASC
LIMIT {{ .limit }};
