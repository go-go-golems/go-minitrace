-- 13-wesen-os-active-vs-wall.sql
-- Active vs wall-clock for just the 3 wesen-os sessions.

SELECT
  id,
  LEFT(title, 70) AS title,
  CAST(metrics->>'turn_count' AS INT) AS turns,
  ROUND(CAST(timing->>'duration_seconds' AS DOUBLE) / 3600, 1) AS wall_h,
  ROUND(CAST(timing->>'active_duration_seconds' AS DOUBLE) / 3600, 1) AS active_h,
  ROUND(CAST(timing->>'active_duration_seconds' AS DOUBLE) /
    NULLIF(CAST(timing->>'duration_seconds' AS DOUBLE), 0) * 100, 1) AS active_pct
FROM sessions_base
WHERE id IN (
  '019d174c-fc68-7c00-8f1b-7fcc067c1fd6',
  '019d376d-0103-7dc3-a96d-650c7c2e1cf7',
  '019d4a35-9c8d-7f10-8fef-ef0650432725'
)
ORDER BY timing->>'started_at';
