---
Title: 'Scanner-First JS Verb Commands for Minitracecmd: Analysis, Design, and Implementation Guide'
Ticket: GMT-007
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
    - Path: go-go-goja/engine/runtime.go
      Note: Explicit Goja runtime lifecycle that should back JS command invocation
    - Path: go-go-goja/pkg/jsverbs/runtime.go
      Note: Late-binding JS invocation path that proves scanner-first execution is feasible
    - Path: go-go-goja/pkg/jsverbs/scan.go
      Note: Existing scanner-first JS metadata extraction model to adapt into minitracecmd
    - Path: go-minitrace/cmd/go-minitrace/cmds/query/command_runtime.go
      Note: Current SQL-only execution branch that must become a runtime dispatcher
    - Path: go-minitrace/pkg/minitracecmd/catalog.go
      Note: Current scanner-first catalog load path that JS command support should extend
    - Path: go-minitrace/pkg/minitracecmd/source_kind.go
      Note: Existing source-kind model that must learn about JS command files
    - Path: go-minitrace/pkg/minitracecmd/types.go
      Note: Current command-spec and runtime command model that is still SQL-shaped
ExternalSources: []
Summary: Detailed intern-oriented analysis and implementation guide for adding scanner-first JS command files to the minitracecmd catalog and runtime.
LastUpdated: 2026-04-20T18:33:27.33008471-04:00
WhatFor: Explain the current command system, the existing jsverbs scanner/runtime, and a concrete implementation plan for making JS command files behave like scanned SQL command files.
WhenToUse: Read this guide before implementing or reviewing scanner-first JS verb support in minitracecmd.
---


# Scanner-First JS Verb Commands for Minitracecmd: Analysis, Design, and Implementation Guide

## Executive Summary

This document explains how to extend `go-minitrace` so JavaScript command files can be discovered and compiled *up front* the same way SQL command files are today. The goal is not merely to execute JavaScript inside Goja. The goal is to make JS command files participate in the same command lifecycle as SQL command files: repository discovery, schema extraction, Cobra/Glazed registration, web-form generation, aliasing, and only then late runtime execution.

That distinction matters. The current SQL path is successful because it separates **definition time** from **execution time**. A `.sql` file is scanned without being executed. Its preamble becomes a `MinitraceCommandSpec`, and only later, when the user invokes the command, the system renders SQL and runs it against DuckDB. The proposed JS path should preserve that contract. A `.js` or `.cjs` file should be scanned statically, yield one or more `MinitraceCommandSpec` values, and be executed only when the user actually runs the command.

The good news is that the repository already contains most of the machinery we need. `go-minitrace` already has a catalog/compiler/runtime pipeline for SQL and alias commands. `go-go-goja` already has a scanner-first JS command system in `pkg/jsverbs`, with strict metadata extraction, command-schema generation, runtime-owned execution, and Promise-aware result handling. The design proposed here is therefore not a greenfield invention. It is an integration plan that bridges two existing systems with a shared mental model.

For a new intern, the single most important sentence in this guide is this: **we are not designing a generic JS scripting engine for minitrace; we are designing a scanner-first command source format whose execution engine happens to be Goja.** If you remember that sentence, most design choices in this document will make immediate sense.

---

## Problem Statement and Scope

### What users want

The user request is precise:

- JS command files should be scanned first, like SQL command files.
- The system should create `minitracecmd` definitions *up front* from those JS files.
- Those commands should appear in CLI help, the command catalog, the serve/web UI, and command listings before any JS code is executed.
- The JavaScript should run only when a command is invoked.

### What that implies technically

A scanner-first JS command system must provide three capabilities simultaneously:

1. **Static metadata extraction**
   - command name
   - help text
   - flags/arguments
   - sections/layout-like structure
   - output mode
   - parents/tags

2. **Catalog compatibility**
   - JS commands must coexist with SQL commands and aliases in the same catalog.
   - folder-based command grouping must continue to work.
   - aliases should be able to target JS commands too.

3. **Runtime execution**
   - when invoked, a JS command should run in a Goja runtime with a minitrace-specific host API.
   - results should be normalized into Glazed rows or text output.
   - asynchronous JS should be supported in the same runtime-owned style already used by `pkg/jsverbs`.

### What is out of scope for the first version

To keep this design implementable, the first version should *not* attempt to solve every possible JS authoring problem. In particular, the first version should avoid:

- dynamic command metadata evaluated at runtime,
- arbitrary mutation-oriented APIs over minitrace archives,
- a full ORM or fluent in-memory query engine,
- cross-file command registration side effects discovered only by executing JS.

The first version should be conservative and deterministic.

---

## Reading Map for a New Intern

Before you modify code, you should know where the current behavior lives. Read these files in this order:

1. `go-minitrace/pkg/minitracecmd/types.go`
2. `go-minitrace/pkg/minitracecmd/parse_sql.go`
3. `go-minitrace/pkg/minitracecmd/catalog.go`
4. `go-minitrace/cmd/go-minitrace/cmds/query/command_runtime.go`
5. `go-go-goja/pkg/jsverbs/model.go`
6. `go-go-goja/pkg/jsverbs/scan.go`
7. `go-go-goja/pkg/jsverbs/command.go`
8. `go-go-goja/pkg/jsverbs/runtime.go`
9. `go-go-goja/engine/runtime.go`

