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
    - /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs
    - /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/01-baseline-measurements.json
ExternalSources: []
Summary: Chronological diary of the performance study for the transcript viewer and broader web UI, including measurements and evidence-gathering commands.
LastUpdated: 2026-04-06T17:20:00-04:00
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
