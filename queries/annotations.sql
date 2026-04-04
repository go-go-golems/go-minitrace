-- annotations.sql
-- Query annotations from the SQLite annotations.db (attached via sqlite_scanner).
-- This file is served alongside sessions_base (loaded via load.sql).
-- Annotations are live — no refresh needed after writes.
-- Usage:
--   duckdb analysis.duckdb -init queries/load.sql -f queries/annotations.sql

SELECT
  a.session_id,
  sb.environment->>'agent_framework' AS framework,
  a.annotator,
  a.category,
  a.title,
  a.scope_type,
  a.target_id,
  a.created_at,
  a.taxonomy_m AS taxonomy_minitrace,
  a.taxonomy_mast AS taxonomy_mast,
  a.taxonomy_tm AS taxonomy_toolemu,
  a.tags,
  a.classification
FROM annotations a
JOIN sessions_base sb ON sb.id = a.session_id
ORDER BY a.created_at DESC;