If you read only one document in addition to code, read:

- `go-minitrace/pkg/doc/structured-query-commands.md`
- `go-go-goja/pkg/doc/08-jsverbs-example-overview.md`

Those two docs capture the intended user-facing story on both sides of the integration.

---

## Current-State Architecture

This section is evidence-based. Every important claim maps back to concrete code.

### 1. The existing `minitracecmd` source model is SQL + alias only

The current source kind detection supports only two command source types: SQL command files and YAML alias files.

Evidence:

- `go-minitrace/pkg/minitracecmd/source_kind.go:5-23`

```go
const (
    SourceUnknown SourceKind = iota
    SourceSQLCommand
    SourceYAMLAlias
)
```

That means the catalog has no concept of JS sources yet. This is the first structural gap.

### 2. The catalog loader is already scanner-first

The catalog system walks one or more source roots, detects supported file kinds, parses each file into a `MinitraceCommandSpec`, compiles it into a `MinitraceCommand`, and then stores the result in a command catalog.

Evidence:

- `go-minitrace/pkg/minitracecmd/catalog.go:25-120`

The key flow is:

```text
fs.WalkDir -> DetectSourceKind -> parse source -> compile spec -> add to catalog
```

This is exactly the shape we want for JS too. The desired feature is not a new architecture; it is a new source branch inside the existing architecture.

### 3. `MinitraceCommandSpec` is currently SQL-shaped

`MinitraceCommandSpec` holds general command metadata, but its verb-specific runtime payload is currently just `Query string`.

Evidence:

- `go-minitrace/pkg/minitracecmd/types.go:17-31`
- `go-minitrace/pkg/minitracecmd/types.go:54-93`

The validator currently says:

- verb requires `name`, `short`, and `query`
- alias requires `name` and `aliasFor`

That is perfect for SQL, but it means a JS-backed verb cannot validate without pretending to be SQL. This is the second structural gap.

### 4. SQL files are already parsed exactly the way the new JS flow should feel

The SQL parser is small and strict:

- detect a `/* sqleton ... */` preamble,
- decode YAML metadata into `MinitraceCommandSpec`,
- copy the body into `spec.Query`,
- validate.

Evidence:

- `go-minitrace/pkg/minitracecmd/parse_sql.go:11-80`

This is the existing “scanner-first” contract. The proposed JS behavior should mirror this in spirit:

```text
scan metadata -> build spec -> validate -> compile -> register
```

### 5. The command compiler is already language-agnostic enough to reuse

The compiler mostly copies normalized spec fields into a runtime command and adds source bookkeeping such as `Folder`, `Path`, `SourceRoot`, and `SourcePath`.

Evidence:

- `go-minitrace/pkg/minitracecmd/compiler.go:9-68`

This is promising because it suggests we do not need a separate catalog/compiler stack for JS. We likely need to extend the command model, not replace the compiler.

### 6. The current runtime branch is SQL-only

`NewMinitraceCatalogGlazeCommand` turns a `MinitraceCommand` into a Glazed command description, but its execution path assumes every non-alias command ultimately renders SQL and executes it.

Evidence:

- `go-minitrace/cmd/go-minitrace/cmds/query/command_runtime.go:33-66`
- `go-minitrace/cmd/go-minitrace/cmds/query/command_runtime.go:68-109`

The critical lines are these:

```go
sqlText, err := minitracecmd.RenderCommand(...)
...
if err := queryengine.ValidateReadOnlyQuery(sqlText); err != nil { ... }
...
return queryengine.RunIntoProcessor(ctx, conn, sqlText, gp)
```

That means even if the catalog learned how to discover JS commands, execution would still fail unless this runtime becomes a dispatcher rather than a hard-coded SQL renderer.

### 7. The UI/runtime settings path is already shared and should remain shared

The query runtime section defines the archive and DuckDB lifecycle settings that every structured command uses today.

Evidence:

- `go-minitrace/cmd/go-minitrace/cmds/query/runtime_section.go:8-20`

Those settings are:

- `archive-glob`
- `db-path`
- `table-name`
- `persist-loaded`

JS commands should keep using these same runtime settings. This is important because it preserves parity between SQL-backed and JS-backed commands in the CLI and the web UI.

### 8. Folder paths already drive command groups

The `query commands` Cobra subtree uses the folder path of each compiled command to build nested command groups.

Evidence:

- `go-minitrace/cmd/go-minitrace/cmds/query/commands.go:12-57`
- `go-minitrace/cmd/go-minitrace/cmds/query/commands.go:60-92`

This means that as long as JS command files compile into `MinitraceCommand` instances with the correct `Folder` and `Path`, the help/UI grouping story should work automatically.

### 9. `pkg/jsverbs` already implements strict, scanner-first metadata extraction

The `jsverbs` scanner recognizes `__package__`, `__section__`, and `__verb__` metadata. It parses only literal metadata and rejects dynamic expressions. This matters because it already matches the determinism we want.

Evidence:

- `go-go-goja/pkg/jsverbs/scan.go:439-553`
- `go-go-goja/pkg/doc/08-jsverbs-example-overview.md:28-37`

