/* sqleton
name: tool-breakdown
short: Count nightly-review tool usage by operation
long: |
  Count tool calls for the target day so the writeup can describe whether the
  day was reading-heavy, editing-heavy, or execution-heavy.
flags:
  - name: day
    type: date
    help: Filter sessions to one calendar day based on started_at
  - name: framework
    type: stringList
    help: Optional agent-framework filter
*/
SELECT
  REPLACE(CAST(json_extract(tc, '$.operation_type') AS VARCHAR), '"', '') AS operation,
  COUNT(*) AS count
FROM {{TABLE_NAME}},
UNNEST(tool_calls) AS t(tc)
WHERE 1=1
{{ if .day -}}
  AND CAST(timing->>'started_at' AS DATE) = CAST({{ .day | sqlDate }} AS DATE)
{{ end -}}
{{ if .framework -}}
  AND (environment->>'agent_framework') IN ({{ .framework | sqlStringIn }})
{{ end -}}
GROUP BY operation
ORDER BY count DESC, operation ASC;
