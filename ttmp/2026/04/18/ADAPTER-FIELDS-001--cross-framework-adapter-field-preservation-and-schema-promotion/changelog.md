# Changelog

## 2026-04-18

- Initial workspace created

## 2026-04-18

Created follow-up ticket `ADAPTER-FIELDS-001` to separate cross-framework field preservation/schema work from the Pi `isError` bug ticket. Added a source-backed field matrix covering Pi, Codex, and Claude Code, plus a ticket-local scanner script and scan output so the findings can be re-run against representative raw transcripts. Recorded in commit `1da7a17065d19c283e88f4154fa6db1fac5bdb1f` (`Add adapter field analysis ticket`).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/analysis/01-cross-framework-field-matrix.md - Source-backed field matrix and promotion recommendations
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/scripts/01-scan-field-representations.py - Reproducible raw-field scanner
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/sources/01-field-scan.txt - Captured scanner output for the sampled raw transcripts

## 2026-04-18

Implemented the first-wave schema promotions from the new ticket: `tool_calls[].output.exit_code` and `tool_calls[].input.justification`. Updated the minitrace schema/builders, populated both fields in the Codex adapter, added regression coverage, and refreshed the schema/adapter docs. Validation included `gofmt -w ...`, `go test ./pkg/adapters/codex ./pkg/minitrace -count=1`, and `go test ./... -count=1`. Recorded in commit `ffbd6a1d62512a01250d9a780b5dcc6be2b73f3a` (`Promote Codex exit code and justification`).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitrace/schema.go - Added first-class `exit_code` and `justification` fields
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitrace/builders.go - Initialized the new tool input/output fields in the shared builder
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/codex/convert.go - Populated `exit_code` and `justification` during Codex conversion
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/codex/convert_test.go - Added assertions covering the promoted fields
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/doc/minitrace-schema.md - Documented the new schema fields
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/doc/adapter-reference.md - Documented Codex preservation of command metadata and justification

## 2026-04-18

Preserved the Codex metadata-only follow-up fields in the adapter. Session/runtime details such as `approval_policy`, detailed `sandbox_policy`, `collaboration_mode_detail`, `truncation_policy`, `rate_limits`, and `session_source` now survive in `operational_context.framework_config`. Turn-level `turn_id`, `phase`, and `memory_citation` now survive in `turns[].framework_metadata`. Tool-call metadata now keeps `source`, `parsed_cmd`, `stdout`, `stderr`, `status`, and `turn_id`. Added regression coverage for both session-jsonl-v1 and exec-jsonl-v1 preservation. Recorded in commit `585db79bab918ce61131144c78918b8591b71c65` (`Preserve Codex adapter metadata`).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/codex/convert.go - Preserved Codex session, turn, and tool metadata in framework_config/framework_metadata
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/codex/convert_test.go - Added metadata preservation regression coverage for session-jsonl-v1 and exec-jsonl-v1

## 2026-04-18

Preserved the Claude Code metadata-only follow-up fields in the adapter. Session-level `entrypoint` now survives in `operational_context.framework_config`. Turn metadata now keeps `entrypoint`, `slug`, `parent_uuid`, `is_sidechain`, `stop_reason`, `stop_sequence`, and detailed `cache_creation` buckets. Tool-call metadata now keeps `caller` plus the skipped tool-result record context so that thread/session metadata is not lost when tool-result pseudo-turns are absorbed into tool calls. Added regression coverage. Recorded in commit `01e5ddccb19e5165dc5248a4afbfd067957d5a63` (`Preserve Claude adapter metadata`).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/claudecode/convert.go - Preserved Claude session, turn, and tool metadata
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/claudecode/convert_test.go - Added regression coverage for Claude metadata preservation

## 2026-04-18

Preserved the Pi metadata-only follow-up fields in the adapter. Assistant turns now keep `stop_reason` and `error_message` in `turns[].framework_metadata`. Tool results now preserve `details.diff` and `firstChangedLine` as tool-call metadata so edit-result diffs survive conversion. Added regression coverage for both assistant-turn metadata and message-level tool-result metadata. Recorded in commit `332a9d7c648a45fa2c2a33a366508366f4aeeb49` (`Preserve Pi adapter metadata`).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/pi/convert.go - Preserved Pi assistant and tool-result metadata
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/pi/convert_test.go - Added regression coverage for Pi metadata preservation

## 2026-04-18

Added detailed documentation for framework-specific metadata storage and mappings. Introduced a new help page that lists the preserved Codex, Claude Code, and Pi metadata keys and where each one is stored (`framework_config`, `turn.framework_metadata`, or `tool_call.framework_metadata`). Updated the schema and adapter reference docs to describe the new conventions and preserved fields. Also updated the ticket tasks/diary to reflect the completed implementation slices.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/doc/framework-metadata-mappings.md - Detailed per-adapter metadata mapping reference
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/doc/minitrace-schema.md - Added framework metadata storage conventions
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/doc/adapter-reference.md - Listed preserved metadata per adapter and linked to the new mapping doc
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/reference/01-diary.md - Detailed work diary for the metadata preservation slices