That existing scanner already gives us:

- package metadata for parents/tags,
- section metadata for grouped fields,
- verb metadata for command names and output modes,
- deterministic diagnostics.

### 10. `pkg/jsverbs` already has a useful intermediate model

The core JS-verb model includes:

- `Registry`
- `FileSpec`
- `SectionSpec`
- `FieldSpec`
- `VerbSpec`

Evidence:

- `go-go-goja/pkg/jsverbs/model.go:74-157`

This is exactly the kind of intermediate representation we need to turn a scanned JS file into one or more `MinitraceCommandSpec` values.

### 11. `pkg/jsverbs` already knows how to build a Glazed schema from scanned JS metadata

The command builder in `pkg/jsverbs` converts scanned field definitions and sections into Glazed command descriptions and supports both row output and writer output.

Evidence:

- `go-go-goja/pkg/jsverbs/command.go:41-100`
- `go-go-goja/pkg/jsverbs/command.go:102-210`

This gives us two important ideas:

1. scanned JS metadata can already be turned into a CLI schema,
2. the metadata surface is already close to what `minitracecmd` wants.

### 12. `pkg/jsverbs` runtime execution already uses the exact late-binding model we want

The runtime path is:

- create/borrow Goja runtime,
- require the scanned module,
- find the scanned function name in a registry overlay,
- build JS arguments from parsed values,
- invoke the function,
- await Promises if necessary,
- normalize the result.

Evidence:

- `go-go-goja/pkg/jsverbs/runtime.go:18-36`
- `go-go-goja/pkg/jsverbs/runtime.go:44-107`
- `go-go-goja/pkg/jsverbs/runtime.go:167-223`
- `go-go-goja/pkg/jsverbs/runtime.go:225-258`

This late-binding model is exactly why scanner-first JS commands are feasible. The metadata can be discovered before execution because the runtime lookup of the function happens later.

### 13. Goja runtime lifecycle is already explicit and suitable for command execution

The Goja runtime wrapper exposes explicit lifecycle and teardown management.

Evidence:

- `go-go-goja/engine/runtime.go:25-116`

That matters because command execution needs predictable resource cleanup. For JS-backed minitrace commands, the runtime should be created after archive loading starts and torn down after the command completes.

### 14. There is already a generic SQL module, but it is too low-level to be the final minitrace API

The `database` module can expose SQL query and exec functions into JS.

Evidence:

- `go-go-goja/modules/database/database.go:59-160`
- `go-go-goja/modules/database/database.go:195-248`

This module is a useful implementation building block, but it is not yet a good end-user minitrace API. It is too generic and does not reflect minitrace-specific concepts such as session access, SQL helper functions, or command context.

---

## Gap Analysis

This section compares the current system to the requested scanner-first JS behavior.

### Gap 1: Unsupported source kind

Current state:

- `DetectSourceKind` does not know about `.js` or `.cjs`.

Required state:

- JS files must be catalog-visible command sources.

### Gap 2: SQL-only verb payload

Current state:

- verb validation requires `Query`.

Required state:

- a verb must be able to validate as either SQL-backed or JS-backed.

### Gap 3: SQL-only execution branch

Current state:

- all verbs render SQL and execute SQL.

Required state:

- the runtime must dispatch by command execution kind.

### Gap 4: No minitrace-specific JS host API

Current state:

- generic `database` exists, but nothing domain-specific exists for minitrace command handlers.

Required state:

- JS command handlers need a host API shaped for minitrace command execution.

### Gap 5: No adapter from `jsverbs.VerbSpec` to `MinitraceCommandSpec`

Current state:

- `pkg/jsverbs` can build Glazed commands directly.

Required state:

- `go-minitrace` needs to normalize scanned JS metadata into its own command model so SQL and JS can coexist in one catalog.

### Gap 6: Alias semantics are only tested against SQL-backed commands

Current state:

- aliases assume a target command exists by name.

Required state:

- aliases should target JS-backed verbs too, without any special-case behavior in user-facing semantics.

---

## Proposed Architecture

The proposal is intentionally conservative: reuse the existing `minitracecmd` catalog/compiler/runtime boundaries and introduce a JS source branch that uses `pkg/jsverbs` as its scanner and invocation substrate.

## Design Principles

Before looking at structs and functions, internalize these principles.

1. **Definition time and execution time stay separate.**
   JS files are scanned during catalog load and executed only during invocation.

2. **`minitracecmd` remains the source of truth for the command catalog.**
   We are not replacing `minitracecmd` with `pkg/jsverbs`; we are teaching `minitracecmd` to ingest JS-backed command definitions.

3. **The JS metadata format stays static and deterministic.**
   Metadata may contain only literals the scanner can understand.

4. **SQL and JS commands share the same command shell.**
   Same repository layout, same grouping, same alias story, same query runtime flags.

5. **Execution kind is explicit.**
   The system should always know whether a command is SQL-backed or JS-backed.

---

## Proposed Command Model Changes

The existing command model needs a small but important generalization.

### Today

A verb is effectively:

```go
kind == verb && query != ""
```

### Proposed

A verb should become:

```go
kind == verb && execution != nil
```

