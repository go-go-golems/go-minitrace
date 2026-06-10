# Tasks

## Completed setup and baseline

- [x] Survey current goja/xgoja APIs and transcript import commands
- [x] Inspect latest Pi, Codex, and Claude Code session formats
- [x] Prototype preview/survey script in ticket scripts folder
- [x] Write intern-oriented design and implementation guide
- [x] Validate ticket and upload bundle to reMarkable

## Phase 1: Codex latest-format semantic support

Goal: make recent Codex `response_item` / `event_msg` sessions load with accurate tool semantics, especially delegated agents and media/image operations.

- [x] Add minimized Codex fixtures for latest tool forms
  - [x] `spawn_agent` function call with task/prompt arguments
  - [x] `wait_agent` function call/output with child-agent outcome
  - [x] `view_image` function call with path/media argument
  - [x] `apply_patch` function call/output
  - [x] `write_stdin` function call/output
  - [x] `custom_tool_call` payload with namespace/status/input
- [x] Update Codex operation classification
  - [x] classify `spawn_agent` and `wait_agent` as `DELEGATE`
  - [x] classify `view_image` as `READ`
  - [x] classify `write_stdin` as `EXECUTE`
  - [x] keep `apply_patch` as `MODIFY`
- [x] Promote Codex spawned-agent metadata
  - [x] populate `ToolCall.SpawnedAgent` for `spawn_agent`
  - [x] preserve agent/task identifiers in framework metadata
  - [x] merge `wait_agent` output into spawned-agent outcome where possible
- [x] Preserve Codex image signals
  - [x] set image/media metadata for `view_image`
  - [x] make importer preview report `hasImageSignals=true`
- [x] Preserve custom tool-call metadata
  - [x] normalize `custom_tool_call` into `ToolCall`
  - [x] keep namespace/status/input/output in framework metadata
- [x] Run targeted tests and commit Phase 1

## Phase 2: Claude Code latest-format metadata and attachments

Goal: preserve recent Claude Code non-message records and make subagent/session metadata visible in normalized sessions and previews.

- [ ] Add minimized Claude Code fixtures
  - [ ] `attachment` records with file/image-like metadata
  - [ ] `mode` records
  - [ ] `permission-mode` records
  - [ ] `ai-title` records
  - [ ] subagent records carrying `agentId`, `parentUuid`, `sessionId`, `slug`, `isSidechain`
- [ ] Preserve Claude attachment records
  - [ ] create annotations or events for attachments
  - [ ] set image/media signal metadata when appropriate
  - [ ] avoid inlining large blobs by default
- [ ] Preserve Claude session mode records
  - [ ] store `mode` in `OperationalContext.FrameworkConfig`
  - [ ] store `permission-mode` in `OperationalContext.FrameworkConfig`
  - [ ] map permission mode to autonomy-level if a stable mapping is obvious
- [ ] Preserve Claude title records
  - [ ] use `ai-title` as a title candidate when present
  - [ ] keep raw title metadata for review
- [ ] Verify Claude subagent linking
  - [ ] ensure `ConvertSubagentLocator` keeps child metadata
  - [ ] ensure parent `Agent` tool calls receive `SubSessionID`
  - [ ] ensure preview reports nonzero subagent signals when normalized data contains them
- [ ] Run targeted tests and commit Phase 2

## Phase 3: First-class preview command

Goal: expose the new importer preview API without requiring users to write JavaScript.

- [ ] Design command shape
  - [ ] decide final command path (`go-minitrace preview session` or equivalent)
  - [ ] define flags: `--source-session`, `--source-dir`, `--framework`, `--latest`, `--sample-limit`, `--privacy`
  - [ ] define output formats supported by Glazed
- [ ] Implement preview command
  - [ ] call `minitracedb.LoadSessionFileAuto` / `minitracejs.ImportBuilder.Preview`
  - [ ] support one-file preview first
  - [ ] add directory/latest-N mode if straightforward
  - [ ] keep structural/snippet privacy defaults bounded
- [ ] Add command tests
  - [ ] unit-test preview object rendering if command harness supports it
  - [ ] smoke-test one Pi/Codex/Claude fixture
- [ ] Update JS/API docs and design guide
- [ ] Run targeted tests and commit Phase 3

## Phase 4: End-to-end validation and documentation refresh

Goal: prove the importer works on latest local sessions and refresh the ticket deliverables.

- [ ] Re-run latest session survey script
- [ ] Run preview/import on latest local Codex sessions
- [ ] Run preview/import on latest local Claude parent and subagent sessions
- [ ] Run preview/import on latest local Pi sessions as regression coverage
- [ ] Update design guide with actual implementation decisions and commands
- [ ] Update diary with validation outcomes and failures
- [ ] Run `docmgr doctor`
- [ ] Upload refreshed bundle to reMarkable
- [ ] Commit final ticket documentation refresh
