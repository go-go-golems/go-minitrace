---
Title: MinitraceCommand implementation plan with glazed parameter definition reuse
Ticket: GMT-002
Status: active
Topics:
    - backend
    - documentation
    - go-minitrace
    - minitrace
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../sqleton/pkg/cmds/spec.go
      Note: Sqleton spec/compiler model mirrored conceptually by the proposed MinitraceCommand
    - Path: cmd/go-minitrace/cmds/query/duckdb.go
      Note: Current CLI execution path that the new MinitraceCommand runtime adapter should reuse
    - Path: cmd/go-minitrace/cmds/serve/handlers_queries.go
      Note: Current raw SQL query-library handlers used as the contrast case for new query-command APIs
    - Path: web/src/pages/QueryEditorPage.tsx
      Note: Current raw-SQL-centric page state that will need a parallel active-command state model
ExternalSources: []
Summary: Detailed implementation guide for introducing a go-minitrace-native MinitraceCommand model that reuses glazed parameter definitions and sqleton-style source files without coupling go-minitrace to sqleton runtime behavior.
LastUpdated: 2026-04-08T18:23:30-04:00
WhatFor: 'Provide a concrete implementation plan for the chosen architecture: a MinitraceCommand catalog/spec layer that mirrors SqletonCommand conceptually while reusing glazed parameter definitions.'
WhenToUse: Read this before implementing the MinitraceCommand type, repository parser/compiler, query-verb APIs, or UI query forms that are backed by sqleton-style SQL metadata preambles.
---


# MinitraceCommand implementation plan with glazed parameter definition reuse

## Executive summary

The right implementation shape is now clear:

- create a **go-minitrace-owned runtime/catalog type named `MinitraceCommand`**,
- make it conceptually mirror sqleton’s `SqlCommand`,
- reuse **glazed parameter definitions** instead of inventing a new parameter schema,
- keep sqleton-style `.sql` and `.alias.yaml` source files,
- but do **not** make sqleton’s runtime command object the canonical model inside go-minitrace.

That gives us a very strong balance of reuse and local control.

### What gets reused

From sqleton / its source model:

- the `.sql` file format with `/* sqleton ... */` YAML preamble,
- the deterministic source-kind split between SQL command files and alias YAML files,
- the idea of repository roots from embedded FS plus config / env / flags,
- the concept of “neutral parsed spec -> compiled runtime command”.

From glazed:

- `fields.Definition` for flags and arguments,
- `layout.Section` for optional layout metadata,
- command-description construction and Cobra compilation at the CLI edge.

### What remains go-minitrace-native

- the canonical command/catalog model,
- the query renderer and DuckDB execution integration,
- the API transport for form-backed query verbs,
- the frontend form model and query editor state,
- the read-only query enforcement.

### Core decision in one sentence

> Implement a local `MinitraceCommand` and supporting `MinitraceCommandSpec` / catalog types, reusing glazed parameter definitions directly inside them.

That is better than either extreme:

- better than re-implementing a whole parameter system from scratch,
- and better than binding go-minitrace directly to sqleton runtime commands.

## Problem statement

The previous design doc established that go-minitrace needs a structured query-catalog layer if it wants repository-backed verbs and UI query forms. The user then refined the direction further:

- mirror sqleton’s naming and conceptual command model,
- but keep the implementation local to go-minitrace,
- and specifically reuse glazed parameter definitions as much as possible.

That creates a more precise design problem:

1. What should the canonical `MinitraceCommand` type contain?
2. How should it relate to a parsed source spec?
3. How much of sqleton’s current parser/command model should be mirrored?
4. How do we reuse glazed types without making Glazed command objects themselves the canonical catalog model?
5. What file-by-file implementation plan gets us there with low risk?

This document answers those questions concretely.

## Design constraints

These constraints are worth stating explicitly because they explain why the final type/package split looks the way it does.

### Constraint 1: the UI needs a catalog object, not a Cobra command

The query-form UI needs metadata such as:

- name,
- path,
- help text,
- parameter list,
- defaults,
- allowed choices,
- tags,
- maybe layout hints.

A Cobra command or even a generic Glazed runtime command is too execution-oriented to serve cleanly as the app’s canonical query catalog object.

### Constraint 2: go-minitrace query execution is specialized

Unlike sqleton, go-minitrace does not need a generic SQL-command runtime for arbitrary DBs. It needs:

- load the archive into DuckDB,
- render a read-only SQL query,
- enforce read-only restrictions,
- execute against `sessions_base` or a validated table name,
- return arbitrary result rows.

That means execution should remain local to go-minitrace.

### Constraint 3: parameter definitions are already well solved

There is no reason to invent a new parameter-definition schema if `glazed/pkg/cmds/fields.Definition` already models the key things we need:

- name,
- type,
- help,
- required/default,
- choices,
- short flag,
- argument semantics.

So we should reuse it directly.

### Constraint 4: source parsing and runtime execution should remain separate

Sqleton’s `spec.go` is useful here because it demonstrates the right seam:

```text
source file
  -> parsed spec
  -> compiled command
```

That same seam should exist in go-minitrace.

## Proposed solution

## Overview

Introduce three layers:

1. **source spec layer**
   - `MinitraceCommandSpec`
2. **runtime/catalog layer**
   - `MinitraceCommand`
3. **edge adapters**
   - CLI compiler
   - API DTO conversion
   - UI form model

That gives us the following flow:

```text
sqleton-style .sql / .alias.yaml source
                 |
                 v
        MinitraceCommandSpec parser
                 |
                 v
          MinitraceCommand compiler
                 |
        +--------+--------+
        |                 |
        v                 v
   CLI command        API / UI model
        |                 |
        +--------+--------+
                 |
                 v
      render SQL + validate readonly
                 |
                 v
          existing DuckDB execution
```

## The canonical type names

