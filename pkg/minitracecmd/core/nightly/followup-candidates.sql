/* sqleton
name: followup-candidates
short: Surface sessions worth revisiting in another context window
long: |
  Identify unusually long or tool-heavy sessions. This is the handoff query for
  multi-window work: it gives the next pass a smaller set of sessions to inspect
  or annotate in detail.

  The day filter matches any session whose active window includes the day, not
  only sessions that started that day (see session-inventory).
flags:
  - name: day
    type: date
    help: Include sessions active on this calendar day (started on/before, ended on/after)
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
  s.working_directory,
  s.session_id AS id,
  s.title,
  s.turn_count AS turns,
  s.tool_call_count AS tools,
  ROUND(s.duration_seconds / 3600, 1) AS hours,
  ROUND(m.read_ratio, 2) AS read_ratio,
  CASE
    WHEN s.tool_call_count >= {{ .min_tools }}
      AND s.turn_count >= {{ .min_turns }}
      THEN 'tool-heavy and turn-heavy'
    WHEN s.duration_seconds / 3600 >= {{ .min_hours }}
      THEN 'long-running'
    WHEN m.read_ratio >= 0.6
      THEN 'research-heavy'
    ELSE 'review-candidate'
  END AS reason,
  s.started_at
FROM sessions s
LEFT JOIN metrics m ON m.session_id = s.session_id
WHERE 1=1
{{ if .day -}}
  AND date(s.started_at) <= date({{ .day | sqlDate }})
  AND date(COALESCE(s.ended_at, s.started_at)) >= date({{ .day | sqlDate }})
{{ end -}}
AND (
  s.tool_call_count >= {{ .min_tools }}
  OR s.turn_count >= {{ .min_turns }}
  OR s.duration_seconds / 3600 >= {{ .min_hours }}
)
ORDER BY hours DESC, tools DESC, s.started_at ASC
LIMIT {{ .limit }};
