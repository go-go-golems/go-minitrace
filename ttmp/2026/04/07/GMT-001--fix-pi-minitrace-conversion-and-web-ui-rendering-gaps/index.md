---
Title: Fix Pi minitrace conversion and web UI rendering gaps
Ticket: GMT-001
Status: active
Topics:
    - pi
    - web-ui
    - conversion
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/go-minitrace/cmds/serve/handlers_sessions.go
      Note: Server - TurnResponse drops Thinking/Model/Usage (Issues 4
    - Path: pkg/adapters/pi/convert.go
      Note: Converter - tool result turn skip (Issue 1)
    - Path: pkg/adapters/pi/convert_test.go
      Note: Test - updated turn count 3→2
    - Path: pkg/minitrace/schema.go
      Note: Schema - Go structs for Turn
    - Path: web/src/components/TranscriptViewer/BlockBody.tsx
      Note: Frontend - markdown rendering
    - Path: web/src/components/TranscriptViewer/ToolCallRow.tsx
      Note: Frontend - summary extraction
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-07T18:02:12.58720616-04:00
WhatFor: ""
WhenToUse: ""
---


# Fix Pi minitrace conversion and web UI rendering gaps

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- pi
- web-ui
- conversion

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
