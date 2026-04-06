---
Title: Migrating go-minitrace web virtualization to TanStack React Virtual
Ticket: TRANSCRIPT-PERF-001
Status: active
Topics:
    - performance
    - frontend
    - react
    - virtualization
    - transcript-analysis
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: web/package.json
      Note: New frontend dependency will be added here when the migration begins.
    - Path: web/src/App.tsx
      Note: Route-level error-boundary containment belongs here after the migration stabilizes.
    - Path: web/src/api/minitrace.ts
      Note: Data fetching does not need major changes, but this file explains the route payloads used by Session Browser and Transcript Viewer.
    - Path: web/src/components/SessionBrowser/SessionBrowser.tsx
      Note: Current fixed-height virtualized table using the custom hook; best low-risk pilot surface for TanStack Virtual.
    - Path: web/src/components/TranscriptViewer/BlockBody.tsx
      Note: Expensive dynamic-height subtree inside each transcript block.
    - Path: web/src/components/TranscriptViewer/BlockCard.tsx
      Note: Block shell and expansion boundary; will need a small measurement callback seam.
    - Path: web/src/components/TranscriptViewer/ToolCallRow.tsx
      Note: Nested dynamic-height rows that amplify transcript measurement complexity.
    - Path: web/src/components/TranscriptViewer/TranscriptViewer.tsx
      Note: Main transcript virtualization integration point and owner of focus/URL/expansion state.
    - Path: web/src/components/TranscriptViewer/stories/BlockCard.stories.tsx
      Note: Storybook compatibility should be re-validated after any BlockCard API change.
    - Path: web/src/components/shared/useVirtualList.ts
      Note: Current custom measured-window hook to be retired after migration.
    - Path: web/src/pages/SessionBrowserPage.tsx
      Note: Route shell for sessions list; mostly unchanged except optional error-boundary wrapping.
    - Path: web/src/pages/TranscriptViewerPage.tsx
      Note: Route shell for transcript loading and the best location for transcript-specific crash containment.
    - Path: ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/reference/02-maximum-update-depth-regression-analysis-and-error-boundary-placement.md
      Note: Companion note explaining the current regression and why this migration is now attractive.
    - Path: ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs
      Note: Existing repeatable measurement script to rerun before and after the migration.
    - Path: ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/01-baseline-measurements.json
      Note: Baseline evidence for transcript hot-path cost before the migration.
ExternalSources:
    - https://tanstack.com/virtual/latest/docs/api/virtualizer
    - https://tanstack.com/virtual/latest/docs/api/virtual-item
    - https://tanstack.com/virtual/latest/docs/framework/react/examples
Summary: Detailed intern-oriented migration guide for replacing the custom virtualization hook with TanStack React Virtual in Session Browser and Transcript Viewer.
LastUpdated: 2026-04-06T23:10:00-04:00
WhatFor: Onboard a new engineer to the current virtualization architecture, explain why the custom hook has become risky, and provide a concrete phased migration plan to TanStack React Virtual.
WhenToUse: Use when implementing or reviewing the migration away from web/src/components/shared/useVirtualList.ts, especially for transcript rendering and Session Browser scalability.
---

# Migrating go-minitrace web virtualization to TanStack React Virtual

## Executive summary

This document explains how to replace the current custom virtualization hook in the `go-minitrace` web UI with `@tanstack/react-virtual`. It is written for a new intern who has never seen this codebase before and needs enough context to make careful changes without breaking transcript navigation, annotation workflows, or Session Browser usability.

The short version is this:

- the app already gained major performance wins from virtualization,
- but the current shared hook (`web/src/components/shared/useVirtualList.ts`) has become a risk surface,
- a real regression (`Maximum update depth exceeded`) showed that we are now paying the complexity cost of custom virtualization code,
- TanStack Virtual gives us a battle-tested measurement and scroll model, which should reduce lifecycle bugs and let us focus on app-specific behavior instead of list math.

This is not a “swap one hook for another” change. The migration touches core assumptions about:

- where list state lives,
- how DOM measurement is triggered,
- how focus and scrolling interact with URL state,
- how fixed-height and dynamic-height lists should be treated differently,
- where error boundaries should contain failures.

If you only remember one design rule from this document, remember this:

> Do not rebuild a second generic virtualization abstraction on top of TanStack on day one. Migrate the two call sites deliberately, keep the code close to the route that owns the behavior, and only extract helpers after the new behavior is stable.

---

## 1. Read this first: what the system is

`go-minitrace` is a Go backend plus a React web frontend for browsing and analyzing session transcripts. The web app has three main routes:

- `/sessions` — Session Browser, a list of all sessions.
- `/sessions/:sessionId` — Transcript Viewer, a detailed page for one session.
- `/query` — Query Editor for DuckDB queries.

The performance work in this ticket focused primarily on the transcript route because it renders the deepest and heaviest React tree.

### Why the transcript route is special

