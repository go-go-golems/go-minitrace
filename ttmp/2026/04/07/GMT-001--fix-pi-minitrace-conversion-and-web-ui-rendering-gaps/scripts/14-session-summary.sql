SELECT 
  id, title,
  CAST(timing->>'started_at' AS TIMESTAMP) AS started,
  CAST(timing->>'duration_seconds' AS DOUBLE) / 60 AS duration_min,
  CAST(metrics->>'turn_count' AS INT) AS turns,
  CAST(metrics->>'tool_call_count' AS INT) AS tools,
  CAST(metrics->>'total_input_tokens' AS BIGINT) AS input_tokens,
  CAST(metrics->>'total_output_tokens' AS BIGINT) AS output_tokens,
  environment->>'model' AS model
FROM sessions_base
