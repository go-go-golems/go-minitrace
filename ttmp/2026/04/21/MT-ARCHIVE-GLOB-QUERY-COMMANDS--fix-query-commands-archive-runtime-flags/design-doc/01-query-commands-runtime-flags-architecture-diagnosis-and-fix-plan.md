---
Title: 'Query commands runtime flags: architecture, diagnosis, and fix plan'
Ticket: MT-ARCHIVE-GLOB-QUERY-COMMANDS
Status: active
Topics:
    - go-minitrace
    - query
    - duckdb
    - cli
    - js
    - archive-glob
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/go-minitrace/cmds/query/command_runtime.go
      Note: Evidence that runtime flags were already wired and the original bug report was a misdiagnosis
    - Path: cmd/go-minitrace/cmds/query/commands.go
      Note: Shows how compiled command folders become Cobra command groups and documents the new path rule
    - Path: cmd/go-minitrace/cmds/query/commands_test.go
      Note: Regression tests for collapsed self-named JS command paths and help output
    - Path: cmd/go-minitrace/cmds/query/runtime_section.go
      Note: Defines archive-glob/db-path/table-name/persist-loaded runtime settings
    - Path: cmd/go-minitrace/cmds/serve/server_test.go
      Note: API routing regression coverage for the new canonical path
    - Path: pkg/doc/structured-query-commands.md
      Note: User-facing structured command documentation updated for the collapsed-path rule
    - Path: pkg/minitracecmd/parse_javascript.go
      Note: Implemented the canonical self-named single-verb JS path collapse
    - Path: ttmp/2026/04/21/MT-ARCHIVE-GLOB-QUERY-COMMANDS--fix-query-commands-archive-runtime-flags/scripts/01-reproduce-js-group-flag-confusion.sh
      Note: Replayable reproduction of the original misleading archive-glob failure
    - Path: ttmp/2026/04/21/MT-ARCHIVE-GLOB-QUERY-COMMANDS--fix-query-commands-archive-runtime-flags/scripts/02-inspect-leaf-vs-group-help.sh
      Note: Shows the help-surface differences between group and executable leaf paths
    - Path: ttmp/2026/04/21/MT-ARCHIVE-GLOB-QUERY-COMMANDS--fix-query-commands-archive-runtime-flags/scripts/03-locate-runtime-flag-plumbing.sh
      Note: Quick grep-based architecture map for runtime flag plumbing
ExternalSources: []
Summary: Investigation and implementation guide for fixing the misleading query-commands archive-glob failure by collapsing redundant self-named JS command paths in minitracecmd.
LastUpdated: 2026-04-22T00:00:00Z
WhatFor: Help an unfamiliar engineer understand how go-minitrace turns SQL/JS query files into catalog entries and Cobra commands, why the archive-glob failure appeared, and how the implemented fix works.
WhenToUse: Read before modifying structured query command path derivation, query command help/docs, or serve/query-command API routing.
---


# Query commands runtime flags: architecture, diagnosis, and fix plan

## Executive Summary

We began this ticket believing that `go-minitrace query commands` had a real runtime-flag bug: users were seeing `unknown flag: --archive-glob` when invoking custom JavaScript-backed query commands. After tracing the implementation and reproducing the failure against both the installed binary and the repository checkout, the deeper diagnosis was different: the runtime flags were already wired correctly on executable leaf commands, but JavaScript command files add an extra CLI path segment based on the file stem, and that extra group level made the user stop one node too early in the command tree.

That mismatch produced a misleading Cobra error. When the user invoked a JS-backed path like `hardware-research research-summary`, Cobra was still sitting on a non-runnable group command, so the leaf-only runtime flags were unavailable and `--archive-glob` appeared unsupported. The implementation we first prototyped at the Cobra layer worked, but it was the wrong abstraction boundary. The cleaner fix is now implemented in `pkg/minitracecmd/parse_javascript.go`: when a JS file defines exactly one verb and that verb has the same name as the file stem, go-minitrace collapses the redundant extra path level while building the minitrace command catalog.

This means the path shape is corrected before Cobra command registration, before serve-mode routing, and before docs/help generation. We deliberately did **not** change go-go-goja/jsverbs scanning and we did **not** retain a Cobra-only special case. The result is a smaller and more coherent command model.

## Problem Statement

### User-visible symptom

A user authoring a custom JS query command saw a failure like:

```bash
go-minitrace query commands hardware-research research-summary --archive-glob ...
# Error: unknown flag: --archive-glob
```

The immediate interpretation was that `query commands` did not actually support archive/runtime flags for structured commands, despite the embedded docs and examples claiming otherwise.

### Why the symptom was confusing

Several facts pointed in different directions:

