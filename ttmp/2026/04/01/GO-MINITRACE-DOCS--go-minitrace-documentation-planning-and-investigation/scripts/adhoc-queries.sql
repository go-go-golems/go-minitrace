-- adhoc-queries.sql
-- Custom queries run during the documentation investigation
-- to understand archive shape and query capabilities.

-- Models used across frameworks
SELECT
  environment->>'agent_framework' AS framework,
  environment->>'model' AS model,
  COUNT(*) AS sessions
FROM sessions_base
GROUP BY framework, model
ORDER BY sessions DESC;

-- Token usage summary by framework
SELECT
  environment->>'agent_framework' AS framework,
  ROUND(AVG(CAST(metrics->>'total_input_tokens' AS BIGINT)), 0) AS avg_input_tokens,
  ROUND(AVG(CAST(metrics->>'total_output_tokens' AS BIGINT)), 0) AS avg_output_tokens,
  ROUND(AVG(CAST(metrics->>'total_cache_read_tokens' AS BIGINT)), 0) AS avg_cache_read,
  ROUND(SUM(CAST(metrics->>'total_input_tokens' AS BIGINT)) / 1e6, 1) AS total_input_M,
  ROUND(SUM(CAST(metrics->>'total_output_tokens' AS BIGINT)) / 1e6, 1) AS total_output_M
FROM sessions_base
GROUP BY framework;

-- Tool name usage across all sessions
SELECT
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool_name,
  COUNT(*) AS invocations
FROM sessions_base,
UNNEST(tool_calls) AS t(tc)
GROUP BY tool_name
ORDER BY invocations DESC
LIMIT 30;

-- Source format breakdown
SELECT
  provenance->>'source_format' AS source_format,
  COUNT(*) AS cnt
FROM sessions_base
GROUP BY source_format
ORDER BY cnt DESC;

-- Longest sessions by turn count (non-subagent)
SELECT
  id,
  environment->>'agent_framework' AS framework,
  title,
  CAST(metrics->>'turn_count' AS INT) AS turns,
  CAST(metrics->>'tool_call_count' AS INT) AS tools,
  ROUND(CAST(timing->>'duration_seconds' AS DOUBLE), 0) AS duration_s
FROM sessions_base
WHERE provenance->>'source_format' NOT LIKE '%subagent%'
ORDER BY turns DESC
LIMIT 20;