Use these names deliberately.

### Parsed source type

```go
type MinitraceCommandSpec struct {
    Name       string
    Short      string
    Long       string
    Layout     []*layout.Section
    Flags      []*fields.Definition
    Arguments  []*fields.Definition
    Tags       []string
    Metadata   map[string]any
    Query      string
    AliasFor   string
    AliasFlags map[string]any
    Kind       MinitraceCommandKind
}
```

### Runtime/catalog type

```go
type MinitraceCommand struct {
    Name        string
    Folder      string
    Path        string
    Short       string
    Long        string
    Layout      []*layout.Section
    Flags       []*fields.Definition
    Arguments   []*fields.Definition
    Tags        []string
    Metadata    map[string]any
    Query       string
    Kind        MinitraceCommandKind
    AliasFor    string
    AliasFlags  map[string]any
    Readonly    bool
    SourceRoot  string
    SourcePath  string
}
```

### Kind enum

```go
type MinitraceCommandKind string

const (
    MinitraceCommandVerb  MinitraceCommandKind = "verb"
    MinitraceCommandAlias MinitraceCommandKind = "alias"
)
```

These names matter because they communicate the intended architecture clearly to future maintainers.

- `MinitraceCommandSpec` = parsed source representation
- `MinitraceCommand` = canonical app command/catalog representation

That mirrors sqleton’s mental model without reusing sqleton’s runtime type directly.

## Why `MinitraceCommand` should contain glazed definitions directly

The cleanest approach is to store these fields directly on the command/spec:

- `Flags []*fields.Definition`
- `Arguments []*fields.Definition`
- `Layout []*layout.Section`

### Benefits

1. **No adapter churn in the middle layer**
   - you parse YAML directly into the reusable Glazed types
2. **One source of truth for CLI and UI forms**
   - the same definitions can drive both runtime surfaces
3. **Less bespoke schema code**
   - no need for a parallel `ParamSpec` type unless the UI later needs a narrower transport DTO
4. **Glazed command compilation becomes straightforward**
   - the compiler can pass these definitions almost directly into `cmds.WithFlags(...)`, `cmds.WithArguments(...)`, etc.

### Cost

The main cost is that your canonical command model becomes coupled to the Glazed field-definition structs.

That is an acceptable tradeoff here because:

- go-minitrace already uses Glazed heavily for commands,
- the parameter-definition problem is not domain-specific enough to justify duplication,
- and the user explicitly asked for this reuse direction.

## Recommended package layout

Create a new focused package cluster.

```text
go-minitrace/
  pkg/
    minitracecmd/
      types.go
      source_kind.go
      parse_sql.go
      parse_alias.go
      normalize.go
      compiler.go
      catalog.go
      render.go
      render_helpers.go
      catalog_test.go
      parse_sql_test.go
      parse_alias_test.go
      compiler_test.go
      render_test.go
      core/
        session-list.sql
        framework-summary.sql
        timing-analysis.sql
        aliases/
          codex-framework-summary.alias.yaml
```

You could call this package `querycatalog`, but because the user explicitly wants `MinitraceCommand`, I recommend `pkg/minitracecmd` so the package name and key type line up.

## Detailed type design

## `types.go`

This file should define the main types and minimal helpers.

Suggested contents:

```go
package minitracecmd

import (
    fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
    "github.com/go-go-golems/glazed/pkg/cmds/layout"
)

type MinitraceCommandKind string

const (
    MinitraceCommandVerb  MinitraceCommandKind = "verb"
    MinitraceCommandAlias MinitraceCommandKind = "alias"
)

type MinitraceCommandSpec struct {
    Name       string                 `yaml:"name"`
    Short      string                 `yaml:"short"`
    Long       string                 `yaml:"long,omitempty"`
    Layout     []*layout.Section      `yaml:"layout,omitempty"`
    Flags      []*fields.Definition   `yaml:"flags,omitempty"`
    Arguments  []*fields.Definition   `yaml:"arguments,omitempty"`
    Tags       []string               `yaml:"tags,omitempty"`
    Metadata   map[string]any         `yaml:"metadata,omitempty"`
    Query      string                 `yaml:"query,omitempty"`
    AliasFor   string                 `yaml:"aliasFor,omitempty"`
    AliasFlags map[string]any         `yaml:"flags,omitempty"`
    Kind       MinitraceCommandKind   `yaml:"-"`
}

type MinitraceCommand struct {
    Name       string
    Folder     string
    Path       string
    Short      string
    Long       string
    Layout     []*layout.Section
    Flags      []*fields.Definition
    Arguments  []*fields.Definition
    Tags       []string
    Metadata   map[string]any
    Query      string
    AliasFor   string
    AliasFlags map[string]any
    Kind       MinitraceCommandKind
    Readonly   bool
    SourceRoot string
    SourcePath string
}
```

### Important note about alias parsing

Do **not** try to use exactly one YAML struct for both verbs and aliases if that makes the code confusing.

A more maintainable shape may be:

```go
type SQLCommandPreamble struct { ... }
type AliasYAML struct { ... }
```

and then compile both into `MinitraceCommandSpec`.

That is probably cleaner.

## `source_kind.go`

This file should keep all source-kind dispatch logic in one place.

```go
type SourceKind int

const (
    SourceUnknown SourceKind = iota
    SourceSQLCommand
    SourceYAMLAlias
)

func DetectSourceKind(path string) SourceKind
```

Rules:

- `.alias.yaml` / `.alias.yml` => alias
- `.sql` => candidate SQL command
- no other kinds in v1

The actual “does it really have a sqleton preamble?” check can be a helper used during repository scanning.

## SQL parser design

## `parse_sql.go`

This file should mirror the best parts of sqleton’s `spec.go`, but return `MinitraceCommandSpec`.

### Main responsibilities

