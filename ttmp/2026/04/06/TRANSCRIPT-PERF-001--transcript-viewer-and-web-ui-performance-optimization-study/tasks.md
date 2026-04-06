# Tasks

## Complete

- [x] Create a new performance optimization ticket for the web UI
- [x] Map the relevant frontend and backend code paths for transcript rendering
- [x] Add a repeatable Playwright baseline measurement script under `scripts/`
- [x] Capture baseline measurements for the current local dev stack
- [x] Write the primary design / implementation guide
- [x] Write the investigation diary
- [x] Relate key files to the ticket docs

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

- [ ] Extract a lightweight always-mounted block header component
- [ ] Extract a lazily mounted block body component
- [ ] Keep collapsed block footprint minimal
- [ ] Verify focused-target and annotation navigation still work correctly
- [ ] Run `npm run build` in `web/`
- [ ] Commit the code changes
- [ ] Update diary and changelog for the step

### Step 7 — Virtualize transcript block rendering

- [ ] Choose and implement a virtualization approach for block lists
- [ ] Render only visible blocks plus overscan
- [ ] Integrate focused target scrolling with virtualized rows
- [ ] Rerun baseline measurements on the same session
- [ ] Commit the code changes
- [ ] Update diary and changelog for the step

### Step 8 — Add Session Browser virtualization

- [ ] Add virtualization or threshold-based virtualization to Session Browser
- [ ] Verify click-through and query-button interactions still work
- [ ] Run `npm run build` in `web/`
- [ ] Capture a route measurement snapshot if practical
- [ ] Commit the code changes
- [ ] Update diary and changelog for the step

### Step 9 — Backend session summary/body shaping

- [ ] Add a summary-only session-detail API path or equivalent shaping mechanism
- [ ] Update the transcript page to fetch summary metadata separately from blocks
- [ ] Run `npm run build` in `web/`
- [ ] Run Go tests/build for the serve package or repo
- [ ] Capture a follow-up measurement snapshot
- [ ] Commit the code changes
- [ ] Update diary and changelog for the step

