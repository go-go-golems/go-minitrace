-- Recorded per-turn usage with explicit provenance; missing fields are zero-filled
SELECT 'turn' AS scope, session_id, turn_index, role,
       COALESCE(input_tokens,0) AS input_tokens,
       COALESCE(output_tokens,0) AS output_tokens,
       (input_tokens IS NOT NULL) + (output_tokens IS NOT NULL)
         + (cache_read_tokens IS NOT NULL) + (cache_creation_tokens IS NOT NULL)
         + (reasoning_tokens IS NOT NULL) + (tool_tokens IS NOT NULL) AS reported_usage_fields,
       CASE WHEN ((input_tokens IS NOT NULL) + (output_tokens IS NOT NULL)
         + (cache_read_tokens IS NOT NULL) + (cache_creation_tokens IS NOT NULL)
         + (reasoning_tokens IS NOT NULL) + (tool_tokens IS NOT NULL)) > 0
            THEN 'recorded_usage' ELSE 'unknown' END AS token_origin,
       'tokens' AS token_unit,
       'recorded_fields_zero_fill' AS token_method
FROM turns
ORDER BY session_id, turn_index;
