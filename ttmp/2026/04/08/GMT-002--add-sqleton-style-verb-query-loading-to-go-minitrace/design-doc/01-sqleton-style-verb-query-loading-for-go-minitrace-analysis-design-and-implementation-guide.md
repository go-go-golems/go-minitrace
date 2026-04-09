---
Title: 'sqleton-style verb query loading for go-minitrace: analysis, design, and implementation guide'
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
    - Path: ../../../../../../../sqleton/cmd/sqleton/doc/topics/06-query-commands.md
      Note: Sqleton query-command documentation that defines the SQL preamble and repository model
    - Path: ../../../../../../../sqleton/cmd/sqleton/main.go
      Note: Sqleton embedded plus external repository mounting and command registration flow
    - Path: ../../../../../../../sqleton/pkg/cmds/spec.go
      Note: Sqleton neutral SQL command spec
    - Path: cmd/go-minitrace/cmds/serve/handlers_queries.go
      Note: Current raw SQL preset and saved-query loader/CRUD model in go-minitrace
    - Path: pkg/query/assets.go
      Note: Current built-in preset registry in go-minitrace
    - Path: proto/go_go_golems/minitrace/api/v1/queries.proto
      Note: Current structured query metadata transport contract that is too small for forms
    - Path: web/src/pages/QueryEditorPage.tsx
      Note: Current query editor page orchestration showing raw-SQL-centric UI state
ExternalSources: []
Summary: Evidence-based design for bringing sqleton-style repository-backed SQL verbs and UI query forms into go-minitrace without coupling go-minitrace to sqleton runtime behavior.
LastUpdated: 2026-04-08T17:34:25-04:00
WhatFor: Explain how sqleton query repositories work today, explain how go-minitrace query loading/UI work today, and propose a concrete implementation plan for repository-backed verbs and query forms.
WhenToUse: Read this before implementing repository-backed query verbs, structured SQL form metadata, new query catalog APIs, or query-form UI work in go-minitrace.
---


# sqleton-style verb query loading for go-minitrace: analysis, design, and implementation guide

## Executive summary

The requested feature is larger than “load a few more SQL files.” `sqleton` works because it has a complete source model: repository discovery, deterministic file-kind detection, a neutral parsed spec, command compilation, and directory-derived command hierarchy. `go-minitrace` does not have that architecture today. It has two simpler systems instead: embedded built-in presets in `pkg/query/presets/*.sql`, and plain filesystem scanning of `.sql` files for the web UI’s “presets” and “saved queries” lists. Those systems expose raw SQL text, not structured verbs or forms.

The clean way to bring sqleton-style behavior into `go-minitrace` is to add a **new query catalog layer** that sits between source files and execution. That catalog should:

1. load an embedded repository plus external repositories,
2. parse sqleton-style `.sql` files with a `/* sqleton ... */` YAML preamble,
3. optionally load `.alias.yaml` files for shortcuts,
4. produce a neutral `VerbSpec` / `CatalogEntry` model,
5. compile that model into both:
   - CLI commands under `go-minitrace query ...`, and
   - API payloads that drive UI query forms,
6. render SQL server-side and then reuse the existing DuckDB execution path and read-only query guard.

The most important design choice in this document is: **reuse the sqleton source format and repository ideas, but do not couple go-minitrace to sqleton’s runtime command implementation.** `sqleton` builds commands for arbitrary SQL databases. `go-minitrace` executes read-only analysis queries against one already-loaded DuckDB table. The shared part is the source format and repository loading pattern, not the runtime behavior.

## Problem statement

The user asked for a new docmgr ticket and a detailed research/design guide for “sqleton-style verb query loading” in `go-minitrace`, with an implementation diary and a final reMarkable upload. The requested end state is:

- load query definitions from an embedded repository and from additional directories,
- do so “like sqleton,” meaning repository-based discovery rather than one-off file paths,
- support SQL/YAML source material that can become verbs,
- expose those verbs in the UI as query forms, not only as raw SQL text,
- and explain the entire system clearly enough for a new intern to implement it safely.

That request touches five subsystems at once:

1. source-file format,
2. repository discovery,
3. CLI command registration,
4. HTTP / protobuf API shape,
5. frontend query editor behavior.

The main architectural question is not “can go-minitrace read `.sql` files?” It already can. The real question is:

> How do we move from raw SQL files to a structured query catalog that can drive both CLI verbs and UI forms?

## Scope

### In scope

- sqleton’s repository-discovery and SQL-command source model
- go-minitrace’s current query engine, serve API, saved-query/preset loading, and query editor UI
- a proposed neutral catalog/spec layer for go-minitrace
- a phased implementation plan for CLI, backend API, and frontend form rendering
- concrete file references and API sketches

### Out of scope

- implementing the feature in this ticket
- redesigning DuckDB loading or the minitrace schema
- allowing write/mutation SQL in the UI
- replacing the ad hoc SQL editor; forms should complement it, not remove the power-user path
- turning the top-level `queries/load.sql` external DuckDB workflow into a form-driven system

## Current state: sqleton

### What sqleton documents as the intended model

The sqleton help page already describes the source model we want to borrow.

- Query commands are regular `.sql` files with a `/* sqleton ... */` YAML preamble (`sqleton/cmd/sqleton/doc/topics/06-query-commands.md:18-46`).
- Repository directory layout becomes CLI hierarchy (`sqleton/cmd/sqleton/doc/topics/06-query-commands.md:48-66`).
- Additional repositories are discovered from config and environment (`sqleton/cmd/sqleton/doc/topics/06-query-commands.md:68-90`).
- Parameter metadata stays structured inside the preamble (`sqleton/cmd/sqleton/doc/topics/06-query-commands.md:108-138`).

That document is valuable because it is not merely conceptual; the current sqleton implementation now matches it.

### How sqleton discovers repositories

`sqleton/cmd/sqleton/config.go` defines a very small app-level repository config:

