---
Title: JS API Options for Minitrace
Ticket: GMT-006
Status: active
Topics:
    - backend
    - documentation
    - glazed
    - minitrace
    - go-minitrace
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/modules/database/database.go
      Note: Generic SQL module shape that informs the first minitrace JS module design
    - Path: go-go-goja/pkg/jsverbs/runtime.go
      Note: Existing Goja command bridge and Promise-aware runtime ownership pattern
    - Path: go-minitrace/cmd/go-minitrace/cmds/query/duckdb.go
      Note: Current DuckDB execution model that a JS API can call through or abstract over
    - Path: go-minitrace/pkg/minitracecmd/catalog.go
      Note: Catalog loading and source-kind dispatch that will need a JS branch
    - Path: go-minitrace/pkg/minitracecmd/parse_sql.go
      Note: Existing SQL-only parsing entrypoint that JS source handling would need to parallel or replace
    - Path: go-minitrace/pkg/minitracecmd/render.go
      Note: SQL rendering boundary that a JS command API may bypass or wrap
    - Path: go-minitrace/pkg/minitracecmd/types.go
      Note: Current command-spec shape and verb/alias constraints
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-20T17:56:39.398640866-04:00
WhatFor: ""
WhenToUse: ""
---


# JS API Options for Minitrace

## Executive Summary

`go-minitrace` can already discover and render repository-backed SQL verbs, and `go-go-goja` already provides a reusable Goja runtime, module registry, async owner-thread helpers, and a working `pkg/jsverbs` command bridge. What is missing is an elegant, domain-specific JavaScript surface for minitrace itself.

The right first step is not to expose raw DuckDB or raw Go structs directly. The better path is to define a small `minitrace` JS module on top of the existing Go runtime and then layer command authoring helpers on top of that module. That gives us a clean separation:

- Go owns archive loading, query execution, and runtime lifecycle.
- JS authors write scripts against a stable, discoverable minitrace API.
- `minitracecmd` can later accept both SQL-backed and JS-backed verbs through the same command catalog.

## Problem Statement

Today the command model is SQL-first:

- `pkg/minitracecmd/parse_sql.go` only understands `/* sqleton ... */` files.
- `pkg/minitracecmd/render.go` only renders SQL templates.
- `pkg/minitracecmd/catalog.go` only loads `.sql` verbs and `.alias.yaml` files.
- `cmd/go-minitrace/cmds/query/duckdb.go` loads archives and executes SQL directly against DuckDB.

That works well for query-centric workflows, but it is not enough for JS-backed scripts. A JS author needs a higher-level way to:

- access loaded sessions and related transcript data,
- run or compose queries without writing boilerplate DuckDB code,
- define a command/verb with typed inputs and output mode,
- and do that in a style that feels native to Goja rather than like a thin SQL shell.

`go-go-goja` already contains the pieces we need, but they are spread across separate abstractions:

- `modules/database` exposes a generic SQL API,
- `modules/timer` demonstrates async Promise settlement,
- `pkg/jsverbs` shows how JS files can become Glazed commands,
- `pkg/runtimebridge` and `pkg/runtimeowner` provide the lifecycle/async plumbing.

The missing piece is a minitrace-specific JS API surface that sits on top of those primitives.

## Proposed Solution

Build a dedicated Goja module for minitrace and keep the first version deliberately small.

### Core module shape

Start with a host-owned module, for example:

```js
const mt = require("minitrace");
```

That module should expose a few stable entrypoints:

- `openArchive(...)` / `loadArchive(...)`
- `query(sql, ...args)`
- `session(id)`
- `sessions()` or `rows()`
- `defineVerb(spec, handler)` or `verb(spec, handler)`
- helper utilities for filters, formatting, and minitrace field access

The module should return plain JS objects, arrays, and primitives. If an operation is asynchronous, return a Promise and settle it through the existing runtime-owner bridge.

### Host ownership model

The Go side should own:

- DuckDB connection lifecycle,
- archive loading,
- query validation,
- runtime creation and teardown,
- logging and cancellation.

JS should receive handles and helpers, not raw `*sql.DB` or `*sql.Conn` values.

That matches the existing `go-go-goja` pattern, where the runtime is explicit and modules cooperate with the owner thread rather than bypassing it.

### Recommended first-cut API

A good first slice is this trio:

1. **Data access**
   - `mt.loadArchive({ globs, tableName, dbPath })`
   - `mt.query(sql, params?)`
   - `mt.session(id)`

2. **Command authoring**
   - `mt.verb({ name, short, flags, arguments, tags, output }, fn)`
   - or `mt.defineVerb(...)` if we want a more explicit registration style

3. **Convenience helpers**
   - `mt.sql.string(...)`, `mt.sql.in(...)`, `mt.sql.like(...)`
   - `mt.sessions.filter(...)` or similar once a fluent layer exists