The transcript page does not render a flat list of strings. It renders a hierarchy:

```text
TranscriptViewerPage
  -> TranscriptViewer
    -> BlockCard x N blocks
      -> turn rows x many turns per block
        -> ToolCallRow x many tool calls per turn
```

Each level adds UI work:

- Material UI containers, chips, buttons, tooltips, collapse regions
- annotation affordances
- focus styling
- URL-driven deep linking
- expansion state
- tool-call detail subtrees

That is why transcript rendering behaves differently from the Session Browser and Query Editor.

### The important backend/frontend boundary

The migration described here is almost entirely a **frontend** migration. The backend already serves the shapes the frontend needs:

- `GET /api/sessions` for Session Browser rows
- `GET /api/sessions/{id}/summary` for transcript header/metadata
- `GET /api/sessions/{id}/blocks` for block-oriented transcript content

Relevant files:

- `web/src/api/minitrace.ts`
- `web/src/pages/SessionBrowserPage.tsx`
- `web/src/pages/TranscriptViewerPage.tsx`
- `cmd/go-minitrace/cmds/serve/handlers_sessions.go`
- `cmd/go-minitrace/cmds/serve/blocks.go`

The backend does **not** need a new virtualization-aware API for this migration.

---

## 2. Why we are doing this migration now

The app already implemented a custom virtualization solution in `web/src/components/shared/useVirtualList.ts`. That custom hook helped us reach strong performance wins:

- transcript initial load dropped significantly,
- transcript tab switching became much faster,
- Session Browser now mounts far fewer rows,
- DOM node counts are much lower on large routes.

However, the shared hook also introduced a high-severity regression:

- `Maximum update depth exceeded`
- triggered by a transcript deep link with focused tool-call and inline composer state
- initially traced to measurement/ref behavior inside `useVirtualList.ts`

Companion analysis:

- `ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/reference/02-maximum-update-depth-regression-analysis-and-error-boundary-placement.md`

### Why this regression matters strategically

This regression is important not only because it crashes the page, but because it tells us something about the architecture:

- we have crossed from “simple list rendering” into “complex virtualized dynamic-height UI,”
- the current hook now owns tricky lifecycle behavior,
- the same hook is shared by more than one route,
- future transcript work will keep stressing measurement, focus, and scrolling.

That is exactly the kind of moment where adopting a focused external library becomes reasonable.

### Evidence that transcript rendering is worth the effort

The existing ticket already captured baseline and follow-up measurements under:

- `sources/01-baseline-measurements.json`
- `sources/03-step-3-unmount-on-exit-measurements.json`
- `sources/04-step-7-transcript-virtualization-measurements.json`
- `sources/05-step-8-session-browser-virtualization-measurements.json`
- `sources/06-step-9-summary-and-blocks-split-measurements.json`

Key baseline findings from the current ticket:

- Session Browser mean load: about `612 ms`
- Transcript initial load: about `3958 ms`
- Transcript DOM nodes before virtualization work: about `15,479`

So the migration is justified by both:

- correctness risk in the custom hook, and
- continued importance of transcript rendering performance.

---

## 3. Current architecture you must understand before changing anything

This section is the orientation pass for a new engineer.

## 3.1 Route and data-flow map

```text
Browser route
  -> React page component
    -> RTK Query hook
      -> Go /api endpoint
        -> normalized JSON response
          -> React route tree
```

Concrete routes and files:

- App routing: `web/src/App.tsx`
- Sessions page shell: `web/src/pages/SessionBrowserPage.tsx`
- Transcript page shell: `web/src/pages/TranscriptViewerPage.tsx`
- API hooks: `web/src/api/minitrace.ts`

## 3.2 Session Browser mental model

The Session Browser is conceptually simple:

1. fetch all session summaries,
2. filter in memory,
3. render the filtered results in a MUI table,
4. let the user click a row or query button.

Current implementation:

- `web/src/components/SessionBrowser/SessionBrowser.tsx`

Important characteristics:

- row height is effectively fixed or near-fixed,
- interaction is simple,
- filtering is synchronous in memory,
- virtualization here is mostly about large row counts, not complicated nested measurement.

This is the easiest place to pilot TanStack Virtual.

## 3.3 Transcript Viewer mental model

The Transcript Viewer is more complicated because it combines:

- virtualized block list rendering,
- URL-driven selected tab state,
- URL-driven focused target state,
- URL-driven inline annotation composer state,
- parent-owned block expansion state,
- nested turn rendering,
- nested tool-call rendering,
- inner anchor scrolling after the outer row becomes visible.

Current implementation:

- `web/src/components/TranscriptViewer/TranscriptViewer.tsx`
- `web/src/components/TranscriptViewer/BlockCard.tsx`
- `web/src/components/TranscriptViewer/BlockBody.tsx`
- `web/src/components/TranscriptViewer/ToolCallRow.tsx`
- `web/src/components/TranscriptViewer/AnnotationPanel.tsx`
- `web/src/components/TranscriptViewer/AnnotationComposer.tsx`

