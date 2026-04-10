---
Title: Repository config and flag support implementation plan
Ticket: GMT-003
Status: active
Topics:
    - backend
    - configuration
    - go-minitrace
    - minitrace
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../sqleton/cmd/sqleton/config.go
      Note: |-
        Reference model for config-file and environment-based repository discovery
        Reference implementation for config-file and environment-based repository discovery
    - Path: ../../../../../../../sqleton/cmd/sqleton/main.go
      Note: |-
        Reference model for composing embedded plus external repositories
        Reference implementation for composing embedded and external repository roots
    - Path: cmd/go-minitrace/cmds/query/commands.go
      Note: |-
        Current query command loader only mounts the embedded catalog
        Current CLI command loader is embedded-only and must switch to resolved source roots
        CLI query commands root now loads configured catalogs and advertises query-repository sources (commit e1c9101)
    - Path: cmd/go-minitrace/cmds/serve/handlers_query_commands_v2.go
      Note: Query-command handlers now load from resolved source roots instead of embedded-only catalog (commit e1c9101)
    - Path: cmd/go-minitrace/cmds/serve/serve.go
      Note: |-
        Current serve command settings that need repository flags added
        Serve settings and flags will need repeated query-repository inputs
        Serve settings now accept repeated query-repository flags and resolve structured command roots (commit e1c9101)
    - Path: cmd/go-minitrace/main.go
      Note: Pre-bootstrap query-repository flag extraction added before building dynamic command tree (commit e1c9101)
    - Path: pkg/minitracecmd/assets.go
      Note: |-
        Current embedded catalog entry point that should become one root among several
        Current embedded source root entry point becomes one source root among many
    - Path: pkg/minitracecmd/repositories.go
      Note: Shared query-repository config/env/flag resolution and source-root composition implementation (commit e1c9101)
ExternalSources: []
Summary: Detailed plan for adding config/env/flag-driven sqleton-style query repository discovery to go-minitrace while preserving the existing go-minitrace-native MinitraceCommand catalog model.
LastUpdated: 2026-04-09T17:25:00-04:00
WhatFor: Provide a concrete implementation plan for loading additional query-command repositories from app config, environment variables, and repeated CLI flags in both serve and query command surfaces.
WhenToUse: Read this before implementing repository config resolution, query-repository CLI flags, multi-root catalog composition, or override-precedence tests for MinitraceCommand catalogs.
---



# Repository config and flag support implementation plan

## Executive Summary

The sqleton-style command format is already integrated into go-minitrace, but the current runtime only mounts the embedded command repository. The next milestone is to make repository discovery configurable so users can layer additional command roots from app config, environment variables, and repeated CLI flags.

The implementation should stay aligned with the current go-minitrace-native catalog model rather than importing sqleton runtime behavior. Concretely, we should add a small repository-discovery helper, expose repeated `--query-repository` flags on both `serve` and `query commands`, and compose source roots in an order that matches the existing catalog’s **first-root-wins** duplicate policy.

## Problem Statement

Today, repository-backed query commands are real, but they are effectively static:

- `pkg/minitracecmd.LoadEmbeddedCatalog()` only loads the embedded `core/` tree.
- `cmd/go-minitrace/cmds/query/commands.go` mounts that embedded catalog directly.
- `cmd/go-minitrace/cmds/serve/handlers_query_commands_v2.go` also loads only the embedded catalog.
- there is no app-level config file or environment variable for additional query-command repositories.
- there is no repeated CLI flag to add or override command repositories per invocation.

That means the current integration is missing one of the most important sqleton ideas: **embedded + external repository composition**.

## Proposed Solution

### 1. Add a small repository-discovery helper

Introduce a helper owned by go-minitrace that resolves additional repository paths from three sources:

1. app config file
2. environment variable
3. repeated CLI flags

Suggested environment variable:

- `GO_MINITRACE_QUERY_REPOSITORIES`

Suggested config shape:

```yaml
queryRepositories:
  - ./queries/team
  - ~/.config/go-minitrace/query-repositories
```

The helper should:

- trim whitespace
- ignore empty entries
- deduplicate paths while preserving priority order
- expand environment variables when converting paths to source roots
- ignore non-existent directories only at the final “turn paths into roots” stage, not while collecting logical inputs

### 2. Compose source roots explicitly

The current catalog loader keeps the **first** command for a duplicate logical path. That means repository precedence is determined entirely by source-root ordering.

Recommended priority order (highest priority first):

1. repeated CLI `--query-repository`
2. `GO_MINITRACE_QUERY_REPOSITORIES`
3. config file `queryRepositories`
4. optional default user directory if we want one later
5. embedded repository root

This order lets user-provided repositories override the embedded built-ins without changing the catalog’s current merge semantics.

### 3. Keep serve and CLI on the same repository-resolution path

Both command execution surfaces should use the same repository-resolution helper:

- `go-minitrace serve`
- `go-minitrace query commands ...`

That avoids a future split where the browser sees one catalog and the CLI sees another.

### 4. Avoid changing raw SQL preset/query-dir behavior in this ticket

This ticket should only affect the structured query-command repository system. It should not redesign:

- `--preset-dir`
- `--query-dir`
- saved raw SQL CRUD behavior