1. The built-in SQL leaf command `overview session-list` clearly showed `--archive-glob`, `--db-path`, `--table-name`, and `--persist-loaded` in `--help` output.
2. The runtime execution layer for structured commands (`cmd/go-minitrace/cmds/query/command_runtime.go`) already decodes a dedicated runtime section with those fields.
3. The docs explicitly advertise `query commands ... --archive-glob ...` usage.
4. Yet the custom JS path still failed with `unknown flag`.

That made the failure feel like a flag-registration bug when it was really a path-shape problem.

## Requested Outcome and Scope

The user asked us to fix the first issue around `archive-glob` support and produce a detailed analysis/design/implementation guide for an unfamiliar intern. During implementation, the user explicitly redirected us away from a Cobra-only shortcut and toward a solution at the minitrace command-model layer.

In scope:

- inspect the `go-minitrace` query command architecture end-to-end,
- reproduce the failure concretely,
- determine whether the bug lives in runtime flag plumbing or in command-path derivation,
- implement the fix in go-minitrace without changing go-go-goja,
- update tests and docs,
- document review/validation steps.

Out of scope:

- changing jsverbs scanner behavior in `go-go-goja`,
- redesigning all JS command path semantics,
- adding alternate aliasing systems for arbitrary JS commands.

## Current-State Architecture

### High-level data flow

Structured query commands in go-minitrace are built in four major stages:

```text
query file (.sql / .js / .alias.yaml)
        |
        v
pkg/minitracecmd catalog loader
  - detect source kind
  - parse source into command specs
  - compute logical paths/folders
        |
        v
compiled MinitraceCommand catalog
  - ByPath
  - ByName
  - Folder / Path metadata
        |
        v
cmd/go-minitrace/cmds/query/commands.go
  - build Cobra groups from Folder
  - build executable leaf commands from MinitraceCommand
        |
        v
runtime execution
  - decode command flags
  - decode query-runtime section
  - load archive into DuckDB
  - run SQL or JS handler
```

### Key files

- `pkg/minitracecmd/parse_javascript.go` — scans JS files, builds command specs, computes JS command paths.
- `pkg/minitracecmd/catalog.go` — loads all command sources, compiles them, and stores them in `Catalog.Commands`, `Catalog.ByPath`, and `Catalog.ByName`.
- `cmd/go-minitrace/cmds/query/commands.go` — turns the catalog into a nested Cobra command tree.
- `cmd/go-minitrace/cmds/query/command_runtime.go` — loads archives and executes SQL or JS structured commands.
- `cmd/go-minitrace/cmds/query/runtime_section.go` — defines the runtime flags section (`archive-glob`, `db-path`, `table-name`, `persist-loaded`).
- `cmd/go-minitrace/cmds/query/commands_test.go` — command-tree and help-surface tests.
- `cmd/go-minitrace/cmds/serve/server_test.go` — API routing tests for query-command execution.

### Runtime flags are real and already wired

The runtime layer was never the problem. The relevant plumbing is visible in these files:

- `cmd/go-minitrace/cmds/query/runtime_section.go:8-19` defines the runtime section and fields.
- `cmd/go-minitrace/cmds/query/command_runtime.go:17-22` defines `MinitraceQueryRuntimeSettings`.
- `cmd/go-minitrace/cmds/query/command_runtime.go:65-71` decodes the runtime section and rejects empty `archive-glob`.
- `cmd/go-minitrace/cmds/query/command_runtime.go:83-88` loads the archive via `queryengine.LoadArchive(...)`.

So the runtime settings existed and were executable on real leaf commands before this ticket started.

### Where the path bug came from

The problematic logic lived in `pkg/minitracecmd/parse_javascript.go`.

Before the fix, JS command paths were derived like this:

```go
func jsCommandPath(sourcePath, commandName string) string {
    groupPath := jsFileGroupPath(sourcePath)
    ...
    return filepath.ToSlash(filepath.Join(groupPath, commandName))
}
```

That rule is correct for a file like `overview/session-tools.js` with verb `session-list`, because the desired path is:

```text
overview/session-tools/session-list
```

But for a file like `hardware-research/research-summary.js` with a single verb also named `research-summary`, the old rule produced:

```text
hardware-research/research-summary/research-summary
```

That path is mechanically consistent, but ergonomically poor and easy to misread. A human naturally tries the shorter path:

```text
hardware-research/research-summary
```

Under the old behavior, that shorter path was still an intermediate group, not the executable leaf.

## Reproduction and Diagnosis

### Reproduction scripts added to the ticket

To preserve the investigation trail, we added these scripts under the ticket’s `scripts/` directory:

- `scripts/01-reproduce-js-group-flag-confusion.sh`
- `scripts/02-inspect-leaf-vs-group-help.sh`
- `scripts/03-locate-runtime-flag-plumbing.sh`

These capture the original failure mode, the CLI help differences between group vs leaf paths, and the source locations for runtime flag registration.

### What we observed

