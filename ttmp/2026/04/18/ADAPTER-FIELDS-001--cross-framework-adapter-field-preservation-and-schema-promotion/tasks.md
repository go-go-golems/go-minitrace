# Tasks

## TODO

- [x] Create a dedicated ticket for cross-framework adapter field preservation and schema promotion work
- [x] Build a source-backed Pi/Codex/Claude field matrix and store it in the new ticket
- [ ] Add `tool_calls[].output.exit_code` to `pkg/minitrace/schema.go`
- [ ] Populate Codex `exit_code` in both session-jsonl-v1 and exec-jsonl-v1 conversion paths
- [ ] Add `tool_calls[].input.justification` to `pkg/minitrace/schema.go`
- [ ] Populate Codex `justification` as a first-class input field when present
- [ ] Add regression tests for the new Codex fields
- [ ] Update schema and adapter docs for `exit_code` and `justification`
- [ ] Preserve metadata-only follow-up fields for Codex, Claude Code, and Pi in a later slice