- `AppConfig.Repositories []string` (`sqleton/cmd/sqleton/config.go:15-17`)
- `collectRepositoryPaths()` merges config and environment (`sqleton/cmd/sqleton/config.go:48-58`)
- `SQLETON_REPOSITORIES` is split with the platform path-list separator (`sqleton/cmd/sqleton/config.go:13`, `60-67`)
- `normalizeRepositoryPaths()` deduplicates and trims paths (`sqleton/cmd/sqleton/config.go:69-85`)

This is important for go-minitrace because it shows the pattern is intentionally simple. There is no heavy registry service. The app gathers directories, normalizes them, and mounts them.

### How sqleton mounts embedded and external repositories

`sqleton/cmd/sqleton/main.go` wires three important ideas together:

1. it always mounts an embedded repository from `queriesFS` (`sqleton/cmd/sqleton/main.go:131-132`, `231-238`),
2. it appends repository directories discovered from config/env plus `$HOME/.sqleton/queries` (`sqleton/cmd/sqleton/main.go:217-226`, `240-253`),
3. it loads those directories through one command loader and repository stack (`sqleton/cmd/sqleton/main.go:228-230`, `255-273`).

That “embedded + local repositories” composition is the behavior the user wants brought into go-minitrace.

### How sqleton parses source files

The core sqleton source model lives in two files.

#### 1. Source-kind detection and loading

`sqleton/pkg/cmds/loaders.go` shows a deterministic source-kind boundary:

- `LoadCommands()` opens one file and dispatches on `DetectSourceKind(entryName)` (`sqleton/pkg/cmds/loaders.go:22-63`)
- `.sql` files with a sqleton preamble are parsed into a spec and compiled (`sqleton/pkg/cmds/loaders.go:35-50`)
- `.alias.yaml` / `.alias.yml` are parsed as aliases (`sqleton/pkg/cmds/loaders.go:52-57`)
- unsupported file kinds are rejected explicitly (`sqleton/pkg/cmds/loaders.go:59-63`)
- `IsFileSupported()` only accepts `.sql` when the preamble is actually present (`sqleton/pkg/cmds/loaders.go:66-76`, `79-93`)

#### 2. Neutral parsed spec

`sqleton/pkg/cmds/spec.go` is the most relevant implementation reference for this ticket.

It defines:

- `SourceKind` with `SourceSQLCommand` and `SourceYAMLAlias` (`sqleton/pkg/cmds/spec.go:16-34`)
- `SqlCommandSpec` as the neutral parsed model (`sqleton/pkg/cmds/spec.go:36-47`)
- `Validate()` on the spec (`sqleton/pkg/cmds/spec.go:49-60`)
- `SqlCommandCompiler.Compile()` that turns the spec into a runtime command (`sqleton/pkg/cmds/spec.go:62-111`)
- `ParseSQLFileSpec()` that splits the preamble, YAML-decodes metadata, rejects subqueries in metadata, and injects the SQL body (`sqleton/pkg/cmds/spec.go:136-156`)
- `splitSqletonSQLPreamble()` and `LooksLikeSqletonSQLCommand()` (`sqleton/pkg/cmds/spec.go:159-200`)
- `MarshalSpecToSQLFile()` that can write the structured format back to disk (`sqleton/pkg/cmds/spec.go:202-244`)

This is the exact architectural seam go-minitrace is missing today.

### How sqleton proves the model works end-to-end

`sqleton/cmd/sqleton/main_test.go` provides smoke tests that are especially useful for an intern because they show the runtime contract clearly.

- `TestSQLiteSmoke` proves `run-command` can execute a generated `.sql` command file (`sqleton/cmd/sqleton/main_test.go:19-60`).
- `TestConfiguredRepositoryDiscoverySmoke` proves repository discovery from `SQLETON_REPOSITORIES`, including alias loading (`sqleton/cmd/sqleton/main_test.go:62-109`).
- `TestConfiguredRepositoryDiscoveryFromConfigFileSmoke` proves config-file repository discovery (`sqleton/cmd/sqleton/main_test.go:111-141`).
- `writeSmokeCommandFile()` shows sqleton generating a command file from the spec (`sqleton/cmd/sqleton/main_test.go:215-237`).

These tests are a good template for the kind of smoke coverage go-minitrace will need.

### What we should copy from sqleton

Copy the pattern, not the whole runtime:

- repository discovery via config + env + embedded repo
- deterministic file-kind detection
- one neutral parsed spec
- separate compile/adapt stage
- tests that prove embedded and local repositories merge correctly

### What we should not copy blindly

Do **not** import sqleton’s runtime command behavior wholesale.

Why:

- sqleton commands connect to arbitrary databases and inject SQL-connection layers
- go-minitrace commands always execute against one already-loaded DuckDB analysis table
- sqleton command semantics are broader than go-minitrace needs

So the shared concept is **source loading**, not execution wiring.

## Current state: go-minitrace

### Current CLI query model

The current `query` root is intentionally small.

- `cmd/go-minitrace/cmds/query/root.go` only mounts the `duckdb` subcommand (`cmd/go-minitrace/cmds/query/root.go:5-20`).
- `DuckDBQuerySettings` only supports `archive-glob`, `db-path`, `table-name`, `preset`, `sql`, `sql-file`, `load-only`, and `persist-loaded` (`cmd/go-minitrace/cmds/query/duckdb.go:25-34`).
- The command explicitly enforces exactly one query source (`cmd/go-minitrace/cmds/query/duckdb.go:90-111`).
- After loading the archive it resolves one SQL text and executes it (`cmd/go-minitrace/cmds/query/duckdb.go:120-144`).

This is a clean ad hoc query command, but it is not a repository-backed verb system.

### Current built-in preset model

`pkg/query/assets.go` embeds six SQL preset files and exposes them by name.

