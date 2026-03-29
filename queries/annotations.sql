-- annotations.sql
-- Unnest and query session annotations.
-- Usage:
--   duckdb analysis.duckdb -init queries/load.sql -f queries/annotations.sql

SELECT
  id AS session_id,
  environment->>'agent_framework' AS framework,
  CAST(json_extract(ann, '$.annotator') AS VARCHAR) AS annotator,
  CAST(json_extract(ann, '$.content.category') AS VARCHAR) AS category,
  CAST(json_extract(ann, '$.content.title') AS VARCHAR) AS title,
  CAST(json_extract(ann, '$.scope.type') AS VARCHAR) AS scope_type
FROM sessions_base,
UNNEST(annotations) AS a(ann)
ORDER BY session_id;
