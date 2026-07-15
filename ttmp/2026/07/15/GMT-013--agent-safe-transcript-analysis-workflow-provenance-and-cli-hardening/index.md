---
Title: Agent-safe transcript analysis workflow provenance and CLI hardening
Ticket: GMT-013
Status: active
Topics:
    - go-minitrace
    - minitrace
    - documentation
    - architecture
    - conversion
    - transcript-analysis
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/07/15/GMT-013--agent-safe-transcript-analysis-workflow-provenance-and-cli-hardening/design-doc/01-agent-safe-transcript-analysis-architecture-and-implementation-guide.md
      Note: Primary architecture and implementation plan
    - Path: repo://ttmp/2026/07/15/GMT-013--agent-safe-transcript-analysis-workflow-provenance-and-cli-hardening/reference/01-investigation-diary.md
      Note: Chronological evidence, failures, decisions, and review guidance
ExternalSources: []
Summary: Design ticket for deterministic transcript attribution, Codex child/parent identity correctness, collision-safe archives, conversion and query receipts, archive validation, valid structured output, strict execution semantics, and consolidation of existing help pages.
LastUpdated: 2026-07-15T18:30:00-04:00
WhatFor: Track the evidence-backed design and future implementation of an agent-safe go-minitrace workflow.
WhenToUse: Start here when reviewing or resuming GMT-013.
---


# Agent-safe transcript analysis workflow provenance and CLI hardening

## Overview

GMT-013 converts lessons from two isolated external-agent evaluations into an implementation-ready go-minitrace plan. The highest-priority defect is Codex child identity corruption: later replayed parent metadata can replace the child ID, after which the archive writer silently overwrites by normalized ID. The ticket also addresses reproducible query evidence, archive/manifests verification, valid empty JSON output, partial/truncated process semantics, repository-backed attribution, and documentation duplication.

No product code is changed by this ticket. The proposed implementation is phased, begins with minimized regression fixtures, and explicitly separates P0 correctness from P1 workflow improvements and P2 schema semantics.

## Key links

- [Architecture and implementation guide](./design-doc/01-agent-safe-transcript-analysis-architecture-and-implementation-guide.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Primary decisions

- First valid Codex session header owns child identity; parent identity is lineage.
- Conflicting archive IDs fail by default; identical source fingerprints reconvert idempotently.
- Conversion and query provenance use versioned JSON sidecar receipts, not a new workflow database.
- Archive/manifests checks extend the existing `validate` command.
- Empty JSON is fixed at the formatter boundary, never with fake rows.
- New documentation is integrated into canonical existing pages; four overlapping/legacy pages are planned for removal.

## Status

- Research and design: complete.
- Documentation validation and reMarkable delivery: complete.
- Product implementation: not started; see the future phases in `tasks.md`.

## Review order

1. Read design sections 1–3 for evidence and current failure mechanism.
2. Read sections 5–9 for contracts, APIs, documentation ownership, and decisions.
3. Read sections 10–11 for implementation phases and tests.
4. Read the diary before beginning any implementation phase.