- embedded preset FS: `//go:embed presets/*.sql` (`pkg/query/assets.go:11-12`)
- static map of six preset names to files (`pkg/query/assets.go:14-21`)
- `ListPresets()` returns names only (`pkg/query/assets.go:23-30`)
- `ResolvePresetSQL()` reads the file and only substitutes `{{TABLE_NAME}}` (`pkg/query/assets.go:32-43`)

This is a name → SQL-text mapping, not a structured command catalog.

### Current DuckDB execution model

`pkg/query/engine.go` is the other major current-state anchor.

- `LoadArchive()` validates the table name, expands archive globs, and executes one generated `CREATE OR REPLACE ... AS SELECT * FROM read_json(...)` statement (`pkg/query/engine.go:44-58`, `68-98`).
- `ResolveSQL()` chooses one of: preset name, inline SQL, or SQL file (`pkg/query/engine.go:151-170`).
- `RunIntoProcessor()` executes one SQL string and streams arbitrary result rows (`pkg/query/engine.go:172-205`).

Again: the engine executes already-rendered SQL. It does not know about source repositories, metadata preambles, aliases, or forms.

### Current `serve` query-library model

`go-minitrace serve` already has some repository-like behavior, but it is not yet structured enough for forms.

`ServeSettings` defines:

- repeated `--preset-dir` for read-only SQL preset roots (`cmd/go-minitrace/cmds/serve/serve.go:29-36`, `69-77`)
- repeated `--query-dir` for read-write saved SQL roots (`cmd/go-minitrace/cmds/serve/serve.go:29-36`, `69-77`)

The server stores those roots as `presetDirs` and `queryDirs` (`cmd/go-minitrace/cmds/serve/server.go:24-34`, `56-75`).

So go-minitrace already has:

- embedded presets,
- extra read-only preset directories,
- extra read-write query directories,
- root ordering and de-duplication by relative path.

That existing behavior is useful, but it is still based on **plain `.sql` file scanning**.

### Current saved-query / preset handlers

`cmd/go-minitrace/cmds/serve/handlers_queries.go` reveals exactly what the current UI contract can represent.

#### Data model

`SavedQuery` only has:

- `name`
- `folder`
- `path`
- `description`
- `sql`
- `readonly`

(`cmd/go-minitrace/cmds/serve/handlers_queries.go:15-22`)

There is no room for:

- parameters / flags
- required vs optional fields
- argument order
- tags
- long help
- alias information
- source kind
- default values beyond whatever raw SQL already contains

#### Preset loading

`handleGetPresets()`:

- turns embedded preset names into `SavedQuery` objects by reading SQL text and deriving the description from the first `-- ` comment line (`cmd/go-minitrace/cmds/serve/handlers_queries.go:42-83`)
- optionally appends external preset dirs through `loadSQLDirs()` (`cmd/go-minitrace/cmds/serve/handlers_queries.go:67-80`)

#### Saved-query loading and CRUD

`handleGetQueries()` and CRUD routes operate on plain files under `queryDirs` (`cmd/go-minitrace/cmds/serve/handlers_queries.go:85-180`, `183-286`).

The helper layer:

- walks each directory looking for `.sql` only (`cmd/go-minitrace/cmds/serve/handlers_queries.go:344-378`)
- loads a single file into `SavedQuery` by folder/path/comment/sql text (`cmd/go-minitrace/cmds/serve/handlers_queries.go:380-405`)
- extracts only the first `-- ` comment as description (`cmd/go-minitrace/cmds/serve/handlers_queries.go:407-418`)
- builds saved files as `-- description\n<sql>` (`cmd/go-minitrace/cmds/serve/handlers_queries.go:420-426`)
- validates paths and keeps writes inside configured roots (`cmd/go-minitrace/cmds/serve/handlers_queries.go:428-588`)

This is a solid raw-SQL file library, but it is not a verb loader.

### Current read-only query execution guard

The server’s raw execution endpoint is intentionally conservative.

- `POST /api/query` requires a raw SQL string (`cmd/go-minitrace/cmds/serve/server.go:155-203`).
- `validateReadOnlyQuery()` only allows normalized SQL beginning with `SELECT`, `WITH`, `EXPLAIN`, `DESCRIBE`, or `SHOW` (`cmd/go-minitrace/cmds/serve/server.go:301-316`).
- `normalizeQueryForValidation()` strips leading comments, rejects multiple statements, and rejects empty SQL (`cmd/go-minitrace/cmds/serve/server.go:318-356`).

That guard should remain in the future design. Form execution should render SQL and then pass through the same guard.

### Current protobuf contract for query metadata

The phase-1 protobuf contract mirrors the same limited raw-query model.

`proto/go_go_golems/minitrace/api/v1/queries.proto` defines:

- `SavedQuery { name, folder, path, description, sql, readonly }` (`proto/go_go_golems/minitrace/api/v1/queries.proto:9-16`)
- list envelopes for presets and queries (`proto/go_go_golems/minitrace/api/v1/queries.proto:18-26`)
- save/update/delete messages for raw query files (`proto/go_go_golems/minitrace/api/v1/queries.proto:28-44`)

There is no message for forms or verbs yet.

### Current frontend model

The frontend is built around the same `SavedQuery` shape.

- `web/src/types/query.ts` defines `SavedQuery` with exactly the same six fields (`web/src/types/query.ts:3-10`).
- `web/src/api/queryProtoAdapters.ts` simply maps protobuf `SavedQuery` into the same TS type (`web/src/api/queryProtoAdapters.ts:10-33`).
- `web/src/pages/QueryEditorPage.tsx` polls presets and saved queries, and selecting one just copies `query.sql` into the editor (`web/src/pages/QueryEditorPage.tsx:34-57`, `105-110`).
- Saving creates a timestamp-based file with `name`, `folder`, `description`, and `sql` only (`web/src/pages/QueryEditorPage.tsx:131-166`).
- `QuerySidebar` only groups entries by folder and renders two sections, Presets and Saved (`web/src/components/QueryEditor/QuerySidebar.tsx:14-18`, `20-37`, `49-136`).
- `QueryEditor` only knows about raw SQL editing, Run, Save, and a file-status banner (`web/src/components/QueryEditor/QueryEditor.tsx:23-37`, `62-207`).

