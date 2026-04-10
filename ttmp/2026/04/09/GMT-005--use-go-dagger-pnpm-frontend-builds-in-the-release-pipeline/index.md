---
Title: Use Go Dagger pnpm frontend builds in the release pipeline
Ticket: GMT-005
Status: complete
Topics:
    - backend
    - frontend
    - configuration
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Replace the release pipeline's npm-based frontend build path with a Go Dagger command that runs reproducible pnpm builds.
LastUpdated: 2026-04-09T21:51:27.780097645-04:00
WhatFor: Track the migration of release/frontend build plumbing from ad hoc Makefile npm commands to a Go Dagger-based pnpm builder.
WhenToUse: Use when reviewing or continuing the Dagger-based frontend build migration for go-minitrace releases.
---


# Use Go Dagger pnpm frontend builds in the release pipeline

## Overview

This ticket migrates the release/frontend build path away from `npm ci && npm run build` in the Makefile and toward a Go-based Dagger builder that performs `pnpm install --frozen-lockfile` plus `pnpm run build` in a containerized environment. The goal is to make the release path more reproducible, keep Node tooling out of the host assumptions for release builds, and simplify the Makefile so it delegates frontend embedding to one purpose-built command.

## Key Links

- Design doc: [design-doc/01-dagger-pnpm-release-pipeline-implementation-guide.md](./design-doc/01-dagger-pnpm-release-pipeline-implementation-guide.md)
- Diary: [reference/01-dagger-release-pipeline-diary.md](./reference/01-dagger-release-pipeline-diary.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **active**
