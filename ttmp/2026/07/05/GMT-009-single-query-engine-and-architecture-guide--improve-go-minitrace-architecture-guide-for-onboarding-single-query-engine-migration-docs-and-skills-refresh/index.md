---
Title: 'Improve go-minitrace: architecture guide for onboarding, single query engine migration, docs and skills refresh'
Ticket: GMT-009-single-query-engine-and-architecture-guide
Status: active
Topics:
    - tooling
    - cli
    - documentation
    - diagnostics
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Architecture guide for onboarding plus the single-query-engine migration design (retire DuckDB; consolidate on normalized SQLite) and a docs/skills refresh plan. Grounded in the DOCMGR-200 field work - 240 converted sessions and file:line source reviews.
LastUpdated: 2026-07-05T19:41:58.067119914-04:00
WhatFor: ""
WhenToUse: ""
---

# Improve go-minitrace: architecture guide for onboarding, single query engine migration, docs and skills refresh

## Overview

Two deliverables, both grounded in the DOCMGR-200 field work (docmgr repo):

1. `design-doc/01-go-minitrace-analysis-design-and-implementation-guide.md` - intern-ready tour of every subsystem (adapters + measured fidelity matrix, schema, archives/manifests, both query engines, JS runtime, query commands, serve/web/proto, annotations, CI) plus assessment and a prioritized backlog including the docs/skills refresh.
2. `design-doc/02-single-query-engine-migrating-go-minitrace-off-the-dual-duckdb-sqlite-stack.md` - the decision (SQLite wins), the complete DuckDB dependency map (3 driver files, 14-column sessions_base, 12 mechanical SQL rewrites, ~50MB binary win), and a five-phase shippable migration plan.

Diary in `reference/01-investigation-diary.md`. Note: files are uncommitted - the parent gitdir is read-only in the authoring environment.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- tooling
- cli
- documentation
- diagnostics

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