So the UI today can do exactly two things:

1. choose an existing raw SQL blob,
2. edit/run/save raw SQL.

It cannot render structured inputs because none are provided.

## Gap analysis

The request and the current implementation differ in several important ways.

### Gap 1: go-minitrace has file scanning, not repository-backed source semantics

What exists now:

- embedded preset name → SQL map
- repeated `--preset-dir`
- repeated `--query-dir`
- plain `.sql` scanning

What is needed:

- embedded repository root
- external repository roots from config/env/flags
- deterministic source-kind detection
- shared catalog view across CLI and UI

### Gap 2: go-minitrace can represent SQL text, but not form metadata

Current metadata contract:

- one-line description from `-- comment`
- name from filename
- folder/path from relative location

Missing metadata:

- parameter definitions
- required/default/choice information
- longer help text
- tags
- alias declarations
- display layout hints

### Gap 3: current preset SQL files are not the same thing as future form-backed verbs

This distinction matters a lot.

There are currently **two SQL worlds** in go-minitrace:

1. `pkg/query/presets/*.sql`
   - embedded app-level presets used by `query duckdb` and `/api/presets`
2. top-level `queries/*.sql` plus `queries/load.sql`
   - external DuckDB CLI workflow documented in `queries/README.md`

The second world is intentionally a raw DuckDB CLI recipe set. `queries/load.sql` is not a form-backed verb candidate; it is an interactive shell helper. If we mix these concerns, the feature will become confusing very quickly.

### Gap 4: the frontend sidebar model is too small

The current sidebar can only display folders of `SavedQuery` entries. It cannot display:

- verbs vs aliases
- raw SQL entries vs form-backed entries
- parameter badges
- tags / categories
- readonly structured commands loaded from repositories

### Gap 5: execution is raw-SQL only

Current server execution receives a string:

```text
UI SQL editor
  -> POST /api/query { sql: "..." }
  -> validateReadOnlyQuery(...)
  -> conn.QueryContext(...)
```

Future form execution needs a new step:

```text
UI form values
  -> resolve verb definition
  -> render SQL template with safe helper funcs
  -> validateReadOnlyQuery(renderedSQL)
  -> execute rendered SQL
```

## Design goals

1. **Reuse sqleton’s source format and repository-loading ideas.**
2. **Do not couple go-minitrace to sqleton runtime command behavior.**
3. **Preserve the current raw SQL editor as a power-user path.**
4. **Make form-backed verbs available to both CLI and UI.**
5. **Keep query execution read-only.**
6. **Keep the source model explicit and deterministic.**
7. **Keep the first version understandable to a new engineer.**

## Proposed solution

## Overview

Add a new subsystem called the **query catalog**.

The query catalog is the missing architectural layer between source files and execution.

```text
embedded repo + external repos
          |
          v
   source-kind detection
          |
          v
   parse SQL/YAML sources
          |
          v
    neutral catalog model
          |
     +----+------------------+
     |                       |
     v                       v
 CLI command compiler   API/UI adapters
     |                       |
     v                       v
go-minitrace query ...   query forms in UI
     \                       /
      \                     /
       v                   v
      render SQL server-side
               |
               v
      existing read-only validation
               |
               v
       existing DuckDB execution
```

## Key design decision

**Reuse the sqleton file format and repository pattern, but build a go-minitrace-native catalog and compiler.**

That means:

- yes to sqleton-style `.sql` preambles,
- yes to explicit `.alias.yaml` shortcuts,
- yes to config/env/flag repository discovery,
- no to importing sqleton’s DB connection layers or runtime command types into go-minitrace execution.

## The proposed source model

### Source kinds

Use explicit source kinds from day one.

```go
type SourceKind int

const (
    SourceUnknown SourceKind = iota
    SourceSQLVerb
    SourceAliasYAML
    SourceRawSQL // optional migration-only kind
)
```

Recommended rules:

- `*.sql` with a valid `/* sqleton ... */` preamble => `SourceSQLVerb`
- `*.alias.yaml` / `*.alias.yml` => `SourceAliasYAML`
- plain `.sql` without a preamble => either reject, or treat as a separate raw-SQL library kind during migration

For the cleanest design, use only the first two kinds for the new catalog.

### Structured SQL files

Use the sqleton command format directly:

```sql
/* sqleton
name: framework-summary
short: Summarize sessions by framework
flags:
  - name: framework
    type: stringList
    help: Restrict to one or more frameworks
  - name: min_turns
    type: int
    default: 0
    help: Minimum turn count
*/
SELECT
  environment->>'agent_framework' AS framework,
  COUNT(*) AS sessions,
  AVG(CAST(metrics->>'turn_count' AS INT)) AS avg_turns
FROM {{TABLE_NAME}}
WHERE 1 = 1
{{ if .framework }}
  AND environment->>'agent_framework' IN ({{ .framework | sqlStringIn }})
{{ end }}
{{ if .min_turns }}
  AND CAST(metrics->>'turn_count' AS INT) >= {{ .min_turns }}
{{ end }}
GROUP BY 1
ORDER BY sessions DESC;
```

Notes for the intern:

- `{{TABLE_NAME}}` remains a server/compiler-controlled substitution, just like current built-in presets.
- user parameters are rendered from the form/CLI values map.
- the file remains human-readable SQL.

### Alias YAML files

Support sqleton-style alias files as lightweight shortcuts:

```yaml
name: codex-framework-summary
aliasFor: framework-summary
flags:
  framework:
    - codex
```

These should resolve into a form/command that points at an underlying SQL verb and pre-applies defaults.