Where execution can be one of two concrete shapes:

- SQL execution
- JS execution

### Suggested Go model

This does not need to be the exact final code, but it shows the shape clearly.

```go
type CommandRuntimeKind string

const (
    CommandRuntimeSQL CommandRuntimeKind = "sql"
    CommandRuntimeJS  CommandRuntimeKind = "js"
)

type JSExecutionSpec struct {
    ModulePath   string   `yaml:"-"`
    FunctionName string   `yaml:"-"`
    OutputMode   string   `yaml:"-"`
    Parents      []string `yaml:"-"`
}

type MinitraceCommandSpec struct {
    Name      string
    Short     string
    Long      string
    Layout    []*layout.Section
    Flags     []*fields.Definition
    Arguments []*fields.Definition
    Tags      []string
    Metadata  map[string]any

    RuntimeKind CommandRuntimeKind `yaml:"-"`

    Query string `yaml:"-"` // SQL only
    JS    *JSExecutionSpec `yaml:"-"` // JS only

    AliasFor   string
    AliasFlags map[string]any
    Kind       MinitraceCommandKind
}
```

### Validation rules

Verb validation should become:

- `name` required
- `short` required
- exactly one of:
  - SQL payload present,
  - JS payload present
- `aliasFor` forbidden on non-alias verbs

Alias validation remains mostly unchanged.

This is the smallest high-value generalization because it preserves the rest of the catalog model.

---

## Proposed Scanner-First JS Source Contract

A JS command file should look like a scanned manifest plus executable function bodies.

### Example file

```js
__section__("filters", {
  title: "Filters",
  fields: {
    framework: {
      type: "stringList",
      help: "Filter by agent framework",
    },
    limit: {
      type: "int",
      default: 20,
      help: "Maximum number of rows",
    },
  },
});

function sessionList(filters, ctx) {
  return ctx.minitrace.query(`
    SELECT
      id,
      title,
      environment->>'agent_framework' AS framework
    FROM ${ctx.runtime.tableName}
    WHERE 1=1
    ${filters.framework?.length
      ? `AND environment->>'agent_framework' IN (${ctx.sql.stringIn(filters.framework)})`
      : ""}
    ORDER BY timing->>'started_at' DESC
    LIMIT ${filters.limit}
  `);
}

__verb__("sessionList", {
  name: "session-list",
  short: "List minitrace sessions",
  long: "List recent sessions with optional framework filtering.",
  sections: ["filters"],
  output: "glaze",
});
```

### Rules for v1

1. Metadata must be statically scanable.
2. `__verb__` must point at a top-level function.
3. One file may define multiple verbs, but authors should strongly prefer one file per primary command.
4. Dynamic metadata is forbidden.
5. Relative `require()` should continue to work the way `pkg/jsverbs` already supports.

These rules keep the system deterministic for discovery while remaining expressive enough for real commands.

---

## Repository Layout and Command Naming

The repository layout should remain identical to SQL command repositories.

```text
query-commands/
├── overview/
│   ├── session-list.sql
│   ├── session-list.js
│   └── aliases/
│       └── codex-session-list.alias.yaml
├── nightly/
│   └── followup-candidates.js
└── tools/
    └── tool-failures.sql
```

### Folder semantics

Folders still mean command groups.

Examples:

- `overview/session-list.js` → `go-minitrace query commands overview session-list`
- `nightly/followup-candidates.js` → `go-minitrace query commands nightly followup-candidates`

### Duplicate rule

For sanity, the catalog should reject duplicate leaf command names in the same resolved path regardless of source language.

For example, if both of these exist and resolve to the same logical command path:

- `overview/session-list.sql`
- `overview/session-list.js`

then the loader should either:

- fail clearly, or
- apply a documented precedence rule.

For v1, I recommend failing loudly because it is easier for authors and reviewers to reason about.

---

## End-to-End Flow

### High-level diagram

```text
                ┌─────────────────────────────┐
                │ source roots (embedded/fs)  │
                └──────────────┬──────────────┘
                               │ WalkDir
                               v
                    ┌──────────────────────┐
                    │ DetectSourceKind     │
                    │ sql / js / alias     │
                    └───────┬───────┬──────┘
                            │       │
                      SQL parse   JS scan
                            │       │
                            v       v
                    ┌──────────────────────┐
                    │ MinitraceCommandSpec │
                    └──────────────┬───────┘
                                   │ Compile
                                   v
                      ┌────────────────────────┐
                      │ MinitraceCommand       │
                      │ runtime kind: sql/js   │
                      └──────────────┬─────────┘
                                     │
                                     v
                           ┌──────────────────┐
                           │ catalog + Cobra  │
                           │ help + web forms │
                           └────────┬─────────┘
                                    │ invoke
                                    v
                     ┌────────────────────────────────┐
                     │ dispatch runtime by kind       │
                     │ sql -> render/exec             │
                     │ js  -> goja load/invoke        │
                     └────────────────────────────────┘
```

### Load-time flow pseudocode