### Transcript rendering layers

```text
TranscriptViewer
  owns:
    - search params to view/focus/compose state
    - expandedBlocks state
    - scroll container ref
    - virtual row calculation

BlockCard
  owns:
    - block shell
    - block header
    - block collapse boundary
    - local showAllTools state

BlockBody
  owns:
    - turn rows
    - tool-call row mapping
    - turn annotation chips
    - tool-call annotation chips

ToolCallRow
  owns:
    - tool-call summary row
    - expandable detail panel
```

### Why this matters for virtualization

The transcript list is **dynamic height**. Row height changes when:

- a block expands or collapses,
- a focused tool call forces a block open,
- `showAllTools` changes from `false` to `true`,
- tool-call rows expand or collapse,
- annotation UI appears or disappears,
- text wrapping changes with layout width.

That means transcript virtualization is the hard case.

## 3.4 The current custom virtualization hook

Current file:

- `web/src/components/shared/useVirtualList.ts`

Current responsibilities:

- read `scrollTop` and container height,
- maintain measured row heights in React state,
- compute cumulative row starts and total size,
- compute visible range with overscan,
- expose virtual items,
- expose top and bottom spacer heights,
- expose `measureElement(index)` ref factory,
- expose `scrollToIndex(...)`.

This hook is elegant in size, but it now owns complicated behavior:

- callback ref identity,
- ResizeObserver lifecycle,
- dynamic height state updates,
- row start recomputation,
- scroll math,
- multiple caller requirements.

That is the piece we want to retire.

---

## 4. TanStack Virtual primer for a new intern

This section explains the library itself in plain language.

## 4.1 What TanStack Virtual gives us

`@tanstack/react-virtual` provides a `useVirtualizer(...)` hook that returns a **virtualizer instance**. That instance owns the bookkeeping for:

- total list size,
- visible virtual items,
- item measurements,
- scroll element tracking,
- scroll-to-index behavior,
- overscan.

You still render the rows yourself. The library is not a UI kit. It is a virtualization engine.

## 4.2 Core concepts

### Virtualizer instance

You create it in a component with options like:

- `count`
- `getScrollElement`
- `estimateSize`
- `overscan`
- optional `getItemKey`

### Virtual items

The instance returns `getVirtualItems()`, each with data like:

- `index`
- `key`
- `start`
- `end`
- `size`

These items tell you which rows to render and where they belong in the scroll coordinate space.

### Measurement

For dynamic-height rows, you attach `virtualizer.measureElement` to the row element you want the library to measure.

The important difference from our current custom hook is:

- measurement lifecycle is handled by the library,
- we no longer need our own generic callback-ref factory that writes React state during ref attachment.

### Total size

`virtualizer.getTotalSize()` gives the total scrollable height.

### Scrolling to a row

`virtualizer.scrollToIndex(index, { align: "center" })` scrolls the outer virtual list.

This is important in the transcript route, where we must first make the block row visible before trying to scroll to a turn or tool-call anchor inside that row.

## 4.3 The API references to keep open while coding

Recommended references:

- Virtualizer API: `https://tanstack.com/virtual/latest/docs/api/virtualizer`
- VirtualItem API: `https://tanstack.com/virtual/latest/docs/api/virtual-item`
- React examples: `https://tanstack.com/virtual/latest/docs/framework/react/examples`

You do not need to memorize every option. For this migration, the most relevant API surface is small.

## 4.4 Old vs new mental model

| Concern | Current custom hook | TanStack Virtual equivalent |
|---|---|---|
| item count | `count` | `count` |
| scroll container | `scrollContainerRef` | `getScrollElement: () => scrollRef.current` |
| estimated size | `estimateSize(index)` | `estimateSize: (index) => ...` |
| overscan | `overscan` | `overscan` |
| visible items | `virtualItems` | `virtualizer.getVirtualItems()` |
| total size | `totalSize` | `virtualizer.getTotalSize()` |
| row measurement | `measureElement(index)` | `ref={virtualizer.measureElement}` on the row element |
| scroll to row | `scrollToIndex(index, behavior, align)` | `virtualizer.scrollToIndex(index, { align })` |
| stable row keys | implicit / caller-managed | `getItemKey: (index) => ...` |

The main simplification is that we stop owning the measurement engine ourselves.

---

## 5. Design goals and non-goals

## 5.1 Goals

1. Replace the shared custom virtualization hook with TanStack Virtual.
2. Keep the current performance wins for transcript and Session Browser.
3. Reduce measurement/ref lifecycle bugs.
4. Preserve transcript deep linking and focus behavior.
5. Preserve annotation workflows.
6. Keep Session Browser table semantics if practical.
7. Avoid introducing a second homemade abstraction layer too early.
8. Make the migration easy to review in small commits.

