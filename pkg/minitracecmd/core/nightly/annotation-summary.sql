/* sqleton
name: annotation-summary
short: Summarize annotations for nightly-review sessions
long: |
  Show annotations that already exist in the archive. This is the bridge between
  the review workflow and the go-minitrace annotation system: once a session is
  marked up, the next window can pick up from here.
flags:
  - name: day
    type: date
    help: Filter sessions to one calendar day based on started_at
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
  AND date(s.started_at) = date({{ .day | sqlDate }})
{{ end -}}
ORDER BY created_at DESC, a.session_id ASC;
