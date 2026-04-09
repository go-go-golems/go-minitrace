SELECT
  sb.id AS session_id,
  sb.environment->>'model' AS model,
  REPLACE(CAST(json_extract(ann, '$.annotator') AS VARCHAR), '"', '') AS annotator,
  REPLACE(CAST(json_extract(ann, '$.content.category') AS VARCHAR), '"', '') AS category,
  REPLACE(CAST(json_extract(ann, '$.content.title') AS VARCHAR), '"', '') AS title,
  REPLACE(CAST(json_extract(ann, '$.scope.type') AS VARCHAR), '"', '') AS scope_type,
  REPLACE(CAST(json_extract(ann, '$.scope.target_id') AS VARCHAR), '"', '') AS target_id,
  REPLACE(CAST(json_extract(ann, '$.timestamp') AS VARCHAR), '"', '') AS created_at,
  REPLACE(CAST(json_extract(ann, '$.classification') AS VARCHAR), '"', '') AS classification
FROM sessions_base sb,
     UNNEST(sb.annotations) AS a(ann)
WHERE REPLACE(CAST(json_extract(ann, '$.content.title') AS VARCHAR), '"', '') IN (
  'phase-1-code-complete',
  'phase-1-bookkeeping-complete'
)
ORDER BY session_id, created_at;
