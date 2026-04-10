WITH start_turns AS (
  SELECT
    sb.id AS session_id,
    MIN(REPLACE(CAST(json_extract(turn, '$.timestamp') AS VARCHAR), '"', '')) AS start_ts
  FROM sessions_base sb,
       UNNEST(sb.turns) AS t(turn)
  WHERE sb.id IN (
      '2d525241-fe32-417b-8576-b29ce3b3e47c',
      '7f61f412-40f0-417f-ab85-4dffdb9927e5'
    )
    AND REPLACE(CAST(json_extract(turn, '$.role') AS VARCHAR), '"', '') = 'user'
    AND CAST(json_extract(turn, '$.content') AS VARCHAR) LIKE '%Add detailed tasks to the ticket%'
  GROUP BY sb.id
),
boundary_annotations AS (
  SELECT
    sb.id AS session_id,
    REPLACE(CAST(json_extract(ann, '$.scope.target_id') AS VARCHAR), '"', '') AS boundary_tool_call_id
  FROM sessions_base sb,
       UNNEST(sb.annotations) AS a(ann)
  WHERE REPLACE(CAST(json_extract(ann, '$.content.title') AS VARCHAR), '"', '') = 'phase-1-code-complete'
),
boundaries AS (
  SELECT
    sb.id AS session_id,
    st.start_ts,
    REPLACE(CAST(json_extract(tc, '$.timestamp') AS VARCHAR), '"', '') AS boundary_ts
  FROM sessions_base sb
  JOIN start_turns st ON st.session_id = sb.id
  JOIN boundary_annotations ba ON ba.session_id = sb.id
  CROSS JOIN UNNEST(sb.tool_calls) AS t(tc)
  WHERE REPLACE(CAST(json_extract(tc, '$.id') AS VARCHAR), '"', '') = ba.boundary_tool_call_id
),
file_touches AS (
  SELECT
    CASE sb.id
      WHEN '2d525241-fe32-417b-8576-b29ce3b3e47c' THEN 'minimax'
      WHEN '7f61f412-40f0-417f-ab85-4dffdb9927e5' THEN 'gpt-5.4'
      ELSE 'other'
    END AS run,
    REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool_name,
    REPLACE(CAST(json_extract(tc, '$.input.file_path') AS VARCHAR), '"', '') AS file_path
  FROM sessions_base sb
  JOIN boundaries b ON b.session_id = sb.id
  CROSS JOIN UNNEST(sb.tool_calls) AS t(tc)
  WHERE REPLACE(CAST(json_extract(tc, '$.timestamp') AS VARCHAR), '"', '') BETWEEN b.start_ts AND b.boundary_ts
    AND REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') IN ('read', 'write', 'edit')
    AND json_extract(tc, '$.input.file_path') IS NOT NULL
)
SELECT run, tool_name, file_path, COUNT(*) AS touches
FROM file_touches
GROUP BY run, tool_name, file_path
ORDER BY run, touches DESC, tool_name, file_path
LIMIT 120;