## 5.2 Non-goals

1. Do not redesign transcript data shapes.
2. Do not refactor all routes to one universal list abstraction.
3. Do not change Query Editor virtualization in this migration.
4. Do not rewrite the transcript UI structure from scratch.
5. Do not add speculative optimizations that are unrelated to the current bug/perf goals.

---

## 6. Proposed migration design

## 6.1 Key design decision: do not replace one generic hook with another generic hook

This is the most important design decision in the document.

It may be tempting to write something like:

- `useVirtualList2.ts`
- or `useTanStackVirtualList.ts`
- and hide TanStack behind a new local abstraction.

Do **not** do that initially.

Why:

- the current regression is partly a result of a shared abstraction trying to serve different surfaces,
- Session Browser and Transcript Viewer have different needs,
- the best way to learn the library is to use it directly at the call sites,
- wrapping too early risks recreating the same bug class under a different name.

Recommended approach:

- use `useVirtualizer(...)` directly in `SessionBrowser.tsx`,
- use `useVirtualizer(...)` directly in `TranscriptViewer.tsx`,
- only extract tiny local helpers if there is obvious duplication after stabilization.

## 6.2 Treat Session Browser and Transcript Viewer as different migration shapes

### Session Browser: fixed-height pilot

Session Browser rows are effectively fixed height. That means we can use a simpler virtualizer setup:

- `estimateSize: () => 76`
- `getItemKey: (index) => filtered[index].id`
- likely no `measureElement` at all
- preserve the current top/bottom spacer pattern in `<TableBody>`

This is a low-risk first migration because:

- row interaction is simple,
- dynamic measurement is unnecessary,
- failures are easier to debug,
- it proves the dependency integration without touching the hardest route first.

### Transcript Viewer: dynamic-height main migration

Transcript block rows are dynamic height and need measurement.

Recommended shape:

- each virtual row corresponds to one `SessionBlock`,
- the outer scroll container remains the transcript content pane,
- the virtualized row element is a wrapper `<Box>` or `<div>` around `BlockCard`,
- that wrapper gets the measurement ref from TanStack,
- block expansion stays parent-owned in `TranscriptViewer`,
- forced expansion for focused targets remains unchanged conceptually,
- inner DOM anchor scrolling still happens after outer row scrolling.

## 6.3 Keep parent-owned durable state in the transcript route

Virtualization means rows unmount and remount. Any state that must survive that must be owned above the row.

This rule already showed up in the current custom implementation and remains true after migration.

State that should remain in `TranscriptViewer`:

- current selected tab (`transcript` vs `annotations`) via URL state
- focused target derived from URL state
- expanded block map
- search-param patching helpers
- focused block index calculation

State that may remain in children if temporary and local:

- `ToolCallRow` local detail expansion, if losing it on virtualization is acceptable
- `BlockCard` local `showAllTools` only if it can be re-derived or safely reset

### Important note about `showAllTools`

`showAllTools` is currently local to `BlockCard`. That may still be acceptable, but the intern should review whether it must survive row unmount/remount for good UX. If we see surprising resets after migration, this state should move upward.

## 6.4 Explicit two-phase focus flow for transcript deep links

The transcript route already needs a two-step navigation flow:

1. scroll the **virtual row** into view,
2. scroll the **inner anchor** (session top / turn / tool call) into view.

That flow remains correct after migration.

Recommended sequence:

```text
URL params change
  -> derive focused target
  -> compute containing block index
  -> force block expanded
  -> virtualizer.scrollToIndex(blockIndex, align=center)
  -> wait for row mount / next frame
  -> query DOM anchor inside block
  -> anchor.scrollIntoView(block=center)
```

This is critical because TanStack can only scroll the outer list; it does not know anything about turn IDs or tool-call IDs inside `BlockCard`.

## 6.5 Add a small explicit height-invalidated seam for transcript blocks

Dynamic-height lists often work better if row components can tell the parent, “my height probably changed.”

Recommended optional addition:

- add `onHeightMayHaveChanged?: () => void` to `BlockCard`
- call it when:
  - block expansion toggles,
  - `showAllTools` flips to `true`,
  - a forced expansion path becomes active,
  - collapse transition finishes if needed

Then `TranscriptViewer` can call:

```ts
requestAnimationFrame(() => virtualizer.measure())
```

or equivalent after those state transitions.

This is not always mandatory, but it is a good safety valve when working with animated MUI `Collapse` and dynamic nested content.

## 6.6 Error boundaries are companion work, not the migration itself

The migration should be paired with route-level error boundaries because transcript and Session Browser remain high-risk interactive surfaces.

Recommended locations:

- `web/src/App.tsx`
- `web/src/pages/TranscriptViewerPage.tsx`
- optionally `web/src/pages/SessionBrowserPage.tsx`

Do not put boundaries per row or per block.

---

## 7. File-by-file implementation guide

