---
Title: Align Go and TypeScript API types with protobuf schema-first contracts
Ticket: PROTO-API-001
Status: active
Topics:
    - backend
    - frontend
    - minitrace
    - go-minitrace
    - documentation
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/go-minitrace/cmds/serve/server.go
      Note: Current route table and JSON transport helpers.
    - Path: cmd/go-minitrace/cmds/serve/handlers_sessions.go
      Note: Current handwritten session API DTO layer.
    - Path: cmd/go-minitrace/cmds/serve/handlers_annotations.go
      Note: Current handwritten annotation API layer.
    - Path: cmd/go-minitrace/cmds/serve/handlers_queries.go
      Note: Saved-query metadata plus dynamic query result surface.
    - Path: web/src/api/minitrace.ts
      Note: Frontend RTK Query layer that will be migrated to generated decoders.
    - Path: web/src/types/session.ts
      Note: Current handwritten frontend transport types for sessions and annotations.
    - Path: web/src/types/query.ts
      Note: Current handwritten frontend transport types for saved-query metadata and query results.
ExternalSources: []
Summary: Schema-first ticket for aligning go-minitrace backend/frontend API contracts with protobuf, Buf codegen, protojson, and generated TypeScript decoders.
LastUpdated: 2026-04-06T20:35:00-04:00
WhatFor: Provide a focused ticket workspace for planning and implementing protobuf-backed API contract alignment.
WhenToUse: Use when working on schema-first API contracts for the go-minitrace web frontend and serve backend.
---

# Align Go and TypeScript API types with protobuf schema-first contracts

## Overview

This ticket studies and implements a schema-first API contract layer for `go-minitrace`, with the goal of replacing duplicated handwritten backend/frontend transport types with protobuf-generated Go and TypeScript contracts.

Current status:

- ticket workspace created
- current API duplication analyzed
- detailed design and implementation guide written
- phased task list created
- implementation work not yet started beyond planning/scaffolding preparation

## Key Links

- **Primary design doc**: `design-doc/01-protobuf-schema-first-api-alignment-analysis-and-implementation-guide.md`
- **Diary**: `reference/01-investigation-diary.md`
- **Tasks**: `tasks.md`
- **Changelog**: `changelog.md`
- **Related Files**: See frontmatter `RelatedFiles`

## Status

Current status: **active**

## Topics

- backend
- frontend
- minitrace
- go-minitrace
- documentation

## Tasks

See [tasks.md](./tasks.md) for the execution plan.

## Changelog

See [changelog.md](./changelog.md) for recent changes.

## Structure

- design/ - Architecture and design documents
- reference/ - Diaries and API-reference notes
- playbooks/ - Operational procedures
- scripts/ - Temporary code and validation helpers
- various/ - Working notes and experiments
- archive/ - Deprecated or reference-only artifacts