```go
func LoadCatalog(roots []SourceRoot) (*Catalog, error) {
    catalog := newCatalog()

    for _, root := range roots {
        walk(root.FS, root.RootDir, func(path string, contents []byte) error {
            switch DetectSourceKind(path) {
            case SourceSQLCommand:
                if !LooksLikeSqletonSQLCommand(contents) {
                    return nil
                }
                spec, err := ParseSQLCommandSpec(path, contents)
                if err != nil { return err }
                return addCompiledSpec(catalog, root, path, spec)

            case SourceJSCommand:
                specs, err := ScanJSCommandSpecs(path, contents, ScanJSOptions{
                    RootName: root.Name,
                    RootDir:  root.RootDir,
                })
                if err != nil { return err }
                for _, spec := range specs {
                    if err := addCompiledSpec(catalog, root, path, spec); err != nil {
                        return err
                    }
                }
                return nil

            case SourceYAMLAlias:
                spec, err := ParseAliasSpec(path, contents)
                if err != nil { return err }
                return addCompiledSpec(catalog, root, path, spec)
            }
            return nil
        })
    }

    sortCatalog(catalog)
    if err := resolveAliases(catalog); err != nil { return nil, err }
    return catalog, nil
}
```

### Run-time flow pseudocode

```go
func (c *MinitraceCatalogGlazeCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
    runtimeSettings := decodeRuntimeSettings(vals)
    commandValues := collectCommandValues(vals, c.command)
    resolvedCommand, resolvedValues := ResolveAliasCommand(...)

    db, conn := openAndLoadArchive(runtimeSettings)
    defer closeEverything()

    switch resolvedCommand.RuntimeKind {
    case CommandRuntimeSQL:
        sqlText := RenderCommand(resolvedCommand, ...)
        ValidateReadOnlyQuery(sqlText)
        return RunIntoProcessor(ctx, conn, sqlText, gp)

    case CommandRuntimeJS:
        runtime := BuildMinitraceRuntime(ctx, conn, runtimeSettings, resolvedCommand, resolvedValues)
        defer runtime.Close(...)
        result := InvokeJSCommand(ctx, runtime, resolvedCommand, resolvedValues)
        return EmitJSResultIntoProcessor(ctx, gp, result)

    default:
        return fmt.Errorf("unsupported runtime kind %q", resolvedCommand.RuntimeKind)
    }
}
```

---

## Integrating `pkg/jsverbs` into `minitracecmd`

This is the heart of the proposal.

### What we should reuse directly

Reuse these pieces from `pkg/jsverbs`:

- scanner metadata format (`__package__`, `__section__`, `__verb__`),
- AST-based static extraction,
- diagnostics model,
- section and field representations,
- runtime invocation mechanics,
- Promise-aware waiting.

### What we should *not* reuse directly as the final public shape

Do **not** make `go-minitrace` depend on `Registry.Commands()` as its primary integration, because that would make `pkg/jsverbs` the owner of the final command object. Instead:

- scan with `pkg/jsverbs`,
- adapt `VerbSpec` to `MinitraceCommandSpec`,
- keep `minitracecmd` as the owner of the final command catalog.

This preserves the architecture people already understand in `go-minitrace`.

### Suggested adapter function

```go
func CommandSpecsFromJSRegistry(reg *jsverbs.Registry) ([]*MinitraceCommandSpec, error)
```

Its job is to:

1. iterate scanned verbs,
2. map `VerbSpec` fields onto `MinitraceCommandSpec`,
3. convert JS field metadata into Glazed `fields.Definition` values,
4. embed JS runtime execution metadata,
5. preserve parents/tags/output mode/source info.

### Important mapping decisions

| jsverbs concept | minitracecmd concept | Notes |
|---|---|---|
| `VerbSpec.Name` | `MinitraceCommandSpec.Name` | command leaf name |
| `VerbSpec.Short` / `Long` | `Short` / `Long` | direct map |
| `VerbSpec.Tags` | `Tags` | direct map |
| `VerbSpec.Parents` + folder path | folder/group path | needs normalization |
| `FieldSpec` / `SectionSpec` | Glazed flags/arguments/layout | same user-visible schema story |
| `FunctionName` + file module path | JS execution metadata | execution-time lookup |
| `OutputMode` | runtime output handling | row vs text |

---

## Proposed Execution Context for JS Commands

JS handlers should not receive raw Go DB objects. That would leak implementation detail and make testing and compatibility harder.

Instead, the runtime should inject a host-owned context object that exposes only the supported minitrace execution surface.

### Suggested JS handler signature

There are two reasonable signatures:

1. `function handler(filters, ctx)`
2. `function handler(allValues, ctx)`

The first aligns better with `pkg/jsverbs` section binding. The second is simpler but less structured. I recommend keeping the `pkg/jsverbs` binding model and using a `ctx` parameter for shared runtime helpers.

### Suggested `ctx` shape

```js
{
  command: {
    name: "session-list",
    sourcePath: "overview/session-list.js",
    sourceRoot: "embedded"
  },
  runtime: {
    tableName: "sessions_base",
    dbPath: ":memory:",
    archiveGlob: ["./output/active/*/*.minitrace.json"],
    persistLoaded: false
  },
  minitrace: {
    query(sql, args?),
    queryOne(sql, args?),
    session(id),
    sessions(),
  },
  sql: {
    string(value),
    stringIn(values),
    like(value)
  },
  values: {
    all: {...},
    sections: {...}
  }
}
```