This section is the “what exactly do I edit?” checklist.

## 7.1 `web/package.json`

Add the dependency:

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web
npm install @tanstack/react-virtual
```

Expected change:

- `dependencies` gains `@tanstack/react-virtual`

No other package should be added unless the migration explicitly includes error-boundary helper libraries.

## 7.2 `web/src/components/SessionBrowser/SessionBrowser.tsx`

This is the recommended pilot migration.

Current behavior:

- uses `useVirtualList(...)`
- renders top and bottom spacer rows
- passes `ref={measureElement(item.index)}` to `TableRow`

Target behavior:

- use `useVirtualizer(...)`
- keep top and bottom spacer rows
- keep MUI table structure intact
- do not use dynamic measurement unless needed

Recommended sketch:

```tsx
const rowVirtualizer = useVirtualizer({
  count: filtered.length,
  getScrollElement: () => scrollContainerRef.current,
  estimateSize: () => 76,
  overscan: 8,
  getItemKey: (index) => filtered[index]?.id ?? index,
});

const virtualRows = rowVirtualizer.getVirtualItems();
const topSpacerHeight = virtualRows[0]?.start ?? 0;
const bottomSpacerHeight = rowVirtualizer.getTotalSize() - (virtualRows[virtualRows.length - 1]?.end ?? 0);
```

Notes for the intern:

- Because rows are stable height, start with no measurement ref.
- If the browser renders rows slightly taller than `76px`, accept a small mismatch first; only add measurement if a real visual issue appears.
- This route is the easiest place to get comfortable with the library.

## 7.3 `web/src/components/TranscriptViewer/TranscriptViewer.tsx`

This is the main migration file.

Current responsibilities already in this file:

- read `useSearchParams`
- derive `view`, focused target, and draft target
- compute `focusedBlockNum`
- own `expandedBlocks`
- own `scrollContainerRef`
- use the custom virtualization hook
- call `scrollToIndex(...)` before inner anchor scrolling

Target behavior:

- replace `useVirtualList(...)` with `useVirtualizer(...)`
- keep URL and expansion logic local to this file
- render virtual items from TanStack
- attach `ref={virtualizer.measureElement}` to the row wrapper element
- preserve focused target scrolling behavior

Recommended transcript row structure:

```tsx
<Box ref={scrollContainerRef} sx={{ overflow: "auto", flex: 1 }}>
  <Box sx={{ height: `${rowVirtualizer.getTotalSize()}px`, position: "relative" }}>
    {rowVirtualizer.getVirtualItems().map((virtualRow) => {
      const block = session.blocks[virtualRow.index];
      return (
        <Box
          key={virtualRow.key}
          data-index={virtualRow.index}
          ref={rowVirtualizer.measureElement}
          sx={{
            position: "absolute",
            top: 0,
            left: 0,
            width: "100%",
            transform: `translateY(${virtualRow.start}px)`,
          }}
        >
          <BlockCard ... />
        </Box>
      );
    })}
  </Box>
</Box>
```

### Why use a wrapper around `BlockCard`

Use a wrapper for measurement rather than measuring `Paper` or inner content directly because:

- it gives us one consistent row element per block,
- it avoids leaking virtualization details into `BlockCard` styling,
- it keeps positioning concerns out of the card component itself.

## 7.4 `web/src/components/TranscriptViewer/BlockCard.tsx`

This file likely needs only small changes.

Potential additions:

- `onHeightMayHaveChanged?: () => void`
- optional hooks on expand/collapse transitions
- optional callback when `showAllTools` changes

Example idea:

```tsx
function handleToggle() {
  if (isControlled) {
    onToggleExpanded?.();
  } else {
    setInternalExpanded((current) => !current);
  }
  onHeightMayHaveChanged?.();
}
```

If MUI `Collapse` timing causes stale height measurements, consider using `onEntered` and `onExited` to notify the parent more accurately.

## 7.5 `web/src/components/TranscriptViewer/BlockBody.tsx`

Likely minimal changes.

Things to watch:

- `showAllTools` can expand row height dramatically.
- if that state change is not visible to the virtualizer quickly enough, the focused scroll position can drift.

If needed, bubble an event upward when `onShowAllTools()` runs.

## 7.6 `web/src/components/TranscriptViewer/ToolCallRow.tsx`

Probably no direct TanStack changes are needed here because we virtualize at the block level, not the tool-call level.

Still, this file matters because it is a major source of dynamic block height. If layout or collapse behavior becomes noisy during migration, inspect this file first.

## 7.7 `web/src/components/shared/useVirtualList.ts`

Migration plan for this file:

- keep it during intermediate commits while one call site is still using it,
- delete it only after both Session Browser and Transcript Viewer are migrated,
- remove its exports/imports afterwards.

Do not delete it too early or you will make review and bisecting harder.

## 7.8 `web/src/App.tsx` and `web/src/pages/TranscriptViewerPage.tsx`

These are not primary virtualization files, but they are good follow-up files for:

- app-shell error boundary
- transcript-specific boundary/fallback

This should be a separate commit after the migration is stable enough to test meaningfully.

---

## 8. Recommended implementation phases

This is the migration plan I would hand to an intern.

## Phase 0 — Preparation and dependency install

### Goal

Introduce the dependency and prepare the work tree for a controlled migration.

### Steps

1. Create a branch.
2. Install `@tanstack/react-virtual`.
3. Read these files in order:
   - `web/src/components/shared/useVirtualList.ts`
   - `web/src/components/SessionBrowser/SessionBrowser.tsx`
   - `web/src/components/TranscriptViewer/TranscriptViewer.tsx`
   - `web/src/components/TranscriptViewer/BlockCard.tsx`
   - companion regression note in `reference/02-...md`
4. Run the existing build once before changing code:

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web
npm run build
```

