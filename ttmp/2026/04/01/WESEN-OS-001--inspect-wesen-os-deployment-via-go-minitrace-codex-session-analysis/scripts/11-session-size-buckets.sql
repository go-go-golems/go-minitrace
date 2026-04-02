-- 11-session-size-buckets.sql
-- Categorize all sessions by size and compute aggregate stats per bucket.
-- Separates safety-assessment subagent stubs from real sessions.

SELECT
  CASE
    WHEN title LIKE 'The following is the Codex agent history%' THEN 'safety-assessment'
    WHEN CAST(metrics->>'turn_count' AS INT) <= 2 THEN 'stub (≤2 turns)'
    WHEN CAST(metrics->>'turn_count' AS INT) <= 10 THEN 'short (3-10 turns)'
    WHEN CAST(metrics->>'turn_count' AS INT) <= 100 THEN 'medium (11-100 turns)'
    WHEN CAST(metrics->>'turn_count' AS INT) <= 500 THEN 'long (101-500 turns)'
    ELSE 'marathon (500+ turns)'
  END AS category,
  COUNT(*) AS sessions,
  SUM(CAST(metrics->>'turn_count' AS INT)) AS total_turns,
  ROUND(SUM(CAST(timing->>'active_duration_seconds' AS DOUBLE)) / 3600, 1) AS active_hours,
  ROUND(SUM(CAST(timing->>'duration_seconds' AS DOUBLE)) / 3600, 1) AS wall_hours
FROM sessions_base
GROUP BY category
ORDER BY sessions DESC;