### Do not overload the top-level `queries/` directory

Keep this distinction explicit:

- top-level `queries/` remains the external DuckDB CLI recipe library
- new form/verb repositories should live in a separate dedicated tree, for example:

```text
pkg/querycatalog/core/
  framework-summary.sql
  session-list.sql
  aliases/
    codex-framework-summary.alias.yaml
```

That separation will save future engineers a lot of confusion.

## Neutral catalog model

Create a new local package, for example `pkg/querycatalog`, with a model like this:

```go
type CatalogEntryKind string

const (
    CatalogEntryVerb  CatalogEntryKind = "verb"
    CatalogEntryAlias CatalogEntryKind = "alias"
)

type ParamType string

const (
    ParamString     ParamType = "string"
    ParamInt        ParamType = "int"
    ParamBool       ParamType = "bool"
    ParamDate       ParamType = "date"
    ParamChoice     ParamType = "choice"
    ParamStringList ParamType = "stringList"
    ParamIntList    ParamType = "intList"
)

type ParamSpec struct {
    Name       string
    Type       ParamType
    Help       string
    Required   bool
    DefaultRaw any
    Choices    []string
    Positional bool
}

type VerbSpec struct {
    Name        string
    Folder      string
    Path        string
    Short       string
    Long        string
    Tags        []string
    Parameters  []ParamSpec
    SQLTemplate string
    SourceKind  SourceKind
    SourceRoot  string
    Readonly    bool
}

type AliasSpec struct {
    Name       string
    Folder     string
    Path       string
    AliasFor   string
    PresetArgs map[string]any
    SourceRoot string
}
```

Important design note: this model is **not** the same as current `SavedQuery`.

`SavedQuery` is a raw editor/view model.

`VerbSpec` is a source/catalog model.

That distinction keeps the system understandable.

## Repository discovery for go-minitrace

Mirror sqleton’s pattern.

### Recommended inputs

1. embedded repository root, always mounted
2. app config file, for example `~/.go-minitrace/config.yaml`
3. env var, for example `GO_MINITRACE_QUERY_REPOSITORIES`
4. repeated CLI flag, for example `--query-repository`

### Suggested config shape

```yaml
queryRepositories:
  - /home/manuel/code/team/minitrace-queries
  - ./queries/minitrace-forms
```

### Suggested env shape

```bash
export GO_MINITRACE_QUERY_REPOSITORIES=/path/a:/path/b
```

### Suggested CLI flags

Add to both `go-minitrace query` and `go-minitrace serve`:

- `--query-repository <dir>` (repeatable)

Why on both commands?

- `serve` needs them for UI form discovery
- `query` needs them for CLI verbs

### Why not reuse `--preset-dir` and `--query-dir` forever

Those current flags describe **raw SQL storage semantics**, not repository-backed verb semantics.

A clean long-term shape is:

- repository flags for structured verb sources
- workspace flags only if you still want writable raw-SQL scratchpads

For implementation safety, you can stage this in phases, but the end state should keep the concepts separate.

## Compilation model

The catalog should feed two adapters.

### Adapter 1: CLI command compiler

Compile `VerbSpec` into a Glazed command mounted under `go-minitrace query ...`.

Suggested runtime flow:

```text
go-minitrace query <repo path> <verb> --framework codex
    |
    v
parse CLI flags into values map
    |
    v
open duckdb connection
    |
    v
load archive into {{TABLE_NAME}}
    |
    v
render SQL template with values + system context
    |
    v
validate read-only SQL
    |
    v
run existing queryengine.RunIntoProcessor(...)
```

### Adapter 2: API / UI adapter

Expose the same `VerbSpec` through a new API.

The UI does not need the whole internal model. It needs a transport DTO like:

- id/path/folder/name
- short/long description
- parameters
- readonly flag
- maybe rendered SQL preview or raw template for “view source” mode

## Backend API proposal

The current `SavedQuery` API is not enough. Add a new query-verb API instead of overloading `SavedQuery`.

### Why a new API is better than extending `SavedQuery`

`SavedQuery` currently means “raw SQL blob on disk.”

A form-backed verb is a different thing:

- it has parameter metadata
- it may have aliases
- it may come from an embedded repo rather than a writable file root
- it should usually be readonly

Overloading one message with both meanings will make the frontend and backend harder to reason about.

### Suggested protobuf messages

```proto
message QueryParam {
  string name = 1;
  string type = 2;
  string help = 3;
  bool required = 4;
  string default_json = 5;
  repeated string choices = 6;
  bool positional = 7;
}

message QueryVerb {
  string name = 1;
  string folder = 2;
  string path = 3;
  string short_description = 4;
  string long_description = 5;
  repeated QueryParam params = 6;
  repeated string tags = 7;
  bool readonly = 8;
  string source_kind = 9;
  string source_root = 10;
  string sql_template = 11;
}

message ListQueryVerbsResponse {
  ApiMeta meta = 1;
  repeated QueryVerb verbs = 2;
}

message ExecuteQueryVerbRequest {
  string path = 1;
  map<string, string> values = 2;
  bool render_only = 3;
}
```

### Execution response recommendation

Keep dynamic query results JSON-native for now, exactly like `POST /api/query`.

Reason:

- the result row shape is still dynamic
- this repo already chose JSON-native execution results for that reason
- the structured part that benefits from protobuf is the verb metadata, not arbitrary SQL result rows

### Suggested new routes

- `GET /api/v2/query-verbs`
- `POST /api/v2/query-verbs/{path...}/execute`
- optional: `POST /api/v2/query-verbs/{path...}/render`

`execute` can internally:

1. load the catalog entry by path,
2. merge alias defaults,
3. render SQL,
4. validate read-only SQL,
5. run the existing query execution pipeline,
6. optionally include `rendered_sql` in the JSON envelope for debugging.

## SQL rendering proposal

This is one of the trickiest parts.

### What we need