### Deliverable

- dependency added
- baseline build passes

## Phase 1 — Migrate Session Browser first

### Why first

It is the easiest surface:

- fixed-height rows,
- flatter UI,
- fewer moving parts,
- little interaction with deep linking or dynamic expansion.

### Steps

1. Replace `useVirtualList` with `useVirtualizer` in `SessionBrowser.tsx`.
2. Preserve the table body with spacer rows.
3. Keep row click and query icon behavior unchanged.
4. Build and manually verify `/sessions`.

### Success criteria

- page loads,
- scrolling works,
- row click opens transcript,
- query button still works,
- no console errors,
- DOM node count remains low on large session sets.

## Phase 2 — Migrate Transcript Viewer second

### Why second

This is the high-value route and the difficult one.

### Steps

1. Replace `useVirtualList` with `useVirtualizer` in `TranscriptViewer.tsx`.
2. Keep `expandedBlocks` parent-owned.
3. Keep focused-target derivation parent-owned.
4. Use a measured wrapper element around `BlockCard`.
5. Preserve the two-phase scroll flow:
   - scroll outer row,
   - scroll inner anchor.
6. If needed, add `onHeightMayHaveChanged` plumbing from `BlockCard`.
7. Re-test the previously failing focused tool-call URL.

### Success criteria

- transcript route loads,
- deep-link focus works,
- inline composer still appears,
- no maximum update depth loop,
- no obvious jumpy scroll errors,
- initial load remains near the current optimized range.

## Phase 3 — Cleanup and removal of custom hook

### Steps

1. Remove `web/src/components/shared/useVirtualList.ts` once no call sites remain.
2. Remove any exports/imports that referenced it.
3. Run full frontend build.
4. Optionally run Storybook build if `BlockCard` props changed.

### Success criteria

- no remaining imports of `useVirtualList`
- frontend build passes
- stories still compile if touched

## Phase 4 — Error boundaries and validation pass

### Steps

1. Add route-level error boundaries.
2. Run the exact failing deep-link smoke.
3. Re-run the baseline measurement script.
4. Record results under `sources/` and update the ticket diary/changelog.

### Success criteria

- regression no longer reproduces,
- a route crash is contained if a future bug appears,
- measurements are no worse than the current optimized baseline.

---

## 9. Pseudocode and implementation sketches

These are intentionally not copy-paste final code. They are guidance.

## 9.1 Session Browser pseudocode

```tsx
import { useVirtualizer } from '@tanstack/react-virtual';

const scrollContainerRef = useRef<HTMLDivElement | null>(null);

const rowVirtualizer = useVirtualizer({
  count: filtered.length,
  getScrollElement: () => scrollContainerRef.current,
  estimateSize: () => 76,
  overscan: 8,
  getItemKey: (index) => filtered[index]?.id ?? index,
});

const virtualRows = rowVirtualizer.getVirtualItems();
const totalSize = rowVirtualizer.getTotalSize();
const topSpacerHeight = virtualRows[0]?.start ?? 0;
const bottomSpacerHeight = totalSize - (virtualRows[virtualRows.length - 1]?.end ?? 0);

return (
  <TableContainer ref={scrollContainerRef} ...>
    <Table ...>
      <TableHead>...</TableHead>
      <TableBody>
        {topSpacerHeight > 0 && <SpacerRow height={topSpacerHeight} />}
        {virtualRows.map((virtualRow) => {
          const session = filtered[virtualRow.index];
          return <SessionTableRow key={session.id} session={session} ... />;
        })}
        {bottomSpacerHeight > 0 && <SpacerRow height={bottomSpacerHeight} />}
      </TableBody>
    </Table>
  </TableContainer>
);
```

## 9.2 Transcript Viewer pseudocode

