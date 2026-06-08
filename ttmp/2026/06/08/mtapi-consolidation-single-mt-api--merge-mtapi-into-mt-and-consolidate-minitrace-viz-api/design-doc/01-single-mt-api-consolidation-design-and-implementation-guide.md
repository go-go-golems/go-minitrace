---
Title: Single mt API consolidation design and implementation guide
Ticket: mtapi-consolidation-single-mt-api
Status: active
Topics:
    - minitrace
    - architecture
    - xgoja
    - goja
    - javascript
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ClubMedMeetup/minitrace-viz/lib/course-session-data.js
      Note: WidgetRenderer transcript/context shaping that should remain app-side
    - Path: ClubMedMeetup/minitrace-viz/lib/session-service.js
      Note: Upload/import call site to migrate to mt.importer fluent builder
    - Path: ClubMedMeetup/minitrace-viz/lib/timeline-data.js
      Note: Timeline call site to migrate from archive turnBlocks to SQL/view helpers
    - Path: ClubMedMeetup/minitrace-viz/pkg/mtapiprovider/provider.go
      Note: Old ClubMed mt provider API to delete after clean cutover
    - Path: ClubMedMeetup/minitrace-viz/xgoja.yaml
      Note: Current dual mt/minitrace module wiring to consolidate
    - Path: go-minitrace/pkg/minitracedb/convert.go
      Note: Existing auto-detection and conversion foundation for mt.importer
    - Path: go-minitrace/pkg/minitracedb/materialize.go
      Note: Existing normalized SQLite materialization and event insertion path
    - Path: go-minitrace/pkg/minitracedb/schema.go
      Note: Schema source of truth for sessions
    - Path: go-minitrace/pkg/minitracejs/db_builder.go
      Note: Current broad fluent DB builder assessed for API cleanup
    - Path: go-minitrace/pkg/minitracejs/module.go
      Note: Current go-minitrace JS module export surface to replace/extend
ExternalSources: []
Summary: Design for replacing ClubMedMeetup mtapi with one clear require("mt") API centered on staged fluent Go-owned builders, transcript sessions, SQL recipes, and opinionated view helpers.
LastUpdated: 2026-06-08T21:05:00-04:00
WhatFor: Use this as the implementation guide for an intern consolidating mtapi into go-minitrace and refactoring minitrace-viz to one mt API.
WhenToUse: Read before changing go-minitrace/pkg/minitracejs, go-minitrace/pkg/minitracedb, or ClubMedMeetup/minitrace-viz data access.
---


# Single mt API Consolidation Design and Implementation Guide

## Executive Summary

This ticket turns the current two-API minitrace JavaScript story into one clean `require("mt")` API. Today the ClubMedMeetup visualization site uses a local `mt` module backed by `ClubMedMeetup/minitrace-viz/pkg/mtapi`, while the reusable `go-minitrace` package exposes `require("minitrace")` with a normalized SQLite builder. The local `mtapi` layer duplicates conversion, archive loading, turn-block projection, and report generation that belong in `go-minitrace`.

The consolidation should be a **clean cutover**, not a compatibility migration. We should remove the local ClubMed provider, remove the old `mt.source(...).detect().convert().save(...)`, `mt.archiveFile(...).turnBlocks(...)`, and `mt.archiveFile(...).report(...)` APIs, and expose a single `mt` module from `go-minitrace` in `ClubMedMeetup/minitrace-viz/xgoja.yaml`.

The new API should also improve the current `go-minitrace` JavaScript shape, but the direction is **builder-first**, not options-map-first. Fluent Go-backed builders are valuable because every intermediate object can be constructed and validated on the Go side. That lets the module carry richer semantics than plain JavaScript maps: typed source sets, cache policies, import policies, query recipes, and view plans can be opaque Go-owned values with explicit methods and validation.

The problem with the current `mt.db()` builder is not that it is fluent. The problem is that one builder owns too many dimensions at once: source selection, conversion behavior, backend/storage, cache behavior, query limits, validation, and build lifecycle are all interleaved. The target is a **staged fluent API with subbuilders**. Subbuilders produce typed Go-owned option objects, and higher-level builders consume those objects.

The recommended target is a layered fluent API:

```js
const mt = require("mt");

// Reusable Go-owned option objects built fluently.
const sources = mt.sources()
  .File(sessionPath)
  .Build();

const cache = mt.cache()
  .Auto()
  .Dir(cacheDir)
  .Build();

// Opinionated path for web apps and interns.
const session = mt.session()
  .Sources(sources)
  .Cache(cache)
  .Open();

const transcript = session.view().Transcript().IncludeTools().Run();
const timeline = session.view().Timeline().Run();
const usage = session.view().TokenUsage().ByTurn().Run();
session.close();

// Explicit SQL path for power users.
const db = mt.db()
  .Sources(sources)
  .Cache(cache)
  .Limits(mt.limits().Rows(1000).CellChars(6000).Build())
  .Build();

const recipe = mt.query()
  .TranscriptRows()
  .SessionID(session.id())
  .IncludeTools()
  .Build();
const rows = db.query(recipe.sql(), ...recipe.args());
db.close();

// Import path for uploads.
const saved = mt.importer()
  .Content(content)
  .Name(name)
  .Into(sessionsDir)
  .SessionID(sessionId)
  .AutoDetect()
  .Convert()
  .Save();
```

In short: **`go-minitrace` owns conversion, normalized storage, query recipes, and generic transcript/timeline/token helpers; `minitrace-viz` owns WidgetRenderer-specific models, teaching notes, and course-specific context-window explanations.**

---

## Goals and Non-Goals

### Goals

- Provide exactly one minitrace JS module in the ClubMedMeetup site: `require("mt")`.
- Move useful `mtapi` behavior into `go-minitrace`:
  - upload/import/save,
  - format detection and diagnostics,
  - normalized SQLite materialization,
  - transcript/turn/timeline/token query recipes,
  - small opinionated view helpers.
- Redesign the `mt` JavaScript API for clarity instead of backwards compatibility.
- Keep SQL access first-class for advanced users.
- Keep the out-of-the-box path friendly for interns and app code.
- Delete the ClubMed-specific Go provider and `mtapi` package after parity.
- Update minitrace-viz to use the new API without compatibility shims.

### Non-Goals

- Do not rebuild the old report builder inside `go-minitrace`.
- Do not preserve old method names merely for compatibility.
- Do not put WidgetRenderer component shapes into `go-minitrace` core.
- Do not hide SQL behind an opaque analytics framework.
- Do not keep both `require("mt")` and `require("minitrace")` in minitrace-viz after cutover.

---

## System Map for a New Intern

### What is minitrace?

Minitrace is a normalized representation of coding-agent transcript sessions. It turns raw logs from systems such as Pi, Codex, and Claude Code into a stable model with sessions, turns, tool calls, files, annotations, metrics, and renderable events. `go-minitrace` can materialize that model into SQLite tables so JavaScript commands and web apps can query transcripts with normal SQL.

### What is ClubMedMeetup/minitrace-viz?

`ClubMedMeetup/minitrace-viz` is a teaching/demo web app. It uploads transcript files, stores them under a sessions directory, and renders transcript, context-window, timeline, and token views through WidgetRenderer pages. It currently has its own `mtapi` Go package because the app needed a quick app-specific API before `go-minitrace` had enough reusable JavaScript support.

### Current module wiring

`ClubMedMeetup/minitrace-viz/xgoja.yaml` registers two separate minitrace-ish modules:

- lines 29-34 register the local `clubmed-minitrace-viz` package and expose it as `mt`;
- lines 55-61 register `go-minitrace` and expose it as `minitrace`.

That means app code sees two concepts:

```js
const mt = require("mt");          // local ClubMed mtapi
const minitrace = require("minitrace"); // reusable go-minitrace module
```

After this ticket, the site should register only the reusable provider and expose it as `mt`:

