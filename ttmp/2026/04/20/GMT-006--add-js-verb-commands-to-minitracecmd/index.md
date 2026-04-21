---
Title: Add JS verb commands to minitracecmd
Ticket: GMT-006
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
Summary: "Research and design workspace for adding JS-backed verbs to minitracecmd and defining a Goja API on top of minitrace."
LastUpdated: 2026-04-20T17:56:29.51865951-04:00
WhatFor: "Track the investigation, API options, and eventual implementation slices for JS and SQL command coexistence."
WhenToUse: "Read this ticket when planning or reviewing JS-backed minitrace verbs, Goja module APIs, or a future mixed SQL/JS command catalog."
---

# Add JS verb commands to minitracecmd

## Overview

This ticket explores how to let `minitracecmd` load and run both SQL-backed verbs and JS-backed verbs. The immediate focus is the Goja API surface on top of minitrace: a small, elegant module for data access plus a command-definition layer that can feed the same command catalog as the existing SQL path.

The workspace tracks the investigation, the design alternatives, and the follow-up implementation slices.

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
