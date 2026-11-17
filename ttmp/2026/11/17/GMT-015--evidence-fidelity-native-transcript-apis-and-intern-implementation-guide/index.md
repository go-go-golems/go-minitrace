---
Title: Evidence fidelity native transcript APIs and intern implementation guide
Ticket: GMT-015
Status: active
Topics: [analysis, minitrace]
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/go-minitrace/cmds/query/run.go
      Note: CLI evidence and cache controls
    - Path: repo://pkg/minitracecmd/catalog.go
      Note: Catalog diagnostics and isolation
    - Path: repo://pkg/minitracedb/query.go
      Note: Loss diagnostics and strict query behavior
    - Path: repo://pkg/minitracejs/typescript.go
      Note: Public JavaScript declaration contracts
    - Path: repo://pkg/nativeevidence/inspect.go
      Note: Native request and compaction inspection
    - Path: repo://pkg/nativeevidence/reader.go
      Note: Bounded streaming and exact source coordinates
ExternalSources: []
Summary: ""
LastUpdated: 2026-11-17T14:01:35.276716-05:00
WhatFor: ""
WhenToUse: ""
---

# Evidence fidelity and native transcript APIs

Implementation and intern onboarding for [upstream issue 34](https://github.com/go-go-golems/go-minitrace/issues/34). Branch `feat/evidence-fidelity`, based on `19d18b9`.

- [Intern architecture and implementation guide](design-doc/01-trustworthy-transcript-analysis-architecture-implementation-and-intern-guide.md)
- [Detailed diary](reference/01-implementation-diary.md)
- [Query fidelity](reference/02-query-fidelity-notes.md)
- [Provenance](reference/03-provenance-notes.md)
- [Native evidence](reference/04-native-evidence-notes.md)

Reproduction scripts are in `scripts/`; validation receipts are in `sources/`. The guide explicitly separates delivered fixes from deferred scanner and replay work. Full-suite validation retains two confirmed baseline configuration failures; focused new regressions pass.

Planned delivery: `/ai/2026/11/17/GMT-015`, document `GMT 015 Evidence Fidelity and Intern Guide`. Upload success will be reported in the final session receipt; all commits precede upload at the user’s request.