1. split preamble and body
2. decode YAML preamble into reusable Glazed-backed fields/layout metadata
3. set `Kind = MinitraceCommandVerb`
4. validate required command fields
5. normalize optional bool flags if needed

### Suggested parser contract

```go
func ParseSQLCommandSpec(path string, contents []byte) (*MinitraceCommandSpec, error)
func ParseSQLCommandSpecFromReader(path string, r io.Reader) (*MinitraceCommandSpec, error)
func LooksLikeSqletonSQLCommand(contents []byte) bool
func splitSqletonSQLPreamble(contents []byte) (meta string, body string, err error)
```

### Validation rules

For verb commands:

- `name` required
- `short` required
- `query body` required
- `aliasFor` must be empty

For now, reject subquery metadata if you do not plan to support it in v1.

### Recommended implementation detail

Normalize optional bool flags exactly once in the compiler or parser, not ad hoc in multiple places.

Sqleton does this because optional bools without defaults can be awkward for CLI behavior. The same logic is useful here.

## Alias parser design

## `parse_alias.go`

This file should parse sqleton-style alias YAML.

Suggested source shape:

```yaml
name: codex-framework-summary
aliasFor: framework-summary
flags:
  framework:
    - codex
```

### Suggested parser contract

```go
type AliasYAML struct {
    Name     string         `yaml:"name"`
    AliasFor string         `yaml:"aliasFor"`
    Flags    map[string]any `yaml:"flags,omitempty"`
    Short    string         `yaml:"short,omitempty"`
}

func ParseAliasSpec(path string, contents []byte) (*MinitraceCommandSpec, error)
```

### Validation rules

- `name` required
- `aliasFor` required
- `query` must be empty
- `Kind = MinitraceCommandAlias`

### Why aliases should still compile into `MinitraceCommand`

Because the UI and CLI both want to see them as selectable commands.

An alias is still a command entry. It just resolves through another command.

## Compiler design

## `compiler.go`

The compiler should turn a `MinitraceCommandSpec` into a `MinitraceCommand` plus metadata derived from path/root context.

### Suggested context object

```go
type CompileOptions struct {
    Folder     string
    Path       string
    SourceRoot string
    SourcePath string
    Readonly   bool
}
```

### Suggested compiler contract

```go
type Compiler struct{}

func (c *Compiler) Compile(spec *MinitraceCommandSpec, opts CompileOptions) (*MinitraceCommand, error)
```

### Responsibilities

- clone fields/layout metadata if you want to avoid mutation leaks
- normalize bool defaults
- store `Folder`, `Path`, `SourceRoot`, `SourcePath`
- mark `Readonly = true` for catalog-loaded commands
- preserve `Flags` / `Arguments` exactly as reusable Glazed definitions

## Catalog loader design

## `catalog.go`

This file should own repository scanning and multi-root merge behavior.

### Suggested types

```go
type SourceRoot struct {
    Name     string
    FS       fs.FS
    RootDir  string
    Readonly bool
}

type Catalog struct {
    Commands []*MinitraceCommand
    ByPath   map[string]*MinitraceCommand
    ByName   map[string]*MinitraceCommand
}
```

### Suggested loader contract

```go
func LoadCatalog(roots []SourceRoot) (*Catalog, error)
```

### Responsibilities

1. walk each root recursively
2. detect source kind
3. parse into spec
4. compile into `MinitraceCommand`
5. merge by path with deterministic precedence
6. resolve aliases after all verb commands are loaded

### Merge rule recommendation

Earlier roots win.

This matches the intuition of:

- embedded defaults loaded first or last by policy,
- user/team repos overriding built-ins if placed earlier,
- one clear deterministic rule instead of “whatever loads last”.

Document the policy explicitly in code comments and docs.

## Renderer design

## `render.go` and `render_helpers.go`

This is where go-minitrace should stay local.

### Do not compile MinitraceCommand directly to sqleton execution

Instead, define a small renderer API:

```go
type RenderContext struct {
    TableName string
    Values    map[string]any
}

func RenderCommand(cmd *MinitraceCommand, ctx RenderContext) (string, error)
```

### Rendering responsibilities

- merge alias defaults with caller-provided values
- expose helper funcs like:
  - `sqlString`
  - `sqlStringIn`
  - `sqlIntIn`
  - `sqlLike`
  - maybe `sqlDate`
- substitute `{{TABLE_NAME}}` safely using validated system context
- return one final SQL string

### Why this stays local

Because go-minitrace’s execution contract is not generic SQL CLI execution. It is:

- render,
- enforce read-only,
- execute in DuckDB against loaded minitrace archive tables.

That final execution path already exists and should remain local.

## CLI compiler design

## Goal

Mount repository-backed commands under the `query` subtree while keeping `query duckdb` intact.

### Recommended UX

Prefer a dedicated subgroup at first:

```bash
go-minitrace query commands session-list --archive-glob '...'
go-minitrace query commands framework-summary --framework codex
```

This is slightly more verbose than `go-minitrace query session-list`, but much cleaner for incremental rollout because it avoids collisions with existing subcommands.

Later, if desired, you can flatten the surface.

### Suggested file

- `cmd/go-minitrace/cmds/query/commands.go`

### Suggested flow

```go
func NewCommandsCommand() (*cobra.Command, error) {
    root := &cobra.Command{Use: "commands", Short: "Run repository-backed minitrace commands"}

    catalog, err := loadConfiguredCatalog(...)
    if err != nil { return nil, err }

    for _, cmd := range catalog.Commands {
        if cmd.Kind != MinitraceCommandVerb && cmd.Kind != MinitraceCommandAlias {
            continue
        }
        cobraCmd, err := BuildCobraCommandFromMinitraceCommand(cmd)
        if err != nil { return nil, err }
        root.AddCommand(cobraCmd)
    }
    return root, nil
}
```

### `BuildCobraCommandFromMinitraceCommand`

