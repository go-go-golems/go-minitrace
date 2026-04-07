# Tasks

## Complete

- [x] Create a new performance optimization ticket for the web UI
- [x] Map the relevant frontend and backend code paths for transcript rendering
- [x] Add a repeatable Playwright baseline measurement script under `scripts/`
- [x] Capture baseline measurements for the current local dev stack
- [x] Write the primary design / implementation guide
- [x] Write the investigation diary
- [x] Relate key files to the ticket docs
- [x] Write a full TanStack React Virtual migration design and implementation guide for a new intern

## Execution plan

### Step 2 — Stabilize already-started transcript tab-switch optimization

- [x] Review the existing local changes in `TranscriptViewer.tsx`, `BlockCard.tsx`, and `ToolCallRow.tsx`
- [x] Validate that the transcript pane stays mounted across tab switches
- [x] Validate that memoized `BlockCard` / `ToolCallRow` do not break annotation navigation
- [x] Run `npm run build` in `web/`
- [x] Capture a post-change measurement snapshot
- [x] Commit the code changes
- [x] Update diary and changelog for the step

### Step 3 — Transcript mount hygiene for collapsed subtrees

- [x] Add `unmountOnExit` to transcript block collapse regions
- [x] Add `unmountOnExit` to tool-call detail collapse regions
- [x] Consider a low-risk paint/layout hint such as `content-visibility: auto` on large transcript bodies (`deferred` — kept this step narrowly focused on unmounting)
- [x] Run `npm run build` in `web/`
- [x] Capture a post-change measurement snapshot
- [x] Commit the code changes
- [x] Update diary and changelog for the step

### Step 4 — Query results table cheap-win optimization

- [x] Memoize `ResultsTable` sorting so unrelated rerenders do not re-sort full result sets
- [x] Run `npm run build` in `web/`
- [x] Commit the code changes
- [x] Update diary and changelog for the step

### Step 5 — Query editor polling hygiene

- [x] Revisit the 3-second polling behavior in `QueryEditorPage.tsx`
- [x] Reduce unnecessary polling pressure without losing the “external file changed” affordance
- [x] Run `npm run build` in `web/`
- [x] Commit the code changes
- [x] Update diary and changelog for the step

## Remaining execution steps

### Step 6 — Split transcript block header from block body

- [x] Extract a lightweight always-mounted block header component
- [x] Extract a lazily mounted block body component
- [x] Keep collapsed block footprint minimal
- [x] Verify focused-target and annotation navigation still work correctly
- [x] Run `npm run build` in `web/`
- [x] Commit the code changes
- [x] Update diary and changelog for the step

### Step 7 — Virtualize transcript block rendering

- [x] Choose and implement a virtualization approach for block lists
- [x] Render only visible blocks plus overscan
- [x] Integrate focused target scrolling with virtualized rows
- [x] Rerun baseline measurements on the same session
- [x] Commit the code changes
- [x] Update diary and changelog for the step

### Step 8 — Add Session Browser virtualization

- [x] Add virtualization or threshold-based virtualization to Session Browser
- [x] Verify click-through and query-button interactions still work
- [x] Run `npm run build` in `web/`
- [x] Capture a route measurement snapshot if practical
- [x] Commit the code changes
- [x] Update diary and changelog for the step

### Step 9 — Backend session summary/body shaping

- [x] Add a summary-only session-detail API path or equivalent shaping mechanism
- [x] Update the transcript page to fetch summary metadata separately from blocks
- [x] Run `npm run build` in `web/`
- [x] Run Go tests/build for the serve package or repo
- [x] Capture a follow-up measurement snapshot
- [x] Commit the code changes
- [x] Update diary and changelog for the step

## Stabilization follow-up after virtualization rollout

### Step 10 — Fix virtual-list maximum-update-depth regression

- [ ] Stabilize virtual-row ref callbacks so they are not recreated every render
- [ ] Remove or minimize synchronous measurement-driven state writes from the ref-attach path
- [ ] Verify `npm run build` in `web/`
- [ ] Commit the code changes
- [ ] Update diary and changelog for the step

### Step 11 — Add route-level error-boundary containment

- [ ] Add a reusable app/route error boundary component
- [ ] Wrap routed pages in boundaries in `App.tsx`
- [ ] Add a transcript-specific boundary with session-aware fallback in `TranscriptViewerPage.tsx`
- [ ] Verify `npm run build` in `web/`
- [ ] Commit the code changes
- [ ] Update diary and changelog for the step

### Step 12 — Validate the exact failing transcript deep link

- [ ] Add a reproducible smoke script under `scripts/` for the maximum-update-depth regression URL
- [ ] Run the smoke against `http://127.0.0.1:5174` or the current dev stack
- [ ] Confirm the focused tool-call + composer URL no longer crashes the app
- [ ] Record the result in `sources/` if useful
- [ ] Update diary and changelog for the step

