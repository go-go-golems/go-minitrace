/* sqleton
name: followup-candidates
short: Surface sessions worth revisiting in another context window
long: |
  Identify unusually long or tool-heavy sessions. This is the handoff query for
  multi-window work: it gives the next pass a smaller set of sessions to inspect
  or annotate in detail.
flags:
  - name: day
    type: date
    help: Filter sessions to one calendar day based on started_at
  - name: min_tools
    type: int
    default: 100
    help: Minimum tool calls before a session is considered follow-up worthy
  - name: min_turns
    type: int
    default: 40
    help: Minimum turns before a session is considered follow-up worthy
  - name: min_hours
    type: float
    default: 3.0
    help: Minimum duration in hours before a session is considered follow-up worthy
  - name: limit
    type: int
    default: 20
    help: Limit the result set
*/
SELECT
  COALESCE(
    operational_context->>'working_directory',
    environment->>'working_directory',
    provenance->>'cwd'
  ) AS working_directory,
  id,
  title,
  CAST(metrics->>'turn_count' AS INT) AS turns,
  CAST(metrics->>'tool_call_count' AS INT) AS tools,
  ROUND(CAST(timing->>'duration_seconds' AS DOUBLE) / 3600, 1) AS hours,
  ROUND(CAST(metrics->>'read_ratio' AS DOUBLE), 2) AS read_ratio,
  CASE
    WHEN CAST(metrics->>'tool_call_count' AS INT) >= {{ .min_tools }}
      AND CAST(metrics->>'turn_count' AS INT) >= {{ .min_turns }}
      THEN 'tool-heavy and turn-heavy'
    WHEN CAST(timing->>'duration_seconds' AS DOUBLE) / 3600 >= {{ .min_hours }}
      THEN 'long-running'
    WHEN CAST(metrics->>'read_ratio' AS DOUBLE) >= 0.6
      THEN 'research-heavy'
    ELSE 'review-candidate'
  END AS reason,
  timing->>'started_at' AS started_at
FROM {{TABLE_NAME}}
WHERE 1=1
{{ if .day -}}
  AND CAST(timing->>'started_at' AS DATE) = CAST({{ .day | sqlDate }} AS DATE)
{{ end -}}
AND (
  CAST(metrics->>'tool_call_count' AS INT) >= {{ .min_tools }}
  OR CAST(metrics->>'turn_count' AS INT) >= {{ .min_turns }}
  OR CAST(timing->>'duration_seconds' AS DOUBLE) / 3600 >= {{ .min_hours }}
)
ORDER BY hours DESC, tools DESC, started_at ASC
LIMIT {{ .limit }};
