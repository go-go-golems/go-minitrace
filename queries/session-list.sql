-- session-list.sql
-- List all sessions with key metadata fields.
-- Usage:
--   duckdb analysis.duckdb -init queries/load.sql -f queries/session-list.sql

SELECT
  id,
  environment->>'agent_framework' AS framework,
  environment->>'model' AS model,
  title,
  CAST(metrics->>'turn_count' AS INT) AS turns,
  CAST(metrics->>'tool_call_count' AS INT) AS tools,
  ROUND(CAST(timing->>'duration_seconds' AS DOUBLE), 1) AS duration_s,
  ROUND(CAST(metrics->>'read_ratio' AS DOUBLE), 2) AS read_ratio,
  timing->>'started_at' AS started_at,
  provenance->>'source_format' AS source_format
FROM sessions_base
ORDER BY timing->>'started_at';
