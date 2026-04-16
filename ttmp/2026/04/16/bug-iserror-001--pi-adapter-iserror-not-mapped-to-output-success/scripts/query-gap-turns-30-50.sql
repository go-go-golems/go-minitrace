-- Query 7: Look for aborted/error turns between 14 and 48
-- especially model errors or overloaded responses
SELECT
  CAST(json_extract(turn, '$.index') AS INT) AS turn_idx,
  json_extract(turn, '$.role') AS role,
  LEFT(CAST(json_extract(turn, '$.content') AS VARCHAR), 300) AS content_preview,
  json_extract(turn, '$.model') AS model,
  json_extract(turn, '$.timestamp') AS ts
FROM sessions_base,
  UNNEST(turns) AS t(turn)
WHERE
  CAST(json_extract(turn, '$.index') AS INT) BETWEEN 30 AND 50
ORDER BY CAST(json_extract(turn, '$.index') AS INT);
