-- 12-marathon-sessions.sql
-- The 500+ turn marathon sessions with active vs wall-clock and tool density.

SELECT
  id,
  LEFT(title, 80) AS title,
  CAST(metrics->>'turn_count' AS INT) AS turns,
  CAST(metrics->>'tool_call_count' AS INT) AS tools,
  ROUND(CAST(timing->>'duration_seconds' AS DOUBLE) / 3600, 1) AS wall_h,
  ROUND(CAST(timing->>'active_duration_seconds' AS DOUBLE) / 3600, 1) AS active_h,
  ROUND(
    CAST(timing->>'active_duration_seconds' AS DOUBLE) /
    NULLIF(CAST(timing->>'duration_seconds' AS DOUBLE), 0) * 100, 1
  ) AS active_pct,
  ROUND(CAST(metrics->>'tool_call_count' AS INT) * 1.0 /
    NULLIF(CAST(metrics->>'turn_count' AS INT), 0), 1) AS tools_per_turn
FROM sessions_base
WHERE CAST(metrics->>'turn_count' AS INT) > 500
ORDER BY CAST(metrics->>'turn_count' AS INT) DESC;
