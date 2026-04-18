---
Title: Cross-framework adapter field preservation and schema promotion
Ticket: ADAPTER-FIELDS-001
Status: active
Topics:
    - minitrace
    - adapters
    - codex
    - claude-code
    - pi
    - schema
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/analysis/01-cross-framework-field-matrix.md
      Note: Source-backed matrix of candidate fields across Pi, Codex, and Claude Code
    - Path: pkg/minitrace/schema.go
      Note: First-wave schema changes will add exit_code and justification here
    - Path: pkg/adapters/codex/convert.go
      Note: Codex adapter is the first implementation target for exit_code and justification
ExternalSources: []
Summary: "Separate ticket for source-backed adapter field preservation and first-wave schema promotion work across Pi, Codex, and Claude Code."
LastUpdated: 2026-04-18T00:00:00Z
WhatFor: "Track the field-matrix research and the first implementation slice for exit_code and justification."
WhenToUse: "Use when working on schema promotion or metadata preservation across the adapter set."
---

# Cross-framework adapter field preservation and schema promotion

## Overview

This ticket owns the follow-up work that came out of the Pi `isError` investigation but is broader than the Pi adapter itself. The immediate purpose is to validate which candidate fields really exist in Pi, Codex, and Claude Code raw transcripts, and then implement the first two low-risk schema promotions backed by that evidence: `tool_calls[].output.exit_code` and `tool_calls[].input.justification`.

## Key Links

- Analysis: [analysis/01-cross-framework-field-matrix.md](./analysis/01-cross-framework-field-matrix.md)
- Diary: [reference/01-diary.md](./reference/01-diary.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **active**

## Topics

- minitrace
- adapters
- codex
- claude-code
- pi
- schema

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- analysis/ - Field matrix and research notes
- reference/ - Diary and continuation docs
- scripts/ - Ticket-local research/verification scripts
- sources/ - Captured research artifacts
- design/ - Future schema or metadata design docs
- playbooks/ - Repeatable validation flows
- various/ - Scratch notes
- archive/ - Deprecated artifacts