```yaml
packages:
  - id: go-minitrace
    import: github.com/go-go-golems/go-minitrace/pkg/minitracejs/provider

modules:
  - package: go-minitrace
    name: minitrace
    as: mt
```

### Current data flow

```text
Upload/raw transcript
        |
        v
ClubMed mt.source(...).detect().convert().save(...)
        |
        v
session.minitrace.json + metadata.json
        |
        +--> mt.archiveFile(...).turnBlocks() ----> timeline-data.js
        |                                           |
        |                                           +--> course-session-data.js transcript model
        |                                           +--> course-session-data.js context-window model
        |
        +--> mt.archiveFile(...).report() ---------> /analyze and /api/report

Separate path:
        go-minitrace require("minitrace").db().File(...).Build()
        used by SQL report experiments, but not the main app path.
```

### Target data flow

```text
Upload/raw transcript
        |
        v
mt.importer().Content(...).Name(...).Into(...).SessionID(...).Save()
        |
        v
session.minitrace.json + metadata.json
        |
        v
mt.session().Sources(mt.sources().File(...).Build()).Cache(...).Open()
        |
        +--> session.view().Transcript().Run()       --> app maps to WidgetRenderer transcript panel
        +--> session.view().ContextParts().Run()     --> app maps to teaching context-window panel
        +--> session.view().Timeline().Run()         --> app maps to timeline cards/minimap
        +--> session.view().TokenUsage().ByTurn().Run() --> app maps to token-size views
        +--> session.db().query(...)                 --> debug/power-user SQL
```

The important conceptual change is that the reusable module becomes **session-centered**, **SQL-backed**, and **builder-composed**, not report-builder-centered or map-options-centered.

---

## Evidence from the Current Codebase

### Local ClubMed provider duplicates reusable API space

`ClubMedMeetup/minitrace-viz/pkg/mtapiprovider/provider.go` exposes the current local `mt` module:

- line 30 exposes `mt.source(content, opts)`;
- line 44 exposes `mt.archiveFile(path)`;
- line 51 exposes `mt.reportPresets()`.

Those functions make the local package responsible for import, archive loading, block projection, and report generation.

### Uploads depend on the old builder chain

`ClubMedMeetup/minitrace-viz/lib/session-service.js` imports `mt` at line 3 and uses this chain at line 73:

```js
const source = mt.source(content, { name })
  .detect()
  .convert();
const saved = source.save(SESSIONS_DIR, sessionId);
```

This should become one clear import call, because upload code should not need to understand the internal detect/convert/save state machine.

### Timeline depends on archive turn blocks

`ClubMedMeetup/minitrace-viz/lib/timeline-data.js` imports `mt` at line 1. `buildTimeline()` calls `mt.archiveFile(sessionPath).turnBlocks(...)` at line 4, then reshapes the returned frames into UI cards.

The useful part is the **turn-frame concept**. The problematic part is that the frame is computed by ClubMed-specific Go structs instead of by reusable SQL over `turns`, `tool_calls`, `turn_tool_calls`, and `events`.

### Transcript and context-window models are app-specific

`ClubMedMeetup/minitrace-viz/lib/course-session-data.js` calls `buildTimeline(sessionPath(sessionId))` from `buildTranscriptModel()` at lines 4-6 and from `buildContextWindowModel()` at lines 194-196. The rest of that file adds teaching annotations, WidgetRenderer metadata, context-window estimates, model-limit guesses, and part labels.

That logic should remain app-side because it describes a teaching UI, not a general minitrace data model.

### Report routes are not central to the desired product

`ClubMedMeetup/minitrace-viz/server.js` still exposes report-oriented endpoints:

- `/analyze` starts around line 132 and writes `result.md` / `result.json`;
- `/api/report/:sessionId` calls `mt.archiveFile(...).report().preset(...).build()` around line 191;
- `analyzeSession()` calls `mt.archiveFile(...).report().preset(...)` around line 268.

These routes should be removed or converted to debug examples. They should not drive the new API.

### go-minitrace already owns conversion and SQLite materialization

`go-minitrace/pkg/minitracedb/convert.go` already implements auto loading and conversion:

- `LoadSessionContentAuto` starts at line 54;
- it detects JSONL formats through `DetectJSONLFormat` at line 94;
- it converts Pi, Codex, and Claude Code JSONL into minitrace sessions.

`go-minitrace/pkg/minitracedb/materialize.go` already materializes sessions into SQL:

- `MaterializeSession` starts at line 33;
- it inserts turns and turn events around lines 56-61;
- it inserts tool calls and tool events around lines 70-74.

### The normalized schema already supports the desired views

`go-minitrace/pkg/minitracedb/schema.go` defines the needed tables:

- `sessions` create SQL starts around line 144;
- `turns` create SQL starts around line 236;
- `tool_calls` create SQL starts around line 301;
- `turn_tool_calls` create SQL starts around line 347;
- `events` create SQL starts around line 531.

This means transcript, timeline, token usage, and turn-block helpers can be implemented as SQL contracts and small JS grouping functions.

### Current go-minitrace JS API is powerful but broad

`go-minitrace/pkg/minitracejs/module.go` exposes `exports.db` at line 29. The builder itself lives in `go-minitrace/pkg/minitracejs/db_builder.go`. It has many chain methods:

- source methods: `File`, `Files`, `Archive`, `Content`, `Glob`, `RuntimeArchives`, `Dir`;
- storage/cache methods: `SQLiteMemory`, `SQLiteDiskCache`, `Cache`, `CacheDir`, `ForceRebuild`;
- conversion methods: `AutoConvert`, `StrictConversion`;
- query limit methods: `MaxRows`, `MaxColumns`, `MaxCellChars`, `Timeout`, `RequireOrderBy`;
- introspection/build methods: `sources`, `CacheKey`, `Validate`, `Build`.

This is usable for power users, but it is too much surface for the default app path. It also forces users to remember details such as `.AutoConvert(true)` for raw transcript content.

---

## API Assessment: Keep Fluent Builders, Fix Builder Structure

The previous revision of this ticket proposed an options-map-first API. That direction is now superseded. The correct direction is **fluent builder APIs as much as possible**, because Goja builders let us construct, validate, and carry rich Go-side values instead of repeatedly marshalling and unmarshalling plain JavaScript maps.

The current `mt.db()` builder has one major virtue: every option is visible, chainable, and controlled by Go code. That is worth preserving. The problem is not fluency; the problem is **too many unrelated responsibilities on one fluent object**.

### Concern 1: one broad builder mixes too many domains

A user building a database needs to make decisions in several categories:

1. Which sources should be loaded?
2. Should raw transcript formats be auto-detected and converted?
3. Should the result be cached, and where?
4. What query limits should be enforced?
5. Which transcript/view contract should be produced?

Today all categories are represented by peer methods on one builder:

```js
const db = mt.db()
  .Content(jsonlText, { name: "upload.jsonl" })
  .AutoConvert(true)
  .StrictConversion(true)
  .SQLiteMemory()
  .Cache("auto")
  .CacheDir(cacheDir)
  .MaxRows(1000)
  .MaxCellChars(6000)
  .Build();
```

This chain is explicit, but it does not communicate structure. Source construction, import policy, cache policy, query limits, and final DB construction are different semantic objects. They should be built by different fluent builders.

### Concern 2: plain options maps throw away Go-side semantics

Plain maps are convenient at the call site, but they weaken the boundary:

- all validation happens after an untyped map is decoded;
- invalid combinations are represented too easily;
- nested options require repetitive `map[string]any` decoding;
- rich behavior cannot live on intermediate option objects;
- future extensions can accidentally become stringly typed.

A builder-first API lets Go own each intermediate value:

```js
const sources = mt.sources().Content(jsonlText).Name("upload.jsonl").Build();
const importPolicy = mt.importPolicy().AutoConvert().Strict().Build();
const cache = mt.cache().Auto().Dir(cacheDir).Build();
const limits = mt.limits().Rows(5000).CellChars(12000).TimeoutMs(10000).Build();

const db = mt.db()
  .Sources(sources)
  .Import(importPolicy)
  .Cache(cache)
  .Limits(limits)
  .Build();
```

`SourceSet`, `ImportPolicy`, `CachePolicy`, and `QueryLimits` should be Go-owned values. They may expose `Summary()` or `toJSON()` for debugging, but callers should not construct them as object literals.

### Concern 3: defaults should be presets on builders

The API should still be concise for common transcript work. The answer is not plain maps; it is builder presets and high-level builders.

Examples:

```js
// Upload path: opinionated by default.
const saved = mt.importer()
  .Content(content)
  .Name(name)
  .Into(SESSIONS_DIR)
  .SessionID(sessionId)
  .Save();

// App path: session builder applies transcript-friendly defaults.
const session = mt.session()
  .File(sessionPath)
  .InteractiveCache(CACHE_DIR)
  .Open();

// Query-command path: runtime archive preset.
const db = mt.db()
  .RuntimeArchives()
  .QueryCommandDefaults()
  .Build();
```

Convenience methods such as `.File(...)`, `.InteractiveCache(...)`, and `.QueryCommandDefaults()` can configure internal subbuilders. The important rule is that Go still controls construction.

### Concern 4: the old report builder mixes too many layers

The old `mtapi.ReportBuilder` is also fluent, but it is the wrong abstraction. It mixes querying, analysis lenses, section composition, and Markdown rendering. The new API should keep fluency for construction and execution, but it should not recreate the old report-builder concept. If report-style outputs are useful, implement them as examples over query/view builders.

---

## Proposed Public API

### Module naming

The public JS module in the ClubMedMeetup app should be:

```js
const mt = require("mt");
```

Inside the provider package, the registered module name can remain `minitrace` if xgoja aliases it as `mt`. The user-facing app should not import `require("minitrace")` after this cutover.

### Top-level builder factories

```ts
type MTModule = {
  importer(): ImportBuilder;
  sources(): SourceSetBuilder;
  importPolicy(): ImportPolicyBuilder;
  cache(): CachePolicyBuilder;
  limits(): QueryLimitsBuilder;
  db(): DBBuilder;
  session(): SessionBuilder;
  query(): QueryRecipeBuilder;
  view(): ViewPlanBuilder;
  sql: SQLHelpers;
  runtime: RuntimeInfo;
};
```

Each factory returns a Go-backed builder object. Builders mutate Go structs directly and return either another builder, the parent builder, a Go-owned option object, or an executable handle.

| Builder | Responsibility | Final value |
|---|---|---|
| `mt.importer()` | Detect, convert, and save uploads/files. | `SavedSession` or `ConvertedSession`. |
| `mt.sources()` | Build a typed set of file/content/glob/runtime sources. | `SourceSet`. |
| `mt.importPolicy()` | Build conversion strictness and auto-detection policy. | `ImportPolicy`. |
| `mt.cache()` | Build memory/disk/auto cache behavior. | `CachePolicy`. |
| `mt.limits()` | Build query bounds and validation policy. | `QueryLimits`. |
| `mt.db()` | Compose sources/policies/limits and build a DB handle. | `DBHandle`. |
| `mt.session()` | Compose transcript-friendly defaults and build a session handle. | `SessionHandle`. |
| `mt.query()` | Build named SQL recipes. | `QueryRecipe`. |
| `mt.view()` / `session.view()` | Build and execute generic transcript/timeline/token view plans. | rows/frames. |

### Builder design rules

1. **Prefer typed builder methods over map arguments.** Maps should be rare escape hatches, not the main path.
2. **Use subbuilders for distinct semantic domains.** Do not put every method on `DBBuilder`.
3. **Return Go-owned option objects from `.Build()`.** A `CachePolicy` or `SourceSet` should be a value constructed by Go, not a JavaScript object literal.
4. **Allow concise high-level shortcuts.** `mt.session().File(path).InteractiveCache(dir).Open()` is acceptable if it delegates to internal subbuilders.
5. **Expose inspection methods.** Every built option object should have `Summary()` and/or `toJSON()` for debugging.
6. **Keep old names only when they remain clear.** Clean cutover means no obligation to preserve `source`, `archiveFile`, or report-builder APIs.
7. **Use PascalCase consistently for builder methods.** This matches the current `DBBuilder` style and keeps Go-exported method semantics obvious in JS.

### Import builder

The import builder replaces `mt.source(...).detect().convert().save(...)` with a clearer Go-owned workflow.

```ts
type ImportBuilder = {
  Content(content: string): ImportBuilder;
  File(path: string): ImportBuilder;
  Name(name: string): ImportBuilder;
  SourcePath(path: string): ImportBuilder;
  AutoDetect(): ImportBuilder;
  Format(format: string): ImportBuilder;
  Strict(enabled?: boolean): ImportBuilder;
  Into(rootDir: string): ImportBuilder;
  SessionID(id: string): ImportBuilder;
  Overwrite(enabled?: boolean): ImportBuilder;
  Detect(): DetectResult;
  Convert(): ImportBuilder;
  Converted(): ConvertedSession;
  Save(): SavedSession;
};
```

Usage in `session-service.js`:

```js
const saved = mt.importer()
  .Content(content)
  .Name(name)
  .Into(SESSIONS_DIR)
  .SessionID(sessionId)
  .AutoDetect()
  .Strict()
  .Convert()
  .Save();
```

`Convert()` returns the same builder so the app can inspect `Detect()`, `Converted()`, or `Diagnostics()` if those methods are added. `Save()` writes `session.minitrace.json` and `metadata.json`.

### SourceSet builder

```ts
type SourceSetBuilder = {
  File(path: string): SourceSetBuilder;
  Archive(path: string): SourceSetBuilder;
  Files(paths: string[]): SourceSetBuilder;
  Dir(path: string): SourceSetBuilder;
  Glob(pattern: string): SourceSetBuilder;
  Content(content: string): SourceSetBuilder;
  Name(name: string): SourceSetBuilder;      // applies to most recent content source
  RuntimeArchives(): SourceSetBuilder;
  Build(): SourceSet;
};
```

Examples:

```js
const uploadSources = mt.sources()
  .Content(jsonlText)
  .Name("upload.jsonl")
  .Build();

const appSources = mt.sources()
  .File(sessionPath)
  .Build();

const commandSources = mt.sources()
  .RuntimeArchives()
  .Build();
```

### ImportPolicy, CachePolicy, and QueryLimits builders

```ts
type ImportPolicyBuilder = {
  AutoConvert(enabled?: boolean): ImportPolicyBuilder;
  NativeOnly(): ImportPolicyBuilder;
  Strict(enabled?: boolean): ImportPolicyBuilder;
  Lenient(): ImportPolicyBuilder;
  Build(): ImportPolicy;
};

type CachePolicyBuilder = {
  None(): CachePolicyBuilder;
  Memory(): CachePolicyBuilder;
  Disk(): CachePolicyBuilder;
  Auto(): CachePolicyBuilder;
  Dir(path: string): CachePolicyBuilder;
  ForceRebuild(enabled?: boolean): CachePolicyBuilder;
  Build(): CachePolicy;
};

type QueryLimitsBuilder = {
  Rows(n: number): QueryLimitsBuilder;
  Columns(n: number): QueryLimitsBuilder;
  CellChars(n: number): QueryLimitsBuilder;
  TimeoutMs(n: number): QueryLimitsBuilder;
  RequireOrderBy(enabled?: boolean): QueryLimitsBuilder;
  Build(): QueryLimits;
};
```

These builders are small and composable. They can be reused across `DBBuilder`, `SessionBuilder`, tests, examples, and future APIs.

### DB builder

