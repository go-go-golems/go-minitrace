---
Title: Clarify DuckDB tool_calls access patterns and troubleshooting
Ticket: MT-DUCKDB-TOOL-CALLS-DOCS
Status: active
Topics:
    - go-minitrace
    - documentation
    - analysis
    - minitrace
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/doc/minitrace-schema.md
      Note: Schema reference page in scope for this ticket
    - Path: pkg/doc/troubleshooting.md
      Note: Main troubleshooting page in scope for this ticket
    - Path: pkg/doc/writing-duckdb-queries.md
      Note: Main docs entrypoint likely to be updated by this ticket
ExternalSources: []
Summary: Proposed repo-local follow-up ticket to improve DuckDB JSON[] tool_calls documentation and troubleshooting without claiming the schema itself is broken.
LastUpdated: 2026-04-22T00:00:00Z
WhatFor: Track the proposed go-minitrace docs ticket around DuckDB tool-call analysis sharp edges.
WhenToUse: Use when reviewing or implementing documentation updates for tool_calls SQL querying.
---


# Clarify DuckDB tool_calls access patterns and troubleshooting

## Overview

This ticket captures a narrow, evidence-based docs proposal: improve how go-minitrace explains DuckDB `JSON[]` tool-call access patterns, common mistakes, and troubleshooting. The proposal deliberately avoids claiming that `tool_calls` loading is broken on current HEAD.

## Key Links

- **Primary proposal:** `design/01-proposed-ticket-duckdb-tool-calls-docs-and-troubleshooting.md`
- **Diary:** `reference/01-diary.md`
- **Tasks:** `tasks.md`
- **Changelog:** `changelog.md`

## Status

Current status: **active**

## Topics

- go-minitrace
- documentation
- analysis
- minitrace
