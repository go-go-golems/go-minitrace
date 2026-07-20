/* sqleton
name: annotation-summary
short: Summarize annotations for nightly-review sessions
long: |
  Show annotations that already exist in the archive. This is the bridge between
  the review workflow and the go-minitrace annotation system: once a session is
  marked up, the next window can pick up from here.

  The day filter matches annotations on any session whose active window
  includes the day, not only sessions that started that day (see
  session-inventory).
flags:
  - name: day
    type: date
    help: Include annotations on sessions active on this calendar day
*/
SELECT
  a.session_id,
  a.scope_type,
  a.category,
  a.title,
  a.detail,
  a.target_id,
  a.timestamp AS created_at
FROM annotations a
JOIN sessions s ON s.session_id = a.session_id
WHERE 1=1
{{ if .day -}}
  AND date(s.started_at) <= date({{ .day | sqlDate }})
  AND date(COALESCE(s.ended_at, s.started_at)) >= date({{ .day | sqlDate }})
{{ end -}}
ORDER BY created_at DESC, a.session_id ASC;