The DB API remains fluent. The change is that it consumes typed subbuilder outputs and has clearer presets.

```ts
type DBBuilder = {
  Sources(sources: SourceSet): DBBuilder;
  Import(policy: ImportPolicy): DBBuilder;
  Cache(policy: CachePolicy): DBBuilder;
  Limits(limits: QueryLimits): DBBuilder;

  // Convenience methods that configure internal subbuilders.
  File(path: string): DBBuilder;
  Content(content: string): DBBuilder;
  Name(name: string): DBBuilder;
  RuntimeArchives(): DBBuilder;
  AutoConvert(enabled?: boolean): DBBuilder;
  Strict(enabled?: boolean): DBBuilder;
  CacheAuto(dir?: string): DBBuilder;
  QueryCommandDefaults(): DBBuilder;
  InteractiveDefaults(cacheDir?: string): DBBuilder;

  Validate(): ValidationResult;
  CacheKey(): CacheKey;
  SourcesSummary(): any;
  Build(): DBHandle;
};
```

Examples:

```js
// Query-command default.
const db = mt.db()
  .RuntimeArchives()
  .QueryCommandDefaults()
  .Build();

// Web app default with explicit subbuilders.
const db = mt.db()
  .Sources(mt.sources().File(sessionPath).Build())
  .Import(mt.importPolicy().AutoConvert().Strict().Build())
  .Cache(mt.cache().Auto().Dir(CACHE_DIR).Build())
  .Limits(mt.limits().Rows(1000).CellChars(6000).Build())
  .Build();

// Concise equivalent.
const db = mt.db()
  .File(sessionPath)
  .AutoConvert()
  .CacheAuto(CACHE_DIR)
  .Build();
```

### Session builder: the opinionated out-of-the-box path

The session builder is the default app path. It opens one logical transcript session and exposes views over it.

```ts
type SessionBuilder = {
  Sources(sources: SourceSet): SessionBuilder;
  Source(source: SourceSet): SessionBuilder; // alias if singular reads better
  Import(policy: ImportPolicy): SessionBuilder;
  Cache(policy: CachePolicy): SessionBuilder;
  Limits(limits: QueryLimits): SessionBuilder;
  SessionID(id: string): SessionBuilder;

  // Convenience methods.
  File(path: string): SessionBuilder;
  Content(content: string): SessionBuilder;
  Name(name: string): SessionBuilder;
  InteractiveCache(dir?: string): SessionBuilder;
  Strict(enabled?: boolean): SessionBuilder;

  Open(): SessionHandle;
};

type SessionHandle = {
  id(): string;
  summary(): SessionSummary;
  diagnostics(): ImportDiagnostic[];
  cacheInfo(): CacheInfo;
  db(): DBHandle;
  query(sql: string, ...args: any[]): Record<string, any>[];
  view(): ViewPlanBuilder;
  close(): void;
};
```

Common usage:

```js
const session = mt.session()
  .File(sessionPath)
  .SessionID(sessionId)
  .InteractiveCache(CACHE_DIR)
  .Open();
try {
  return {
    summary: session.summary(),
    transcript: session.view().Transcript().IncludeTools().Run(),
    timeline: session.view().Timeline().Run(),
    usage: session.view().TokenUsage().ByTurn().Run(),
  };
} finally {
  session.close();
}
```

### Query recipe builder

`mt.query()` returns a builder for named SQL recipes. Recipes are Go-owned objects that expose SQL and args.

```ts
type QueryRecipeBuilder = {
  SessionSummary(): QueryRecipeBuilder;
  TurnRows(): QueryRecipeBuilder;
  ToolRows(): QueryRecipeBuilder;
  EventRows(): QueryRecipeBuilder;
  TurnBlockRows(): QueryRecipeBuilder;
  TokenUsageRows(): QueryRecipeBuilder;
  TranscriptRows(): QueryRecipeBuilder;
  TimelineRows(): QueryRecipeBuilder;

  SessionID(id: string): QueryRecipeBuilder;
  IncludeTools(enabled?: boolean): QueryRecipeBuilder;
  BySession(): QueryRecipeBuilder;
  ByTurn(): QueryRecipeBuilder;
  ByRole(): QueryRecipeBuilder;
  ByTool(): QueryRecipeBuilder;
  Build(): QueryRecipe;
};

type QueryRecipe = {
  name(): string;
  sql(): string;
  args(): any[];
  description(): string;
  output(): string;
  toJSON(): Record<string, any>;
};
```

Example:

```js
const recipe = mt.query()
  .TimelineRows()
  .SessionID("sess-123")
  .Build();

const rows = db.query(recipe.sql(), ...recipe.args());
```

### View plan builder

`mt.view()` and `session.view()` build executable view plans. A standalone view plan receives a DB handle; a session view plan is already bound to the session DB and session id.

```ts
type ViewPlanBuilder = {
  DB(db: DBHandle): ViewPlanBuilder;
  SessionID(id: string): ViewPlanBuilder;

  Transcript(): ViewPlanBuilder;
  TurnFrames(): ViewPlanBuilder;
  Timeline(): ViewPlanBuilder;
  TokenUsage(): ViewPlanBuilder;
  SessionSummary(): ViewPlanBuilder;

  IncludeTools(enabled?: boolean): ViewPlanBuilder;
  IncludeThinking(enabled?: boolean): ViewPlanBuilder;
  IncludeToolResults(enabled?: boolean): ViewPlanBuilder;
  CollapseLongTextAt(chars: number): ViewPlanBuilder;
  BySession(): ViewPlanBuilder;
  ByTurn(): ViewPlanBuilder;
  ByRole(): ViewPlanBuilder;
  ByTool(): ViewPlanBuilder;

  Plan(): QueryRecipe | ViewPlan;
  Run(): any;
};
```

Examples:

```js
const transcript = session.view()
  .Transcript()
  .IncludeTools()
  .Run();

const frames = mt.view()
  .DB(db)
  .SessionID(sessionId)
  .TurnFrames()
  .IncludeThinking()
  .IncludeToolResults()
  .CollapseLongTextAt(2400)
  .Run();
```

The view builder should still be thin: it should run query recipes and group rows. It should not own WidgetRenderer-specific labels or course-specific annotations.

### View output contracts remain plain data

Builder inputs should be Go-owned and fluent. Query/view outputs should remain plain JSON-serializable rows so templates, WidgetRenderer adapters, tests, and HTTP responses can use them easily.

#### TranscriptRow

```ts
type TranscriptRow = {
  id: string;
  session_id: string;
  turn_index: number;
  ordinal: number;
  role: "system" | "developer" | "user" | "assistant" | "tool" | "other";
  kind: "message" | "thinking" | "tool_call" | "tool_result" | "tool_error" | "annotation";
  name?: string;
  title?: string;
  text: string;
  timestamp?: string;
  tokens?: number;
  tool_call_id?: string;
  severity?: string;
  collapsed_by_default?: boolean;
  metadata?: Record<string, any>;
};
```

#### TurnFrame

```ts
type TurnFrame = {
  session_id: string;
  turn_index: number;
  turn: TurnRow | null;
  blocks: TurnBlockRow[];
  toolCalls: ToolRow[];
  stats: {
    chars: number;
    toolCalls: number;
    failedToolCalls: number;
    estimatedTokens: number;
  };
};
```

#### TokenUsageRow

```ts
type TokenUsageRow = {
  scope: "session" | "turn" | "role" | "tool" | "block";
  session_id: string;
  turn_index?: number;
  role?: string;
  tool_call_id?: string;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
  reasoning_tokens: number;
  tool_tokens: number;
  estimated_tokens?: number;
  total_tokens: number;
};
```

#### TimelineRow

```ts
type TimelineRow = {
  session_id: string;
  turn_index: number;
  role: string;
  timestamp?: string;
  preview: string;
  total_tokens: number;
  tool_call_count: number;
  failed_tool_count: number;
  file_count: number;
  has_thinking: boolean;
  has_annotations: boolean;
};
```

