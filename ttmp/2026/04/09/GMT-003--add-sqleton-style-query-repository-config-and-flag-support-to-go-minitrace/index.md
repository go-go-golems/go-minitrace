---
Title: Add sqleton-style query repository config and flag support to go-minitrace
Ticket: GMT-003
Status: complete
Topics:
    - backend
    - configuration
    - go-minitrace
    - minitrace
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Follow-up ticket for loading sqleton-style query-command repositories from app config, environment variables, and repeated CLI flags in both serve and query-command surfaces.
LastUpdated: 2026-04-09T17:26:41.085522421-04:00
WhatFor: Track the design and implementation work needed to move from an embedded-only command catalog to embedded plus configurable external query-command repositories.
WhenToUse: Read this ticket when implementing repository config/env/flag support, source-root precedence, or external command-repository overrides for MinitraceCommand catalogs.
---


# Add sqleton-style query repository config and flag support to go-minitrace

## Overview

This ticket is the next sqleton-integration milestone after `GMT-002`. The command catalog, API, and UI are already in place, but repository discovery is still embedded-only. The goal here is to add sqleton-style repository configuration so go-minitrace can compose embedded commands with external repository roots resolved from config, environment variables, and repeated CLI flags.

The main deliverables are:

- a detailed implementation plan for repository config/flag support,
- a granular task list covering shared repository resolution, CLI integration, serve integration, precedence tests, and docs,
- and the follow-up implementation work when this ticket is picked up.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- backend
- configuration
- go-minitrace
- minitrace

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
