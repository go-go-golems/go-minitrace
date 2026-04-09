---
Title: Add sqleton-style verb query loading to go-minitrace
Ticket: GMT-002
Status: active
Topics:
    - backend
    - documentation
    - go-minitrace
    - minitrace
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Research/design ticket for bringing sqleton-style repository-backed SQL verbs and UI query forms into go-minitrace."
LastUpdated: 2026-04-08T17:34:25-04:00
WhatFor: "Track the investigation, design, validation, and delivery work for a future go-minitrace query-catalog / verb-form feature."
WhenToUse: "Read this ticket when planning or reviewing repository-backed query verbs, structured query forms, and sqleton-inspired query loading in go-minitrace."
---

# Add sqleton-style verb query loading to go-minitrace

## Overview

This ticket investigates how to bring sqleton-style repository-backed SQL command loading into go-minitrace so query definitions can be loaded from embedded and external repositories, exposed as CLI/query verbs, and surfaced in the web UI as structured query forms.

The main deliverables are:

- a detailed architecture / design / implementation guide,
- a follow-up implementation-plan guide centered on `MinitraceCommand` and glazed parameter-definition reuse,
- a chronological investigation diary,
- ticket bookkeeping and validation,
- and a reMarkable-ready document bundle.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

Research and design documentation are complete. Final bookkeeping, validation, and reMarkable delivery are tracked in `tasks.md` and `changelog.md`.

## Topics

- backend
- documentation
- go-minitrace
- minitrace

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- `design-doc/01-sqleton-style-verb-query-loading-for-go-minitrace-analysis-design-and-implementation-guide.md` — primary architecture/design guide
- `design-doc/02-minitracecommand-implementation-plan-with-glazed-parameter-definition-reuse.md` — follow-up implementation plan centered on `MinitraceCommand`
- `reference/01-investigation-diary.md` — chronological research diary
- `tasks.md` — delivery checklist
- `changelog.md` — key ticket updates
