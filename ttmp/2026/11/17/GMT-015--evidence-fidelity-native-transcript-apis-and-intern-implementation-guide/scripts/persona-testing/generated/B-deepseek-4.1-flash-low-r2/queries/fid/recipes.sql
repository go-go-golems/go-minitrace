-- Bound-argument SQL recipe (used by fidelity.js boundArgs via queryResult)
SELECT tool_call_id, tool_name, source_ordinal
FROM tool_calls
WHERE tool_name = ?
ORDER BY source_ordinal;

-- Archive order vs chronological order (fidelity.js toolFidelity)
SELECT tool_call_id, source_ordinal, arguments_json, output_status, outcome_status, success, timestamp, emitting_turn_index
FROM tool_calls ORDER BY source_ordinal;

SELECT tool_call_id, source_ordinal, arguments_json, output_status, outcome_status, success, timestamp, emitting_turn_index
FROM tool_calls ORDER BY timestamp, tool_call_id;

-- Per-framework summary with join (fidelity.js realSummary)
SELECT s.agent_framework, tc.tool_name, COUNT(*) AS n,
       SUM(CASE WHEN tc.outcome_status='failed' THEN 1 ELSE 0 END) AS failed
FROM tool_calls tc JOIN sessions s ON s.session_id = tc.session_id
GROUP BY s.agent_framework, tc.tool_name ORDER BY n DESC LIMIT 8;

-- Token provenance split (recorded vs unknown), turns
SELECT s.agent_framework,
       CASE WHEN t.input_tokens IS NULL AND t.output_tokens IS NULL AND t.cache_read_tokens IS NULL
                 AND t.cache_creation_tokens IS NULL AND t.reasoning_tokens IS NULL AND t.tool_tokens IS NULL
            THEN 'unknown' ELSE 'recorded_usage' END AS token_origin,
       COUNT(*) AS n
FROM turns t JOIN sessions s ON s.session_id = t.session_id
GROUP BY 1,2 ORDER BY 1,2;
