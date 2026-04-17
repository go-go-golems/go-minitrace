# Tasks

## TODO

### Pi adapter follow-up

- [x] Fix: read isError from message-level toolResult in pi convert.go line 175
- [x] Add test case with isError=true message-level toolResult fixture
- [x] Verify fix against jellyfin session using ticket script `scripts/01-verify-real-session.go`
- [ ] Python adapter: add message-level toolResult handling (~15 lines)
- [ ] Python adapter: suppress spurious user turns from toolResult messages
- [ ] Capture details.diff from edit tool results

### Validated schema-gap follow-up

- [ ] Add `tool_calls[].output.exit_code` to `pkg/minitrace/schema.go`
- [ ] Populate Codex `exit_code` consistently in both session-jsonl-v1 and exec-jsonl-v1 conversion paths
- [ ] Add `tool_calls[].input.justification` to `pkg/minitrace/schema.go`
- [ ] Populate Codex `justification` from `function_call.arguments` as a first-class field, not only in `framework_metadata`
- [ ] Preserve richer Codex metadata in `framework_config` / `framework_metadata`: approval_policy, sandbox_policy, collaboration_mode, truncation_policy, rate_limits, turn_id, phase, memory_citation, command source, parsed_cmd, stdout, stderr
- [ ] Preserve richer Claude Code metadata in `framework_config` / `framework_metadata`: caller, entrypoint, stop_reason, stop_sequence, slug, parentUuid, isSidechain, cache_creation bucket detail
- [ ] Add regression tests for newly promoted/preserved Codex fields
- [ ] Add regression tests for newly preserved Claude Code fields
- [ ] Update `pkg/doc/minitrace-schema.md` and `pkg/doc/adapter-reference.md` for any new schema fields and metadata conventions
- [ ] After the metadata-preservation pass, decide whether `stdout/stderr`, `stop_reason`, or normalized sandbox policy should be promoted into first-class schema fields
