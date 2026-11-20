SELECT failure.session_id, failure.tool_call_id, failure.source_ordinal,
       (SELECT MIN(later.source_ordinal)
        FROM tool_calls later
        WHERE later.session_id = failure.session_id AND later.tool_name = failure.tool_name
          AND later.source_ordinal > failure.source_ordinal AND later.success = 1
          AND COALESCE(later.exit_code, 0) = 0) AS next_success_ordinal
FROM tool_calls failure
WHERE failure.success = 0 OR failure.exit_code != 0
ORDER BY failure.session_id, failure.source_ordinal, failure.tool_call_id;