### Why this shape is good

It is good for a new intern because it keeps responsibilities separate:

- `command` describes what command was invoked,
- `runtime` describes how the archive/database were loaded,
- `minitrace` exposes domain operations,
- `sql` exposes safe SQL string helpers,
- `values` exposes raw parsed inputs when needed.

The handler body stays focused on command logic rather than runtime wiring.

---

## SQL and JS Command Coexistence

A successful design must make SQL and JS feel like siblings, not like two different products.

### User-facing invariants

The following should be true after implementation:

1. SQL and JS commands live in the same repository tree.
2. SQL and JS commands appear together in `go-minitrace query commands --help`.
3. SQL and JS commands use the same runtime section flags.
4. SQL and JS commands are visible in the same serve/web UI command listing.
5. YAML aliases can target either SQL or JS commands by name.
6. The user should not need to care whether a command is backed by SQL or JS unless they are authoring the command file.

### Why this matters

This uniformity is not just aesthetic. It reduces cognitive load. An intern reading the codebase should see one command system with two backends, not two separate command systems awkwardly glued together.

---

## Detailed Implementation Plan

This section is the practical part. If you are assigned the feature, follow these phases in order.

## Phase 1: Extend the command model

### Files to update

- `go-minitrace/pkg/minitracecmd/types.go`
- `go-minitrace/pkg/minitracecmd/errors.go`
- `go-minitrace/pkg/minitracecmd/compiler.go`

### Goals

- add explicit runtime/execution kind for verbs,
- add JS execution metadata,
- update validation rules,
- preserve alias behavior.

### Concrete tasks

- Add a command runtime kind enum.
- Add a JS execution payload struct.
- Update `Validate()` so verbs can be SQL-backed or JS-backed.
- Add explicit error values for invalid execution configurations.
- Teach the compiler to copy JS execution metadata into `MinitraceCommand`.

### Why start here

Because every later phase depends on the command model being able to represent JS commands without pretending they are SQL.

---

## Phase 2: Add JS source-kind detection and scanning adapter

### Files to update

- `go-minitrace/pkg/minitracecmd/source_kind.go`
- `go-minitrace/pkg/minitracecmd/catalog.go`
- new file, for example: `go-minitrace/pkg/minitracecmd/parse_js.go`
- possibly new file: `go-minitrace/pkg/minitracecmd/js_adapter.go`

### Goals

- recognize `.js` and `.cjs` as command sources,
- scan them with `pkg/jsverbs`,
- normalize scanned verbs into `MinitraceCommandSpec` values.

### Concrete tasks

1. Add `SourceJSCommand` to `DetectSourceKind`.
2. Implement a JS scan function that returns one or more specs.
3. Extend catalog load dispatch to handle JS source files.
4. Ensure source path and folder handling are preserved correctly.
5. Decide how to handle multiple verbs in one file.

### Recommended rule for v1

Allow multiple verbs per file technically, but document “one file, one primary command” as the repository convention.

### Pseudocode

```go
func ParseJSCommandSpecs(path string, contents []byte, opts JSParseOptions) ([]*MinitraceCommandSpec, error) {
    registry, err := jsverbs.ScanSources([]jsverbs.SourceFile{{
        Path: path,
        Source: contents,
    }})
    if err != nil {
        return nil, err
    }
    return CommandSpecsFromJSRegistry(registry)
}
```

---

## Phase 3: Add runtime dispatch for SQL vs JS

### Files to update

- `go-minitrace/cmd/go-minitrace/cmds/query/command_runtime.go`
- new file, for example: `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go`

### Goals

- preserve the current SQL path,
- add a JS branch,
- share the same archive/DuckDB setup.

### Concrete tasks

1. Refactor `RunIntoGlazeProcessor` into a dispatcher.
2. Keep archive loading before execution dispatch.
3. Add a JS invocation branch.
4. Normalize JS results into `middlewares.Processor` rows.
5. Add a writer-output path if text-mode JS commands are supported in the first slice.

### Important design choice

Load the archive once using the existing query runtime settings, then expose that already-loaded table/connection to the JS runtime. Do **not** ask JS commands to configure their own DB connection using the generic `database.configure(...)` path.

That would weaken the uniform CLI/serve command model.

---

## Phase 4: Build a minitrace-specific Goja host module

### Files to add

Likely in `go-go-goja`, for example:

- `go-go-goja/modules/minitrace/minitrace.go`
- maybe `go-go-goja/modules/minitrace/types.go`

And/or Go-side wiring in `go-minitrace`:

- `go-minitrace/pkg/minitracecmd/jsruntime/...`

### Goals

- expose a minimal, safe JS API for command handlers,
- avoid leaking raw DB internals,
- make command authoring pleasant and explicit.

### Minimum viable API

```js
const mt = require("minitrace");

mt.query(sql, args?)
mt.queryOne(sql, args?)
mt.session(id)
```

### Recommended additions

```js
ctx.sql.string(value)
ctx.sql.stringIn(values)
ctx.sql.like(value)
```

These helpers mirror the SQL templating helpers already used by `RenderCommand`.

### Why not start with a large API

