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
  tc.operation_type AS operation,
  COUNT(*) AS count
FROM tool_calls tc
JOIN sessions s ON s.session_id = tc.session_id
WHERE 1=1
{{ if .day -}}
  AND date(s.started_at) = date({{ .day | sqlDate }})
{{ end -}}
{{ if .framework -}}
  AND s.agent_framework IN ({{ .framework | sqlStringIn }})
{{ end -}}
GROUP BY operation
ORDER BY count DESC, operation ASC;
