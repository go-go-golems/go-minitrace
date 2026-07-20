/* sqleton
name: tool-breakdown
short: Count nightly-review tool usage by operation
long: |
  Count tool calls for the target day so the writeup can describe whether the
  day was reading-heavy, editing-heavy, or execution-heavy.

  The day filter matches on each tool call's own timestamp, not the session's
  start date. A session that started days earlier but ran tool calls on the
  target day contributes exactly those calls, and only those — so the count
  reflects work performed on the day rather than every call of every session
  that happened to begin on it.
flags:
  - name: day
    type: date
    help: Count tool calls whose timestamp falls on this calendar day
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
  AND date(tc.timestamp) = date({{ .day | sqlDate }})
{{ end -}}
{{ if .framework -}}
  AND s.agent_framework IN ({{ .framework | sqlStringIn }})
{{ end -}}
GROUP BY operation
ORDER BY count DESC, operation ASC;