This adapter is where Glazed reuse pays off.

Suggested shape:

```go
func BuildCobraCommandFromMinitraceCommand(cmd *MinitraceCommand) (*cobra.Command, error)
```

Internally it should:

- create a `cmds.CommandDescription`
- attach:
  - `cmds.WithShort(cmd.Short)`
  - `cmds.WithLong(cmd.Long)`
  - `cmds.WithFlags(cmd.Flags...)`
  - `cmds.WithArguments(cmd.Arguments...)`
  - `cmds.WithLayout(...)`
  - `cmds.WithMetadata(...)`
- add shared analysis flags/sections needed for archive loading and output formatting
- run through the standard `common.BuildCobraCommand(...)` path

### Runtime behavior of a CLI MinitraceCommand

When executed, the command should:

1. decode CLI values into a values map,
2. open DuckDB connection,
3. load archive using existing `pkg/query/engine.go`,
4. render SQL from the selected `MinitraceCommand`,
5. call `validateReadOnlyQuery(...)`,
6. run existing `queryengine.RunIntoProcessor(...)`.

That adapter should be in a separate runtime file, not mixed into the parser or catalog.

## API design for forms

## New transport model

Do not overload `SavedQuery`.

Create a new protobuf file, for example:

```text
proto/go_go_golems/minitrace/api/v1/query_commands.proto
```

### Suggested messages

```proto
message QueryCommandParam {
  string name = 1;
  string type = 2;
  string help = 3;
  bool required = 4;
  string default_json = 5;
  repeated string choices = 6;
  bool positional = 7;
}

message QueryCommand {
  string name = 1;
  string folder = 2;
  string path = 3;
  string short_description = 4;
  string long_description = 5;
  repeated QueryCommandParam flags = 6;
  repeated QueryCommandParam arguments = 7;
  repeated string tags = 8;
  bool readonly = 9;
  string kind = 10;
  string alias_for = 11;
}

message ListQueryCommandsResponse {
  ApiMeta meta = 1;
  repeated QueryCommand commands = 2;
}

message ExecuteQueryCommandRequest {
  map<string, string> values = 1;
  bool render_only = 2;
}
```

### Why not expose Glazed structs directly in protobuf

Because protobuf should be a stable transport contract, not a mirror of Go runtime structs. The canonical in-memory type can reuse glazed definitions; the transport should still be a narrowed explicit schema.

## Serve handler implementation

Add a new file:

- `cmd/go-minitrace/cmds/serve/handlers_query_commands_v2.go`

### Suggested handlers

- `GET /api/v2/query-commands`
- `POST /api/v2/query-commands/{path...}/execute`

### `GET /api/v2/query-commands`

Responsibilities:

- load catalog once per request or via cached loader
- convert each `MinitraceCommand` into proto DTO
- optionally include aliases in the same list with `kind=alias`

### `POST /api/v2/query-commands/{path...}/execute`

Responsibilities:

- resolve path to catalog command
- decode form values
- render SQL
- if `render_only`, return rendered SQL in a debugging envelope
- else validate read-only and execute using the existing query path

### Important compatibility note

Keep the existing raw SQL endpoints and saved-query UI behavior intact during the first implementation slice.

That gives you a low-risk additive rollout.

## Frontend implementation design

## Type additions

Add:

- `web/src/types/queryCommand.ts`
- `web/src/api/queryCommandProtoAdapters.ts`

Suggested TS type:

```ts
export interface QueryCommandParam {
  name: string;
  type: string;
  help: string;
  required: boolean;
  defaultJson: string;
  choices: string[];
  positional: boolean;
}

export interface QueryCommand {
  name: string;
  folder: string;
  path: string;
  shortDescription: string;
  longDescription: string;
  flags: QueryCommandParam[];
  arguments: QueryCommandParam[];
  tags: string[];
  readonly: boolean;
  kind: "verb" | "alias";
  aliasFor: string;
}
```

## New components

Add:

- `web/src/components/QueryEditor/QueryCommandForm.tsx`
- optional `QueryCommandSidebarSection.tsx`

### `QueryCommandForm`

This component should be driven by the transport type, not by Go/Glazed types directly.

Render rules:

- `string` -> text field
- `int` -> numeric field
- `bool` -> checkbox / switch
- `choice` -> select
- `stringList` -> chips / repeated text input
- `intList` -> repeated numeric input

### QueryEditorPage state changes

Current page state is raw-SQL-centric. Add a separate active-command state:

```ts
interface ActiveQueryCommandState {
  path: string;
  values: Record<string, unknown>;
}
```

Do not try to overload the existing `SavedQuery` selection flow.

### UX recommendation

For the first UI slice:

- keep current sidebar sections:
  - Presets
  - Saved
- add a new top section:
  - Commands
- selecting a command opens the form pane
- selecting a raw preset/saved query still loads SQL directly into the editor

This keeps the transition legible.

## Concrete file-by-file implementation plan

## Phase 1: types and parsing

### 1. `pkg/minitracecmd/types.go`
Implement:

- `MinitraceCommandKind`
- `MinitraceCommandSpec`
- `MinitraceCommand`

### 2. `pkg/minitracecmd/source_kind.go`
Implement:

- `SourceKind`
- `DetectSourceKind(path string)`
- `LooksLikeSqletonSQLCommand(contents []byte)`

### 3. `pkg/minitracecmd/parse_sql.go`
Implement:

- preamble splitter
- SQL spec parser
- validation helpers

### 4. `pkg/minitracecmd/parse_alias.go`
Implement:

- alias YAML parser
- alias validation

### 5. tests
Add:

- `parse_sql_test.go`
- `parse_alias_test.go`

Required test cases:

- valid SQL command file
- missing preamble
- bad preamble marker
- empty body
- empty short/name
- valid alias YAML
- alias missing target name

## Phase 2: compilation and catalog

