---
Title: Add scanner-first JS verb command support to minitracecmd
Ticket: GMT-007
Status: active
Topics:
    - backend
    - documentation
    - glazed
    - minitrace
    - go-minitrace
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Research and implementation-planning ticket for adding scanner-first JS command files to minitracecmd while preserving the current SQL-like catalog lifecycle."
LastUpdated: 2026-04-20T18:33:27.33008471-04:00
WhatFor: "Track the architecture analysis, intern-oriented design guide, and delivery artifacts for scanner-first JS verb support in go-minitrace."
WhenToUse: "Read this ticket when implementing, reviewing, or onboarding onto the scanner-first JS command integration between go-minitrace and go-go-goja."
---

# Add scanner-first JS verb command support to minitracecmd

## Overview

This ticket documents how to extend `minitracecmd` so `.js` and `.cjs` command files are scanned up front, compiled into normal `MinitraceCommand` values, and executed only when invoked. The guiding constraint is to preserve the successful shape of the current SQL command system: discovery first, registration second, execution last.

The primary deliverable is a detailed analysis, design, and implementation guide written for a new engineer joining the codebase. It maps the current system, explains the required glue between `go-minitrace` and `go-go-goja/pkg/jsverbs`, and proposes a phased plan for implementation and testing.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- backend
- documentation
- glazed
- minitrace
- go-minitrace

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
