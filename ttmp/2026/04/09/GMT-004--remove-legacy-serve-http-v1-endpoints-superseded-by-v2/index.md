---
Title: Remove legacy serve HTTP v1 endpoints superseded by v2
Ticket: GMT-004
Status: complete
Topics:
    - backend
    - documentation
    - minitrace
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Remove obsolete non-v2 serve endpoints while deliberately keeping POST /api/query as the one remaining JSON-native route.
LastUpdated: 2026-04-09T21:27:25.184597598-04:00
WhatFor: Track the cleanup of duplicate serve HTTP route families that the first-party frontend no longer uses.
WhenToUse: Use when reviewing or continuing the HTTP API cleanup that removes legacy v1-style serve endpoints superseded by protobuf-backed v2 routes.
---


# Remove legacy serve HTTP v1 endpoints superseded by v2

## Overview

This ticket removes the old `go-minitrace serve` HTTP endpoints that are now duplicated by the protobuf-backed `/api/v2/...` surface. The cleanup is intentionally narrow: session, saved-query/preset, and annotation legacy routes are removed, while `POST /api/query` remains because the current frontend still uses it and earlier protobuf-rollout docs explicitly kept it as the JSON-native exception.

## Key Links

- Design doc: [design-doc/01-legacy-serve-http-cleanup-implementation-guide.md](./design-doc/01-legacy-serve-http-cleanup-implementation-guide.md)
- Diary: [reference/01-cleanup-implementation-diary.md](./reference/01-cleanup-implementation-diary.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Current focus

1. Remove dead route registrations from `cmd/go-minitrace/cmds/serve/server.go`.
2. Delete the corresponding dead v1 handler methods while preserving shared helper/model code still used by v2.
3. Update tests, mocks, and docs so the repo reflects the cleaned surface instead of carrying stale `/api/...` examples.

## Non-goals

- Do **not** remove `POST /api/query` in this ticket.
- Do **not** redesign the protobuf schemas or frontend API adapters beyond what is needed to reflect the removed routes.
- Do **not** change structured query-command execution routes; they are already v2-only.

## Status

Current status: **active**

## Topics

- backend
- documentation
- minitrace

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.