### 6. `pkg/minitracecmd/compiler.go`
Implement:

- `Compiler`
- `Compile(spec, opts)`
- bool default normalization

### 7. `pkg/minitracecmd/catalog.go`
Implement:

- root walker
- parse + compile pipeline
- merge behavior
- alias resolution pass

### 8. tests
Add:

- `compiler_test.go`
- `catalog_test.go`

Required tests:

- embedded + external merge
- duplicate path precedence
- alias resolution after merge
- folder/path derivation correctness

## Phase 3: rendering and execution

### 9. `pkg/minitracecmd/render_helpers.go`
Implement minimal helper funcs:

- `sqlString`
- `sqlStringIn`
- `sqlIntIn`
- `sqlLike`
- maybe `sqlDate`

### 10. `pkg/minitracecmd/render.go`
Implement:

- merge alias defaults + provided values
- render command SQL
- table-name substitution

### 11. tests
Add:

- `render_test.go`

Required tests:

- string escaping
- list rendering
- alias default merge precedence
- rendered SQL remains single-statement where expected

## Phase 4: CLI integration

### 12. `cmd/go-minitrace/cmds/query/commands.go`
Implement repository-backed commands group.

### 13. `cmd/go-minitrace/cmds/query/command_runtime.go`
Implement runtime adapter for executing a `MinitraceCommand`.

### 14. `cmd/go-minitrace/cmds/query/root.go`
Mount the new subgroup.

### 15. tests
Add command smoke tests against a temp archive fixture.

## Phase 5: API integration

### 16. `proto/.../query_commands.proto`
Define transport schema.

### 17. generate code
Run existing proto generation flow.

### 18. `cmd/go-minitrace/cmds/serve/handlers_query_commands_v2.go`
Implement list + execute handlers.

### 19. `cmd/go-minitrace/cmds/serve/server.go`
Mount routes.

### 20. tests
Add HTTP handler tests.

## Phase 6: frontend integration

### 21. `web/src/types/queryCommand.ts`
Add UI-facing types.

### 22. `web/src/api/queryCommandProtoAdapters.ts`
Decode protobuf responses.

### 23. `web/src/api/minitrace.ts`
Add:

- `getQueryCommands`
- `executeQueryCommand`

### 24. `web/src/components/QueryEditor/QueryCommandForm.tsx`
Implement form renderer.

### 25. `web/src/pages/QueryEditorPage.tsx`
Add active command state and execution wiring.

### 26. `web/src/components/QueryEditor/QuerySidebar.tsx`
Add commands section.

### 27. stories/tests
Add Storybook coverage for each parameter type.

## Suggested first PR slices

To keep implementation safe, split the work like this.

### PR 1: parser + catalog only

Includes:

- `pkg/minitracecmd/types.go`
- `source_kind.go`
- `parse_sql.go`
- `parse_alias.go`
- tests

No CLI, no server, no UI yet.

### PR 2: compiler + renderer + tests

Includes:

- `compiler.go`
- `catalog.go`
- `render.go`
- helper funcs
- tests

Still no UI.

### PR 3: CLI subgroup

Includes:

- `query commands` subgroup
- one or two embedded test commands
- smoke tests

### PR 4: API transport + serve handlers

Includes:

- protobuf schema
- new endpoints
- handler tests

### PR 5: frontend commands section + one basic form renderer

Includes:

- command list fetch
- form rendering for string/int/bool
- command execution

### PR 6: polish

Includes:

- aliases in UI
- list/choice/date field support
- “show rendered SQL”
- docs and examples

## Testing checklist for the intern

When the intern thinks the feature works, they should verify all of the following.

### Parser / catalog

- [ ] valid `.sql` command file loads
- [ ] malformed `.sql` file fails clearly
- [ ] `.alias.yaml` loads
- [ ] alias target resolves
- [ ] root precedence works deterministically

### CLI

- [ ] `go-minitrace query commands <name> ...` runs
- [ ] command flags from glazed definitions appear in help
- [ ] rendered SQL is read-only validated
- [ ] normal `query duckdb` still works unchanged

### Server

- [ ] `GET /api/v2/query-commands` returns the expected entries
- [ ] `POST /api/v2/query-commands/{path}/execute` returns results
- [ ] alias execution works
- [ ] invalid/missing params fail cleanly

### Frontend

- [ ] commands show in sidebar
- [ ] selecting a command shows a form
- [ ] required validation works
- [ ] execution returns results in existing results table
- [ ] raw SQL editor still works for old saved queries/presets

## Risks and mitigations

### Risk: Glazed field definitions are too CLI-shaped for the UI

Possible issue:

- some `fields.Definition` details may not map 1:1 to clean UI controls.

Mitigation:

- keep `fields.Definition` as the in-memory canonical parameter model,
- but convert it into a narrower transport/UI model in protobuf/TS.

### Risk: command and alias YAML diverge awkwardly

Possible issue:

- trying to force one YAML struct for both verbs and aliases may make parsing hard to read.

Mitigation:

- use separate parser structs,
- compile both into `MinitraceCommandSpec`.

### Risk: route/UX confusion with raw SavedQuery flows

Possible issue:

- users and developers confuse raw SQL library entries with structured commands.

Mitigation:

- keep separate API families,
- keep separate sidebar sections,
- use a dedicated embedded source tree for commands.

## Alternatives considered

### Alternative 1: make `MinitraceCommand` just a wrapper around sqleton `SqlCommand`

Rejected.

This keeps too much runtime coupling and still does not provide a clean canonical model for UI/API use.

### Alternative 2: make `fields.Definition` transport-visible everywhere

Rejected.

Good for Go internals; bad as a long-term wire contract. Keep protobuf/TS transport types explicit.

### Alternative 3: invent a new `ParamSpec` type instead of reusing Glazed fields

Rejected.

