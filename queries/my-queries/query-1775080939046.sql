SELECT id, title,
  CAST(metrics->>'turn_count' AS INT) AS turns
FROM sessions_base
LIMIT 20;