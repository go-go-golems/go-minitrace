---
type: changelog
title: Changelog
status: active
---

# Changelog

## 2026-04-07

### Fixed
- **Issue 1**: Tool result messages no longer appear as fake "user" turns. Added `continue` in Pi converter after `applyToolResult()`. Turns: 1391 → 654.
- **Issue 2**: Tool call summary row now shows file path (read/write/edit), query (web_search), or command (bash) instead of generic tool name.
- **Issue 3**: Assistant turn content rendered as Markdown via `react-markdown`. User turns remain plain text.

### Added
- Ticket GMT-001 created with design doc, diary, 19 investigation scripts, 6 related source files.
- Design doc catalogs all 7 issues with data flow, schema references, and fix sketches.

### Changed
- `pkg/adapters/pi/convert.go`: Skip turn creation for `role == "toolResult"` messages.
- `pkg/adapters/pi/convert_test.go`: Expected turn count 3 → 2.
- `web/src/components/TranscriptViewer/ToolCallRow.tsx`: Improved summary extraction chain.
- `web/src/components/TranscriptViewer/BlockBody.tsx`: Markdown rendering for assistant turns.
- `web/package.json`: Added `react-markdown` dependency.

### Fixed (continued)
- **Issue 4**: Thinking traces now surface in the web UI as collapsible 💭 blocks. 150/654 turns have thinking content.
- **Issue 5**: Edit tool calls render a line-based diff (red `-` / green `+`) from `edits[].oldText/newText` in the expanded view.
- **Issue 6**: Write tool calls render the full file content in a scrollable code block with auto-truncation.
- **Issue 7**: Model name shown as chip badge and token counts (`in/out/cache`) shown in the turn header bar.

### Changed (continued)
- `cmd/go-minitrace/cmds/serve/handlers_sessions.go`: Added `TurnUsageResponse`, extended `TurnResponse` with `Thinking`/`Model`/`Usage`, added `normalizeUsage()`.
- `web/src/types/session.ts`: Extended `Turn` interface with `thinking`, `model`, `usage`.
- `web/src/components/TranscriptViewer/BlockBody.tsx`: Added `TurnMetaChips`, `ThinkingBlock` components. Model/usage in header, thinking as collapsible.
- `web/src/components/TranscriptViewer/ToolCallRow.tsx`: Major refactor — extracted `ToolCallDetail`, `DiffView`, `ContentBlock`. Edit shows diffs, write shows content, bash shows command+output.