That duplicates solved work and creates another schema translation layer for little benefit.

## Recommended implementation order

If I were supervising an intern, I would give them this exact order:

1. implement `MinitraceCommandSpec` and `MinitraceCommand`
2. parse sqleton-style SQL files into specs
3. parse aliases
4. compile specs into commands
5. load a multi-root catalog
6. render commands into SQL
7. execute commands from a new CLI subgroup
8. expose the catalog over a new API
9. render one simple UI form
10. only after that add alias UX polish and helper-surface expansion

That order maximizes learning and minimizes debugging complexity.

## Exact first-PR coding checklist and starter stubs

This section is for the person who will actually open the first coding PR. It is intentionally more literal than the rest of the document.

## First PR goal

The first PR should do **only this**:

- introduce the new local types,
- parse sqleton-style `.sql` command files,
- parse `.alias.yaml` files,
- compile both into `MinitraceCommand`,
- load a multi-root in-memory catalog,
- and cover that behavior with tests.

It should **not** yet:

- add CLI commands,
- add protobuf messages,
- add HTTP handlers,
- add frontend components,
- add SQL rendering helpers beyond what parser tests need,
- or change the existing saved-query/editor flows.

That narrow scope is important because it keeps the first PR easy to review and easy to revert if the source-model shape needs adjustment.

## First PR files to create

Create exactly these files first:

```text
pkg/minitracecmd/
  types.go
  source_kind.go
  parse_sql.go
  parse_alias.go
  compiler.go
  catalog.go
  parse_sql_test.go
  parse_alias_test.go
  compiler_test.go
  catalog_test.go
```

## Starter stub: `pkg/minitracecmd/types.go`

```go
package minitracecmd

import (
    fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
    "github.com/go-go-golems/glazed/pkg/cmds/layout"
)

type MinitraceCommandKind string

const (
    MinitraceCommandVerb  MinitraceCommandKind = "verb"
    MinitraceCommandAlias MinitraceCommandKind = "alias"
)

type MinitraceCommandSpec struct {
    Name      string                 `yaml:"name"`
    Short     string                 `yaml:"short"`
    Long      string                 `yaml:"long,omitempty"`
    Layout    []*layout.Section      `yaml:"layout,omitempty"`
    Flags     []*fields.Definition   `yaml:"flags,omitempty"`
    Arguments []*fields.Definition   `yaml:"arguments,omitempty"`
    Tags      []string               `yaml:"tags,omitempty"`
    Metadata  map[string]any         `yaml:"metadata,omitempty"`

    Query      string               `yaml:"-"`
    AliasFor   string               `yaml:"aliasFor,omitempty"`
    AliasFlags map[string]any       `yaml:"-"`
    Kind       MinitraceCommandKind `yaml:"-"`
}

type MinitraceCommand struct {
    Name      string
    Folder    string
    Path      string
    Short     string
    Long      string
    Layout    []*layout.Section
    Flags     []*fields.Definition
    Arguments []*fields.Definition
    Tags      []string
    Metadata  map[string]any

    Query      string
    AliasFor   string
    AliasFlags map[string]any
    Kind       MinitraceCommandKind
    Readonly   bool
    SourceRoot string
    SourcePath string
}

func (s *MinitraceCommandSpec) Validate() error {
    switch s.Kind {
    case MinitraceCommandVerb:
        if s.Name == "" {
            return ErrMissingName
        }
        if s.Short == "" {
            return ErrMissingShort
        }
        if s.Query == "" {
            return ErrMissingQuery
        }
        if s.AliasFor != "" {
            return ErrVerbCannotSetAliasFor
        }
    case MinitraceCommandAlias:
        if s.Name == "" {
            return ErrMissingName
        }
        if s.AliasFor == "" {
            return ErrMissingAliasTarget
        }
        if s.Query != "" {
            return ErrAliasCannotSetQuery
        }
    default:
        return ErrUnknownCommandKind
    }
    return nil
}
```

You will also want a small `errors.go`, or you can keep the errors in `types.go` initially if you want the first PR smaller.

## Starter stub: `pkg/minitracecmd/source_kind.go`

```go
package minitracecmd

import "strings"

type SourceKind int

const (
    SourceUnknown SourceKind = iota
    SourceSQLCommand
    SourceYAMLAlias
)

func DetectSourceKind(path string) SourceKind {
    lower := strings.ToLower(path)
    switch {
    case strings.HasSuffix(lower, ".alias.yaml"), strings.HasSuffix(lower, ".alias.yml"):
        return SourceYAMLAlias
    case strings.HasSuffix(lower, ".sql"):
        return SourceSQLCommand
    default:
        return SourceUnknown
    }
}
```

## Starter stub: `pkg/minitracecmd/parse_sql.go`

