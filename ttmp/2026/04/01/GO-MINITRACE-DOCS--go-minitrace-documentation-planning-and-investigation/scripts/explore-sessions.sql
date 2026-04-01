-- explore-sessions.sql
-- Exploratory queries for understanding the minitrace archive shape

-- 1) Session classification breakdown
SELECT
  classification,
  COUNT(*) AS cnt
FROM sessions_base
GROUP BY classification;

-- 2) Source format breakdown
SELECT
  provenance->>'source_format' AS source_format,
  COUNT(*) AS cnt
FROM sessions_base
GROUP BY source_format
ORDER BY cnt DESC;

-- 3) Non-subagent sessions with >= 10 turns (substantial conversations)
SELECT
  id,
  environment->>'agent_framework' AS framework,
  environment->>'model' AS model,
  title,
  CAST(metrics->>'turn_count' AS INT) AS turns,
  CAST(metrics->>'tool_call_count' AS INT) AS tools,
  ROUND(CAST(timing->>'duration_seconds' AS DOUBLE), 1) AS duration_s,
  provenance->>'source_format' AS source_format
FROM sessions_base
WHERE CAST(metrics->>'turn_count' AS INT) >= 10
  AND provenance->>'source_format' NOT LIKE '%subagent%'
ORDER BY turns DESC
LIMIT 30;

-- 4) Quality distribution
SELECT
  quality,
  COUNT(*) AS cnt
FROM sessions_base
GROUP BY quality
ORDER BY cnt DESC;

-- 5) Hour-of-day activity pattern
SELECT
  CAST(timing->>'hour_of_day' AS INT) AS hour,
  COUNT(*) AS sessions
FROM sessions_base
WHERE timing->>'hour_of_day' IS NOT NULL
GROUP BY hour
ORDER BY hour;

-- 6) Tool names used across sessions
SELECT
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool_name,
  COUNT(*) AS invocations
FROM sessions_base,
UNNEST(tool_calls) AS t(tc)
GROUP BY tool_name
ORDER BY invocations DESC
LIMIT 30;
