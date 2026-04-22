---
Title: 'Proposed ticket: clarify DuckDB tool_calls access patterns and troubleshooting'
Ticket: MT-DUCKDB-TOOL-CALLS-DOCS
Status: active
Topics:
    - go-minitrace
    - documentation
    - analysis
    - minitrace
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../trace-analysis/ttmp/2026/04/21/TKT-2026-0422-hardware-research-methodology--hardware-research-documentation-methodology/design-doc/02-bug-tool-calls-duckdb-schema.md
      Note: External evidence base that narrowed the problem from a schema-bug claim to a docs ticket
    - Path: pkg/doc/analysis-guide.md
      Note: Updated adjacent examples so tool-call queries match the verified current schema
    - Path: pkg/doc/minitrace-schema.md
      Note: Target doc for clarifying input.file_path vs input.arguments on tool calls
    - Path: pkg/doc/query.md
      Note: Updated query overview page with JSON[] and 1-based indexing notes for tool_calls access
    - Path: pkg/doc/troubleshooting.md
      Note: Target doc for JSON[] anti-patterns and operator-precedence troubleshooting notes
    - Path: pkg/doc/writing-duckdb-queries.md
      Note: Primary tutorial targeted for clearer JSON[] and 1-based indexing guidance
    - Path: pkg/query/engine.go
      Note: Evidence that tool_calls is intentionally loaded as JSON[] and does not require an engine-ticket conclusion by itself
ExternalSources: []
Summary: Narrow repo-ticket proposal for improving docs and troubleshooting around DuckDB JSON[] tool_calls access patterns, based on evidence from transcript-analysis work.
LastUpdated: 2026-04-22T00:00:00Z
WhatFor: Define a small, defensible documentation ticket in go-minitrace without claiming the tool_calls schema itself is broken.
WhenToUse: Read before implementing doc changes or deciding whether a future ticket should touch engine code instead of docs.
---



# Proposed ticket: clarify DuckDB tool_calls access patterns and troubleshooting

## Recommendation

Open a **documentation/troubleshooting** ticket, not a schema-bug ticket.

The evidence gathered in the external research workspace shows:

- `tool_calls` is intentionally loaded as DuckDB `JSON[]`
- current `go-minitrace` already supports the normal access pattern:
  - `UNNEST(tool_calls) AS t(tc)`
  - `tc->>'tool_name'`
- the strongest earlier claim, that `tool_calls` is not queryable without extra casting, does not hold on current HEAD

So the right repo-local follow-up is to improve documentation around the real sharp edges rather than changing the load model.

## Problem statement

Users doing ad hoc DuckDB analysis can still get tripped up by a few non-obvious behaviors:

1. DuckDB list indexing is **1-based**
   - `tool_calls[1]` is the first element
   - `tool_calls[0]` returns `NULL`
2. `tool_calls` is a `JSON[]` list column, not a single JSON blob
   - `UNNEST(tool_calls)` works
   - but some intuitive container-level JSON-path attempts are misleading or return `NULL`
3. Nested tool input fields are easy to guess incorrectly
   - for many read/write/edit tool calls, the stable path is `input.file_path`
   - the raw argument path often lives at `input.arguments.path`
4. `->>` combined with `LIKE` can require parentheses in practice
   - e.g. `(tc->'input'->>'command') LIKE '%docmgr%'`
5. Tool-name casing should not be guessed from memory
   - the inspected Pi archive uses lowercase names such as `read`, `write`, `edit`, `bash`

These are exactly the kinds of issues that create “the schema is broken” impressions during exploratory SQL work.

## Non-goals

This ticket should **not**:

- redesign the DuckDB load schema
- normalize tool calls into a separate table
- change `pkg/query/engine.go`
- add compatibility casts or helper views unless docs prove insufficient later

## Proposed scope

### 1. Strengthen the DuckDB tutorial docs

Update:

- `pkg/doc/writing-duckdb-queries.md`
- `pkg/doc/query.md`

Add a short section near the existing `UNNEST(tool_calls)` guidance covering:

- `JSON[]` means list of JSON values
- recommended pattern: `UNNEST(tool_calls) AS t(tc)`
- example element access with both:
  - `tc->>'tool_name'`
  - `json_extract(tc, '$.tool_name')`
- 1-based list indexing example:
  - `tool_calls[1]`
  - not `tool_calls[0]`

### 2. Add explicit anti-patterns to troubleshooting

Update:

- `pkg/doc/troubleshooting.md`

Add a small subsection like “DuckDB JSON[] sharp edges” with concrete examples:

- `tool_calls[0]` returns `NULL`
- prefer `UNNEST(tool_calls)` over guessing container-level JSON paths
- if a `LIKE` filter on `->>` behaves strangely, wrap the extraction in parentheses
- check actual tool names and nested keys before assuming casing/path layout

### 3. Tighten schema-reference examples for tool-call inputs

Update:

- `pkg/doc/minitrace-schema.md`

Add a short note under tool-call input fields:

- `input.file_path` is the normalized path field when available
- `input.arguments` preserves adapter/tool-specific raw arguments
- analysts may need `input.arguments.path`, `input.arguments.query`, etc. depending on tool

### 4. Optionally refresh one or two built-in examples

If helpful, adjust one example in:

- `pkg/doc/duckdb-query-recipes.md`
- or `pkg/doc/analysis-guide.md`

so there is at least one visible recipe showing a field-path fallback like:

```sql
COALESCE(tc->'input'->>'file_path', tc->'input'->'arguments'->>'path')
```

This is optional, but it would make the docs more robust for exploratory users.

## Evidence base

The proposal is based on verified follow-up work in:

- `/home/manuel/code/wesen/trace-analysis/ttmp/2026/04/21/TKT-2026-0422-hardware-research-methodology--hardware-research-documentation-methodology/design-doc/02-bug-tool-calls-duckdb-schema.md`

Key outcomes from that investigation:

- `pkg/query/engine.go` explicitly loads `tool_calls` as `JSON[]`
- `UNNEST(tool_calls)` works on current HEAD
- several earlier failures came from incorrect local SQL/JS assumptions, not from a broken engine
- the remaining rough edges are documentation-quality issues, not yet engine bugs

## Acceptance criteria

This ticket is done when:

- `pkg/doc/writing-duckdb-queries.md` explicitly calls out 1-based indexing and the preferred `UNNEST(tool_calls)` pattern
- `pkg/doc/troubleshooting.md` includes a short anti-patterns section for JSON[] tool-call analysis
- `pkg/doc/minitrace-schema.md` clarifies `input.file_path` vs `input.arguments`
- at least one example in the docs shows a robust path extraction pattern for tool-call inputs
- no engine code changes are required

## Suggested implementation order

1. Update `pkg/doc/writing-duckdb-queries.md`
2. Update `pkg/doc/troubleshooting.md`
3. Update `pkg/doc/minitrace-schema.md`
4. Decide whether one recipe/example update is still needed in `analysis-guide.md` or `duckdb-query-recipes.md`
5. Validate help pages and cross-references

## Why this is the right ticket shape

This ticket is small, evidence-backed, and safe.

It avoids reopening the earlier over-claim that “tool_calls is broken,” while still addressing the real frustration users hit during exploratory analysis. It also preserves the option to open a deeper engine/adapter ticket later if new evidence appears from additional frameworks or archives.
