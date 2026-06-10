# Tasks

## TODO

- [x] 1. Establish backend session schema primitives.
  - Add `Session.Events []Event` and `Session.Attachments []Attachment` to `pkg/minitrace/schema.go`.
  - Add builder helpers in `pkg/minitrace/builders.go` so adapters do not hand-roll default slices or timestamps.
  - Keep JSON fields optional with `omitempty` so existing archives remain readable.

- [ ] 2. Materialize explicit source events and attachments.
  - Extend the existing `events` table with explicit-event fields needed by the new `Event` struct.
  - Add an `attachments` table to `pkg/minitracedb/schema.go`.
  - Insert `session.Events` separately from already-derived turn/tool/annotation events.
  - Insert `session.Attachments` with bounded metadata and raw JSON.
  - Add tests in `pkg/minitracedb/materialize_test.go` and update schema tests.

- [ ] 3. Validate and document native JSON behavior.
  - Update `pkg/validate/json.go` for `events` and `attachments` shape checks.
  - Add validator tests for valid, empty, null, and malformed arrays.
  - Update query/API documentation that lists normalized tables and JSON arrays.

- [ ] 4. Map Pi non-message records onto first-class events.
  - Convert `session_info` to `Session.Title` and a `session_info` event.
  - Convert `custom` records to `custom.<customType>` events and preserve payloads in `FrameworkMetadata`/`RawJSON`.
  - Convert `compaction` records to `compaction` events with privacy-safe summaries.
  - Preserve existing `model_change` and `thinking_level_change` config behavior while adding lifecycle events.
  - Add minimized adapter tests in `pkg/adapters/pi/convert_test.go`.

- [ ] 5. Map Claude Code attachments and lifecycle records.
  - Convert `attachment` records to `Session.Attachments` instead of annotation-only preservation.
  - Add timeline events for `mode`, `permission-mode`, `ai-title`, and `attachment` records.
  - Keep existing framework config/title behavior intact.
  - Add or update tests in `pkg/adapters/claudecode/convert_test.go`.

- [ ] 6. Map Codex image/subagent/source lifecycle details.
  - Convert `view_image` calls to image attachments linked to tool calls.
  - Add events for `spawn_agent`, `wait_agent`, token/rate-limit signals, and source lifecycle records where appropriate.
  - Keep existing normalized tool-call semantics intact.
  - Add or update tests in `pkg/adapters/codex/convert_test.go`.

- [ ] 7. Expose the new primitives in preview/import surfaces.
  - Extend `pkg/minitracejs` preview summaries with event and attachment counts and sample rows.
  - Update `go-minitrace preview session` output and tests.
  - Keep default preview privacy-safe: structural summaries first, snippets only when requested.

- [ ] 8. Update long-form docs and examples.
  - Update `pkg/doc/js-api-reference.md`, `pkg/doc/adapter-reference.md`, and query docs for events/attachments.
  - Add query examples for timeline events and image/file attachments.
  - Ensure the ticket design doc reflects actual implementation decisions.

- [ ] 9. Validate, doctor, and deliver.
  - Run targeted tests after each phase and a broader test set before handoff.
  - Run `docmgr --root "$(pwd)/ttmp" doctor --ticket session-events-attachments --stale-after 30`.
  - Upload the ticket bundle to reMarkable under `/ai/2026/06/10/session-events-attachments`.

## DONE

