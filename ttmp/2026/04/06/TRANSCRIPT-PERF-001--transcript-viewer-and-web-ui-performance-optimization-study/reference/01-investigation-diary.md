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
    - Path: cmd/go-minitrace/cmds/serve/handlers_sessions.go
      Note: Step 9 separated session summary normalization from heavy block payload generation (commit 0835f29)
    - Path: cmd/go-minitrace/cmds/serve/server.go
      Note: Step 9 registered the summary-only session API route (commit 0835f29)
    - Path: cmd/go-minitrace/cmds/serve/server_test.go
      Note: Step 9 added a summary-endpoint handler test (commit 0835f29)
    - Path: ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs
    - Path: ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/02-session-summary-blocks-split-perf.mjs
      Note: API measurement script for the summary-vs-blocks backend shaping step
    - Path: ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/01-baseline-measurements.json
    - Path: ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/02-step-2-persistent-mount-measurements.json
      Note: Post-Step-2 snapshot for mounted transcript pane behavior
    - Path: ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/03-step-3-unmount-on-exit-measurements.json
      Note: Post-Step-3 snapshot showing large mounted-tree reduction
    - Path: ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/04-step-7-transcript-virtualization-measurements.json
      Note: Post-Step-7 transcript virtualization snapshot
    - Path: ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/05-step-8-session-browser-virtualization-measurements.json
      Note: Post-Step-8 Session Browser virtualization snapshot
    - Path: ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/06-step-9-summary-and-blocks-split-measurements.json
      Note: Post-Step-9 API shaping snapshot
    - Path: web/src/api/minitrace.ts
      Note: Step 9 added the session summary RTK Query endpoint (commit 0835f29)
    - Path: web/src/components/QueryEditor/ResultsTable.tsx
      Note: Step 4 memoized full-table sorting (commit 7a6e30c)
    - Path: web/src/components/SessionBrowser/SessionBrowser.tsx
      Note: Step 8 virtualized Session Browser rows (commit c4bb6ca)
    - Path: web/src/components/TranscriptViewer/BlockBody.tsx
      Note: Step 6 extracted the lazily mounted transcript block body (commit e6fa2c8)
    - Path: web/src/components/TranscriptViewer/BlockCard.tsx
      Note: Steps 2-3 memoized heavy block rows and unmounted collapsed block bodies (commits 22aafff
    - Path: web/src/components/TranscriptViewer/BlockHeader.tsx
      Note: Step 6 extracted the lightweight always-mounted transcript block header (commit e6fa2c8)
    - Path: web/src/components/TranscriptViewer/ToolCallRow.tsx
      Note: Steps 2-3 memoized heavy tool-call rows and unmounted collapsed tool-call details (commits 22aafff
    - Path: web/src/components/TranscriptViewer/TranscriptViewer.tsx
      Note: Step 2 kept transcript pane mounted across tab switches (commit 22aafff)
    - Path: web/src/components/shared/useVirtualList.ts
      Note: Steps 7-8 added a shared measured-window virtualization hook for transcript blocks and session rows (commits 6abe053
    - Path: web/src/pages/QueryEditorPage.tsx
      Note: Step 5 reduced background polling pressure while preserving active-source refreshes (commit 17600ec)
    - Path: web/src/pages/TranscriptViewerPage.tsx
      Note: Step 9 now composes the transcript view from summary metadata plus blocks (commit 0835f29)
    - Path: web/src/types/session.ts
      Note: Step 9 introduced a summary-detail type distinct from full transcript detail (commit 0835f29)
ExternalSources: []
Summary: Chronological diary of the performance study for the transcript viewer and broader web UI, including measurements and evidence-gathering commands.
LastUpdated: 2026-04-06T22:20:00-04:00
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

## Step 6: Split transcript block header from block body

This step turned the transcript block into an explicit performance boundary in code. Instead of one large `BlockCard` component holding both cheap summary UI and expensive turn/tool-call UI in a single file, the block now has a lightweight header and a lazily mounted body.

This refactor was partly about performance and partly about maintainability. The transcript optimization work had reached the point where the component structure itself was getting in the way of the next step. By making the block shell, header, and body more distinct, I created a cleaner foundation for virtualization and future tuning.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue executing the remaining performance tasks in order, with small commits and a diary trail.

**Inferred user intent:** Finish the execution side of the performance ticket rather than leaving the large structural transcript work as a future note.

**Commit (code):** `e6fa2c8` — `web: split transcript block header and body`

### What I did

- Added:
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/BlockHeader.tsx`
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/BlockBody.tsx`
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/types.ts`
- Refactored:
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/BlockCard.tsx`
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/TranscriptViewer.tsx`
- Moved the cheap summary/header UI into `BlockHeader`.
- Moved artifact, turn, and tool-call rendering into `BlockBody`.
- Kept the expensive body behind `Collapse ... unmountOnExit`.
- Added `content-visibility: auto` plus `containIntrinsicSize` around the mounted body wrapper as an incremental paint/layout hint.
- Ran:
  - `cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build`
- Committed the code change.

### Why

- The transcript block needed a sharper separation between always-mounted and expensive content.
- This makes the performance model easier to reason about and makes the following virtualization step much less invasive.

### What worked

- The build passed cleanly.
- The new structure reduced the amount of logic concentrated in `BlockCard.tsx`.
- The refactor did not require API or data-shape changes.

### What didn't work

- N/A — no build/runtime failures on this step.
- I did not create a separate measurement snapshot for this refactor alone because the next step (virtualization) was immediately dependent on it and more meaningful to measure.

### What I learned

- Once collapsed subtree unmounting was in place, the next limit was not only mounted DOM size; it was also component organization. The transcript tree became easier to optimize once the block boundary was explicit.

### What was tricky to build

- The tricky part was preserving behavior while moving rendering logic across files. The block still needed to preserve focus styling, annotation chips, and tool-call affordances exactly as before.
- Another subtle point was not accidentally losing the uncontrolled Storybook/demo behavior for `BlockCard`, which mattered once expansion control started shifting toward the parent for the next step.

### What warrants a second pair of eyes

- Whether the `content-visibility: auto` wrapper has any surprising interaction with focused target scrolling in edge cases.
- Whether the new component split leaves prop surfaces stable enough to keep memoization effective.

### What should be done in the future

- Keep state that must survive virtualization or unmount/remount boundaries in the parent viewer rather than in row-local state.

### Code review instructions

Review in this order:

1. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/BlockCard.tsx`
2. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/BlockHeader.tsx`
3. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/BlockBody.tsx`
4. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/types.ts`

Validation:

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build
```

### Technical details

The important architectural change is that `BlockCard` now acts primarily as a shell/coordinator:

- `BlockHeader` is always mounted and cheap.
- `BlockBody` is only mounted when the block is expanded or force-expanded.

## Step 7: Virtualize transcript block rendering

This was the biggest frontend structural optimization in the ticket. Instead of rendering every block row up front, the transcript viewer now window-renders only the visible block range plus overscan, while tracking measured row heights and scrolling focused blocks into view.

The result is the clearest transcript performance win after subtree unmounting. The saved measurement snapshot shows that the transcript route now mounts only a handful of blocks on initial load, and the route timings dropped dramatically again.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue through the remaining structural performance work, including the larger transcript rendering changes.

**Inferred user intent:** Finish the core transcript scalability work rather than stopping at cheap wins.

**Commit (code):** `6abe053` — `web: virtualize transcript block rendering`

### What I did

- Added:
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/shared/useVirtualList.ts`
- Updated:
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/TranscriptViewer.tsx`
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/BlockCard.tsx`
- Introduced a shared measured-window virtualization hook.
- Switched transcript block expansion state from row-local state toward viewer-managed state so virtualization would not lose important expansion behavior when rows unmount.
- Rendered only the visible transcript block range plus overscan, using top/bottom spacers.
- Added focused-block scroll-to-index behavior before the existing DOM-anchor `scrollIntoView` path.
- Ran:
  - `cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build`
  - `cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && node ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs > ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/04-step-7-transcript-virtualization-measurements.json`
- Committed the code change.

### Why

- The transcript route was still paying for every block row even after subtree unmounting.
- Virtualization is the main structural answer once wasted collapsed-content mounting has already been removed.

### What worked

- The build passed cleanly.
- The measurement snapshot was strongly positive:
  - mean transcript initial load: `763 ms`
  - mean transcript → annotations: `169 ms`
  - mean transcript back: `38 ms`
  - mounted blocks on the sampled state: `6`
  - DOM nodes: `1419`
- Focus navigation still had a viable path because the viewer now scrolls to the virtual row before trying to find the specific turn/tool-call anchor.

### What didn't work

- No build failure on the final implementation.
- One intermediate issue surfaced during the refactor: Storybook stories still passed `defaultExpanded` after `BlockCard` became more controlled. `npm run build` failed with TypeScript errors like:

```text
Object literal may only specify known properties, and 'defaultExpanded' does not exist in type ...
```

- I resolved this by restoring an uncontrolled fallback path in `BlockCard` while keeping the controlled mode for the viewer.

### What I learned

- Transcript virtualization required one prerequisite that was easy to underestimate: block expansion state had to stop living purely inside virtual rows.
- A small shared virtualization hook was enough for this app; I did not need to bring in a new external library to get the first strong win.

### What was tricky to build

- The hardest part was coordinating virtualization with deep-link/focus behavior. A tool-call target cannot be scrolled into view if its containing block is not currently rendered, so the viewer had to scroll the virtual row first and only then perform the anchor-level scroll.
- Dynamic heights also mattered because expanded blocks are much taller than collapsed ones. The measured-height hook plus overscan was a practical compromise.

### What warrants a second pair of eyes

- Whether very tall expanded blocks need additional measurement stabilization or finer-grained virtualization later.
- Whether the overscan settings are optimal across both small and large transcript sets.

### What should be done in the future

- If the transcript grows beyond block-level comfort again, the next frontier would be turn-level virtualization inside extremely large expanded blocks.

### Code review instructions

Review in this order:

1. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/shared/useVirtualList.ts`
2. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/TranscriptViewer.tsx`
3. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/BlockCard.tsx`

Validation:

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && node ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs
```

### Technical details

Measurement artifact:

- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/04-step-7-transcript-virtualization-measurements.json`

The virtualized transcript path now uses:

- measured item heights,
- top/bottom spacer boxes,
- parent-owned expansion state,
- pre-anchor virtual row scrolling for focused targets.

## Step 8: Add Session Browser virtualization

After transcript virtualization, I applied the same measured-window idea to the Session Browser table. The route was not the top hotspot, but it still rendered every filtered row into the table body, which was a clear scalability risk once archives grow.

The measured follow-up snapshot confirms the expected shape: the Session Browser now mounts far fewer rows and far fewer DOM nodes while preserving the same interaction model.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue the remaining performance tasks across the app, not only inside the transcript route.

**Inferred user intent:** Finish the full app-side execution plan, including secondary scalability surfaces.

**Commit (code):** `c4bb6ca` — `web: virtualize session browser rows`

### What I did

- Updated:
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/SessionBrowser/SessionBrowser.tsx`
- Reused the shared `useVirtualList` hook for the session table body.
- Virtualized the filtered session rows when the filtered set exceeds a threshold.
- Preserved row click-through and the query-button interaction.
- Ran:
  - `cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build`
  - `cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && node ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs > ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/05-step-8-session-browser-virtualization-measurements.json`
- Committed the code change.

### Why

- The Session Browser was not broken today, but it had the same “render everything” pattern as the earlier transcript implementation.
- This was a low-risk way to make archive growth less dangerous.

### What worked

- The build passed cleanly.
- The route measurement snapshot showed that the browser now mounted only a subset of rows:
  - mounted rows in the sampled state: `27`
  - DOM nodes: `1035`
- The row click and query-icon click behavior were preserved.

### What didn't work

- N/A — no build/runtime failures on this step.
- I did not add a dedicated browser-interaction smoke beyond the route measurement, so validation here is still lighter than the transcript path.

### What I learned

- The shared virtualization hook was general enough to pay off twice immediately.
- Even the simpler table route benefits materially in DOM size from the same windowing approach.

### What was tricky to build

- Virtualizing a `<TableBody>` is slightly awkward because it needs spacer rows rather than generic div spacers. The measured hook still worked, but the rendering shape had to respect table semantics.

### What warrants a second pair of eyes

- Whether the threshold (`filtered.length > 40`) is the right cutoff.
- Whether row height estimation remains stable enough when annotation badges wrap more than expected.

### What should be done in the future

- If the Session Browser gets richer row content, consider whether a grid/list component with built-in virtualization would be simpler long-term.

### Code review instructions

Review:

- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/SessionBrowser/SessionBrowser.tsx`
- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/shared/useVirtualList.ts`

Validation:

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && node ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs
```

### Technical details

Measurement artifact:

- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/05-step-8-session-browser-virtualization-measurements.json`

## Step 9: Split session summary metadata from transcript blocks at the API layer

This step completed the backend-shaping part of the execution plan. The server now exposes a summary-only session endpoint, and the transcript page fetches summary metadata separately from the heavy block payload.

The most important part of this step is not raw latency reduction on its own. It is that the API surface now expresses the real data boundary: session metadata is small, while transcript blocks are large. The follow-up measurement artifact confirms just how different those payload sizes are.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Finish the remaining execution work, including backend shaping where it is justified.

**Inferred user intent:** Complete the ticket rather than leaving the API-side optimization only as a design recommendation.

**Commit (code):** `0835f29` — `serve: split session summary and blocks endpoints`

### What I did

- Updated backend:
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/server.go`
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/handlers_sessions.go`
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/server_test.go`
- Updated frontend:
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/types/session.ts`
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/api/minitrace.ts`
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/pages/TranscriptViewerPage.tsx`
- Added a new route:
  - `GET /api/sessions/{id}/summary`
- Added a shared normalization helper for session summary detail on the server.
- Switched `TranscriptViewerPage` to fetch:
  - summary metadata via `useGetSessionSummaryQuery`
  - blocks via `useGetSessionBlocksQuery`
- Added a new API measurement script:
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/02-session-summary-blocks-split-perf.mjs`
- Ran:
  - `cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build`
  - `cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && go test ./cmd/go-minitrace/cmds/serve/...`
  - pre-commit also ran `go test ./...`
- Captured:
  - `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/06-step-9-summary-and-blocks-split-measurements.json`
- Committed the code change.

### Why

- The UI and server already knew that transcript blocks are the expensive part, but the API surface still bundled metadata and blocks together for the main detail route.
- Separating those concerns makes the contract clearer and gives the frontend more room to load and reason about them independently.

### What worked

- The frontend build passed.
- The new serve-package test passed.
- The full repo test suite passed via the pre-commit hook on the final code commit.
- The API measurement snapshot clearly showed the payload split:
  - summary endpoint mean size: `987 bytes`
  - blocks endpoint mean size: `1,801,428 bytes`
  - full detail endpoint mean size: `1,802,424 bytes`
- The summary endpoint is therefore dramatically smaller and correctly omits the `blocks` field.

### What didn't work

- My first serve-package test assertion used the wrong fixture values and failed twice:

```text
unexpected session title "Fixture Session"
unexpected source format "fixture"
```

- I corrected the test to match the actual fixture.
- I also attempted to reuse the Playwright route measurement script against a temporary standalone `serve` process, but the embedded non-dev frontend path was not reliable enough for that exact script in this environment. I therefore captured the step with a dedicated API-level measurement script instead, which was a better fit for this particular backend-shaping change.

### What I learned

- The data-shape split is real and large. The blocks payload dominates both bytes and backend work; the summary payload is tiny.
- For this step, an API-focused measurement was more informative than a browser-route measurement because the change was specifically about transport and endpoint separation.

### What was tricky to build

- The tricky part was not the route itself. It was validating the change in a way that matched the change. The generic browser timing script was built for route responsiveness, while this step was primarily about request shape and payload separation.
- Another subtle point was keeping the transcript page wiring simple: fetch summary and blocks separately, then compose the existing `SessionDetail` shape in the page layer.

### What warrants a second pair of eyes

- Whether the now-less-used `GET /api/sessions/{id}` full-detail route should remain as-is for compatibility or eventually become an internal/debug convenience.
- Whether the transcript page should eventually render the header immediately from summary data and stream blocks in visually afterward rather than waiting for both.

### What should be done in the future

- If desired, the next UX refinement would be to render the summary/header chrome while blocks are still loading.
- If the full-detail route stops being useful, consider de-emphasizing or deprecating it in favor of summary + blocks.

### Code review instructions

Review in this order:

1. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/server.go`
2. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/handlers_sessions.go`
3. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/server_test.go`
4. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/api/minitrace.ts`
5. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/types/session.ts`
6. `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/pages/TranscriptViewerPage.tsx`

Validation:

```bash
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web && npm run build
cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace && go test ./cmd/go-minitrace/cmds/serve/...
```

API-shape measurement:

```bash
BASE_URL=http://127.0.0.1:18081 \
SESSION_ID=019d0295-d06b-7033-b154-a991a94672b6 \
ITERATIONS=5 \
node ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/02-session-summary-blocks-split-perf.mjs
```

### Technical details

Measurement artifact:

- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/06-step-9-summary-and-blocks-split-measurements.json`

New endpoint:

```text
GET /api/sessions/{id}/summary
```

Representative API snapshot:

- summary endpoint mean duration: `31 ms`
- summary endpoint mean bytes: `987`
- blocks endpoint mean duration: `49 ms`
- blocks endpoint mean bytes: `1,801,428`
- full detail endpoint mean duration: `50 ms`
- full detail endpoint mean bytes: `1,802,424`
