---
Title: Diary
Ticket: MT-DUCKDB-TOOL-CALLS-DOCS
Status: active
Topics:
    - go-minitrace
    - documentation
    - analysis
    - minitrace
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/doc/analysis-guide.md
      Note: Corrected tool frequency and failed-tool examples
    - Path: pkg/doc/minitrace-schema.md
      Note: Clarified tool input field access and tool-name casing guidance
    - Path: pkg/doc/query.md
      Note: Added high-level JSON[] access note near the sessions_base table
    - Path: pkg/doc/troubleshooting.md
      Note: Added JSON[] sharp-edge troubleshooting notes
    - Path: pkg/doc/writing-duckdb-queries.md
      Note: Main docs implementation file for JSON[] and UNNEST guidance
ExternalSources: []
Summary: Chronological notes for the repo-local docs proposal about DuckDB tool_calls access patterns.
LastUpdated: 2026-04-22T00:00:00Z
WhatFor: Resume context for the proposed narrow documentation ticket without rereading the entire external investigation first.
WhenToUse: Read before editing go-minitrace docs related to DuckDB JSON[] tool-call analysis.
---


# Diary

## 2026-04-22

Created this repo-local ticket after the external transcript-analysis investigation narrowed the issue substantially.

Important conclusion carried over:

- do **not** treat this as a confirmed schema/engine bug
- do treat it as a likely docs/troubleshooting improvement ticket

Evidence from the external workspace showed:

- `tool_calls` is explicitly loaded as `JSON[]`
- `UNNEST(tool_calls)` works on current HEAD
- direct element access like `tc->>'tool_name'` works after unnesting
- the most confusing failures came from a mix of:
  - 1-based DuckDB list indexing
  - container-vs-element JSON semantics
  - incorrect local SQL/JS assumptions about nested fields and tool-name casing

Planned next step in this repo:

- update the DuckDB docs only, unless fresh evidence appears that engine behavior is inconsistent across adapters/frameworks.

## 2026-04-22 — docs implementation

Implemented the docs-only version of the ticket.

Updated files:

- `pkg/doc/writing-duckdb-queries.md`
- `pkg/doc/troubleshooting.md`
- `pkg/doc/minitrace-schema.md`
- `pkg/doc/analysis-guide.md`
- `pkg/doc/query.md`

Key changes:

- clarified that `tool_calls`, `turns`, and `annotations` are loaded as DuckDB `JSON[]`
- emphasized the recommended `UNNEST(tool_calls) AS t(tc)` pattern
- documented DuckDB 1-based indexing with `tool_calls[1]` vs `tool_calls[0]`
- added safer examples for nested tool inputs:
  - `input.file_path`
  - `input.arguments.path`
  - `input.arguments.query`
- added a warning to parenthesize `->>` when combining it with `LIKE`
- corrected adjacent analysis examples so they use `tool_name`, `operation_type`, and `output.error` consistently

No engine code was changed.