---

## SQL Contracts

### Session summary

```sql
SELECT
  s.session_id,
  s.title,
  s.summary,
  s.agent_framework,
  s.model,
  s.working_directory,
  s.started_at,
  s.ended_at,
  s.turn_count,
  s.tool_call_count,
  COALESCE(m.total_input_tokens, 0) AS total_input_tokens,
  COALESCE(m.total_output_tokens, 0) AS total_output_tokens,
  COALESCE(m.total_reasoning_tokens, 0) AS total_reasoning_tokens,
  COALESCE(m.total_tool_tokens, 0) AS total_tool_tokens
FROM sessions s
LEFT JOIN metrics m ON m.session_id = s.session_id
WHERE (? = '' OR s.session_id = ?)
ORDER BY s.started_at DESC, s.session_id
LIMIT 1;
```

### Transcript rows

```sql
WITH turn_messages AS (
  SELECT
    'turn-' || turn_index AS id,
    session_id,
    turn_index,
    0 AS ordinal,
    role,
    CASE WHEN thinking IS NOT NULL AND thinking != '' THEN 'message' ELSE 'message' END AS kind,
    model AS name,
    role || ' message' AS title,
    content AS text,
    timestamp,
    COALESCE(input_tokens, 0) + COALESCE(output_tokens, 0) + COALESCE(reasoning_tokens, 0) AS tokens,
    NULL AS tool_call_id,
    'info' AS severity,
    0 AS collapsed_by_default,
    raw_json AS metadata
  FROM turns
  WHERE (? = '' OR session_id = ?)
    AND COALESCE(content, '') != ''
), thinking_rows AS (
  SELECT
    'turn-' || turn_index || '-thinking' AS id,
    session_id,
    turn_index,
    1 AS ordinal,
    role,
    'thinking' AS kind,
    model AS name,
    'Thinking' AS title,
    thinking AS text,
    timestamp,
    COALESCE(reasoning_tokens, 0) AS tokens,
    NULL AS tool_call_id,
    'info' AS severity,
    1 AS collapsed_by_default,
    raw_json AS metadata
  FROM turns
  WHERE (? = '' OR session_id = ?)
    AND COALESCE(thinking, '') != ''
), tool_rows AS (
  SELECT
    'tool-' || tc.tool_call_id AS id,
    tc.session_id,
    COALESCE(ttc.turn_index, tc.emitting_turn_index, 0) AS turn_index,
    10 + COALESCE(ttc.ordinal, 0) AS ordinal,
    'tool' AS role,
    CASE WHEN COALESCE(tc.success, 1) = 0 THEN 'tool_error' ELSE 'tool_result' END AS kind,
    tc.tool_name AS name,
    tc.tool_name || ' / ' || COALESCE(tc.operation_type, 'operation') AS title,
    COALESCE(tc.error, tc.result, tc.command, tc.file_path, '') AS text,
    tc.timestamp,
    CAST(ROUND(COALESCE(tc.full_bytes, LENGTH(COALESCE(tc.result, tc.error, ''))) / 4.0) AS INTEGER) AS tokens,
    tc.tool_call_id,
    CASE WHEN COALESCE(tc.success, 1) = 0 THEN 'error' ELSE 'info' END AS severity,
    1 AS collapsed_by_default,
    tc.raw_json AS metadata
  FROM tool_calls tc
  LEFT JOIN turn_tool_calls ttc
    ON ttc.session_id = tc.session_id AND ttc.tool_call_id = tc.tool_call_id
  WHERE (? = '' OR tc.session_id = ?)
)
SELECT * FROM turn_messages
UNION ALL SELECT * FROM thinking_rows
UNION ALL SELECT * FROM tool_rows
ORDER BY turn_index, ordinal, id;
```

### Token usage by turn

```sql
SELECT
  'turn' AS scope,
  session_id,
  turn_index,
  role,
  NULL AS tool_call_id,
  COALESCE(input_tokens, 0) AS input_tokens,
  COALESCE(output_tokens, 0) AS output_tokens,
  COALESCE(cache_read_tokens, 0) AS cache_read_tokens,
  COALESCE(cache_creation_tokens, 0) AS cache_creation_tokens,
  COALESCE(reasoning_tokens, 0) AS reasoning_tokens,
  COALESCE(tool_tokens, 0) AS tool_tokens,
  NULL AS estimated_tokens,
  COALESCE(input_tokens, 0)
    + COALESCE(output_tokens, 0)
    + COALESCE(cache_read_tokens, 0)
    + COALESCE(cache_creation_tokens, 0)
    + COALESCE(reasoning_tokens, 0)
    + COALESCE(tool_tokens, 0) AS total_tokens
FROM turns
WHERE (? = '' OR session_id = ?)
ORDER BY turn_index;
```

### Timeline rows

```sql
SELECT
  t.session_id,
  t.turn_index,
  t.role,
  t.timestamp,
  SUBSTR(COALESCE(t.content, t.thinking, ''), 1, 600) AS preview,
  COALESCE(t.input_tokens, 0)
    + COALESCE(t.output_tokens, 0)
    + COALESCE(t.cache_read_tokens, 0)
    + COALESCE(t.cache_creation_tokens, 0)
    + COALESCE(t.reasoning_tokens, 0)
    + COALESCE(t.tool_tokens, 0) AS total_tokens,
  COUNT(tc.tool_call_id) AS tool_call_count,
  SUM(CASE WHEN COALESCE(tc.success, 1) = 0 THEN 1 ELSE 0 END) AS failed_tool_count,
  COUNT(DISTINCT f.path) AS file_count,
  CASE WHEN COALESCE(t.thinking, '') != '' THEN 1 ELSE 0 END AS has_thinking,
  CASE WHEN COUNT(a.annotation_id) > 0 THEN 1 ELSE 0 END AS has_annotations
FROM turns t
LEFT JOIN turn_tool_calls ttc ON ttc.session_id = t.session_id AND ttc.turn_index = t.turn_index
LEFT JOIN tool_calls tc ON tc.session_id = t.session_id AND tc.tool_call_id = ttc.tool_call_id
LEFT JOIN files f ON f.session_id = t.session_id AND f.tool_call_id = tc.tool_call_id
LEFT JOIN annotations a ON a.session_id = t.session_id AND a.target_id = CAST(t.turn_index AS TEXT)
WHERE (? = '' OR t.session_id = ?)
GROUP BY t.session_id, t.turn_index
ORDER BY t.turn_index;
```

---

## Pseudocode for the New Intern

### Implement `mt.importer()`