We need sqleton-style template rendering helpers for user parameters.

Examples:

- `sqlString`
- `sqlStringIn`
- `sqlIntIn`
- `sqlLike`
- maybe `sqlDate`

### Recommended approach

Implement a small go-minitrace-local renderer package that intentionally supports the subset of sqleton helpers needed by the go-minitrace verb library.

For example:

```text
pkg/queryrender/
  funcs.go
  render.go
  render_test.go
```

Why local first:

- go-minitrace does not currently depend on `clay/pkg/sql`
- a local subset makes the runtime surface explicit
- the execution contract is easier for an intern to understand

### Alternative approach

If you want full sqleton template compatibility immediately, extract the reusable helper surface into a small shared package and depend on that from both repos.

That is a valid long-term direction, but it is a broader refactor than this feature needs.

### Rendering order

The safe rendering order should be:

1. load `VerbSpec`
2. apply alias defaults
3. validate user input types
4. render parameterized SQL template
5. replace `{{TABLE_NAME}}` using validated system table name
6. run `validateReadOnlyQuery(renderedSQL)`
7. execute

The read-only validation must happen **after** rendering.

## UI proposal

## High-level UI shape

Keep the current query editor page, but add a new structured-form path.

```text
Query Editor page
├── Verbs / Forms
│   ├── core
│   ├── team
│   └── aliases
├── Presets / Raw SQL (optional migration slice)
├── Saved SQL
└── Ad hoc editor
```

### Recommended user experience

When a user selects a verb:

1. the sidebar selects the verb,
2. a form panel appears above or next to the SQL editor,
3. fields are rendered from `QueryParam[]`,
4. pressing Run sends form values to `execute-verb`,
5. the results table reuses the existing result renderer,
6. optionally a “Show rendered SQL” button reveals the generated SQL.

### Why keep the raw SQL editor

The raw SQL editor is still valuable because:

- analysts need ad hoc exploration
- debugging rendered SQL is easier with a text view
- the current UI and server already support it well

So the structured forms should be an additive capability, not a replacement.

### Current frontend files that will need to change

- `web/src/types/query.ts` currently only models `SavedQuery` (`web/src/types/query.ts:3-10`)
- `web/src/api/queryProtoAdapters.ts` only adapts `SavedQuery` (`web/src/api/queryProtoAdapters.ts:10-33`)
- `web/src/pages/QueryEditorPage.tsx` only knows how to select a raw SQL source and copy its `sql` field into Redux (`web/src/pages/QueryEditorPage.tsx:50-57`, `105-110`)
- `web/src/components/QueryEditor/QuerySidebar.tsx` only renders two hard-coded groups (`web/src/components/QueryEditor/QuerySidebar.tsx:49-136`)
- `web/src/components/QueryEditor/QueryEditor.tsx` only renders the SQL editor and action buttons, not a parameter form (`web/src/components/QueryEditor/QueryEditor.tsx:62-207`)

### Suggested new frontend pieces

- `web/src/types/queryVerb.ts`
- `web/src/api/queryVerbProtoAdapters.ts`
- `web/src/components/QueryEditor/VerbForm.tsx`
- `web/src/components/QueryEditor/VerbBadge.tsx` (optional)
- `web/src/components/QueryEditor/QuerySidebar.tsx` update to show verbs/aliases/workspace
- `web/src/pages/QueryEditorPage.tsx` update to manage active verb state and form values

### Suggested UI state model

```ts
interface ActiveVerbSource {
  path: string;
  values: Record<string, unknown>;
}
```

Do not try to stuff verb state into the existing `SavedQuery` type.

## Detailed architecture walkthrough for an intern

## 1. Repository layer

This layer answers: “Which directories should be scanned?”

Inputs:

- embedded repository FS
- config file entries
- env var entries
- repeated CLI flag entries

Outputs:

- normalized ordered repository roots

Responsibilities:

- trim whitespace
- dedupe repeated paths
- preserve root ordering
- mark which roots are writable vs readonly if needed later

## 2. Discovery layer

This layer answers: “Which files inside those roots are catalog entries?”

Inputs:

- one repository root

Outputs:

- candidate source files with relative paths

Responsibilities:

- recurse through directories
- ignore non-source files
- classify `.sql` with preamble vs alias YAML vs unsupported files
- keep the relative path because the UI and CLI both use it

## 3. Parse layer

This layer answers: “What does this source file mean?”

Inputs:

- file bytes
- relative path
- source kind

Outputs:

- `VerbSpec` or `AliasSpec`

Responsibilities:

- split SQL preamble/body
- YAML-decode the preamble
- validate required metadata
- resolve alias target name/path semantics
- produce a neutral model

## 4. Catalog layer

This layer answers: “What is the final list of verbs visible to the app?”

Inputs:

- parsed entries from all roots

Outputs:

- ordered catalog with de-duplication rules

Responsibilities:

- merge embedded and external roots
- handle path collisions deterministically
- optionally let earlier roots override later roots
- resolve alias references against loaded verbs

Recommended rule: earlier roots win, matching the current multi-root query-dir behavior where the first resolved path is the authoritative one.

## 5. Compiler / adapter layer

This layer answers: “How should this catalog entry appear in each runtime surface?”

Outputs:

- CLI Glazed commands
- protobuf API DTOs
- optional debug output / list commands

This is where the catalog is adapted, not where it is parsed.

## 6. Execution layer

This layer answers: “How does a selected verb actually run?”

Inputs:

- `VerbSpec`
- user values
- system context (`archive-glob`, `db-path`, `table-name`, `persist-loaded`)

Outputs:

- query results

Responsibilities:

- open/load DuckDB exactly like current `query duckdb`
- render SQL template
- validate read-only SQL
- run the current execution path

That is the full mental model. If an intern keeps those six stages separate, the implementation will stay understandable.

## API references and file references

## Sqleton references

