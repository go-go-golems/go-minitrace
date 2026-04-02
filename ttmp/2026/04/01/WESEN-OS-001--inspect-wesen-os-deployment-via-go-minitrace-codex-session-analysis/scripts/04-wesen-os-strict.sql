-- 04-wesen-os-strict.sql
-- Strict filter: sessions where workdir or first-turn content explicitly
-- references the wesen-os codebase/project (not just coolify generically).
SELECT
  id,
  timing->>'started_at'                                          AS started_at,
  ROUND(CAST(timing->>'duration_seconds' AS DOUBLE) / 3600, 2)  AS hours,
  CAST(metrics->>'turn_count'      AS INT)                       AS turns,
  CAST(metrics->>'tool_call_count' AS INT)                       AS tools,
  operational_context->>'working_directory'                      AS workdir,
  title,
  LEFT(CAST(turns[1]->>'content' AS VARCHAR), 500)               AS prompt_preview
FROM sessions_base
WHERE
  LOWER(operational_context->>'working_directory') LIKE '%wesen-os%'
  OR LOWER(title)                                  LIKE '%wesen-os%'
  OR LOWER(CAST(turns[1]->>'content' AS VARCHAR))  LIKE '%wesen-os%'
  OR LOWER(CAST(turns[1]->>'content' AS VARCHAR))  LIKE '%wesen_os%'
ORDER BY timing->>'started_at';
