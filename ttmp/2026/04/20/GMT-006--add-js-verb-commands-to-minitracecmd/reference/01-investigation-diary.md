---
Title: Investigation Diary
Ticket: GMT-006
Status: active
Topics:
    - backend
    - documentation
    - glazed
    - minitrace
    - go-minitrace
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/modules/timer/timer.go
      Note: Observed async Promise settlement pattern for future minitrace async helpers
    - Path: go-go-goja/pkg/jsverbs/runtime.go
      Note: Observed the JS command bridge that already demonstrates a runtime-owned execution pattern
    - Path: go-minitrace/cmd/go-minitrace/cmds/query/command_runtime.go
      Note: Observed query runtime glue and the current execution path for loaded archives
    - Path: go-minitrace/cmd/go-minitrace/cmds/query/duckdb.go
      Note: Observed DuckDB command shape and runtime settings
    - Path: go-minitrace/pkg/minitracecmd/catalog.go
      Note: Observed command-source loading and where JS sources could be introduced
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-20T17:56:39.403527426-04:00
WhatFor: ""
WhenToUse: ""
---


# Investigation Diary

## Goal

Capture the investigation behind adding JS-backed verb commands to `minitracecmd`, including the ticket workspace setup, the current-state codebase survey, and the API shapes worth considering for a future Goja layer on top of minitrace.

## Step 1: Ticket Workspace Creation

I created a new docmgr ticket workspace for the JS-verbs effort and used it to establish the initial planning documents. The ticket is intentionally narrow: the near-term goal is not to implement the runtime yet, but to get the design space organized so the eventual JS API can support both script-style analysis and `minitracecmd` verbs.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to add support for jsverb commands in minitracecmd, so that they can be both sql and js commands.

For that we first need to have a go-go-goja // goja elegant JS API on top of minitrace.

Create the ticket, then analyze the codebase and give me a set of different JS API possibilities to build scripts on top of what we have.

Keep a diary as you work."

**Assistant interpretation:** Create a new docmgr ticket, inspect the repository for the current SQL/JS command architecture, and record the investigation in a diary while proposing several possible JS API shapes.

**Inferred user intent:** Prepare the design space for adding JS-backed commands to `minitracecmd` without jumping straight into implementation.

### What I did
- Created ticket `GMT-006` with `docmgr ticket create-ticket`.
- Created the ticket workspace at `ttmp/2026/04/20/GMT-006--add-js-verb-commands-to-minitracecmd`.
- Added the first two ticket documents:
  - `design-doc/01-js-api-options-for-minitrace.md`
  - `reference/01-investigation-diary.md`

### Why
- The feature spans both command loading and Goja runtime design, so it needs an explicit workspace before implementation begins.
- A ticket workspace gives us a stable place to store decisions, alternatives, and later implementation notes.

### What worked
- `docmgr ticket create-ticket` succeeded and populated the workspace skeleton.
- `docmgr doc add` created both the design doc and the diary with the expected metadata.

### What didn't work
- Nothing blocked the ticket setup.

### What I learned
- The workspace should stay centered on `minitracecmd`, but the API design depends heavily on `go-go-goja` runtime patterns.
- The ticket should treat “JS commands” as a command-authoring problem, not only a query-execution problem.

### What was tricky to build
- Choosing the ticket boundary: the first instinct is to jump into implementation, but the codebase suggests a design-first step because the JS API needs to support both script usage and verb registration.

### What warrants a second pair of eyes
- Whether the ticket should prioritize a command DSL first or a data/query module first.
- Whether the eventual JS verb surface should mirror the existing `__verb__` macro style or move to explicit exported functions.

### What should be done in the future
- Populate the ticket tasks with concrete investigation and implementation slices.
- Connect the new documents to the specific source files that shaped the design.
- Implement the chosen JS API slice in `go-go-goja` after the design is settled.

### Code review instructions
- Start with the ticket workspace under `ttmp/2026/04/20/GMT-006--add-js-verb-commands-to-minitracecmd`.
- Review the `design-doc/01-js-api-options-for-minitrace.md` first; it frames the implementation choices.
- Validate the workspace with `docmgr doctor --ticket GMT-006 --stale-after 30`.

### Technical details
- Ticket ID: `GMT-006`
- Ticket title: `Add JS verb commands to minitracecmd`
- Initial docs created: design doc + diary

## Step 2: Codebase Survey and API Option Mapping

I inspected the current `minitracecmd` and `go-go-goja` surfaces to see where a JS API can attach cleanly. The codebase already has a strong split between command discovery, SQL rendering, DuckDB execution, and Goja runtime ownership, so the main design question is the shape of the API layer that sits above those primitives.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Survey the codebase and extract practical JS API options that could power minitrace scripts and future JS verbs.

**Inferred user intent:** Understand the viable API shapes before committing to one implementation path.

### What I did
- Read the core command-model files in `go-minitrace/pkg/minitracecmd/`:
  - `types.go`
  - `parse_sql.go`
  - `render.go`
  - `catalog.go`
  - `source_kind.go`
  - `errors.go`
- Read the query runtime glue in `go-minitrace/cmd/go-minitrace/cmds/query/`.
- Read the existing Goja surface in `go-go-goja/`:
  - `README.md`
  - `modules/common.go`
  - `modules/database/database.go`
  - `modules/timer/timer.go`
  - `pkg/runtimebridge/runtimebridge.go`
  - `pkg/jsverbs/runtime.go`
  - `pkg/jsverbs/command.go`
  - `pkg/jsverbs/binding.go`
  - `pkg/jsverbs/model.go`
