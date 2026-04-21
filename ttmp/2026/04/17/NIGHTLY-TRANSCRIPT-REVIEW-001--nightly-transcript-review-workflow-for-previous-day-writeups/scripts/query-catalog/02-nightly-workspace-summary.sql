/* sqleton
name: nightly-workspace-summary
short: Summarize nightly-review work by workspace
long: |
  Group the prior day's transcripts by working directory so the nightly review
  can be written as a sequence of workspace stories instead of a single flat
  list.
flags:
  - name: day
    type: date
    help: Filter sessions to one calendar day based on started_at
  - name: framework
    type: stringList
    help: Optional agent-framework filter
*/
SELECT
  COALESCE(
    operational_context->>'working_directory',
    environment->>'working_directory',
    provenance->>'cwd'
  ) AS working_directory,
  COUNT(*) AS sessions,
  ROUND(SUM(CAST(timing->>'duration_seconds' AS DOUBLE)) / 3600, 1) AS hours,
  SUM(CAST(metrics->>'tool_call_count' AS INT)) AS tools,
  SUM(CAST(metrics->>'turn_count' AS INT)) AS turns,
  ROUND(AVG(CAST(metrics->>'read_ratio' AS DOUBLE)), 2) AS avg_read_ratio,
  MIN(timing->>'started_at') AS first_started_at,
  MAX(timing->>'started_at') AS last_started_at
FROM {{TABLE_NAME}}
WHERE 1=1
{{ if .day -}}
  AND CAST(timing->>'started_at' AS DATE) = CAST({{ .day | sqlDate }} AS DATE)
{{ end -}}
{{ if .framework -}}
  AND environment->>'agent_framework' IN ({{ .framework | sqlStringIn }})
{{ end -}}
GROUP BY working_directory
ORDER BY hours DESC, tools DESC, working_directory ASC;