```tsx
import { useVirtualizer } from '@tanstack/react-virtual';

const scrollContainerRef = useRef<HTMLDivElement | null>(null);

const rowVirtualizer = useVirtualizer({
  count: session.blocks.length,
  getScrollElement: () => scrollContainerRef.current,
  estimateSize: (index) => estimateBlockSize(index),
  overscan: 4,
  getItemKey: (index) => session.blocks[index]?.block_num ?? index,
});

useEffect(() => {
  if (view !== 'transcript' || focusedBlockIndex == null) return;
  rowVirtualizer.scrollToIndex(focusedBlockIndex, { align: 'center' });
}, [focusedBlockIndex, rowVirtualizer, view]);

useEffect(() => {
  if (view !== 'transcript' || !urlFocusedTarget) return;
  const timer = window.setTimeout(() => {
    const el = findTranscriptAnchor(urlFocusedTarget);
    el?.scrollIntoView({ behavior: 'smooth', block: 'center' });
  }, 120);
  return () => window.clearTimeout(timer);
}, [urlFocusedTarget, view]);

return (
  <Box ref={scrollContainerRef} sx={{ flex: 1, overflow: 'auto' }}>
    <Box sx={{ height: `${rowVirtualizer.getTotalSize()}px`, position: 'relative' }}>
      {rowVirtualizer.getVirtualItems().map((virtualRow) => {
        const block = session.blocks[virtualRow.index];
        return (
          <Box
            key={virtualRow.key}
            data-index={virtualRow.index}
            ref={rowVirtualizer.measureElement}
            sx={{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              transform: `translateY(${virtualRow.start}px)`,
            }}
          >
            <BlockCard
              block={block}
              expanded={isBlockExpanded(block.block_num)}
              forceExpanded={focusedBlockNum === block.block_num}
              onToggleExpanded={() => handleToggleBlock(block.block_num)}
              onHeightMayHaveChanged={() => requestAnimationFrame(() => rowVirtualizer.measure())}
              ...
            />
          </Box>
        );
      })}
    </Box>
  </Box>
);
```

## 9.3 Transcript focus flow diagram

```text
URL search params
  -> derive focusType + focusId
  -> locate containing block index
  -> mark that block forceExpanded
  -> rowVirtualizer.scrollToIndex(blockIndex)
  -> virtual row mounts/measures
  -> DOM anchor exists inside BlockBody
  -> anchor.scrollIntoView()
  -> highlight target
```

## 9.4 Height invalidation pseudocode

```tsx
function TranscriptViewer() {
  const handleHeightMayHaveChanged = useCallback(() => {
    requestAnimationFrame(() => {
      rowVirtualizer.measure();
    });
  }, [rowVirtualizer]);

  ...
}
```

Use this sparingly. Do not call `measure()` in a tight loop or every render.

---

## 10. Risks, failure modes, and how to avoid them

## 10.1 Risk: recreating a generic abstraction too early

Symptom:

- a new `useSomethingVirtualized.ts` appears immediately,
- transcript and Session Browser are forced into the same abstraction,
- route-specific behavior is hidden again.

Prevention:

- use TanStack directly at the call sites first.

## 10.2 Risk: transcript row local state gets lost on unmount/remount

Symptom:

- users scroll away and back,
- some row-local UI state silently resets.

Prevention:

- keep durable state in `TranscriptViewer`,
- audit whether `showAllTools` must remain local.

## 10.3 Risk: dynamic-height rows drift during MUI Collapse animation

Symptom:

- row positions jump,
- scroll target lands slightly wrong,
- focused item is almost but not quite centered.

Prevention:

- use `measureElement` on a stable wrapper,
- optionally trigger a post-transition `virtualizer.measure()`.

## 10.4 Risk: inner anchors are scrolled before the row is mounted

Symptom:

- `document.querySelector(...)` returns `null`,
- deep links work inconsistently,
- focus only works after manual scrolling.

Prevention:

- keep the current two-phase flow,
- outer row first, inner anchor second.

## 10.5 Risk: table virtualization in Session Browser breaks semantics

Symptom:

- sticky headers stop behaving,
- row alignment looks wrong,
- keyboard or pointer behavior becomes odd.

Prevention:

- keep the current spacer-row strategy for Session Browser,
- avoid converting the table to an absolutely positioned grid unless required.

## 10.6 Risk: premature cleanup removes the old hook too early

Symptom:

- one route still imports `useVirtualList`,
- build breaks mid-migration,
- review becomes hard.

Prevention:

- delete the old hook only after both call sites are migrated.

---

## 11. Testing and validation plan

## 11.1 Mandatory build validation

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web
npm run build
```

Run this after each migration phase.

## 11.2 Route smoke tests

### Session Browser

Verify:

- `/sessions` loads
- scrolling works
- click row opens transcript
- query icon still works
- annotation badges still display correctly

### Transcript route

Verify:

- `/sessions/:id` loads
- transcript blocks render correctly
- expanding/collapsing blocks works
- turn annotation chips still open the annotation panel
- tool-call annotation chips still work
- focused turn deep link works
- focused tool-call deep link works
- inline composer appears for URL-driven compose state

## 11.3 Regression URL validation

The previously failing URL shape is the most important correctness test. Use the exact live session fixture if available.

Expected outcome after migration:

- no `Maximum update depth exceeded`
- page remains interactive
- focused tool-call row becomes visible
- inline composer appears

## 11.4 Measurement validation

Reuse the existing ticket scripts.

Primary script:

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace
node ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs
```

