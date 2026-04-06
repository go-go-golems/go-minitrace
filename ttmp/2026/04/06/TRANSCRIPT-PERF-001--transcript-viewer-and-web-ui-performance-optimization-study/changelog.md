# Changelog

## 2026-04-06

- Initial workspace created


## 2026-04-06

Created the transcript/web UI performance optimization ticket, added a repeatable Playwright baseline measurement script, captured current route timings/DOM counts, and wrote a detailed intern-oriented optimization guide plus diary.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/design-doc/01-transcript-viewer-and-web-ui-performance-optimization-study-and-implementation-guide.md — Primary design and implementation guide
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/reference/01-investigation-diary.md — Chronological diary of the investigation step
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs — Repeatable baseline measurement script for Session Browser
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/01-baseline-measurements.json — Captured baseline route timings and DOM counts for the current dev stack


## 2026-04-06

Step 2: stabilized transcript tab switches by keeping the transcript pane mounted and memoizing heavy transcript rows (commit 22aafff).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/02-step-2-persistent-mount-measurements.json — Post-Step-2 measurement snapshot
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/BlockCard.tsx — Memoize heavy transcript block rows
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/ToolCallRow.tsx — Memoize heavy tool-call rows
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/TranscriptViewer.tsx — Keep transcript and annotation panes mounted across tab switches


## 2026-04-06

Step 3: unmounted collapsed transcript block and tool-call subtrees; post-change measurement showed a much smaller mounted DOM and faster transcript load path (commit 6bf9596).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/sources/03-step-3-unmount-on-exit-measurements.json — Post-Step-3 measurement snapshot
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/BlockCard.tsx — Use unmountOnExit for collapsed blocks
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/TranscriptViewer/ToolCallRow.tsx — Use unmountOnExit for collapsed tool-call details


## 2026-04-06

Step 4: memoized query result sorting so unrelated rerenders do not re-sort the full result set (commit 7a6e30c).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/components/QueryEditor/ResultsTable.tsx — Memoize sortedRows by rows and sort state


## 2026-04-06

Step 5: reduced background query-editor polling pressure by polling the active source more frequently, the inactive source more slowly, and skipping polling while unfocused (commit 17600ec).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/pages/QueryEditorPage.tsx — Active/inactive polling split with skipPollingIfUnfocused

