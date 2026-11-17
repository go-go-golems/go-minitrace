/* sqleton
name: tool-ordinal-order
short: Archive order vs chronological order for tool calls
flags:
  - name: limit
    type: int
    default: 12
    help: Rows per ordering
*/
SELECT tool_call_id, source_ordinal, emitting_turn_index AS turn_index, timestamp,
       output_status, outcome_status, success
FROM tool_calls
ORDER BY source_ordinal
LIMIT {{ .limit }};
