---
Title: Fix query commands archive runtime flags
Ticket: MT-ARCHIVE-GLOB-QUERY-COMMANDS
Status: active
Topics:
    - go-minitrace
    - minitrace
    - analysis
    - documentation
    - glazed
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/go-minitrace/cmds/query/command_runtime.go
      Note: Runtime flag decoding and archive loading evidence
    - Path: cmd/go-minitrace/cmds/query/commands.go
      Note: CLI command-tree entrypoint and updated long help
    - Path: cmd/go-minitrace/cmds/query/runtime_section.go
      Note: Definition of runtime settings exposed to structured query commands
    - Path: pkg/doc/structured-query-commands.md
      Note: Primary user-facing documentation for the new JS path rule
    - Path: pkg/minitracecmd/parse_javascript.go
      Note: Primary implementation file for collapsed self-named JS paths
ExternalSources: []
Summary: We initially suspected a real archive-glob/runtime-flag bug in structured query commands, but the root cause was a misleading JS command-path shape. The implemented fix collapses redundant self-named single-verb JS file-stem paths during minitrace command creation, avoiding the confusing doubled path and eliminating the misleading Cobra flag failure.
LastUpdated: 2026-04-22T00:00:00Z
WhatFor: Track the analysis and implementation for the structured query-command path/runtime-flag issue.
WhenToUse: Use this ticket when reviewing or extending JS-backed structured query command pathing, help text, or execution behavior.
---


# Fix query commands archive runtime flags

## Overview

This ticket documents and implements the fix for the misleading `query commands ... --archive-glob` failure that appeared while invoking custom JS-backed structured query commands. The original report looked like a runtime flag bug, but the underlying issue was a redundant doubled JS command path for self-named single-verb files. The final implementation fixes that path shape in `minitracecmd` rather than in Cobra or go-go-goja.

## Key Links

- **Primary analysis:** `design-doc/01-query-commands-runtime-flags-architecture-diagnosis-and-fix-plan.md`
- **Diary:** `reference/01-diary.md`
- **Reproduction scripts:** `scripts/01-reproduce-js-group-flag-confusion.sh`, `scripts/02-inspect-leaf-vs-group-help.sh`, `scripts/03-locate-runtime-flag-plumbing.sh`

## Status

Current status: **active**

## Topics

- go-minitrace
- query
- duckdb
- cli
- js
- archive-glob

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
