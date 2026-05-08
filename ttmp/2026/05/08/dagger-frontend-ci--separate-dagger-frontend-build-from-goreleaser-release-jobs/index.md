---
Title: Separate Dagger frontend build from GoReleaser release jobs
Ticket: dagger-frontend-ci
Status: active
Topics:
    - ci
    - deployment
    - frontend
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: .github/workflows/release.yaml
      Note: GoReleaser split-build workflow that fails on macOS
    - Path: .goreleaser.yaml
      Note: GoReleaser config with before hooks including go generate
    - Path: cmd/build-web/main.go
      Note: Dagger-based frontend build that requires container runtime
    - Path: cmd/go-minitrace/cmds/serve/generate.go
      Note: go:generate directive that triggers Dagger build
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-08T06:03:53.478946826-04:00
WhatFor: ""
WhenToUse: ""
---


# Separate Dagger frontend build from GoReleaser release jobs

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- ci
- deployment
- frontend

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
