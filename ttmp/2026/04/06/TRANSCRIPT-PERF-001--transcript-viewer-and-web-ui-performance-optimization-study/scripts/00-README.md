---
Title: Performance investigation scripts
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
RelatedFiles: []
ExternalSources: []
Summary: Script index for the performance investigation ticket.
LastUpdated: 2026-04-06T18:55:00-04:00
WhatFor: Explain the purpose and usage of the ticket-local measurement scripts.
WhenToUse: Use when rerunning baseline measurements or extending the ticket with more profiling scripts.
---

# Performance Investigation Scripts

These scripts support the `TRANSCRIPT-PERF-001` ticket.

## Scripts

| File | Purpose |
|------|---------|
| `01-web-ui-baseline-perf.mjs` | Measure baseline page-load and transcript tab-switch timings against a running dev stack using Playwright |
| `02-session-summary-blocks-split-perf.mjs` | Measure summary-vs-blocks-vs-full-detail API timings and payload sizes for the backend shaping step |

## Usage

From the repo root:

```bash
node ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs
```

Environment overrides:

- `BASE_URL` — frontend base URL (default `http://127.0.0.1:5174`)
- `API_URL` — backend API base URL (default `http://127.0.0.1:8080`)
- `SESSION_ID` — session to profile (default `019d0295-d06b-7033-b154-a991a94672b6`)
- `ITERATIONS` — repeat count (default `3`)

Example:

```bash
BASE_URL=http://127.0.0.1:5174 \
API_URL=http://127.0.0.1:8080 \
SESSION_ID=019d0295-d06b-7033-b154-a991a94672b6 \
ITERATIONS=5 \
node ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/01-web-ui-baseline-perf.mjs
```

For the backend summary/body split measurement:

```bash
BASE_URL=http://127.0.0.1:18081 \
SESSION_ID=019d0295-d06b-7033-b154-a991a94672b6 \
ITERATIONS=5 \
node ttmp/2026/04/06/TRANSCRIPT-PERF-001--transcript-viewer-and-web-ui-performance-optimization-study/scripts/02-session-summary-blocks-split-perf.mjs
```