```go
type ImportBuilder struct {
    ctx context.Context
    content string
    path string
    name string
    sourcePath string
    rootDir string
    sessionID string
    strict bool
    autoDetect bool
    converted *minitracedb.LoadedSession
    diagnostics []minitracedb.ConversionDiagnostic
}

func importerObject(vm *goja.Runtime, b *ImportBuilder) *goja.Object {
    obj := vm.NewObject()
    _ = obj.Set("Content", func(content string) *goja.Object { b.content = content; return importerObject(vm, b) })
    _ = obj.Set("File", func(path string) *goja.Object { b.path = path; return importerObject(vm, b) })
    _ = obj.Set("Name", func(name string) *goja.Object { b.name = name; return importerObject(vm, b) })
    _ = obj.Set("Into", func(root string) *goja.Object { b.rootDir = root; return importerObject(vm, b) })
    _ = obj.Set("SessionID", func(id string) *goja.Object { b.sessionID = id; return importerObject(vm, b) })
    _ = obj.Set("AutoDetect", func() *goja.Object { b.autoDetect = true; return importerObject(vm, b) })
    _ = obj.Set("Strict", func(call goja.FunctionCall) goja.Value { b.strict = optionalBool(call, true); return importerObject(vm, b) })
    _ = obj.Set("Convert", func() (*goja.Object, error) {
        if err := b.Convert(); err != nil { return importerObject(vm, b), err }
        return importerObject(vm, b), nil
    })
    _ = obj.Set("Save", func() (map[string]any, error) {
        saved, err := b.Save()
        return toPlainMap(saved), err
    })
    return obj
}

func (b *ImportBuilder) Convert() error {
    if b.path != "" {
        loaded, err := minitracedb.LoadSessionFileAuto(b.path, minitracedb.LoadOptions{SourcePath: b.path, SourceName: firstNonEmpty(b.name, filepath.Base(b.path)), AutoConvert: true})
        b.converted = loaded
        return err
    }
    loaded, err := minitracedb.LoadSessionContentAuto([]byte(b.content), minitracedb.LoadOptions{SourceName: b.name, SourcePath: b.sourcePath, AutoConvert: true})
    b.converted = loaded
    return err
}

func (b *ImportBuilder) Save() (SavedSession, error) {
    if b.converted == nil {
        if err := b.Convert(); err != nil { return SavedSession{}, err }
    }
    // Write session.minitrace.json and metadata.json exactly once from the Go-owned converted session.
    return saveConvertedSession(b.converted, SaveOptions{RootDir: b.rootDir, SessionID: b.sessionID, OriginalName: b.name})
}
```

### Implement typed subbuilders

```go
type SourceSet struct { sources []dbSource }
type ImportPolicy struct { autoConvert bool; strict bool; forcedFormat string }
type CachePolicy struct { mode string; dir string; forceRebuild bool }
type QueryLimits struct { maxRows int; maxColumns int; maxCellChars int; timeout time.Duration; requireOrderBy bool }

func sourcesBuilderObject(vm *goja.Runtime, b *SourceSetBuilder) *goja.Object {
    obj := vm.NewObject()
    _ = obj.Set("File", func(path string) *goja.Object { b.AddFile(path); return sourcesBuilderObject(vm, b) })
    _ = obj.Set("Content", func(content string) *goja.Object { b.AddContent(content); return sourcesBuilderObject(vm, b) })
    _ = obj.Set("Name", func(name string) *goja.Object { b.NameMostRecent(name); return sourcesBuilderObject(vm, b) })
    _ = obj.Set("RuntimeArchives", func() *goja.Object { b.AddRuntimeArchives(); return sourcesBuilderObject(vm, b) })
    _ = obj.Set("Build", func() (*SourceSet, error) { return b.Build() })
    return obj
}
```

Implementation rule: the object returned by `Build()` is a Go-owned value. Higher-level builders accept that value directly instead of decoding a JavaScript object literal.

### Implement `mt.db()` as a composition builder

```go
func dbBuilderObject(vm *goja.Runtime, b *DBBuilder) *goja.Object {
    obj := vm.NewObject()

    _ = obj.Set("Sources", func(s *SourceSet) *goja.Object {
        b.SetSources(s)
        return dbBuilderObject(vm, b)
    })
    _ = obj.Set("Import", func(p *ImportPolicy) *goja.Object {
        b.SetImportPolicy(p)
        return dbBuilderObject(vm, b)
    })
    _ = obj.Set("Cache", func(p *CachePolicy) *goja.Object {
        b.SetCachePolicy(p)
        return dbBuilderObject(vm, b)
    })
    _ = obj.Set("Limits", func(l *QueryLimits) *goja.Object {
        b.SetQueryLimits(l)
        return dbBuilderObject(vm, b)
    })

    // Convenience methods configure internal subbuilders.
    _ = obj.Set("File", func(path string) *goja.Object { b.InternalSources().AddFile(path); return dbBuilderObject(vm, b) })
    _ = obj.Set("RuntimeArchives", func() *goja.Object { b.InternalSources().AddRuntimeArchives(); return dbBuilderObject(vm, b) })
    _ = obj.Set("AutoConvert", func(call goja.FunctionCall) goja.Value { b.InternalImportPolicy().AutoConvert(optionalBool(call, true)); return dbBuilderObject(vm, b) })
    _ = obj.Set("CacheAuto", func(call goja.FunctionCall) goja.Value { b.InternalCachePolicy().Auto(optionalString(call)); return dbBuilderObject(vm, b) })

    _ = obj.Set("Build", func() (*goja.Object, error) {
        h, err := b.Build()
        if err != nil { return nil, err }
        return handleObject(vm, h), nil
    })
    return obj
}
```

### Implement `mt.session()` as a high-level builder

```go
func sessionBuilderObject(vm *goja.Runtime, b *SessionBuilder) *goja.Object {
    obj := vm.NewObject()
    _ = obj.Set("Sources", func(s *SourceSet) *goja.Object { b.SetSources(s); return sessionBuilderObject(vm, b) })
    _ = obj.Set("Cache", func(p *CachePolicy) *goja.Object { b.SetCachePolicy(p); return sessionBuilderObject(vm, b) })
    _ = obj.Set("File", func(path string) *goja.Object { b.InternalSources().AddFile(path); return sessionBuilderObject(vm, b) })
    _ = obj.Set("SessionID", func(id string) *goja.Object { b.SessionID = id; return sessionBuilderObject(vm, b) })
    _ = obj.Set("InteractiveCache", func(dir string) *goja.Object { b.InternalCachePolicy().Auto(dir); return sessionBuilderObject(vm, b) })
    _ = obj.Set("Open", func() (*goja.Object, error) {
        handle, err := b.Open()
        if err != nil { return nil, err }
        return sessionHandleObject(vm, handle), nil
    })
    return obj
}
```

### Refactor minitrace-viz timeline

```js
const mt = require("mt");
const { CACHE_DIR } = require("./config");

function buildTimeline(sessionPath, sessionId) {
  const session = mt.session()
    .File(sessionPath)
    .SessionID(sessionId)
    .InteractiveCache(CACHE_DIR)
    .Open();
  try {
    const rows = session.view().Timeline().Run();
    const usage = session.view().TokenUsage().ByTurn().Run();
    return shapeTimelineFromRows(session.summary(), rows, usage, session.cacheInfo(), session.diagnostics());
  } finally {
    session.close();
  }
}
```



---

## Clean Cutover Plan

Because backwards compatibility is explicitly not required, the implementation should be simpler and more decisive than a migration with shims.

### Phase 1: Implement the new builder-composed `go-minitrace` `mt` API

Files:

- `go-minitrace/pkg/minitracejs/module.go`
- `go-minitrace/pkg/minitracejs/db_builder.go`
- new `go-minitrace/pkg/minitracejs/import_builder.go`
- new `go-minitrace/pkg/minitracejs/source_builder.go`
- new `go-minitrace/pkg/minitracejs/policy_builders.go`
- new `go-minitrace/pkg/minitracejs/session_builder.go`
- new `go-minitrace/pkg/minitracejs/query_builder.go`
- new `go-minitrace/pkg/minitracejs/view_builder.go`
- tests under `go-minitrace/pkg/minitracejs/provider/` and/or `go-minitrace/pkg/minitracejs/`

Tasks:

1. Add `mt.importer()` with `Content`, `File`, `Name`, `AutoDetect`, `Convert`, `Into`, `SessionID`, and `Save`.
2. Add `mt.sources()` returning a `SourceSetBuilder` and Go-owned `SourceSet`.
3. Add `mt.importPolicy()`, `mt.cache()`, and `mt.limits()` subbuilders.
4. Refactor `mt.db()` so it composes `SourceSet`, `ImportPolicy`, `CachePolicy`, and `QueryLimits` while preserving concise convenience methods.
5. Add `mt.session()` as the high-level transcript/session builder.
6. Add `mt.query()` as the SQL recipe builder.
7. Add `mt.view()` and `session.view()` as view-plan builders.
8. Remove old ClubMed API concepts (`source`, `archiveFile`, report builder) from the target API, but do not remove fluency itself.

