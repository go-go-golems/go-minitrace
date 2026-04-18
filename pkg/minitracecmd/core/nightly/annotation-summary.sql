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
  id AS session_id,
  REPLACE(CAST(json_extract(ann, '$.scope.type') AS VARCHAR), '"', '') AS scope_type,
  REPLACE(CAST(json_extract(ann, '$.content.category') AS VARCHAR), '"', '') AS category,
  REPLACE(CAST(json_extract(ann, '$.content.title') AS VARCHAR), '"', '') AS title,
  REPLACE(CAST(json_extract(ann, '$.content.detail') AS VARCHAR), '"', '') AS detail,
  REPLACE(CAST(json_extract(ann, '$.scope.target_id') AS VARCHAR), '"', '') AS target_id,
  REPLACE(CAST(json_extract(ann, '$.created_at') AS VARCHAR), '"', '') AS created_at
FROM {{TABLE_NAME}},
UNNEST(annotations) AS a(ann)
WHERE 1=1
{{ if .day -}}
  AND CAST(timing->>'started_at' AS DATE) = CAST({{ .day | sqlDate }} AS DATE)
{{ end -}}
ORDER BY created_at DESC, session_id ASC;
