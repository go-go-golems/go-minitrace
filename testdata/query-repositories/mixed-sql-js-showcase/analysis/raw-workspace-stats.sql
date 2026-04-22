/* sqleton
name: raw-workspace-stats
short: Raw SQL workspace aggregates
flags:
  - name: framework
    type: stringList
    help: Restrict to specific agent frameworks
  - name: min_tool_calls
    type: int
    default: 0
    help: Minimum tool-call count per session
  - name: limit
    type: int
    default: 10
    help: Maximum number of rows to return
*/
SELECT
  COALESCE(operational_context->>'working_directory', '(none)') AS working_directory,
  COUNT(*) AS session_count,
  AVG(CAST(metrics->>'tool_call_count' AS DOUBLE)) AS avg_tool_calls,
  AVG(CAST(metrics->>'turn_count' AS DOUBLE)) AS avg_turns
FROM {{TABLE_NAME}}
WHERE 1=1
{{ if .framework -}}
AND (environment->>'agent_framework') IN ({{ .framework | sqlStringIn }})
{{ end -}}
{{ if .min_tool_calls -}}
AND CAST(metrics->>'tool_call_count' AS BIGINT) >= {{ .min_tool_calls }}
{{ end -}}
GROUP BY 1
ORDER BY session_count DESC, avg_tool_calls DESC, working_directory ASC
LIMIT {{ .limit }};
