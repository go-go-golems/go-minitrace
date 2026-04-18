# Tasks

## TODO

### Completed foundation

- [x] Create a dedicated ticket for cross-framework adapter field preservation and schema promotion work
- [x] Build a source-backed Pi/Codex/Claude field matrix and store it in the new ticket
- [x] Add `tool_calls[].output.exit_code` to `pkg/minitrace/schema.go`
- [x] Populate Codex `exit_code` in both session-jsonl-v1 and exec-jsonl-v1 conversion paths
- [x] Add `tool_calls[].input.justification` to `pkg/minitrace/schema.go`
- [x] Populate Codex `justification` as a first-class input field when present
- [x] Add regression tests for the new Codex fields
- [x] Update schema and adapter docs for `exit_code` and `justification`

### Detailed metadata mapping docs

- [x] Add a dedicated doc describing framework-specific metadata storage conventions (`framework_config`, `turn.framework_metadata`, `tool_call.framework_metadata`)
- [x] Document Codex raw-to-minitrace metadata mappings
- [x] Document Claude Code raw-to-minitrace metadata mappings
- [x] Document Pi raw-to-minitrace metadata mappings

### Codex metadata preservation

- [x] Preserve Codex session/runtime metadata in `operational_context.framework_config`: `approval_policy`, detailed `sandbox_policy`, `collaboration_mode` detail, `truncation_policy`, `rate_limits`, session `source`
- [x] Preserve Codex turn metadata in `turns[].framework_metadata`: `turn_id`, `phase`, `memory_citation`
- [x] Preserve Codex tool metadata in `tool_calls[].framework_metadata`: command `source`, `parsed_cmd`, `stdout`, `stderr`, raw `status`, `turn_id` when available
- [x] Add/extend regression tests covering Codex metadata preservation

### Claude Code metadata preservation

- [x] Preserve Claude session-level metadata in `operational_context.framework_config`: `entrypoint` when available
- [x] Preserve Claude turn metadata in `turns[].framework_metadata`: `entrypoint`, `stop_reason`, `stop_sequence`, `slug`, `parentUuid`, `isSidechain`, cache bucket detail
- [x] Preserve Claude tool metadata in `tool_calls[].framework_metadata`: `caller` plus skipped tool-result record metadata needed to avoid loss
- [x] Add/extend regression tests covering Claude metadata preservation

### Pi metadata preservation

- [x] Preserve Pi assistant turn metadata in `turns[].framework_metadata`: `stopReason`, `errorMessage`
- [x] Preserve Pi tool-result metadata in `tool_calls[].framework_metadata`: `details.diff` and related edit-result detail
- [x] Add/extend regression tests covering Pi metadata preservation

### Final doc/bookkeeping pass

- [x] Update `pkg/doc/minitrace-schema.md` for framework metadata conventions and newly preserved fields
- [x] Update `pkg/doc/adapter-reference.md` with preserved metadata details per adapter
- [x] Add a detailed public mapping doc for preserved framework metadata
- [x] Add non-embedded example queries demonstrating `exit_code`, `justification`, and framework metadata analysis
- [x] Smoke-test the new example queries against real converted Codex/Claude/Pi transcripts
- [x] Update ticket diary/changelog/tasks after each implementation slice