Validation:

```bash
cd go-minitrace
go test ./pkg/minitracedb ./pkg/minitracejs/... ./cmd/go-minitrace/cmds/query -count=1
```

### Phase 2: Rewrite JS API documentation and examples

Files:

- `go-minitrace/pkg/doc/js-api-reference.md`
- `go-minitrace/pkg/doc/analysis-guide.md`
- `go-minitrace/testdata/query-repositories/js-showcase/**/*.js`
- `go-minitrace/examples/xgoja/minitrace-command-provider/**/*.js`

Tasks:

1. Rewrite the docs around builder-composed `const mt = require("mt")` examples.
2. Replace old monolithic chains with staged examples such as `mt.sources().RuntimeArchives().Build()` plus `mt.db().Sources(...).Build()`.
3. Add concise examples for `mt.db().RuntimeArchives().QueryCommandDefaults().Build()`.
4. Add upload/import examples using `mt.importer().Content(...).Name(...).Into(...).Save()`.
5. Add app-style examples using `mt.session().File(...).InteractiveCache(...).Open()` and `session.view().Transcript().Run()`.
6. Keep SQL recipes visible and copy/pasteable through `mt.query().<Recipe>().Build()`.

Validation:

```bash
cd go-minitrace
go test ./cmd/go-minitrace/cmds/query -count=1
rg "mt\.db\.open|mt\.session\.open|OpenDBOptions|SessionOpenOptions" pkg/doc testdata examples
```

The final `rg` should return no references except historical notes in this ticket.

### Phase 3: Refactor minitrace-viz xgoja wiring

Files:

- `ClubMedMeetup/minitrace-viz/xgoja.yaml`

Tasks:

1. Remove package `clubmed-minitrace-viz`.
2. Remove module registration for the local package.
3. Register only `go-minitrace` for minitrace data access.
4. Alias it as `mt`.
5. Remove `as: minitrace` to avoid two names.

Validation:

```bash
rg "require\(\"minitrace\"\)|require\('minitrace'\)" ClubMedMeetup/minitrace-viz
rg "clubmed-minitrace-viz|mtapiprovider" ClubMedMeetup/minitrace-viz/xgoja.yaml
```

Both commands should return no app references after cutover.

### Phase 4: Refactor minitrace-viz upload and data access

Files:

- `ClubMedMeetup/minitrace-viz/lib/session-service.js`
- `ClubMedMeetup/minitrace-viz/lib/timeline-data.js`
- `ClubMedMeetup/minitrace-viz/lib/course-session-data.js`
- `ClubMedMeetup/minitrace-viz/server.js`
- optionally `ClubMedMeetup/minitrace-viz/lib/sql-reports.js`

Tasks:

1. Replace `mt.source(...).detect().convert().save(...)` with `mt.importer().Content(...).Name(...).Into(...).SessionID(...).Save()`.
2. Replace `mt.archiveFile(...).turnBlocks(...)` with `mt.session(...).views.turnFrames()` or `views.timeline()`.
3. Update transcript model construction to start from `views.transcript()` where useful.
4. Keep WidgetRenderer-specific annotation and context-window shaping in `course-session-data.js`.
5. Remove `/analyze`, `/api/report/:sessionId`, and `/api/report-presets`, unless explicitly preserved as debug endpoints over SQL recipes.
6. Update route smoke tests or manual curl instructions.

Validation:

```bash
rg "mt\.source|mt\.archiveFile|reportPresets|\.report\(" ClubMedMeetup/minitrace-viz
```

This should return no application references.

### Phase 5: Delete old ClubMed `mtapi`

Files to remove:

- `ClubMedMeetup/minitrace-viz/pkg/mtapi/*.go`
- `ClubMedMeetup/minitrace-viz/pkg/mtapiprovider/provider.go`

Additional cleanup:

- remove now-unused Go dependencies in `ClubMedMeetup/minitrace-viz/go.mod`, if any;
- regenerate xgoja output if generated files are tracked;
- remove stale docs or screenshots referencing old report endpoints.

Validation:

```bash
rg "pkg/mtapi|pkg/mtapiprovider|clubmed-minitrace-viz|mt\.source|mt\.archiveFile" .
```

Only this ticket documentation should still mention the old names.

### Phase 6: End-to-end validation

Run unit tests, xgoja build, and app smoke tests.

Suggested smoke flow:

```bash
# go-minitrace tests
cd go-minitrace
go test ./pkg/minitracedb ./pkg/minitracejs/... ./cmd/go-minitrace/cmds/query -count=1

# minitrace-viz build and launch
cd ../ClubMedMeetup/minitrace-viz
# use the repo's actual build command for xgoja/dist regeneration
./dist/minitrace-viz verbs site start \
  --sessions-dir /tmp/minitrace-viz-sessions \
  --cache-dir /tmp/minitrace-viz-cache

# API smoke checks
curl http://127.0.0.1:8787/api/widget/health
curl http://127.0.0.1:8787/api/sessions
curl http://127.0.0.1:8787/api/timeline/<session-id>
curl http://127.0.0.1:8787/api/sessions/<session-id>/transcript-data
curl http://127.0.0.1:8787/api/sessions/<session-id>/context-window-data
```

---

## Design Decisions

### Decision: Use a clean API cutover instead of compatibility shims

- **Context:** The user explicitly requested no backwards compatibility and asked to optimize for clarity, elegance, and expressiveness.
- **Options considered:** Preserve old APIs with wrappers; expose both old and new APIs temporarily; remove old APIs and update call sites directly.
- **Decision:** Remove old public APIs during this ticket and update all call sites.
- **Rationale:** Compatibility wrappers would keep `source`, `archiveFile`, and report-builder concepts alive and obscure the new model.
- **Consequences:** The implementation must update all tests, examples, docs, and app code in one coordinated cutover. This is more disruptive but easier to reason about afterward.
- **Status:** proposed.

### Decision: Make `mt.session()` the default out-of-the-box transcript builder

- **Context:** Interns and app code need a simple way to work with transcript data without learning every SQLite/cache/import option first.
- **Options considered:** Keep only a DB builder; use options maps; add a session-centered fluent builder.
- **Decision:** Add `mt.session()` as the opinionated high-level builder while retaining `mt.db()` for SQL power users.
- **Rationale:** Transcript work naturally centers on one session and its views. A fluent Go-backed session builder keeps construction typed while still giving app code a concise path.
- **Consequences:** Session handles need clear lifecycle rules (`close()` in `finally`) and stable view contracts.
- **Status:** proposed.

### Decision: Split the monolithic builder into staged fluent subbuilders

- **Context:** The current builder has many methods and mixes sources, cache, conversion, storage, limits, validation, and lifecycle. The user clarified that fluent builders are preferred because they preserve Go-side construction semantics.
- **Options considered:** Keep the builder unchanged; replace public usage with options maps; split the API into subbuilders that produce Go-owned option objects.
- **Decision:** Keep fluent APIs and introduce `mt.sources()`, `mt.importPolicy()`, `mt.cache()`, `mt.limits()`, `mt.db()`, `mt.session()`, `mt.query()`, and `mt.view()` builders.
- **Rationale:** Subbuilders preserve the benefits of Go-backed construction while making the API clearer than one giant chain.
- **Consequences:** Existing query-command tests and examples must be updated, but they should move to clearer staged fluent examples rather than map literals.
- **Status:** proposed.

### Decision: SQL recipes are public API

- **Context:** The normalized SQLite schema is the most stable reusable substrate. Users need both convenience and power.
- **Options considered:** Hide SQL inside view helpers; expose only raw SQL docs; expose named recipes as data.
- **Decision:** Expose `mt.query().<Recipe>().Build()` as the named, documented SQL recipe-builder contract.
- **Rationale:** Recipes are inspectable, testable, copy/pasteable, and easy to override.
- **Consequences:** Query builder outputs must be versioned through tests and docs.
- **Status:** proposed.

