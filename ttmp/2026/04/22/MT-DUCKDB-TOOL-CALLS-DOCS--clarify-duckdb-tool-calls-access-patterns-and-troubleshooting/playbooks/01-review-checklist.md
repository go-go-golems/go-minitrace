---
Title: Review checklist
Ticket: MT-DUCKDB-TOOL-CALLS-DOCS
Status: active
DocType: playbook
Intent: short-term
Topics:
  - go-minitrace
  - documentation
  - analysis
  - minitrace
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Quick checklist for reviewing the future docs-only implementation of the tool_calls DuckDB ticket.
LastUpdated: 2026-04-22T00:00:00Z
WhatFor: Keep the docs ticket small and prevent it from drifting into engine changes.
WhenToUse: Use during implementation and review of the docs-only follow-up.
---

# Review checklist

- Confirm no changes are made to `pkg/query/engine.go` unless new evidence requires it.
- Confirm all examples still use the working pattern `UNNEST(tool_calls) AS t(tc)`.
- Confirm at least one doc explicitly states DuckDB list indexing is 1-based.
- Confirm troubleshooting includes at least one anti-pattern example.
- Confirm nested tool input guidance mentions `input.file_path` and `input.arguments`.