Using a temporary repository containing:

```text
hardware-research/research-summary.js
```

with one verb named `research-summary`, the pre-fix behavior was:

- `go-minitrace query commands --query-repository <repo> hardware-research --help`
  - shows `research-summary` as a subgroup
- `go-minitrace query commands --query-repository <repo> hardware-research research-summary --help`
  - still shows a group, not a runnable leaf
- `go-minitrace query commands --query-repository <repo> hardware-research research-summary --archive-glob ...`
  - fails with `unknown flag: --archive-glob`
- `go-minitrace query commands --query-repository <repo> hardware-research research-summary research-summary --archive-glob ...`
  - works

That proved the real issue: the user had stopped at the file-stem group, not a runnable command.

### Why `archive-glob` looked broken

Cobra was behaving correctly:

- non-runnable groups do not expose the leaf’s runtime flags,
- so the parser rejected `--archive-glob` before any query-runtime code executed.

The error text was technically true, but operationally misleading.

## Design Decision

### Chosen design

**Collapse redundant self-named single-verb JS paths in `minitracecmd` during JS command parsing.**

In other words:

- if a JS file defines **exactly one** verb,
- and the verb name equals the file stem,
- then the logical command path becomes the file-stem path itself,
- instead of `file-stem/file-stem`.

### Why this is the right layer

We deliberately chose the catalog/path derivation layer instead of a Cobra-only patch because:

1. **The problem is semantic, not just cosmetic.** The awkward doubled path was part of the command model produced by go-minitrace.
2. **The fix should apply everywhere.** By changing the logical path early, CLI help, serve-mode routing, `Catalog.ByPath`, and future tooling all agree on the same canonical path.
3. **No go-go-goja changes are needed.** JS scanning still returns the same verbs and metadata; only go-minitrace’s path shaping changes.
4. **The implementation stays local.** We avoid spreading special-case alias logic across Cobra registration and API route handling.

### Alternatives considered

#### 1. Cobra-only shorthand aliasing

We prototyped a version that left the catalog path unchanged but made a self-named group executable by copying the leaf command’s behavior into the group node.

Why rejected:

- created CLI-only magic rather than a clean command model,
- risked divergence between CLI and API behavior,
- complicated Cobra tree building,
- harder to reason about for future maintainers.

#### 2. Change go-go-goja/jsverbs

Why rejected:

- jsverbs is correctly scanning verbs; it has no opinion about go-minitrace command-tree ergonomics,
- the doubled path is a go-minitrace pathing policy, not a JS scanning bug.

#### 3. Add explicit shorthand aliases in the catalog

Why rejected for this slice:

- more moving parts than necessary,
- alias collision rules become more complex,
- we only needed to remove one redundant level in a narrow, deterministic case.

## Implemented Solution

### Core change

The key change is in `pkg/minitracecmd/parse_javascript.go:18-109`.

The parser now:

1. scans JS verbs,
2. filters out nil verbs,
3. checks whether the file has exactly one verb,
4. builds its command description,
5. compares the file stem against the verb name,
6. passes a `collapseSelfNamedSingleVerb` boolean into `jsCommandPath(...)`.

### Relevant implementation excerpt

```go
collapseSelfNamedSingleVerb := false
if len(nonNilVerbs) == 1 {
    description, err := registry.CommandDescriptionForVerb(nonNilVerbs[0])
    ...
    collapseSelfNamedSingleVerb = jsFileStemMatchesCommandName(path, description.Name)
}
```

Then:

```go
Path: jsCommandPath(path, description.Name, collapseSelfNamedSingleVerb)
```

And inside `jsCommandPath(...)`:

```go
if collapseSelfNamedSingleVerb {
    return groupPath
}
return filepath.ToSlash(filepath.Join(groupPath, commandName))
```

### Helper semantics

`jsFileStemMatchesCommandName(...)` compares:

- file stem path basename, e.g. `research-summary`
- command name, e.g. `research-summary`

Only an exact self-named match triggers the collapse.

### Pseudocode summary

```text
scan JS file
collect verbs
if verb_count == 1:
    derive command description
    if basename(file_stem_path) == command_name:
        path = file_stem_path
    else:
        path = file_stem_path + "/" + command_name
else:
    for each verb:
        path = file_stem_path + "/" + command_name
```

## Testing Strategy and Validation

### Unit/integration tests added or updated

#### `pkg/minitracecmd/parse_javascript_test.go`

- `TestParseJSCommandSpecs_CollapsesSelfNamedSingleVerbPath`
  - verifies `overview/research-summary.js` with a single `research-summary` verb becomes `overview/research-summary`
- `TestLoadCatalog_RejectsDuplicateLogicalCommandPathAcrossSQLAndCollapsedSelfNamedJS`
  - verifies collision detection still works after the collapse, now against `queries/overview/session-list.sql`

