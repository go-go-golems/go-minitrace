-- Bound-argument tool evidence recipe (three ? placeholders; pass "", "", tool_name)
SELECT session_id,
       tool_call_id,
       source_ordinal,
       emitting_turn_index AS turn_index,
       timestamp,
       tool_name,
       arguments_json,
       output_status,
       outcome_status,
       success
FROM tool_calls
WHERE (? = '' OR session_id = ?)
  AND tool_name = ?
ORDER BY source_ordinal;
