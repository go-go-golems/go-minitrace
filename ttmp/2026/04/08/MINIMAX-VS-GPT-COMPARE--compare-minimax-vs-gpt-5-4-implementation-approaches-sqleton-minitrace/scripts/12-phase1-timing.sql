WITH start_turns AS (
  SELECT
    sb.id AS session_id,
    MIN(REPLACE(CAST(json_extract(turn, '$.timestamp') AS VARCHAR), '"', '')) AS implementation_start_ts
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
code_boundaries AS (
  SELECT
    sb.id AS session_id,
    REPLACE(CAST(json_extract(tc, '$.timestamp') AS VARCHAR), '"', '') AS code_complete_ts
  FROM sessions_base sb,
       UNNEST(sb.annotations) AS a(ann),
       UNNEST(sb.tool_calls) AS t(tc)
  WHERE REPLACE(CAST(json_extract(ann, '$.content.title') AS VARCHAR), '"', '') = 'phase-1-code-complete'
    AND REPLACE(CAST(json_extract(tc, '$.id') AS VARCHAR), '"', '') = REPLACE(CAST(json_extract(ann, '$.scope.target_id') AS VARCHAR), '"', '')
),
bookkeeping_boundaries AS (
  SELECT
    sb.id AS session_id,
    REPLACE(CAST(json_extract(tc, '$.timestamp') AS VARCHAR), '"', '') AS bookkeeping_complete_ts
  FROM sessions_base sb,
       UNNEST(sb.annotations) AS a(ann),
       UNNEST(sb.tool_calls) AS t(tc)
  WHERE REPLACE(CAST(json_extract(ann, '$.content.title') AS VARCHAR), '"', '') = 'phase-1-bookkeeping-complete'
    AND REPLACE(CAST(json_extract(tc, '$.id') AS VARCHAR), '"', '') = REPLACE(CAST(json_extract(ann, '$.scope.target_id') AS VARCHAR), '"', '')
)
SELECT
  CASE st.session_id
    WHEN '2d525241-fe32-417b-8576-b29ce3b3e47c' THEN 'minimax'
    WHEN '7f61f412-40f0-417f-ab85-4dffdb9927e5' THEN 'gpt-5.4'
    ELSE 'other'
  END AS run,
  st.session_id,
  st.implementation_start_ts,
  cb.code_complete_ts,
  bb.bookkeeping_complete_ts,
  ROUND(DATE_DIFF('second', CAST(st.implementation_start_ts AS TIMESTAMP), CAST(cb.code_complete_ts AS TIMESTAMP)) / 60.0, 1) AS minutes_to_code_complete,
  ROUND(DATE_DIFF('second', CAST(st.implementation_start_ts AS TIMESTAMP), CAST(bb.bookkeeping_complete_ts AS TIMESTAMP)) / 60.0, 1) AS minutes_to_bookkeeping_complete
FROM start_turns st
JOIN code_boundaries cb ON cb.session_id = st.session_id
JOIN bookkeeping_boundaries bb ON bb.session_id = st.session_id
ORDER BY run;
