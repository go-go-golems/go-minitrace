-- 09-deploy-timeline.sql
-- Chronological timeline of all deploy/wesen-os/k3s/hetzner related sessions
-- with workdir, hours, and prompt summary for a concise narrative.
SELECT
  timing->>'started_at'                                          AS started_at,
  id,
  ROUND(CAST(timing->>'duration_seconds' AS DOUBLE) / 3600, 1)  AS hours,
  CAST(metrics->>'turn_count'      AS INT)                       AS turns,
  CAST(metrics->>'tool_call_count' AS INT)                       AS tools,
  operational_context->>'working_directory'                      AS workdir,
  LEFT(title, 100)                                               AS title_short,
  LEFT(CAST(turns[1]->>'content' AS VARCHAR), 400)               AS prompt_snippet
FROM sessions_base
WHERE
  LOWER(operational_context->>'working_directory') LIKE '%wesen-os%'
  OR LOWER(operational_context->>'working_directory') LIKE '%hetzner%'
  OR LOWER(operational_context->>'working_directory') LIKE '%k3s%'
  OR LOWER(operational_context->>'working_directory') LIKE '%hair-booking%'
  OR LOWER(CAST(turns[1]->>'content' AS VARCHAR))  LIKE '%wesen-os%'
  OR LOWER(CAST(turns[1]->>'content' AS VARCHAR))  LIKE '%hetzner%'
  OR LOWER(CAST(turns[1]->>'content' AS VARCHAR))  LIKE '%k3s%'
  OR LOWER(title)                                  LIKE '%wesen-os%'
  OR LOWER(title)                                  LIKE '%hetzner%'
  OR LOWER(title)                                  LIKE '%deploy%'
ORDER BY timing->>'started_at';
