/* sqleton
name: annotations
short: List annotations across sessions
flags:
  - name: framework
    type: stringList
    help: Restrict to selected frameworks
  - name: limit
    type: int
    default: 100
    help: Limit the number of rows returned
*/
SELECT
  id AS session_id,
  environment->>'agent_framework' AS framework,
  CAST(json_extract(ann, '$.annotator') AS VARCHAR) AS annotator,
  CAST(json_extract(ann, '$.content.category') AS VARCHAR) AS category,
  CAST(json_extract(ann, '$.content.title') AS VARCHAR) AS title,
  CAST(json_extract(ann, '$.scope.type') AS VARCHAR) AS scope_type
FROM {{TABLE_NAME}},
  UNNEST(annotations) AS a(ann)
WHERE 1=1
{{ if .framework -}}
AND (environment->>'agent_framework') IN ({{ .framework | sqlStringIn }})
{{ end -}}
ORDER BY session_id
LIMIT {{ .limit }};