Compare results to the existing saved snapshots rather than to vague intuition.

## 11.5 Storybook/build compatibility validation

If `BlockCard` props change, also run:

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web
npm run build-storybook
```

This matters because Storybook previously surfaced a `defaultExpanded` API mismatch during transcript refactoring.

---

## 12. Suggested commit plan

Keep commits small and reviewable.

Recommended commit sequence:

1. `web: add tanstack virtual dependency`
2. `web: migrate session browser to tanstack virtual`
3. `web: migrate transcript viewer to tanstack virtual`
4. `web: remove custom virtual list hook`
5. `web: add route-level error boundaries`
6. `docs(ticket): record tanstack virtual migration and validation`

Do not combine all of this into one giant commit.

---

## 13. Rollback plan

If the transcript migration becomes unstable mid-way:

1. keep the Session Browser migration if it is already stable,
2. revert only the transcript migration commit,
3. keep the documentation and measurement evidence,
4. decide whether a short-term hardening patch to `useVirtualList.ts` is still needed.

This is another reason to migrate routes in separate commits.

---

## 14. Quick-reference implementation checklist for the intern

### Before coding

- [ ] Read `useVirtualList.ts`
- [ ] Read `SessionBrowser.tsx`
- [ ] Read `TranscriptViewer.tsx`
- [ ] Read `BlockCard.tsx`
- [ ] Read the regression note in `reference/02-...md`
- [ ] Run `npm run build`

### During migration

- [ ] Add `@tanstack/react-virtual`
- [ ] Migrate Session Browser first
- [ ] Build and smoke test `/sessions`
- [ ] Migrate Transcript Viewer second
- [ ] Re-test deep-link focus/composer path
- [ ] Remove old hook only after both routes are green
- [ ] Add route boundaries after core behavior is stable

### Before handing off

- [ ] Run `npm run build`
- [ ] Run `npm run build-storybook` if needed
- [ ] Re-run the measurement script
- [ ] Save artifacts into `sources/`
- [ ] Update diary/changelog/index

---

## 15. API reference quick sheet

### Library imports

```ts
import { useVirtualizer } from '@tanstack/react-virtual';
```

### Most relevant options for this repo

```ts
useVirtualizer({
  count,
  getScrollElement: () => scrollContainerRef.current,
  estimateSize: (index) => number,
  overscan: number,
  getItemKey: (index) => string | number,
})
```

### Most relevant instance methods/properties for this repo

```ts
const items = virtualizer.getVirtualItems();
const total = virtualizer.getTotalSize();
virtualizer.scrollToIndex(index, { align: 'center' });
virtualizer.measure();
```

### Dynamic-height measurement pattern

```tsx
<div data-index={virtualRow.index} ref={virtualizer.measureElement}>
  ...row content...
</div>
```

Use this pattern for transcript block rows, not necessarily for Session Browser rows.

---

## 16. References and reading order

Recommended reading order for a new engineer:

1. `web/src/components/shared/useVirtualList.ts`
2. `web/src/components/SessionBrowser/SessionBrowser.tsx`
3. `web/src/components/TranscriptViewer/TranscriptViewer.tsx`
4. `web/src/components/TranscriptViewer/BlockCard.tsx`
5. `web/src/components/TranscriptViewer/BlockBody.tsx`
6. `web/src/components/TranscriptViewer/ToolCallRow.tsx`
7. `web/src/pages/TranscriptViewerPage.tsx`
8. `web/src/api/minitrace.ts`
9. `reference/02-maximum-update-depth-regression-analysis-and-error-boundary-placement.md`
10. `design-doc/01-transcript-viewer-and-web-ui-performance-optimization-study-and-implementation-guide.md`

External references:

- TanStack Virtualizer API: `https://tanstack.com/virtual/latest/docs/api/virtualizer`
- TanStack VirtualItem API: `https://tanstack.com/virtual/latest/docs/api/virtual-item`
- TanStack React examples: `https://tanstack.com/virtual/latest/docs/framework/react/examples`

---

## Final recommendation

Adopt TanStack Virtual, but do it in a disciplined way.

The migration should be treated as an architectural simplification, not as a rushed bug workaround. The simplest safe order is:

1. install the dependency,
2. migrate Session Browser first,
3. migrate Transcript Viewer second,
4. remove the old hook,
5. add route-level boundaries,
6. validate the failing deep-link and rerun measurements.

That sequence keeps learning local, keeps commits reviewable, and gives the intern a clear definition of done.