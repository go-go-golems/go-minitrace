SELECT 
  id AS session_id,
  REPLACE(CAST(json_extract(ann, '$.content.category') AS VARCHAR), '"', '') AS category,
  REPLACE(CAST(json_extract(ann, '$.content.title') AS VARCHAR), '"', '') AS title,
  REPLACE(CAST(json_extract(ann, '$.content.detail') AS VARCHAR), '"', '') AS detail,
  REPLACE(CAST(json_extract(ann, '$.annotator') AS VARCHAR), '"', '') AS annotator
FROM sessions_base,
     UNNEST(annotations) AS a(ann)
ORDER BY session_id
