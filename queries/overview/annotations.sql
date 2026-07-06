-- annotations: List annotations joined with their sessions
-- Annotations are materialized into the normalized database alongside sessions
-- Usage:
--   go-minitrace query run --archive-glob './output/active/*/*.minitrace.json' --sql-file queries/overview/annotations.sql

SELECT
  a.session_id,
  s.agent_framework AS framework,
  a.annotator,
  a.category,
  a.title,
  a.scope_type,
  a.target_id,
  a.timestamp AS created_at,
  a.minitrace_taxonomy_json AS taxonomy_minitrace,
  a.mast_taxonomy_json AS taxonomy_mast,
  a.toolemu_taxonomy_json AS taxonomy_toolemu,
  a.tags_json AS tags,
  a.classification
FROM annotations a
JOIN sessions s ON s.session_id = a.session_id
ORDER BY a.timestamp DESC;