### Decision: Keep WidgetRenderer and teaching heuristics in minitrace-viz

- **Context:** The context-window view in `course-session-data.js` includes teaching notes, estimated system/tool policy tokens, model-limit guesses, and WidgetRenderer part kinds.
- **Options considered:** Move all context-window logic into `go-minitrace`; move only raw context rows; keep everything app-side.
- **Decision:** `go-minitrace` provides generic transcript, turn-frame, timeline, and token rows. The app maps those rows into WidgetRenderer models.
- **Rationale:** This keeps the core library reusable beyond this course app.
- **Consequences:** App code still has meaningful logic, but it is presentation-specific instead of data-ingestion-specific.
- **Status:** proposed.

### Decision: Do not recreate the report builder

- **Context:** The old report API is broad and not aligned with the desired transcript/context/token/timeline product surface.
- **Options considered:** Port `ReportBuilder` to `go-minitrace`; keep it as a ClubMed package; remove it and build debug reports from SQL recipes if needed.
- **Decision:** Remove report-builder APIs from the core cutover.
- **Rationale:** Reports mix analysis lenses, prose rendering, and app routes. SQL recipes plus examples are more composable.
- **Consequences:** `/analyze` and `/api/report*` routes should be removed or rewritten as debug-only SQL endpoints.
- **Status:** proposed.

---

## Testing Strategy

### Unit tests for import

- `mt.importer().Content(...).Name(...).AutoDetect().Detect()` identifies native minitrace JSON, Pi JSONL, Codex JSONL, and Claude Code JSONL.
- `mt.importer().Content(...).Convert().Converted()` returns a minitrace session and diagnostics.
- `mt.importer().Save()` writes `session.minitrace.json` and `metadata.json` with stable fields.
- Empty content, unsupported JSON, and malformed JSONL return clear errors.

### Unit tests for builder-composed `mt.db()`

- Runtime archives can be opened with `mt.db().RuntimeArchives().QueryCommandDefaults().Build()`.
- `SourceSet`, `ImportPolicy`, `CachePolicy`, and `QueryLimits` subbuilders are accepted by `mt.db().Sources(...).Import(...).Cache(...).Limits(...)`.
- Files, content, globs, and dirs map to the same materialization behavior as the old builder.
- Cache policies produce the expected cache key, hit/miss behavior, and close semantics.
- Query limits still bound rows, columns, cell characters, and timeouts.

### Unit tests for query recipes

For every recipe:

1. Build a SQLite DB from a sample session.
2. Execute the recipe.
3. Assert required columns are present.
4. Assert deterministic ordering.
5. Assert empty sessions return an empty array or clear null result, not a panic.

### Integration tests for `mt.session()`

```js
const mt = require("mt");
const session = mt.session()
  .Content(jsonlContent)
  .Name("pi.jsonl")
  .Open();
try {
  if (!session.summary().session_id) throw new Error("missing session id");
  if (!session.view().Transcript().Run().length) throw new Error("empty transcript");
  if (!session.view().Timeline().Run().length) throw new Error("empty timeline");
  if (!session.view().TokenUsage().ByTurn().Run().length) throw new Error("empty token usage");
} finally {
  session.close();
}
```

### Web app regression checks

Compare old and new app behavior at semantic level:

- uploads still produce sessions and links;
- transcript page shows user/assistant/tool rows in turn order;
- context-window page can select a turn and show system/message/thinking/tool/result/free-space parts;
- timeline page shows turn cards and minimap;
- failed tools remain visible;
- token totals remain plausible, even if not byte-for-byte identical to old estimates.

---

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Query-command examples break during builder restructuring. | Update docs, tests, and showcase repositories in the same phase; prefer staged fluent examples over options maps. |
| `session_id` is unknown when querying a one-session DB. | Let `mt.session().Open()` derive it from `sessions` when not provided. Query builders should accept omitted `SessionID` as "all/current" where safe. |
| Event ordering does not preserve exact text/tool interleaving for some adapters. | Validate `events` and `turn_tool_calls` ordering against sample sessions; add missing ordinals where needed before relying on views. |
| Tool-call rows currently represent call and result as one event. | Transcript recipes can produce one combined tool result row initially; split call/result rows later only if source data supports it. |
| Removing report routes surprises users. | If needed, add `/api/debug/query/:name` over `mt.query()` recipes, but do not preserve old report-builder concepts. |
| Context-window token estimates are heuristic. | Mark estimates clearly in app metadata; keep exact token fields when available and `/4 bytes` estimates only for tool output. |

---

## File Reference Map

| File | Role in this ticket |
|---|---|
| `go-minitrace/pkg/minitracejs/module.go` | Add top-level builder factories: `importer`, `sources`, `importPolicy`, `cache`, `limits`, `db`, `session`, `query`, and `view`. |
| `go-minitrace/pkg/minitracejs/db_builder.go` | Refactor the broad fluent DB builder into a composition builder that accepts typed subbuilder outputs. |
| `go-minitrace/pkg/minitracedb/convert.go` | Reuse conversion/detection for `mt.importer()` and auto-converting DB/session builders. |
| `go-minitrace/pkg/minitracedb/materialize.go` | Existing normalized SQLite materialization path. Validate event ordering and fields. |
| `go-minitrace/pkg/minitracedb/schema.go` | Source of truth for SQL contracts. |
| `go-minitrace/pkg/doc/js-api-reference.md` | Rewrite around new `mt` API. |
| `ClubMedMeetup/minitrace-viz/xgoja.yaml` | Remove local provider and alias go-minitrace as `mt`. |
| `ClubMedMeetup/minitrace-viz/lib/session-service.js` | Replace old upload builder with `mt.importer().Content(...).Name(...).Into(...).Save()`. |
| `ClubMedMeetup/minitrace-viz/lib/timeline-data.js` | Replace old archive turn blocks with `mt.session()` and `session.view()` rows/frames. |
| `ClubMedMeetup/minitrace-viz/lib/course-session-data.js` | Keep WidgetRenderer shaping; feed it new rows/frames. |
| `ClubMedMeetup/minitrace-viz/server.js` | Remove old report endpoints and old turn-block route implementation. |
| `ClubMedMeetup/minitrace-viz/pkg/mtapi/*` | Delete after parity. |
| `ClubMedMeetup/minitrace-viz/pkg/mtapiprovider/provider.go` | Delete after xgoja wiring cutover. |

---

## Intern Reading Order

1. Read this document fully.
2. Read `go-minitrace/pkg/doc/js-api-reference.md` to understand the current JS API being replaced.
3. Read `go-minitrace/pkg/minitracedb/schema.go` and focus on `sessions`, `turns`, `tool_calls`, `turn_tool_calls`, `events`, and `metrics`.
4. Read `go-minitrace/pkg/minitracedb/convert.go` to understand auto conversion.
5. Read `go-minitrace/pkg/minitracejs/module.go` and `db_builder.go` to understand current JS export mechanics.
6. Read `ClubMedMeetup/minitrace-viz/pkg/mtapiprovider/provider.go` to see the old API to delete.
7. Read `ClubMedMeetup/minitrace-viz/lib/session-service.js`, `timeline-data.js`, and `course-session-data.js` to see the app call sites.
8. Implement Phase 1 tests before refactoring minitrace-viz.

---

## Final Recommendation

Implement a clean, builder-first `mt` API in `go-minitrace`, make `mt.session()` the default transcript workflow, keep `mt.db()` plus `mt.query()` for SQL power users, and split construction into fluent Go-owned subbuilders rather than plain options maps. Refactor minitrace-viz directly to the new shape, delete the ClubMed `mtapi` package, and do not preserve the old `source`/`archiveFile`/report-builder concepts. This gives the project one coherent API that is easier for interns, still expressive for advanced users, and centered on the transcript views the product actually needs.