A large API will slow down implementation and increase ambiguity. The first version only needs enough functionality to make JS-backed commands viable and ergonomic.

---

## Phase 5: Test the catalog story thoroughly

### Files to add or update

- `go-minitrace/pkg/minitracecmd/catalog_test.go`
- new tests: `go-minitrace/pkg/minitracecmd/parse_js_test.go`
- `go-minitrace/cmd/go-minitrace/cmds/query/commands_test.go`
- new runtime tests for JS execution

### What to test

1. JS files are discovered.
2. invalid JS metadata fails catalog loading with useful diagnostics.
3. command path and folder grouping match repository layout.
4. aliases can target JS commands.
5. SQL and JS commands coexist.
6. JS commands execute and return rows correctly.
7. Promise-returning JS commands work.
8. text-output JS commands work, if supported.

### Example test cases

- `overview/session-list.js` with one scanned verb
- `overview/broken.js` with invalid `__verb__` metadata
- mixed repo with `.sql`, `.js`, `.alias.yaml`
- duplicate logical command path across `.sql` and `.js`

---

## Phase 6: Update documentation and examples

### Files to update

- `go-minitrace/pkg/doc/structured-query-commands.md`
- `go-minitrace/pkg/doc/query.md`
- maybe `go-minitrace/README.md`

### What to document

- `.js` and `.cjs` as scanner-first command sources,
- the JS metadata conventions,
- the static-metadata rule,
- how aliases work with JS commands,
- examples of scanner-first JS command authoring,
- runtime differences between SQL-backed and JS-backed commands.

Documentation should explicitly preserve the SQL-first mental model and then explain JS as a sibling backend.

---

## API Reference Sketches

This section is not executable code; it is a concrete contract sketch for implementors and reviewers.

### Go: source kind detection

```go
const (
    SourceUnknown SourceKind = iota
    SourceSQLCommand
    SourceJSCommand
    SourceYAMLAlias
)
```

### Go: JS execution metadata on commands

```go
type JSExecution struct {
    ModulePath   string
    FunctionName string
    OutputMode   string
}
```

### Go: catalog parse contract

```go
func ParseJSCommandSpecs(path string, contents []byte, opts JSParseOptions) ([]*MinitraceCommandSpec, error)
```

### Go: runtime invocation contract

```go
func RunJSCommandIntoProcessor(
    ctx context.Context,
    conn *sql.Conn,
    cmd *minitracecmd.MinitraceCommand,
    runtimeSettings MinitraceQueryRuntimeSettings,
    values map[string]any,
    gp middlewares.Processor,
) error
```

### JS: scanner metadata contract

```js
__section__("filters", {
  fields: {
    framework: { type: "stringList" },
    limit: { type: "int", default: 20 },
  },
});

function sessionList(filters, ctx) {
  return ctx.minitrace.query(`SELECT ...`);
}

__verb__("sessionList", {
  name: "session-list",
  short: "List sessions",
  sections: ["filters"],
  output: "glaze",
});
```

---

## ASCII Sequence Diagrams

### Sequence: catalog load for a JS command

```text
LoadCatalog
  |
  |-- WalkDir(path=overview/session-list.js)
  |
  |-- DetectSourceKind(.js) -> SourceJSCommand
  |
  |-- ParseJSCommandSpecs
  |     |
  |     |-- jsverbs scanner parses AST
  |     |-- extracts __section__ metadata
  |     |-- extracts __verb__ metadata
  |     |-- builds VerbSpec
  |     `-- adapts VerbSpec -> MinitraceCommandSpec
  |
  |-- Compiler.Compile(spec)
  |
  `-- catalog.Add(command)
```

### Sequence: runtime execution for a JS command

```text
User invokes command
  |
  |-- decode command flags + runtime flags
  |-- resolve alias defaults
  |-- open DuckDB connection
  |-- load archive into sessions_base
  |-- build Goja runtime
  |-- inject minitrace host API + command context
  |-- require JS module
  |-- lookup scanned handler function
  |-- invoke handler
  |-- await Promise if needed
  |-- normalize result to rows/text
  `-- close runtime + DB resources
