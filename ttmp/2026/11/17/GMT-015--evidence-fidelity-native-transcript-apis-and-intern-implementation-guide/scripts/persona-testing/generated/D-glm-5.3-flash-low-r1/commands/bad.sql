/* sqleton
name: bad-sql
short: Probe negative int default in SQL preamble
flags:
  - name: offset
    type: int
    default: -1
    help: Negative default probe
*/
SELECT {{ .offset }} AS offset_value, COUNT(*) AS tool_calls FROM tool_calls;