#### `cmd/go-minitrace/cmds/query/commands_test.go`

- `TestNewCommandsCommand_SelfNamedSingleVerbJSCommandUsesCollapsedPath`
  - verifies the shortened CLI path resolves and executes
- `TestNewCommandsCommand_SelfNamedSingleChildGroupHelpShowsRuntimeFlags`
  - verifies the shortened path exposes runtime flags in help

#### `cmd/go-minitrace/cmds/serve/server_test.go`

- updated the JS query-command execution test path from:
  - `overview/session-list/session-list/execute`
- to:
  - `overview/session-list/execute`

This confirms the API-side routing now reflects the same canonical path.

### Validation commands

```bash
gofmt -w pkg/minitracecmd/parse_javascript.go \
  pkg/minitracecmd/parse_javascript_test.go \
  cmd/go-minitrace/cmds/query/commands.go \
  cmd/go-minitrace/cmds/query/commands_test.go \
  cmd/go-minitrace/cmds/serve/server_test.go

go test ./pkg/minitracecmd ./cmd/go-minitrace/cmds/query ./cmd/go-minitrace/cmds/serve -count=1
go test ./... -count=1
```

These all passed on the implementation commit.

## Documentation Changes

We updated user-facing docs so the new path rule is explicit:

- `cmd/go-minitrace/cmds/query/commands.go:18-49`
- `pkg/doc/structured-query-commands.md:55-107, 199-216, 378-496`
- `pkg/doc/analysis-guide.md:236-289`

The documentation now states:

- JS files **usually** add a file-stem group,
- self-named single-verb JS files collapse the redundant extra level,
- `unknown flag: --archive-glob` on a JS command often means the user is not invoking the executable leaf they think they are.

## Risks and Edge Cases

### 1. Multi-verb JS files

These still keep the extra file-stem group. That is intentional, because collapsing a multi-verb file would destroy an important namespace boundary.

### 2. Collision behavior changes

Because self-named single-verb JS files now occupy the shorter path directly, collision detection also moves earlier to the shorter path. This is correct, but it means some formerly non-colliding layouts may now conflict. We added a test to lock that down.

### 3. Backward compatibility of doubled paths

The old doubled path is no longer the canonical logical path for the self-named single-verb case. That is a behavior change, but it is the exact improvement we wanted. It aligns the command model with user intuition and fixes the practical failure mode.

If compatibility pressure appears later, an explicit alias layer could be added — but that would be a separate feature, not part of this cleanup.

## Review Guidance for a New Intern

### Start here

1. Read `pkg/minitracecmd/parse_javascript.go` first.
2. Then read `pkg/minitracecmd/catalog.go` to understand how parsed command specs become compiled command entries.
3. Then read `cmd/go-minitrace/cmds/query/commands.go` to see how `Folder` metadata becomes a Cobra tree.
4. Finally read the tests in:
   - `pkg/minitracecmd/parse_javascript_test.go`
   - `cmd/go-minitrace/cmds/query/commands_test.go`
   - `cmd/go-minitrace/cmds/serve/server_test.go`

### Mental model to keep in mind

- JS scanning lives in go-go-goja/jsverbs.
- go-minitrace owns command-path semantics.
- runtime flags were never broken; the user hit the wrong node in the command tree.
- therefore the right fix is to improve command-path derivation, not DuckDB loading and not Goja scanning.

## Implementation Plan Recap

The implementation is already complete in this ticket. The final shape was:

1. reproduce the misleading `--archive-glob` failure,
2. verify runtime flags actually exist on executable leaf commands,
3. prototype and reject a Cobra-only solution,
4. move the fix into `minitracecmd` JS path derivation,
5. update tests,
6. update docs,
7. validate with `go test ./...`.

## References

### Source files

- `pkg/minitracecmd/parse_javascript.go:18-109`
- `pkg/minitracecmd/catalog.go:20-118,164-171`
- `cmd/go-minitrace/cmds/query/commands.go:12-71`
- `cmd/go-minitrace/cmds/query/command_runtime.go:17-104`
- `cmd/go-minitrace/cmds/query/runtime_section.go:8-19`
- `cmd/go-minitrace/cmds/query/commands_test.go:244-333`
- `cmd/go-minitrace/cmds/serve/server_test.go:1082-1155`
- `pkg/doc/structured-query-commands.md:55-107,199-216,378-496`
- `pkg/doc/analysis-guide.md:236-289`

### Related investigation artifacts in this ticket

- `scripts/01-reproduce-js-group-flag-confusion.sh`
- `scripts/02-inspect-leaf-vs-group-help.sh`
- `scripts/03-locate-runtime-flag-plumbing.sh`

### Commit

- `e2d6c37b140edcc8a3dd8ccf4557c668de94d2d9` — `query: collapse self-named JS command paths`