- SQL preamble and repository docs: `sqleton/cmd/sqleton/doc/topics/06-query-commands.md:18-106`
- config/env repository discovery: `sqleton/cmd/sqleton/config.go:13-67`
- embedded + external repo composition: `sqleton/cmd/sqleton/main.go:217-273`
- source-kind dispatch and loading: `sqleton/pkg/cmds/loaders.go:22-93`
- neutral spec + parser + marshaller: `sqleton/pkg/cmds/spec.go:36-244`
- smoke tests proving repository discovery and execution: `sqleton/cmd/sqleton/main_test.go:19-169`, `215-249`

## go-minitrace references

- built-in preset registry: `go-minitrace/pkg/query/assets.go:11-43`
- query engine loading / resolving / executing: `go-minitrace/pkg/query/engine.go:44-205`
- ad hoc CLI query command: `go-minitrace/cmd/go-minitrace/cmds/query/duckdb.go:25-145`
- serve flags for preset/query dirs: `go-minitrace/cmd/go-minitrace/cmds/serve/serve.go:29-77`
- server routes and query execution: `go-minitrace/cmd/go-minitrace/cmds/serve/server.go:24-116`, `155-205`, `301-356`
- raw saved-query model and CRUD: `go-minitrace/cmd/go-minitrace/cmds/serve/handlers_queries.go:15-588`
- query metadata proto: `go-minitrace/proto/go_go_golems/minitrace/api/v1/queries.proto:9-44`
- frontend query type: `go-minitrace/web/src/types/query.ts:3-30`
- frontend adapters: `go-minitrace/web/src/api/queryProtoAdapters.ts:10-33`
- query editor page orchestration: `go-minitrace/web/src/pages/QueryEditorPage.tsx:22-168`
- query sidebar: `go-minitrace/web/src/components/QueryEditor/QuerySidebar.tsx:14-138`
- query editor widget: `go-minitrace/web/src/components/QueryEditor/QueryEditor.tsx:23-207`

## Proposed file layout

A concrete, intern-friendly layout could be:

```text
go-minitrace/
  pkg/
    querycatalog/
      types.go
      discover.go
      parse_sql.go
      parse_alias.go
      catalog.go
      catalog_test.go
    queryrender/
      funcs.go
      render.go
      render_test.go
    queryverbs/
      compiler.go
      compiler_test.go
      core/
        session-list.sql
        framework-summary.sql
        aliases/
          codex-framework-summary.alias.yaml
  cmd/go-minitrace/cmds/query/
    root.go
    duckdb.go
    repository_commands.go
  cmd/go-minitrace/cmds/serve/
    handlers_query_verbs_v2.go
    handlers_queries.go
    serve.go
    server.go
    server_test.go
  proto/go_go_golems/minitrace/api/v1/
    query_verbs.proto
  web/src/
    types/queryVerb.ts
    api/queryVerbProtoAdapters.ts
    components/QueryEditor/VerbForm.tsx
    pages/QueryEditorPage.tsx
```

This keeps the layers visible and keeps query-catalog code out of the existing low-level DuckDB engine.

## Phased implementation plan

## Phase 1: introduce a local query-catalog package

Create:

- `pkg/querycatalog/types.go`
- `pkg/querycatalog/parse_sql.go`
- `pkg/querycatalog/parse_alias.go`
- `pkg/querycatalog/discover.go`
- `pkg/querycatalog/catalog.go`

Tasks:

1. implement `SourceKind` detection
2. implement sqleton-style SQL preamble splitting
3. implement `VerbSpec` parsing
4. implement `.alias.yaml` parsing
5. load one repository root into a catalog
6. add unit tests for malformed preambles, empty body, bad YAML, and alias resolution

Deliverable: a testable in-memory catalog with no CLI/UI code yet.

## Phase 2: add repository discovery inputs

Create a small go-minitrace config helper modeled after sqleton’s `config.go`.

Tasks:

1. add app config file support
2. add `GO_MINITRACE_QUERY_REPOSITORIES`
3. add `--query-repository` to `serve`
4. add `--query-repository` to the `query` group / loader setup
5. define root-order precedence rules and test them

Deliverable: go-minitrace can load an embedded catalog plus additional roots.

## Phase 3: add CLI verbs under `go-minitrace query`

Tasks:

1. write a compiler that turns `VerbSpec` into a Glazed command
2. mount those commands under the `query` root alongside `duckdb`
3. inject shared analysis flags (`archive-glob`, `db-path`, `table-name`, `persist-loaded`)
4. render SQL server-side / command-side before execution
5. reuse `queryengine.LoadArchive()` and `queryengine.RunIntoProcessor()`

Suggested CLI shape:

```bash
go-minitrace query session-list --archive-glob './output/active/*/*.minitrace.json'
go-minitrace query framework-summary --framework codex
```

Deliverable: repository-backed verbs work from the CLI.

## Phase 4: add form-backed backend APIs

Tasks:

1. define `query_verbs.proto`
2. implement `GET /api/v2/query-verbs`
3. implement `POST /api/v2/query-verbs/{path...}/execute`
4. reuse current read-only validation after rendering
5. keep result rows JSON-native

Deliverable: the frontend can fetch structured verbs and execute them without building SQL on the client.

## Phase 5: add UI query forms

Tasks:

1. add frontend types and adapters for query verbs
2. add a `VerbForm` component that renders fields by parameter type
3. update `QuerySidebar` to show verbs separately from raw saved SQL
4. update `QueryEditorPage` to manage active verb + form values + rendered results
5. optionally add “Show rendered SQL” / “Open in editor” actions

Deliverable: the UI can browse repository-backed verbs and run them through forms.

## Phase 6: migrate core built-ins into the new catalog

Tasks:

1. create a new embedded structured repository
2. port current built-in presets into sqleton-style `.sql` files with metadata preambles
3. keep top-level `queries/` as external DuckDB CLI recipes
4. document the difference clearly