```

---

## Testing and Validation Strategy

A design like this should be validated at four levels.

### 1. Unit tests for source parsing

Validate:

- `.js` detection,
- static metadata scanning,
- adapter conversion into `MinitraceCommandSpec`,
- invalid metadata diagnostics.

### 2. Unit tests for command compilation

Validate:

- JS command specs compile into runtime commands,
- validation errors are precise,
- aliases still resolve by command name.

### 3. Runtime tests for JS invocation

Validate:

- host API wiring,
- section/value binding,
- Promise-returning commands,
- row normalization,
- text output normalization.

### 4. End-to-end CLI tests

Validate:

- mixed SQL/JS repository load,
- `--help` output,
- command invocation through Cobra,
- command execution against a real or fixture archive.

### Suggested smoke commands once implemented

```bash
go test ./pkg/minitracecmd/... ./cmd/go-minitrace/cmds/query/...
go run ./cmd/go-minitrace query commands overview session-list --archive-glob './output/active/*/*.minitrace.json'
go run ./cmd/go-minitrace query commands nightly followup-candidates --archive-glob './output/active/*/*.minitrace.json'
```

---

## Risks and Sharp Edges

### Risk 1: drift between scanned metadata and runtime behavior

If authors start treating metadata as a place for dynamic expressions, the scanner and runtime can diverge. The fix is policy and validation: metadata must remain literal-only.

### Risk 2: too much dependence on `pkg/jsverbs` internal details

If `go-minitrace` directly treats `pkg/jsverbs` as its final command system, future refactors in that package may destabilize the `go-minitrace` command model. The fix is to own an adapter layer in `go-minitrace`.

### Risk 3: duplicate command resolution across languages

If a repo contains both SQL and JS definitions for the same logical command path, user expectations will become ambiguous. The fix is explicit duplicate detection or a documented precedence rule.

### Risk 4: JS runtime lifecycle leaks

Goja runtimes and DB resources must be created and closed predictably for every command execution. The existing `engine.Runtime` lifecycle APIs help here, but the new branch must use them consistently.

### Risk 5: result-shape ambiguity

JS can return many kinds of values. The runtime must explicitly document what is supported in v1.

Recommended supported result contract:

- `object` -> one row
- `array<object>` -> many rows
- primitive -> one row with `value`
- `string` in text mode -> writer output
- `Promise<...>` -> awaited and then normalized

---

## Alternatives Considered

### Alternative A: runtime-first JS registration

In this model the system would execute JS at catalog-build time and let the module register commands dynamically.

Why not choose it for v1:

- it breaks the clean definition-time/execution-time separation,
- it makes catalog building depend on runtime side effects,
- it is harder to reason about and debug,
- it is not what the user asked for.

### Alternative B: direct reuse of `pkg/jsverbs.Registry.Commands()`

In this model `go-minitrace` would let `pkg/jsverbs` produce Glazed commands directly and skip `minitracecmd` normalization.

Why not choose it:

- it would fragment the command system,
- aliases and mixed SQL/JS semantics would become harder to unify,
- the user-facing command model would no longer have a single owner.

### Alternative C: manual JS command API only

In this model authors would write explicit `mt.verb(...)` calls without scanner metadata.

Why not choose it for this ticket:

- it violates the requested scanner-first model,
- it requires executing JS to discover commands,
- it weakens the current SQL-like authoring symmetry.

---

## Open Questions

These questions should be resolved during implementation review.

1. Should one JS file be allowed to define multiple verbs in production repositories, or merely tolerated by the scanner?
2. Should text-mode JS commands be supported in the first version, or only row-producing commands?
3. Should aliases resolve by command leaf name only, or by a fuller logical path once JS multi-verb files are possible?
4. Should the host API expose only query helpers in v1, or also convenience helpers like `session(id)`?
5. Should JS command execution be constrained to read-only operations at the API layer, or only by policy/documentation?

My recommendation is:

- allow but discourage multi-verb files,
- support text mode only if the implementation remains simple,
- keep aliases by leaf name for compatibility in v1,
- expose query helpers and `session(id)` if easy,
- keep the host API read-oriented in the first slice.

---

## Conclusion

The feature requested in this ticket is both feasible and architecturally clean. The current codebase already contains:

- a scanner-first command catalog architecture in `go-minitrace`,
- a scanner-first JS metadata and invocation system in `go-go-goja/pkg/jsverbs`,
- a runtime lifecycle abstraction in `go-go-goja/engine`.

The implementation task is therefore an integration exercise with some command-model generalization, not a wholesale redesign.

If you are a new intern joining the project, your mental model should be:

- **SQL and JS are two command-definition languages.**
- **`minitracecmd` is the single catalog and registration system.**
- **The scanner builds commands; the runtime executes them later.**

Once that model is in your head, the rest of the work becomes a sequence of concrete, reviewable patches.

---

## References

### Core go-minitrace files

- `go-minitrace/pkg/minitracecmd/source_kind.go:5-23`
- `go-minitrace/pkg/minitracecmd/catalog.go:25-120`
- `go-minitrace/pkg/minitracecmd/types.go:17-93`
- `go-minitrace/pkg/minitracecmd/parse_sql.go:11-80`
- `go-minitrace/pkg/minitracecmd/compiler.go:9-68`
- `go-minitrace/pkg/minitracecmd/render.go:14-73`
- `go-minitrace/cmd/go-minitrace/cmds/query/commands.go:12-92`
- `go-minitrace/cmd/go-minitrace/cmds/query/command_runtime.go:18-132`
- `go-minitrace/cmd/go-minitrace/cmds/query/runtime_section.go:8-20`
- `go-minitrace/pkg/doc/structured-query-commands.md:26-247`

### Core go-go-goja files

- `go-go-goja/pkg/jsverbs/model.go:74-157`
- `go-go-goja/pkg/jsverbs/scan.go:439-553`
- `go-go-goja/pkg/jsverbs/command.go:41-210`
- `go-go-goja/pkg/jsverbs/runtime.go:18-107`
- `go-go-goja/pkg/jsverbs/runtime.go:167-258`
- `go-go-goja/modules/database/database.go:59-248`
- `go-go-goja/engine/runtime.go:25-116`
- `go-go-goja/pkg/doc/08-jsverbs-example-overview.md:22-120`
