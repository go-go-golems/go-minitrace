---
Title: Investigation diary
Ticket: TRANSCRIPT-PERF-001
Status: active
Topics:
    - performance
    - frontend
    - react
    - web-ui
    - transcript-analysis
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs
    - Path: ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/01-baseline-measurements.json
    - Path: ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/02-step-2-persistent-mount-measurements.json
      Note: Post-Step-2 snapshot for mounted transcript pane behavior
    - Path: ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/03-step-3-unmount-on-exit-measurements.json
      Note: Post-Step-3 snapshot showing large mounted-tree reduction
    - Path: web/src/components/QueryEditor/ResultsTable.tsx
      Note: Step 4 memoized full-table sorting (commit 7a6e30c)
    - Path: web/src/components/TranscriptViewer/BlockCard.tsx
      Note: Steps 2-3 memoized heavy block rows and unmounted collapsed block bodies (commits 22aafff
    - Path: web/src/components/TranscriptViewer/ToolCallRow.tsx
      Note: Steps 2-3 memoized heavy tool-call rows and unmounted collapsed tool-call details (commits 22aafff
    - Path: web/src/components/TranscriptViewer/TranscriptViewer.tsx
      Note: Step 2 kept transcript pane mounted across tab switches (commit 22aafff)
    - Path: web/src/pages/QueryEditorPage.tsx
      Note: Step 5 reduced background polling pressure while preserving active-source refreshes (commit 17600ec)
ExternalSources: []
Summary: Chronological diary of the performance study for the transcript viewer and broader web UI, including measurements and evidence-gathering commands.
LastUpdated: 2026-04-06T18:35:00-04:00
WhatFor: Preserve the reasoning, commands, measurements, and evidence used to produce the performance optimization study.
WhenToUse: Use when continuing transcript-performance work, reviewing how the measurements were gathered, or validating the rationale behind the optimization phases.
---


# Investigation diary

## Goal

Capture the exact investigation work used to create the `TRANSCRIPT-PERF-001` performance ticket, gather code-backed evidence across the app, measure the current dev stack with Playwright, and write a detailed optimization guide suitable for a new intern.

## Step 1: Create the ticket, gather evidence, and establish a baseline

This step created a fresh ticket workspace for performance work, mapped the relevant frontend and backend files, added a repeatable Playwright measurement script, and captured a baseline against the currently running dev stack. The result is not a vague “the transcript feels slow” note. It is a file-backed performance study with at least one concrete baseline and an explicit phased optimization plan.

The biggest practical outcome of the step is that the transcript route is now clearly separated from the rest of the app in performance terms. The Session Browser and Query Editor are not free, but they are nowhere near the same cost profile as the transcript view. That distinction matters because it changes the implementation strategy: optimize the transcript path first, then address the table-heavy routes as second-wave scalability work.

### Prompt Context

**User prompt (verbatim):** "Create a new performance optimization ticket, and write a detailed performance optimization study across the app, with findings like these. 

Create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable.

I don't know if it's possible for you to do proper measurements using playwright, but you could if you wanted. Write any scripts in the scripts/ directory of the new ticket."

**Assistant interpretation:** Create a new documentation ticket focused on web UI performance, investigate the current architecture and bottlenecks with concrete evidence, add at least some repeatable measurements using Playwright, store the study and scripts in the ticket, and upload the deliverables to reMarkable.

**Inferred user intent:** Build an evidence-backed optimization plan for the app rather than reacting to isolated symptoms, so future work on transcript performance is coordinated, measurable, and easy for another engineer to continue.

**Commit (code):** N/A — investigation/documentation phase only at this step

### What I did

- Created a new ticket workspace:
  - `docmgr ticket create-ticket --ticket TRANSCRIPT-PERF-001 --title "Transcript viewer and web UI performance optimization study" --topics performance,frontend,react,web-ui,transcript-analysis`
- Created the main design doc:
  - `docmgr doc add --ticket TRANSCRIPT-PERF-001 --doc-type design-doc --title "Transcript viewer and web UI performance optimization study and implementation guide"`
- Created the diary doc:
  - `docmgr doc add --ticket TRANSCRIPT-PERF-001 --doc-type reference --title "Investigation diary"`
- Mapped key frontend files:
  - `web/src/App.tsx`
  - `web/src/pages/TranscriptViewerPage.tsx`
  - `web/src/components/TranscriptViewer/TranscriptViewer.tsx`
  - `web/src/components/TranscriptViewer/BlockCard.tsx`
  - `web/src/components/TranscriptViewer/ToolCallRow.tsx`
  - `web/src/components/SessionBrowser/SessionBrowser.tsx`
  - `web/src/pages/QueryEditorPage.tsx`
  - `web/src/components/QueryEditor/ResultsTable.tsx`
- Mapped key backend files:
  - `cmd/go-minitrace/cmds/serve/handlers_sessions.go`
  - `cmd/go-minitrace/cmds/serve/blocks.go`
- Added a repeatable measurement script under the ticket:
  - `scripts/01-web-ui-baseline-perf.mjs`
- Added a scripts README:
  - `scripts/00-README.md`
- Ran the measurement script and saved the output to:
  - `sources/01-baseline-measurements.json`
- Related the key files to the design doc and diary with `docmgr doc relate`

### Why

- The user explicitly asked for a new performance optimization ticket rather than an informal note.
- The user also asked for a detailed intern-oriented implementation guide, which means the deliverable needs architecture mapping, evidence, and a phased plan rather than only a list of optimizations.
- The user specifically mentioned Playwright measurements, so a repeatable script in the ticket’s `scripts/` directory is a better artifact than ad hoc shell history.
- A performance plan without even one measured baseline tends to drift into speculation, especially around React rendering costs.

### What worked

- The app structure is compact enough that the hot paths were easy to identify quickly.
- The route boundaries are clear: Session Browser, Transcript Viewer, Query Editor.
- The backend session-detail flow made it straightforward to identify where blocks are built before sending the response.
- The Playwright measurement script worked well against the existing dev stack on `127.0.0.1:5174` and `127.0.0.1:8080`.
- The baseline output was informative enough to support prioritization:
  - Session Browser mean load ~612 ms
  - Transcript mean initial load ~3958 ms
  - Transcript → Annotations ~1527 ms
  - back to Transcript ~66 ms on the current local stack
  - Query page mean load ~134 ms
  - transcript DOM size ~15.5k nodes on the sampled session

### What didn't work

- A first attempt to search for a `FileViewer` component based on an unrelated React hook-order issue was irrelevant to this repository and led into transcript artifacts rather than source files. That search was a distraction and had to be abandoned.
- One broad `find` under `/home/manuel/workspaces` aborted because of unrelated permission/noise paths. That did not block the work because the relevant repo-local files were already known.
- The measurement script is not a full profiler. It does not produce Chrome traces, React commit breakdowns, or flamegraphs. It provides route timings and rough DOM counts only.

### What I learned

- The app does not have one general performance problem. It has one dominant hotspot: transcript rendering.
- The Session Browser and Query Editor are acceptable at current scale, but both contain obvious future scalability risks.
- The transcript route’s cost comes from multiplicative nested rendering, not from route shell complexity.
- The backend already does useful shaping work by building transcript blocks server-side, but that also means every session detail request pays for full block construction.
- The current local stack already appears to include a mitigation for the transcript remount-on-tab-switch problem, because the “back to transcript” timing is much lower than the initial transcript render timing.

### What was tricky to build

- The main tricky part was separating “what feels slow” from “what is structurally expensive.” The transcript route feels slow for one reason, but the study needed to describe the entire app without flattening all routes into one bucket.
- A second tricky part was measuring something useful without overengineering the measurement harness. Full browser profiling would be stronger, but the ticket needed a repeatable script now, not a week-long instrumentation project.
- Another subtle point was interpreting the current local state fairly. Because the current dev server likely included recent local transcript-tab optimizations, the study had to avoid accidentally presenting the old remount regression as if it were still the dominant current cost. The remaining dominant cost is the large initial transcript render.

### What warrants a second pair of eyes

- Whether `unmountOnExit` on the transcript block and tool-call collapse regions is truly a safe phase-1 change in the presence of all focus and expansion behaviors.
- Whether transcript virtualization should happen only at block level or whether some very large blocks will need deeper turn-level handling.
- Whether backend response splitting is worth the complexity if frontend mount-hygiene work already cuts most of the user-facing latency.
- Whether the current 3-second query-page polling is materially contributing to perceived UI churn in real use.

### What should be done in the future

- Run the new measurement script after each meaningful optimization phase and keep the JSON outputs in the ticket.
- Implement the optimization plan in phases instead of mixing virtualization, mount hygiene, and backend changes together.
- Add stronger profiling evidence later if one phase underperforms expectations, for example Chrome Performance traces or React Profiler captures on the same baseline session.

### Code review instructions

Start in this order:

1. `web/src/components/TranscriptViewer/TranscriptViewer.tsx`
   - understand route-level composition and the `session.blocks.map(...)` path
2. `web/src/components/TranscriptViewer/BlockCard.tsx`
   - inspect the block header/body shape and current collapse behavior
3. `web/src/components/TranscriptViewer/ToolCallRow.tsx`
   - inspect heavy expanded output rendering
4. `cmd/go-minitrace/cmds/serve/handlers_sessions.go`
   - see how session detail is shaped
5. `cmd/go-minitrace/cmds/serve/blocks.go`
   - understand backend block construction
6. `web/src/components/SessionBrowser/SessionBrowser.tsx`
   - note non-virtualized table rendering
7. `web/src/components/QueryEditor/ResultsTable.tsx`
   - inspect full-array sorting and all-row rendering
8. `web/src/pages/QueryEditorPage.tsx`
   - inspect polling behavior

Then run:

```bash
node ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs
```

and compare the result to:

```text
ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/01-baseline-measurements.json
```

### Technical details

Commands used during the investigation step:

```bash
docmgr status --summary-only
docmgr ticket create-ticket --ticket TRANSCRIPT-PERF-001 --title "Transcript viewer and web UI performance optimization study" --topics performance,frontend,react,web-ui,transcript-analysis
docmgr doc add --ticket TRANSCRIPT-PERF-001 --doc-type design-doc --title "Transcript viewer and web UI performance optimization study and implementation guide"
docmgr doc add --ticket TRANSCRIPT-PERF-001 --doc-type reference --title "Investigation diary"
rg --files web/src cmd/go-minitrace/cmds/serve pkg/query pkg/doc | sort
rg -n "TranscriptViewer|SessionBrowser|QueryEditor|BlockCard|ToolCallRow|AnnotationPanel" web/src cmd/go-minitrace/cmds/serve -S
nl -ba web/src/components/TranscriptViewer/TranscriptViewer.tsx | sed -n '1,420p'
nl -ba web/src/components/TranscriptViewer/BlockCard.tsx | sed -n '1,340p'
nl -ba web/src/components/TranscriptViewer/ToolCallRow.tsx | sed -n '1,260p'
nl -ba web/src/components/SessionBrowser/SessionBrowser.tsx | sed -n '1,260p'
nl -ba web/src/components/QueryEditor/ResultsTable.tsx | sed -n '1,220p'
nl -ba web/src/pages/QueryEditorPage.tsx | sed -n '1,220p'
nl -ba cmd/go-minitrace/cmds/serve/handlers_sessions.go | sed -n '1,260p'
nl -ba cmd/go-minitrace/cmds/serve/blocks.go | sed -n '1,220p'
node ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs > ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/01-baseline-measurements.json
```

Key measured outputs captured in the saved JSON:

- session browser mean load: `612 ms`
- transcript mean initial load: `3958 ms`
- transcript mean switch to annotations: `1527 ms`
- transcript mean back to transcript: `66 ms`
- query page mean load: `134 ms`
- transcript DOM nodes on sampled session: `15479`

## Step 2: Stabilize transcript tab switches by keeping the transcript pane mounted

This step took the already-started local transcript performance changes, validated them, and turned them into a real commit instead of leaving them as ambient working-tree state. The main idea was simple: if the user switches from Transcript to Annotations and then back again, the app should not pay the full cost of rebuilding the transcript tree.

In practice, this step did two things together. First, `TranscriptViewer` now keeps both panes mounted and only toggles visibility, which avoids an unnecessary unmount/remount cycle. Second, the heavy row components were wrapped in `React.memo` so stable props can short-circuit rerenders more effectively during tab and URL-state churn.

### Prompt Context

**User prompt (verbatim):** "Ok, add tasks to the ticket, and then execute them one by one, committing at appropriate intervals, and keeping a detailed diary."

**Assistant interpretation:** Convert the design ticket into an active execution plan, perform the optimization steps in sequence, commit each logical change set separately, and record the work in the ticket diary.

**Inferred user intent:** Move from planning into implementation while preserving evidence, traceability, and continuation-friendly documentation.

**Commit (code):** `22aafff` — `web: keep transcript pane mounted across tab switches`

### What I did

- Reviewed the existing uncommitted diff in:
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/TranscriptViewer.tsx`
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/BlockCard.tsx`
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/ToolCallRow.tsx`
- Confirmed that `TranscriptViewer` now keeps both the transcript pane and annotation pane mounted and swaps them with `display`/`aria-hidden` rather than conditional rendering.
- Converted `setUrlState`, `handleNavigateToAnnotationTarget`, `handleOpenAnnotation`, and `handleCreateScopedAnnotation` to `useCallback`-backed handlers.
- Wrapped `BlockCard` and `ToolCallRow` exports in `memo(...)`.
- Ran:
  - `cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build`
- Captured a measurement snapshot:
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/02-step-2-persistent-mount-measurements.json`
- Committed the code change.

### Why

- The working tree already contained a meaningful transcript-tab optimization that should not remain uncommitted while the ticket moves forward.
- Keeping the transcript pane mounted directly addresses one of the most visible user complaints: slow return from Annotations back to Transcript.
- Memoizing the heavy child components is a low-risk multiplier for that change because it reduces avoidable rerenders triggered by tab-state and URL-state updates.

### What worked

- The build passed cleanly.
- The code change was small and contained to the transcript viewer path.
- The post-change snapshot still showed the desired fast “back to transcript” path, with mean `backToTranscriptMs` around `56 ms` on that run.

### What didn't work

- The first post-change measurement snapshot was noisy for `initialLoadMs` and much worse than expected on that run set:
  - mean transcript initial load `8995 ms`
  - mean transcript → annotations `2105 ms`
  - mean transcript back `56 ms`
- Command used:

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && \
node ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs > \
ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/02-step-2-persistent-mount-measurements.json
```

- That result reinforces that the lightweight Playwright script is useful for trend checking, but still subject to dev-server/runtime variance.

### What I learned

- The “keep the transcript mounted” fix is specifically a **tab-switch return-speed** optimization, not a complete solution for heavy initial transcript render cost.
- The noisy `02` measurement made it clearer that I should not treat one dev-stack timing sample as definitive without checking the shape of the DOM and mounted subtree size.

### What was tricky to build

- The tricky part here was not coding the change; it was deciding how to package already-started local work into a traceable step. If I had mixed this change together with `unmountOnExit`, it would have been harder to explain which improvement produced which effect.
- Another subtle point was handler stability. Once `BlockCard` and `ToolCallRow` are memoized, unstable callback props can quietly erode the win. Using `useCallback` for the URL-state handlers was therefore part of making the memoization meaningful rather than cosmetic.

### What warrants a second pair of eyes

- Whether the current prop identity on `turnAnnotations` and `toolCallAnnotations` is stable enough in realistic navigation scenarios to let `memo(...)` do useful work consistently.
- Whether keeping both panes mounted causes any hidden memory/regression cost on very large sessions.

### What should be done in the future

- Follow immediately with explicit subtree unmounting for collapsed transcript content.
- If transcript initial load remains dominant after that, move on to virtualization rather than adding many more small memoization tweaks.

### Code review instructions

Start with:

1. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/TranscriptViewer.tsx`
2. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/BlockCard.tsx`
3. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/ToolCallRow.tsx`

Validation:

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && node ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs
```

### Technical details

Key structural change:

- transcript and annotation panes remain mounted; visibility is toggled with:
  - `sx={{ display: view === "transcript" ? "block" : "none" }}`
  - `sx={{ display: view === "annotations" ? "block" : "none" }}`

Related measurement artifact:

- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/02-step-2-persistent-mount-measurements.json`

## Step 3: Unmount collapsed transcript subtrees

This step applied the first high-confidence optimization from the study: if a transcript block or tool-call detail panel is collapsed, it should not stay mounted. This is the cleanest way to reduce the amount of React tree and DOM the browser has to keep alive for a large transcript.

The result was much clearer than Step 2. After adding `unmountOnExit` to the two key `Collapse` boundaries, the rerun measurement showed a dramatically smaller mounted transcript subtree in the sampled session, along with much better transcript route timings.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue executing the planned optimization tasks in order, keeping each step narrowly scoped and separately committed.

**Inferred user intent:** Reduce real user-facing transcript cost while preserving evidence about which change produced which effect.

**Commit (code):** `6bf9596` — `web: unmount collapsed transcript subtrees`

### What I did

- Updated:
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/BlockCard.tsx`
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/ToolCallRow.tsx`
- Added `unmountOnExit` to both MUI `Collapse` components.
- Intentionally did **not** add `content-visibility: auto` in the same step, to keep the causal surface small.
- Ran:
  - `cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build`
- Captured a new measurement snapshot:
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/03-step-3-unmount-on-exit-measurements.json`
- Committed the code change.

### Why

- Collapsed transcript content that stays mounted is the clearest form of wasted UI work in this route.
- `unmountOnExit` is a focused change with a strong expected payoff and very low conceptual complexity.
- Keeping this step isolated makes the measurement easier to interpret.

### What worked

- The build passed cleanly.
- The new measurement snapshot showed a much smaller mounted subtree and materially better timings:
  - mean transcript initial load: `1912 ms`
  - mean transcript → annotations: `698 ms`
  - mean transcript back: `55 ms`
  - mounted turns in DOM on the sampled state: `12`
  - mounted tool calls in DOM on the sampled state: `34`
  - DOM node count: `1758`
- Compared with the earlier large-tree measurements, this is the first step that clearly attacked the heavy initial transcript render cost rather than only the tab-return path.

### What didn't work

- N/A in terms of build/runtime failures.
- I deliberately did not mix in `content-visibility: auto` because that would have made this measurement harder to interpret.

### What I learned

- Explicit subtree unmounting is a much stronger optimization here than generic memoization alone.
- The transcript route’s performance problem was indeed dominated by mounted subtree size; the DOM/node count collapse in the measurement is too large to be accidental.
- This also means the future block header/body split and virtualization work can now start from a cleaner baseline instead of compensating for always-mounted collapsed bodies.

### What was tricky to build

- The tricky part was guarding against hidden UX regressions. `Collapse` with `unmountOnExit` can reset local state inside the subtree, so I needed to keep the change limited to transcript content that is genuinely disposable when collapsed.
- There was also a sequencing constraint: I wanted the persistent-pane work committed first, so that this step could be evaluated on top of a stable tab-switch baseline rather than an uncommitted mix of ideas.

### What warrants a second pair of eyes

- Whether any annotation-driven focus path relies on previously mounted collapsed content in a way that only shows up in edge cases.
- Whether `showAllTools` behavior remains intuitive when a block is collapsed and re-expanded after the subtree is unmounted.

### What should be done in the future

- Re-run the live annotation-navigation smoke on a session that exercises focused tool-call expansion, not just the baseline measurement script.
- Use this leaner mounted-tree baseline as the starting point for the next structural step: block header/body separation or block virtualization.

### Code review instructions

Review these exact lines first:

1. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/BlockCard.tsx`
2. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/ToolCallRow.tsx`

Validation:

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && node ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs
```

Then compare:

- `sources/01-baseline-measurements.json`
- `sources/02-step-2-persistent-mount-measurements.json`
- `sources/03-step-3-unmount-on-exit-measurements.json`

### Technical details

Exact code changes:

```tsx
<Collapse in={isExpanded} unmountOnExit>
<Collapse in={expanded} unmountOnExit>
```

Related measurement artifact:

- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/03-step-3-unmount-on-exit-measurements.json`

## Step 4: Memoize query result sorting

This step addressed a smaller but very obvious inefficiency outside the transcript route. `ResultsTable` was sorting the full row set on every render path even when the inputs to the sort had not changed. That is cheap for tiny result sets and wasteful for large ones.

This was intentionally a small, isolated change: wrap the full-array sort in `useMemo`, validate the build, and commit it separately so future regressions in query rendering are easy to attribute.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue executing the next planned optimization task with a narrow scope and separate commit.

**Inferred user intent:** Pick off the low-risk, high-confidence performance wins across the app while preserving a clear audit trail.

**Commit (code):** `7a6e30c` — `web: memoize query results sorting`

### What I did

- Updated:
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/QueryEditor/ResultsTable.tsx`
- Added `useMemo` around the existing `sortedRows` computation.
- Kept the sorting behavior itself unchanged.
- Ran:
  - `cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build`
- Committed the code change.

### Why

- The app-wide study identified `ResultsTable` as a clear future scaling risk.
- This is a straightforward optimization that reduces unnecessary O(n log n) work on rerenders without changing the UI contract.

### What worked

- The build passed cleanly.
- The change surface stayed extremely small.
- The implementation did not require any new state, API changes, or UI behavior changes.

### What didn't work

- N/A — no runtime or build errors on this step.
- I did not run a dedicated large-result benchmark for this step yet, so the gain is currently justified by code-path analysis rather than a new saved timing artifact.

### What I learned

- There are still several cheap wins available in the non-transcript parts of the app, but they should stay secondary to transcript work.
- Keeping this change behavior-preserving made it a good “background optimization” while the transcript route remains the main focus.

### What was tricky to build

- The main subtlety was to avoid changing sort semantics accidentally while converting to `useMemo`. The safest move was to memoize the existing expression rather than rewrite the comparison logic.

### What warrants a second pair of eyes

- Whether `result.rows` identity from RTK Query stays stable in the ways I expect, because that determines how often the memoized sort is invalidated.
- Whether very large result sets will still want virtualization even after sort memoization.

### What should be done in the future

- Add row virtualization or paging if query result sizes become large enough that DOM size, not sorting, becomes the dominant cost.

### Code review instructions

Review:

- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/QueryEditor/ResultsTable.tsx`

Validation:

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build
```

### Technical details

Core change:

```tsx
const sortedRows = useMemo(() => [...result.rows].sort(...), [result.rows, sortCol, sortDir]);
```

## Step 5: Reduce background query-editor polling pressure

This step revisited the query page’s 3-second polling behavior. The goal was not to remove the useful “external file changed” affordance, but to stop paying the same polling cost for every source all the time, especially when the page is not focused.

The chosen change is intentionally conservative. The active source type still polls quickly every 3 seconds, the inactive source type polls much more slowly, and both queries stop polling when the tab is unfocused.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue the task list with the next bounded optimization and record the result cleanly.

**Inferred user intent:** Improve the app incrementally without turning the ticket into a single large, hard-to-review change.

**Commit (code):** `17600ec` — `web: reduce background query editor polling`

### What I did

- Updated:
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/pages/QueryEditorPage.tsx`
- Added source-aware polling intervals:
  - active source kind → `3000 ms`
  - inactive source kind → `15000 ms`
- Added `skipPollingIfUnfocused: true` to both RTK Query hooks.
- Kept `refetchOnFocus` and `refetchOnReconnect` enabled.
- Ran:
  - `cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build`
- Committed the code change.

### Why

- Polling every 3 seconds for both presets and saved queries all the time is simple but needlessly chatty.
- The page only needs high-frequency polling for the currently active source if we want timely external-update detection.
- Hidden tabs should not keep polling aggressively.

### What worked

- The build passed cleanly.
- The implementation stayed local to one page component.
- The change preserves the important behavior: external changes to the active query source can still surface quickly.

### What didn't work

- N/A — no build/runtime failure.
- I did not create a dedicated network-request measurement artifact for this step yet, so the benefit is inferred from reduced polling frequency rather than a saved request-count trace.

### What I learned

- Query-page performance hygiene is partly about render cost and partly about background work. This step improved the latter.
- RTK Query already had the right knob (`skipPollingIfUnfocused`) for a safe win without extra plumbing.

### What was tricky to build

- The tricky part was reducing load without degrading the “this source changed on disk” operator experience. The active/inactive split was a good compromise because it preserves responsiveness where the user is actually working.

### What warrants a second pair of eyes

- Whether `15000 ms` is the right inactive-source interval or should be tuned further.
- Whether the saved query and preset sidebars need a manual refresh affordance if the polling policy gets even more conservative later.

### What should be done in the future

- If query-page background activity still matters, measure actual network request volume with Playwright/network logging before changing the polling model further.

### Code review instructions

Review:

- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/pages/QueryEditorPage.tsx`

Validation:

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build
```

### Technical details

New policy:

```ts
const presetPollingInterval = activeSource?.kind === "preset" ? 3000 : 15000;
const savedPollingInterval = activeSource?.kind === "saved" ? 3000 : 15000;
```

Hook options now include:

```ts
skipPollingIfUnfocused: true,
refetchOnFocus: true,
refetchOnReconnect: true,
```