```go
package minitracecmd

import (
    "io"
    "strings"

    "github.com/pkg/errors"
    "gopkg.in/yaml.v3"
)

func ParseSQLCommandSpec(path string, contents []byte) (*MinitraceCommandSpec, error) {
    metaText, body, err := splitSqletonSQLPreamble(contents)
    if err != nil {
        return nil, errors.Wrapf(err, "parse sqleton sql preamble: %s", path)
    }

    spec := &MinitraceCommandSpec{Kind: MinitraceCommandVerb}
    dec := yaml.NewDecoder(strings.NewReader(metaText))
    if err := dec.Decode(spec); err != nil {
        return nil, errors.Wrapf(err, "decode sqleton sql metadata: %s", path)
    }

    spec.Query = strings.TrimSpace(body)
    if err := spec.Validate(); err != nil {
        return nil, errors.Wrapf(err, "validate sqleton sql command: %s", path)
    }
    return spec, nil
}

func ParseSQLCommandSpecFromReader(path string, r io.Reader) (*MinitraceCommandSpec, error) {
    contents, err := io.ReadAll(r)
    if err != nil {
        return nil, errors.Wrapf(err, "read sqleton sql command: %s", path)
    }
    return ParseSQLCommandSpec(path, contents)
}

func splitSqletonSQLPreamble(contents []byte) (string, string, error) {
    s := strings.TrimLeft(string(contents), "\ufeff\r\n\t ")
    if !strings.HasPrefix(s, "/*") {
        return "", "", ErrMissingPreamble
    }

    end := strings.Index(s, "*/")
    if end == -1 {
        return "", "", ErrUnterminatedPreamble
    }

    raw := strings.TrimSpace(s[2:end])
    if !strings.HasPrefix(raw, "sqleton") {
        return "", "", ErrInvalidPreambleMarker
    }

    metadata := strings.TrimSpace(strings.TrimPrefix(raw, "sqleton"))
    body := strings.TrimSpace(s[end+2:])
    if metadata == "" {
        return "", "", ErrEmptyPreambleMetadata
    }
    if body == "" {
        return "", "", ErrMissingQuery
    }
    return metadata, body, nil
}

func LooksLikeSqletonSQLCommand(contents []byte) bool {
    _, _, err := splitSqletonSQLPreamble(contents)
    return err == nil
}
```

## Starter stub: `pkg/minitracecmd/parse_alias.go`

```go
package minitracecmd

import (
    "io"
    "strings"

    "github.com/pkg/errors"
    "gopkg.in/yaml.v3"
)

type aliasYAML struct {
    Name     string         `yaml:"name"`
    Short    string         `yaml:"short,omitempty"`
    AliasFor string         `yaml:"aliasFor"`
    Flags    map[string]any `yaml:"flags,omitempty"`
}

func ParseAliasSpec(path string, contents []byte) (*MinitraceCommandSpec, error) {
    payload := &aliasYAML{}
    if err := yaml.Unmarshal(contents, payload); err != nil {
        return nil, errors.Wrapf(err, "decode alias yaml: %s", path)
    }

    spec := &MinitraceCommandSpec{
        Name:       strings.TrimSpace(payload.Name),
        Short:      strings.TrimSpace(payload.Short),
        AliasFor:   strings.TrimSpace(payload.AliasFor),
        AliasFlags: payload.Flags,
        Kind:       MinitraceCommandAlias,
    }
    if err := spec.Validate(); err != nil {
        return nil, errors.Wrapf(err, "validate alias yaml: %s", path)
    }
    return spec, nil
}

func ParseAliasSpecFromReader(path string, r io.Reader) (*MinitraceCommandSpec, error) {
    contents, err := io.ReadAll(r)
    if err != nil {
        return nil, errors.Wrapf(err, "read alias yaml: %s", path)
    }
    return ParseAliasSpec(path, contents)
}
```

## Starter stub: `pkg/minitracecmd/compiler.go`

```go
package minitracecmd

import (
    "path/filepath"

    fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
)

type CompileOptions struct {
    Folder     string
    Path       string
    SourceRoot string
    SourcePath string
    Readonly   bool
}

type Compiler struct{}

func (c *Compiler) Compile(spec *MinitraceCommandSpec, opts CompileOptions) (*MinitraceCommand, error) {
    if spec == nil {
        return nil, ErrNilSpec
    }
    if err := spec.Validate(); err != nil {
        return nil, err
    }

    return &MinitraceCommand{
        Name:       spec.Name,
        Folder:     filepath.ToSlash(opts.Folder),
        Path:       filepath.ToSlash(opts.Path),
        Short:      spec.Short,
        Long:       spec.Long,
        Layout:     spec.Layout,
        Flags:      normalizeOptionalBoolFlags(spec.Flags),
        Arguments:  spec.Arguments,
        Tags:       spec.Tags,
        Metadata:   spec.Metadata,
        Query:      spec.Query,
        AliasFor:   spec.AliasFor,
        AliasFlags: spec.AliasFlags,
        Kind:       spec.Kind,
        Readonly:   opts.Readonly,
        SourceRoot: opts.SourceRoot,
        SourcePath: opts.SourcePath,
    }, nil
}

func normalizeOptionalBoolFlags(flags []*fields.Definition) []*fields.Definition {
    if len(flags) == 0 {
        return nil
    }

    ret := make([]*fields.Definition, 0, len(flags))
    for _, flag := range flags {
        if flag == nil {
            ret = append(ret, nil)
            continue
        }
        cloned := flag.Clone()
        if cloned.Type == fields.TypeBool && !cloned.Required && cloned.Default == nil {
            v := any(false)
            cloned.Default = &v
        }
        ret = append(ret, cloned)
    }
    return ret
}
```

## Starter stub: `pkg/minitracecmd/catalog.go`

