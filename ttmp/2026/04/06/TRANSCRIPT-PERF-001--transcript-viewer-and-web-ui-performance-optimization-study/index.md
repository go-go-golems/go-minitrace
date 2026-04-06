---
Title: Transcript viewer and web UI performance optimization study
Ticket: TRANSCRIPT-PERF-001
Status: active
Topics:
    - performance
    - frontend
    - react
    - web-ui
    - transcript-analysis
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-06T18:40:00-04:00
WhatFor: ""
WhenToUse: ""
---

# Transcript viewer and web UI performance optimization study

## Overview

This ticket studies the performance characteristics of the `go-minitrace` web UI, with special emphasis on large transcript rendering. The main deliverable is an intern-oriented design and implementation guide that explains the app structure, the current bottlenecks, the measured baseline, and a phased optimization plan.

Current status:

- architecture mapping complete
- baseline Playwright measurement script added
- baseline and follow-up measurement captures saved into `sources/`
- design doc and diary written
- Step 2 committed: keep transcript pane mounted across tab switches (`22aafff`)
- Step 3 committed: unmount collapsed transcript subtrees (`6bf9596`)
- Step 4 committed: memoize query result sorting (`7a6e30c`)
- Step 5 committed: reduce background query-editor polling (`17600ec`)
- next major work remains: transcript block header/body split and eventual virtualization

## Key Links

- **Primary design doc**: `design-doc/01-transcript-viewer-and-web-ui-performance-optimization-study-and-implementation-guide.md`
- **Diary**: `reference/01-investigation-diary.md`
- **Measurement script**: `scripts/01-web-ui-baseline-perf.mjs`
- **Captured baseline**: `sources/01-baseline-measurements.json`
- **Step 2 snapshot**: `sources/02-step-2-persistent-mount-measurements.json`
- **Step 3 snapshot**: `sources/03-step-3-unmount-on-exit-measurements.json`
- **Related Files**: See frontmatter `RelatedFiles`
- **External Sources**: See frontmatter `ExternalSources`

## Status

Current status: **active**

## Topics

- performance
- frontend
- react
- web-ui
- transcript-analysis

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
