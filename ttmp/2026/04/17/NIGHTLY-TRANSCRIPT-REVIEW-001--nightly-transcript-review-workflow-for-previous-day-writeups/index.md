---
Title: Nightly transcript review workflow for previous-day writeups
Ticket: NIGHTLY-TRANSCRIPT-REVIEW-001
Status: active
Topics:
    - transcript-analysis
    - minitrace
    - documentation
    - go-minitrace
    - codex
    - analysis
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/minitracecmd/render.go
      Note: Renderer helper reuse for clay sqlDate/sqlDateTime
    - Path: pkg/minitracecmd/render_helpers.go
      Note: Local safe string helpers and merge/copy helpers
    - Path: pkg/minitracecmd/core/nightly/session-inventory.sql
      Note: Embedded nightly session inventory command
    - Path: pkg/minitracecmd/core/nightly/workspace-summary.sql
      Note: Embedded nightly workspace summary command
    - Path: pkg/minitracecmd/core/nightly/followup-candidates.sql
      Note: Embedded nightly follow-up candidate command
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-17T12:39:47-04:00
WhatFor: ""
WhenToUse: ""
---


# Nightly transcript review workflow for previous-day writeups

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- transcript-analysis
- minitrace
- documentation
- go-minitrace
- codex
- analysis

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
