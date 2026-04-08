---
type: tasklist
title: Tasks
status: active
---

# Tasks

## Completed

- [x] Fix tool result turns rendered as user messages (Issue 1) — `convert.go` + test
- [x] Fix tool call summary showing "read read" instead of file path (Issue 2) — `ToolCallRow.tsx`
- [x] Add markdown rendering for assistant turns (Issue 3) — `BlockBody.tsx` + `react-markdown`
- [x] Rebuild frontend and embed in go-minitrace binary
- [x] Re-convert session and re-sync annotations
- [x] Create ticket GMT-001 with design doc, diary, and investigation scripts
- [x] Surface thinking traces in web UI (Issue 4)
  - [x] Add `Thinking`, `Model`, `Usage` to `TurnResponse` in `handlers_sessions.go`
  - [x] Wire fields in `normalizeTurn()` + `normalizeUsage()` helper
  - [x] Add to frontend `Turn` type in `session.ts`
  - [x] Render thinking as collapsible `ThinkingBlock` with 💭 icon
  - [x] Show model as chip badge on each turn via `TurnMetaChips`
- [x] Show model and usage info per turn (Issue 7)
  - [x] Token counts (`in:8.4k out:459 cache:704`) in turn header
- [x] Render edit tool call diffs in expanded view (Issue 5)
  - [x] `DiffView` component: red/green line diff from `edits[].oldText/newText`
  - [x] Multi-edit support with numbered labels
- [x] Render write tool call content in expanded view (Issue 6)
  - [x] `ContentBlock` component with auto-truncation at 2000 chars + "Show all"

## No remaining issues

All 7 issues identified in the design doc have been fixed.