```go
package minitracecmd

import (
    "io/fs"
    "path/filepath"
    "sort"
    "strings"
)

type SourceRoot struct {
    Name     string
    FS       fs.FS
    RootDir  string
    Readonly bool
}

type Catalog struct {
    Commands []*MinitraceCommand
    ByPath   map[string]*MinitraceCommand
    ByName   map[string]*MinitraceCommand
}

func LoadCatalog(roots []SourceRoot) (*Catalog, error) {
    compiler := &Compiler{}
    catalog := &Catalog{
        Commands: []*MinitraceCommand{},
        ByPath:   map[string]*MinitraceCommand{},
        ByName:   map[string]*MinitraceCommand{},
    }

    for _, root := range roots {
        if err := fs.WalkDir(root.FS, root.RootDir, func(path string, d fs.DirEntry, err error) error {
            if err != nil {
                return err
            }
            if d.IsDir() {
                return nil
            }

            kind := DetectSourceKind(path)
            if kind == SourceUnknown {
                return nil
            }

            b, err := fs.ReadFile(root.FS, path)
            if err != nil {
                return err
            }

            var spec *MinitraceCommandSpec
            switch kind {
            case SourceSQLCommand:
                if !LooksLikeSqletonSQLCommand(b) {
                    return nil
                }
                spec, err = ParseSQLCommandSpec(path, b)
            case SourceYAMLAlias:
                spec, err = ParseAliasSpec(path, b)
            }
            if err != nil {
                return err
            }

            rel, err := filepath.Rel(root.RootDir, path)
            if err != nil {
                return err
            }
            rel = filepath.ToSlash(rel)
            folder := filepath.ToSlash(filepath.Dir(rel))
            if folder == "." {
                folder = ""
            }

            cmd, err := compiler.Compile(spec, CompileOptions{
                Folder:     folder,
                Path:       rel,
                SourceRoot: root.Name,
                SourcePath: path,
                Readonly:   root.Readonly,
            })
            if err != nil {
                return err
            }

            if _, exists := catalog.ByPath[cmd.Path]; exists {
                return nil // first root wins
            }
            catalog.ByPath[cmd.Path] = cmd
            catalog.Commands = append(catalog.Commands, cmd)
            if cmd.Kind == MinitraceCommandVerb {
                if _, exists := catalog.ByName[cmd.Name]; !exists {
                    catalog.ByName[cmd.Name] = cmd
                }
            }
            return nil
        }); err != nil {
            return nil, err
        }
    }

    sort.Slice(catalog.Commands, func(i, j int) bool {
        return strings.Compare(catalog.Commands[i].Path, catalog.Commands[j].Path) < 0
    })
    return catalog, resolveAliases(catalog)
}

func resolveAliases(c *Catalog) error {
    for _, cmd := range c.Commands {
        if cmd.Kind != MinitraceCommandAlias {
            continue
        }
        if _, ok := c.ByName[cmd.AliasFor]; !ok {
            return ErrAliasTargetNotFound
        }
    }
    return nil
}
```

## First PR test cases to write literally

Write these tests before adding more features.

### `parse_sql_test.go`

- `TestParseSQLCommandSpec_ValidFile`
- `TestParseSQLCommandSpec_MissingPreamble`
- `TestParseSQLCommandSpec_UnterminatedPreamble`
- `TestParseSQLCommandSpec_InvalidMarker`
- `TestParseSQLCommandSpec_MissingShort`
- `TestParseSQLCommandSpec_MissingQueryBody`

### `parse_alias_test.go`

- `TestParseAliasSpec_ValidAlias`
- `TestParseAliasSpec_MissingAliasFor`
- `TestParseAliasSpec_MissingName`

### `compiler_test.go`

- `TestCompiler_CompileVerb`
- `TestCompiler_CompileAlias`
- `TestCompiler_NormalizesOptionalBoolFlags`

### `catalog_test.go`

- `TestLoadCatalog_LoadsSQLAndAlias`
- `TestLoadCatalog_FirstRootWinsOnDuplicatePath`
- `TestLoadCatalog_AliasTargetMustExist`
- `TestLoadCatalog_DerivesFolderAndPath`

## First PR acceptance criteria

Do not merge the first PR until all of these are true.

- [ ] parser package exists and is self-contained
- [ ] no CLI/UI/server behavior changed yet
- [ ] all new tests pass
- [ ] catalog can load at least one sqleton-style `.sql` file from a `fstest.MapFS`
- [ ] catalog can load at least one `.alias.yaml` file from a `fstest.MapFS`
- [ ] duplicate path precedence is deterministic
- [ ] alias target resolution is validated
- [ ] `fields.Definition` reuse is direct; no new parallel param schema exists in Go internals

## Code review instructions for the first PR

Tell the reviewer to read the files in this order:

1. `pkg/minitracecmd/types.go`
2. `pkg/minitracecmd/parse_sql.go`
3. `pkg/minitracecmd/parse_alias.go`
4. `pkg/minitracecmd/compiler.go`
5. `pkg/minitracecmd/catalog.go`
6. all tests in `pkg/minitracecmd/*_test.go`

Then ask them to verify these architecture claims:

- parsing is separate from compilation
- `MinitraceCommand` is the canonical local model
- glazed parameter definitions are reused directly
- no sqleton runtime command type is imported into the canonical model
- the PR remains small enough that rollback would be easy

## Open questions

1. Should the first CLI surface be `query commands ...` for safety, or do you want immediate flattening under `query ...`?
2. Should alias entries appear as first-class rows in the UI sidebar, or should they be resolved away into derived defaults on the target verb?
3. Should embedded command roots override external ones, or should external ones override embedded defaults by default?
4. Do you want `render_only` in the first API slice, or can that wait until after command execution works?
5. Which helper funcs beyond `sqlString`, `sqlStringIn`, `sqlIntIn`, and `sqlLike` are required by the first built-in command set?

## References

- `sqleton/pkg/cmds/spec.go`
- `sqleton/pkg/cmds/loaders.go`
- `sqleton/cmd/sqleton/doc/topics/06-query-commands.md`
- `go-minitrace/pkg/query/engine.go`
- `go-minitrace/cmd/go-minitrace/cmds/query/duckdb.go`
- `go-minitrace/cmd/go-minitrace/cmds/serve/handlers_queries.go`
- `go-minitrace/cmd/go-minitrace/cmds/serve/server.go`
- `go-minitrace/proto/go_go_golems/minitrace/api/v1/queries.proto`
- `go-minitrace/web/src/pages/QueryEditorPage.tsx`
- `go-minitrace/web/src/components/QueryEditor/QuerySidebar.tsx`
- `go-minitrace/web/src/components/QueryEditor/QueryEditor.tsx`
- `go-minitrace/ttmp/2026/04/08/GMT-002--add-sqleton-style-verb-query-loading-to-go-minitrace/design-doc/01-sqleton-style-verb-query-loading-for-go-minitrace-analysis-design-and-implementation-guide.md`