- Confirmed there is already a JS command-discovery pattern in `pkg/jsverbs` via `__package__`, `__section__`, and `__verb__`.
- Hit one path mistake while looking for the TS declaration spec and corrected it by locating the real `pkg/tsgen/spec/types.go` and `helpers.go` files.

### Why
- The current `minitracecmd` implementation is SQL-first and does not yet expose a JS-facing minitrace module.
- The Goja repo already shows the likely runtime and module patterns, so reusing those patterns should minimize surprise.

### What worked
- The architecture boundaries are clear:
  - `minitracecmd` owns cataloging and command rendering.
  - `query/duckdb.go` owns archive loading and SQL execution.
  - `go-go-goja` owns runtime lifecycle, module registration, and Promise settlement.
- The existing `pkg/jsverbs` implementation proves that Goja can already become a first-class Glazed command authoring surface.

### What didn't work
- I first tried to read `go-go-goja/pkg/tsgen/spec/spec.go`, which does not exist.
- The failure was:
  - `ENOENT: no such file or directory, access '/home/manuel/workspaces/2026-04-20/minitrace-js/go-go-goja/pkg/tsgen/spec/spec.go'`
- I corrected the path by enumerating the `pkg/tsgen/spec/` directory and reading `types.go` plus `helpers.go` instead.

### What I learned
- There are really two separate design surfaces:
  1. a minitrace data/query API for scripts,
  2. a command-definition API for turning JS files into verbs.
- The second surface can reuse much of `pkg/jsverbs`, but the first surface still needs a dedicated minitrace module.
- The most natural JS return types are plain objects, arrays, and Promises; the codebase already treats those well.

### What was tricky to build
- The main challenge is avoiding an API that is either too raw (`database.query(...)` everywhere) or too magical (`__verb__` macros with no visible module boundary).
- The command model wants typed flags and stable metadata, while JS authors want composable code; those pressures point toward a small procedural core plus a separate command DSL.

### What warrants a second pair of eyes
- Whether the initial API should be a procedural `minitrace` module or a fluent query builder.
- Whether JS command registration should export a spec object or use a macro-style registration function.
- Whether the first JS-backed verbs should be pure/read-only only, or whether the runtime should allow mutation-oriented helpers like annotation writing.

### What should be done in the future
- Prototype the smallest viable `minitrace` module in `go-go-goja`.
- Decide whether `minitracecmd` should ingest JS source by scanning static metadata or by running a JS entrypoint to collect verb specs.
- Add tests for the chosen API shape before expanding the surface.

### Code review instructions
- Start with `go-minitrace/pkg/minitracecmd/types.go` and `go-minitrace/pkg/minitracecmd/parse_sql.go` to see the current command contract.
- Then inspect `go-go-goja/pkg/jsverbs/runtime.go` and `go-go-goja/pkg/jsverbs/command.go` for the runtime and command-bridge patterns.
- Validate future work with `go test ./...` in both repositories plus a focused CLI smoke test for whichever JS entrypoint is added.

### Technical details
- Current minitracecmd command sources are SQL (`.sql`) and aliases (`.alias.yaml`).
- `pkg/jsverbs` already supports JS discovery, metadata, shared sections, and Promise-aware execution.
- `go-go-goja` already has reusable module patterns for SQL access and async timing.

## Step 3: API Direction

The codebase points toward a two-layer JS surface rather than a single giant abstraction. The most practical approach is a small procedural `minitrace` module for loading/querying/reading sessions, plus a separate command DSL that turns JS scripts into the same command-spec shape that SQL verbs already use.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Present the JS API options and bias them toward an implementation path that can support both scripts and future JS verbs.

**Inferred user intent:** Get a concrete menu of API shapes, not just a raw scan of the repository.

### What I did
- Compared the procedural, fluent, functional, DSL, and hybrid host API options against the current codebase.
- Narrowed the recommendation to a staged design:
  1. tiny minitrace data/query helpers,
  2. a command definition layer built on top.

### Why
- The repository already separates query execution from cataloging, so the JS API should keep the same separation.
- A minimal core is easier to expose through docs, TypeScript declarations, and tests.

### What worked
- The option matrix is now explicit enough to use as a follow-up implementation plan.
- The existing code strongly favors host-owned runtime/lifecycle management, which keeps the API clean.

### What didn't work
- No implementation was attempted yet, so there was no runtime validation to report.

### What I learned
- The procedural-vs-DSL split matches the actual product needs:
  - people who want to explore data need the procedural layer,
  - people who want reusable commands need the DSL layer.

### What was tricky to build
- The hardest part is keeping the abstraction small enough that it still feels like Goja and not like a reimplemented ORM.

### What warrants a second pair of eyes
- The command DSL naming, especially whether to keep `verb`, `defineVerb`, or a macro-like `__verb__` style.
- The shape of the “read” API: return rows, return iterators, or return query builder objects.

### What should be done in the future
- Build a proof-of-concept `minitrace` module.
- Decide whether the API should be public-facing immediately or first used only by internal command loading.
- Add one or two sample JS scripts that demonstrate the chosen style.

### Code review instructions
- Re-read `go-go-goja/modules/database/database.go` and `go-go-goja/pkg/jsverbs/runtime.go` before implementing.
- Make sure the first code slice only introduces one new abstraction at a time.

### Technical details
- Best candidate API set:
  - `mt.loadArchive(...)`
  - `mt.query(...)`
  - `mt.session(...)`
  - `mt.verb(...)` / `mt.defineVerb(...)`
  - optional `mt.sql.*` helpers
- Likely runtime model:
  - Go owns archive/query resources,
  - JS consumes plain objects and Promises,
  - command metadata is compiled back into `MinitraceCommandSpec`.
