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
phase1_files AS (
  SELECT DISTINCT
    CASE sb.id
      WHEN '2d525241-fe32-417b-8576-b29ce3b3e47c' THEN 'minimax'
      WHEN '7f61f412-40f0-417f-ab85-4dffdb9927e5' THEN 'gpt-5.4'
      ELSE 'other'
    END AS run,
    CASE
      WHEN REPLACE(CAST(json_extract(tc, '$.input.file_path') AS VARCHAR), '"', '') LIKE '%/go-minitrace/%' THEN
        SUBSTR(REPLACE(CAST(json_extract(tc, '$.input.file_path') AS VARCHAR), '"', ''), STRPOS(REPLACE(CAST(json_extract(tc, '$.input.file_path') AS VARCHAR), '"', ''), '/go-minitrace/') + 13)
      ELSE REPLACE(CAST(json_extract(tc, '$.input.file_path') AS VARCHAR), '"', '')
    END AS normalized_path
  FROM sessions_base sb
  JOIN boundaries b ON b.session_id = sb.id
  CROSS JOIN UNNEST(sb.tool_calls) AS t(tc)
  WHERE REPLACE(CAST(json_extract(tc, '$.timestamp') AS VARCHAR), '"', '') BETWEEN b.start_ts AND b.boundary_ts
    AND REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') IN ('read', 'write', 'edit')
    AND json_extract(tc, '$.input.file_path') IS NOT NULL
),
pivoted AS (
  SELECT
    normalized_path,
    MAX(CASE WHEN run = 'minimax' THEN 1 ELSE 0 END) AS touched_by_minimax,
    MAX(CASE WHEN run = 'gpt-5.4' THEN 1 ELSE 0 END) AS touched_by_gpt
  FROM phase1_files
  GROUP BY normalized_path
)
SELECT normalized_path, touched_by_minimax, touched_by_gpt,
  CASE
    WHEN touched_by_minimax = 1 AND touched_by_gpt = 1 THEN 'both'
    WHEN touched_by_minimax = 1 THEN 'minimax-only'
    WHEN touched_by_gpt = 1 THEN 'gpt-only'
    ELSE 'neither'
  END AS overlap_type
FROM pivoted
ORDER BY overlap_type, normalized_path;
