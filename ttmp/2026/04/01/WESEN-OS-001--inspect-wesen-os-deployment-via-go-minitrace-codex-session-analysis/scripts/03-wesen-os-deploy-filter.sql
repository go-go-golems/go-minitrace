-- 03-wesen-os-deploy-filter.sql
-- Find all sessions mentioning wesen-os, coolify, or deployment topics
-- in their title, working directory, OR the first user turn content.
SELECT
  id,
  timing->>'started_at'                                          AS started_at,
  title,
  CAST(metrics->>'turn_count'      AS INT)                       AS turns,
  CAST(metrics->>'tool_call_count' AS INT)                       AS tools,
  ROUND(CAST(timing->>'duration_seconds' AS DOUBLE) / 3600, 2)  AS hours,
  operational_context->>'working_directory'                      AS workdir,
  -- first user turn content (first 300 chars for preview)
  LEFT(CAST(turns[1]->>'content' AS VARCHAR), 300)               AS prompt_preview
FROM sessions_base
WHERE
  LOWER(title) LIKE '%wesen-os%'
  OR LOWER(title) LIKE '%wesen_os%'
  OR LOWER(title) LIKE '%deploy%'
  OR LOWER(title) LIKE '%coolify%'
  OR LOWER(operational_context->>'working_directory') LIKE '%wesen-os%'
  OR LOWER(CAST(turns[1]->>'content' AS VARCHAR)) LIKE '%wesen-os%'
  OR LOWER(CAST(turns[1]->>'content' AS VARCHAR)) LIKE '%wesen_os%'
  OR LOWER(CAST(turns[1]->>'content' AS VARCHAR)) LIKE '%coolify%'
ORDER BY timing->>'started_at';
