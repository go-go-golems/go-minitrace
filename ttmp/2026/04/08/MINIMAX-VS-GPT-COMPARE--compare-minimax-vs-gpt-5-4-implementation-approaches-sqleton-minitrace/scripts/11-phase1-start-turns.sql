SELECT
  CASE sb.id
    WHEN '2d525241-fe32-417b-8576-b29ce3b3e47c' THEN 'minimax'
    WHEN '7f61f412-40f0-417f-ab85-4dffdb9927e5' THEN 'gpt-5.4'
    ELSE 'other'
  END AS run,
  sb.id AS session_id,
  CAST(json_extract(turn, '$.index') AS INT) AS turn_index,
  REPLACE(CAST(json_extract(turn, '$.role') AS VARCHAR), '"', '') AS role,
  REPLACE(CAST(json_extract(turn, '$.timestamp') AS VARCHAR), '"', '') AS turn_ts,
  SUBSTR(REPLACE(CAST(json_extract(turn, '$.content') AS VARCHAR), '"', ''), 1, 220) AS content_snippet
FROM sessions_base sb,
     UNNEST(sb.turns) AS t(turn)
WHERE sb.id IN (
    '2d525241-fe32-417b-8576-b29ce3b3e47c',
    '7f61f412-40f0-417f-ab85-4dffdb9927e5'
  )
  AND REPLACE(CAST(json_extract(turn, '$.role') AS VARCHAR), '"', '') = 'user'
  AND CAST(json_extract(turn, '$.content') AS VARCHAR) LIKE '%Add detailed tasks to the ticket%'
ORDER BY run, turn_index;