Those remain separate legacy/raw-SQL surfaces.

## Design Decisions

### Decision 1: Reuse the current first-root-wins loader policy

Do **not** change `pkg/minitracecmd.LoadCatalog(...)` precedence semantics. Instead, make repository collection produce source roots in the correct order.

Why:

- it minimizes risk
- the current precedence behavior is already implemented and tested
- changing merge semantics would ripple through existing embedded-catalog assumptions

### Decision 2: Centralize repository path resolution in one package/helper

Do not let `serve`, `query commands`, and the HTTP handlers each assemble repository paths independently.

Preferred shape:

- one helper that resolves ordered repository paths
- one helper that converts them into `[]minitracecmd.SourceRoot`

This can live in either:

- `pkg/minitracecmd/repositories.go`, or
- a shared command/common helper if you want config resolution to remain closer to CLI code

My preference is `pkg/minitracecmd` because both CLI and server handlers need the same source-root assembly behavior.

### Decision 3: Keep handlers loading catalogs from resolved roots, not from globals

The server handlers should not rely on global process state or cached mutable singletons in this first pass. They should load from resolved repository roots based on the command settings already available to the server instance.

That keeps behavior explicit and testable.

### Decision 4: Add repeated `--query-repository` flags only where structured commands matter

Add the new flag to:

- `go-minitrace serve`
- `go-minitrace query commands ...`

Do not add it to unrelated commands.

## Alternatives Considered

### Alternative A: Hardcode one external user directory only

Example:

- always load `~/.go-minitrace/queries`

Why reject it:

- too inflexible
- does not match the requested config/env/flag layering
- makes testing/overrides clumsy

### Alternative B: Change loader precedence to last-root-wins

Why reject it:

- unnecessary churn
- would require revisiting already-working loader behavior
- makes the current embedded-catalog ordering assumptions harder to reason about

### Alternative C: Cache one global resolved catalog at startup only

Why reject it for now:

- couples config resolution too tightly to one process startup path
- makes unit tests and per-command configuration harder
- complicates future hot-reload/file-watcher work before it is needed

## Implementation Plan

## Phase 1: Shared repository resolution

Add a new helper layer that resolves ordered repository paths and source roots.

Tasks:

1. add a config/env resolution helper modeled after `sqleton/cmd/sqleton/config.go`
2. define the environment variable `GO_MINITRACE_QUERY_REPOSITORIES`
3. define the config file key `queryRepositories`
4. normalize, deduplicate, and preserve precedence order
5. convert resolved paths into `minitracecmd.SourceRoot` values plus the embedded root
6. add unit tests for path normalization, deduplication, and precedence ordering

Deliverable:

- one reusable function that returns ordered source roots for the current process/settings

## Phase 2: CLI/query command integration

Wire repository resolution into `go-minitrace query commands`.

Tasks:

1. add repeated `--query-repository` flags to the query commands root/runtime settings
2. stop calling `LoadEmbeddedCatalog()` directly in `cmd/go-minitrace/cmds/query/commands.go`
3. load the catalog from resolved source roots instead
4. preserve current command mounting behavior for the embedded commands when no extra roots are configured
5. add tests proving that external repositories are mounted and can override embedded commands by logical path

Deliverable:

- `go-minitrace query commands ...` can load embedded + external repositories

## Phase 3: Serve integration

Wire the same repository resolution into `go-minitrace serve` and the v2 query-command handlers.

Tasks:

1. extend `ServeSettings` with repeated `query-repository` paths
2. expose `--query-repository` in `cmd/go-minitrace/cmds/serve/serve.go`
3. update server construction so the handlers can access resolved repository roots
4. stop loading only `LoadEmbeddedCatalog()` inside query-command handlers
5. add tests for handler listing/execution against an external temporary repository root

Deliverable:

- browser/API command discovery matches the CLI command catalog

## Phase 4: Config-file and environment wiring

Make config and env inputs first-class, not just ad hoc flags.

Tasks:

1. resolve the app config path for go-minitrace
2. parse `queryRepositories` from config
3. merge config, env, and flag inputs in explicit documented order
4. add tests for config-only, env-only, and combined precedence cases
5. document the final precedence rules in the ticket and help text

Deliverable:

- the same command repository stack can be configured without passing flags every time

## Phase 5: Docs and examples

Tasks:

1. add help text examples showing repeated `--query-repository`
2. add a small example external repository fixture for tests/docs
3. document override behavior clearly: user roots can shadow embedded commands by path because the catalog is first-root-wins

Deliverable:

- repository configuration is understandable and reproducible

## Open Questions

1. Do we want a default user repository directory in this ticket, or should that wait until after config/env/flags are working cleanly?
2. Should the server keep resolved source roots as settings-derived data only, or should we eventually support dynamic repository refresh without restart?
3. Do we want repository paths to be accepted only on `query commands`, or eventually at the top-level `query` command too if the CLI surface expands further?

## References

- `sqleton/cmd/sqleton/config.go`
- `sqleton/cmd/sqleton/main.go`
- `go-minitrace/cmd/go-minitrace/cmds/serve/serve.go`
- `go-minitrace/cmd/go-minitrace/cmds/query/commands.go`
- `go-minitrace/pkg/minitracecmd/assets.go`