Deliverable: one coherent built-in query catalog for both CLI verbs and UI forms.

## Testing and validation strategy

## Parser tests

Add focused unit tests for:

- valid sqleton-style SQL file
- missing preamble
- invalid preamble marker
- unterminated preamble
- empty metadata
- empty SQL body
- unsupported file kind
- valid alias YAML
- alias pointing at missing target

## Catalog tests

Add tests for:

- embedded root + external root merge
- earlier root wins on duplicate path
- nested folders preserve correct `folder` / `path`
- aliases resolve after root merging

## CLI tests

Mirror sqleton’s smoke tests.

Examples:

- load one repository via `--query-repository`
- execute a repository-backed verb against a tiny temp archive / temp DuckDB fixture
- verify parameter defaults and bool flags work
- verify invalid non-read-only rendered SQL is rejected

## Server tests

Add handler tests for:

- `GET /api/v2/query-verbs`
- `POST /api/v2/query-verbs/{path}/execute`
- missing verb path
- invalid parameter values
- alias execution
- repository override precedence

## Frontend tests / stories

Add stories and tests for:

- string/int/bool/list field rendering
- required-field validation
- alias badge display
- sidebar grouping for verbs vs raw SQL
- rendered-SQL preview state

## Risks and tradeoffs

### Risk 1: importing too much sqleton runtime code

If the implementation directly depends on sqleton command execution types, go-minitrace will inherit runtime concerns it does not need.

Mitigation: reuse source format and ideas, keep go-minitrace execution local.

### Risk 2: confusing raw SQL libraries with structured verbs

If the new catalog scans every `.sql` file in the repo, engineers will accidentally try to load `queries/load.sql` and other external DuckDB helper files as UI verbs.

Mitigation: use a dedicated structured repository root.

### Risk 3: parameter rendering helper scope grows too fast

If the first implementation tries to support every sqleton helper and every edge case, this feature will sprawl.

Mitigation: start with the helper subset needed by the built-in go-minitrace verb catalog.

### Risk 4: frontend state becomes tangled with raw SQL editor state

If verb-form state is squeezed into the existing `SavedQuery` / raw editor flow, the page will become fragile.

Mitigation: add a separate active-verb state model.

## Alternatives considered

### Alternative 1: extend `SavedQuery` with `params[]`

Rejected as the main design.

Reason:

- `SavedQuery` currently means “raw SQL file”
- overloading it will blur two different concepts
- aliases and structured repository metadata do not fit naturally into that type

### Alternative 2: import sqleton wholesale and compile its commands directly

Rejected as the main design.

Reason:

- too much runtime coupling
- sqleton’s database/runtime model is broader than go-minitrace’s needs
- it hides the important difference between source parsing and execution environment

### Alternative 3: build forms client-side from comments in raw SQL files

Rejected.

Reason:

- comments are not a stable schema
- you would re-invent a parser in the frontend
- the CLI would still not get verbs

### Alternative 4: keep only raw SQL editor and add a few hard-coded forms

Rejected.

Reason:

- it does not satisfy the repository-backed verb-loading goal
- it creates hand-maintained UI forms disconnected from source files

## Open questions

1. Do you want pure YAML command specs in go-minitrace, or is “SQL command files + alias YAML” enough for v1?
2. Should raw `--preset-dir` / `--query-dir` survive as a separate legacy surface during migration, or do you want a hard cut to repository-backed verbs plus ad hoc SQL only?
3. Should aliases be supported in v1, or can they wait until after plain SQL verbs work?
4. Should go-minitrace keep a local rendering-helper subset, or should we extract a shared helper package from sqleton/clay immediately?
5. Should the UI allow saving new form-backed verbs, or is file-authoring on disk enough for the first version?
6. Do you want repository-backed verbs mounted directly under `go-minitrace query ...`, or under a dedicated subcommand like `go-minitrace query verb ...`?

## Recommended next steps

If I were handing this to an intern tomorrow, I would tell them to do the following in order:

1. read sqleton’s query-command doc and parser files first
2. implement the local go-minitrace query-catalog parser with tests only
3. wire repository discovery second
4. wire one backend list API before touching the UI
5. render one simple form end-to-end in the UI before designing fancy layout features
6. only then port the current built-in presets into the new format

That order keeps the risky parts isolated and makes debugging much easier.

## References

- `sqleton/cmd/sqleton/doc/topics/06-query-commands.md:18-138`
- `sqleton/cmd/sqleton/config.go:13-85`
- `sqleton/cmd/sqleton/main.go:217-273`
- `sqleton/pkg/cmds/loaders.go:22-93`
- `sqleton/pkg/cmds/spec.go:16-244`
- `sqleton/cmd/sqleton/main_test.go:19-169`
- `go-minitrace/pkg/query/assets.go:11-43`
- `go-minitrace/pkg/query/engine.go:44-205`
- `go-minitrace/cmd/go-minitrace/cmds/query/duckdb.go:25-145`
- `go-minitrace/cmd/go-minitrace/cmds/serve/serve.go:29-155`
- `go-minitrace/cmd/go-minitrace/cmds/serve/server.go:24-116`
- `go-minitrace/cmd/go-minitrace/cmds/serve/server.go:155-205`
- `go-minitrace/cmd/go-minitrace/cmds/serve/server.go:301-356`
- `go-minitrace/cmd/go-minitrace/cmds/serve/handlers_queries.go:15-588`
- `go-minitrace/proto/go_go_golems/minitrace/api/v1/queries.proto:9-44`
- `go-minitrace/web/src/types/query.ts:3-30`
- `go-minitrace/web/src/api/queryProtoAdapters.ts:10-33`
- `go-minitrace/web/src/pages/QueryEditorPage.tsx:22-168`
- `go-minitrace/web/src/components/QueryEditor/QuerySidebar.tsx:14-138`
- `go-minitrace/web/src/components/QueryEditor/QueryEditor.tsx:23-207`