That is enough to support scripts that are both readable and easy to embed into `minitracecmd`.

## JS API Possibilities

Here are the most plausible API styles, ordered from lowest to highest abstraction.

| Option | Shape | Best for | Pros | Cons |
|---|---|---|---|---|
| 1. Procedural module | `mt.query(...)`, `mt.session(...)`, `mt.loadArchive(...)` | Small scripts, thin wrappers | Easy to implement, easy to document, easy to test | Still SQL-heavy |
| 2. Fluent query builder | `mt.open().sessions().filter(...).select(...).run()` | Reusable analysis scripts | Ergonomic, composable, hides DuckDB details | More machinery, more design work |
| 3. Functional pipeline | `mt.sessions().map(...).filter(...).groupBy(...).collect()` | JS-native data transforms | Feels natural to JS users, great for small script logic | Harder to optimize and validate |
| 4. Command DSL | `mt.verb({ ... }, fn)` | Future `jsverb` command authoring | Directly matches `minitracecmd` needs | Needs metadata schema and command compiler |
| 5. Hybrid host API | `const { db, log, ctx, query, command } = mt.host()` | Power-user embedding | Very flexible and embeddable | Largest surface, hardest to teach |

### My recommendation

Use a **two-layer design**:

- **Layer 1:** a tiny procedural `minitrace` module that gives scripts data access and safe query helpers.
- **Layer 2:** a command DSL built on top of that module for verb authoring.

That gives us one small API for data and one explicit API for commands, which is easier to evolve than trying to make one object do everything.

## Design Decisions

- Prefer plain JS data over custom class hierarchies.
- Prefer host-owned connections and runtimes over JS-owned resource creation.
- Keep the first version domain-specific to minitrace instead of trying to design a general-purpose SQL abstraction.
- Reuse `go-go-goja` runtime-owner and Promise patterns for async operations.
- Keep command metadata separate from execution logic so SQL verbs and JS verbs can share the same catalog shape later.
- Make the API discoverable through TypeScript declarations or generated docs once the shape stabilizes.

## Alternatives Considered

### 1) Expose only `modules/database`

This is the simplest option, but it is too low-level. It would make JS authors write SQL directly and would not solve the “elegant minitrace API” requirement.

### 2) Reuse the existing `pkg/jsverbs` macro style only

The current `__package__` / `__section__` / `__verb__` pattern is already useful, and it maps nicely to command discovery. However, it is a discovery/registration mechanism, not a rich minitrace data API. It is best treated as one possible command surface, not the whole solution.

### 3) Build a full ORM-like object model

A rich object graph would be tempting, but minitrace data is JSON-first and query-first. An ORM would add complexity without enough benefit.

### 4) Go straight to a full fluent DSL

This is attractive for ergonomics, but it is a lot of surface area before we even know which script patterns matter most. It is better as a second layer.

## Implementation Plan

1. **Survey the runtime primitives**
   - confirm the minitrace loading and query lifecycle
   - confirm which `go-go-goja` helpers we want to reuse

2. **Define a minimal `minitrace` module**
   - data loading/query helpers
   - a few convenience functions
   - logging and error conventions

3. **Add typed declarations and docs**
   - use the existing `go-go-goja` declaration-generation pattern where it helps
   - document the API with examples

4. **Add JS command authoring support**
   - let JS export or register a verb spec
   - compile that into the same `MinitraceCommandSpec` shape used by SQL commands

5. **Teach `minitracecmd` to load both kinds of sources**
   - SQL source files remain supported
   - JS source files register command specs through the runtime
   - aliases continue to work on the shared command model

6. **Add tests**
   - API-level tests for the new module
   - command-loading tests for JS verbs
   - smoke tests for CLI execution

## Open Questions

- Should JS command files export a spec object, call `defineVerb(...)`, or use the existing `__verb__` macro style?
- Should the first release allow direct SQL execution from JS, or only JS helpers that eventually emit SQL behind the scenes?
- Should the module expose read-only session objects only, or also mutation helpers for annotations and derived artifacts?
- Do we want the API to be synchronous-first, async-first, or mixed?
- How much of the API should be public documentation versus internal scaffolding for `minitracecmd`?

## References

- `go-minitrace/pkg/minitracecmd/types.go`
- `go-minitrace/pkg/minitracecmd/parse_sql.go`
- `go-minitrace/pkg/minitracecmd/render.go`
- `go-minitrace/pkg/minitracecmd/catalog.go`
- `go-minitrace/cmd/go-minitrace/cmds/query/duckdb.go`
- `go-go-goja/pkg/jsverbs/runtime.go`
- `go-go-goja/pkg/jsverbs/command.go`
- `go-go-goja/modules/database/database.go`
- `go-go-goja/modules/timer/timer.go`
- `go-go-goja/README.md`
