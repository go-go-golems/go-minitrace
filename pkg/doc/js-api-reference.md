---
Title: JS API Reference
Slug: js-api-reference
Short: JavaScript runtime API available to minitrace JS command handlers
Topics:
- minitrace
- javascript
- js-commands
- js-api
- sqlite
- goja
Commands:
- query commands
Flags:
- archive-glob
- db-path
- table-name
- persist-loaded
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

JavaScript command handlers run in a Goja-powered runtime with access to a `minitrace` module, Goja NodeJS-style helper modules, and scanner markers that turn `.js` files into typed `go-minitrace query commands` verbs.

The current API is **builder-composed**. Use fluent Go-backed builders to construct typed sources, policies, cache settings, query recipes, views, and session handles. The builders keep construction and validation on the Go side while view/query outputs remain plain JSON-serializable rows.

```js
const mt = require("minitrace");
```

> In xgoja apps this provider can be aliased as `require("mt")`. Query-command examples in this repository still use `require("minitrace")` because that is the provider's registered module name.

## Quick patterns

### Query-command default

```js
function sessionList(filters) {
  const mt = require("minitrace");
  const db = mt.db()
    .RuntimeArchives()
    .QueryCommandDefaults()
    .Build();
  try {
    return db.query(`
      SELECT session_id, title, agent_framework, turn_count, tool_call_count
      FROM sessions
      WHERE 1=1
      ${filters.framework?.length ? `AND agent_framework IN (${mt.sql.stringIn(filters.framework)})` : ""}
      ORDER BY started_at DESC
      LIMIT ${filters.limit ?? 25}
    `);
  } finally {
    db.close();
  }
}
```

### Explicit staged construction

```js
const sources = mt.sources()
  .RuntimeArchives()
  .Build();

const cache = mt.cache()
  .Memory()
  .Build();

const limits = mt.limits()
  .Rows(500)
  .CellChars(8000)
  .Build();

const db = mt.db()
  .Sources(sources)
  .Cache(cache)
  .Limits(limits)
  .Build();
```

### Session-oriented app flow

```js
const session = mt.session()
  .File(sessionPath)
  .InteractiveCache(cacheDir)
  .Open();
try {
  const transcript = session.view().Transcript().IncludeTools().Run();
  const timeline = session.view().Timeline().Run();
  const usage = session.view().TokenUsage().ByTurn().Run();
  return { summary: session.summary(), transcript, timeline, usage };
} finally {
  session.close();
}
```

### Upload/import flow

```js
const importer = mt.importer()
  .Content(uploadText)
  .Name(filename)
  .AutoDetect()
  .Strict()
  .Convert();

const preview = importer.Preview();
// Inspect preview.hasSystemPrompt, preview.sampleTurns, preview.sampleTools,
// preview.sampleEvents, preview.sampleAttachments, preview.eventCounts,
// preview.attachmentCounts, preview.subagentCount, and preview.hasImageSignals before saving.

const saved = importer
  .Into(sessionsDir)
  .SessionID(sessionId)
  .Save();
```

## Builder factories

| Factory | Purpose | Final value |
|---|---|---|
| `mt.importer()` | Detect, convert, and save uploaded content/files. | `SavedSession` or `ConvertedSession` maps. |
| `mt.sources()` | Build file/content/dir/glob/runtime source sets. | Go-owned `SourceSet`. |
| `mt.importPolicy()` | Build conversion strictness/auto-convert policy. | Go-owned `ImportPolicy`. |
| `mt.cache()` | Build cache policy. | Go-owned `CachePolicy`. |
| `mt.limits()` | Build query limits. | Go-owned `QueryLimits`. |
| `mt.db()` | Compose sources/policies/limits into a normalized SQLite DB handle. | `DBHandle`. |
| `mt.session()` | Open an opinionated transcript/session handle. | `SessionHandle`. |
| `mt.query()` | Build named SQL recipes. | `QueryRecipe`. |
| `mt.view()` | Build and run generic transcript/timeline/token view plans. | Plain rows/frames. |

## `mt.importer()`

Methods:

| Method | Description |
|---|---|
| `.Content(text)` | Use in-memory upload content. |
| `.File(path)` | Use a source file. |
| `.Name(name)` | Set original/source name. |
| `.SourcePath(path)` | Set source path metadata. |
| `.AutoDetect()` | Enable supported raw transcript conversion. |
| `.Format(format)` | Reserve a forced format hint for future use. |
| `.Strict(true|false)` | Set strict conversion behavior. |
| `.Into(rootDir)` | Set sessions root for `Save()`. |
| `.SessionID(id)` | Override saved session id. |
| `.Overwrite(true|false)` | Allow replacing an existing session directory. |
| `.Detect()` | Load/detect and return `{ format, adapter, diagnostics }`. |
| `.Convert()` | Convert and keep the Go-owned converted session in the builder. |
| `.Converted()` | Return a plain count-level summary of the converted session. |
| `.Preview()` | Return a transcript preview for validation: roles, tools, source events, attachments, sample turns/tool calls/events/attachments, system-prompt/thinking/image/subagent signals, and diagnostics. |
| `.Diagnostics()` | Return conversion diagnostics. |
| `.Save()` | Write `session.minitrace.json` and `metadata.json`. |

### Preview from the CLI

Use the CLI when you want the same importer preview behavior without writing JavaScript:

```bash
go-minitrace preview session --source-session /path/to/session.jsonl --output yaml
```

Directory/latest-N mode uses framework discovery and emits one row per previewed session:

```bash
go-minitrace preview session --framework codex --latest 5 --privacy structural --output json
go-minitrace preview session --framework claude-code --source-dir ~/.claude/projects --latest 3 --sample-limit 2 --output yaml
go-minitrace preview session --framework pi --source-dir ~/.pi/agent/sessions --latest 1
```

Preview privacy controls:

| Privacy | Behavior |
|---|---|
| `structural` | Counts, roles, tools, event/attachment kinds, booleans, and diagnostics only; content snippets/commands/paths are suppressed in samples. |
| `snippets` | Default; includes bounded snippets and commands for quick validation. |
| `full` | Includes full sampled turn text and command strings; use only on trusted local output paths. |

Important preview fields:

| Field | Meaning |
|---|---|
| `turnCount`, `toolCallCount` | Normalized conversation/action counts. |
| `eventCount`, `attachmentCount` | First-class source event/artifact counts. |
| `roleCounts`, `toolCounts` | Breakdown maps for turns and tools. |
| `eventCounts`, `attachmentCounts` | Breakdown maps by event kind and attachment kind. |
| `sampleTurns`, `sampleTools` | Bounded samples for validating role/tool conversion. |
| `sampleEvents`, `sampleAttachments` | Bounded samples for lifecycle facts and artifacts; raw JSON is not included. |
| `hasImageSignals` | True if turns, tools, events, or attachments include image-like signals. |

## `mt.sources()`

Methods: `.File(path)`, `.Archive(path)`, `.Files(paths)`, `.Dir(path)`, `.Glob(pattern)`, `.Content(text)`, `.Name(name)`, `.RuntimeArchives()`, `.Validate()`, `.Build()`.

`Build()` returns a Go-owned `SourceSet`. Pass it to `mt.db().Sources(...)` or `mt.session().Sources(...)`.

```js
const sources = mt.sources()
  .Dir("./output/active")
  .Glob("./extra/*.minitrace.json")
  .Build();
```

## Policy subbuilders

```js
const importPolicy = mt.importPolicy()
  .AutoConvert()
  .Strict()
  .Build();

const cache = mt.cache()
  .Auto()
  .Dir("./.cache/minitrace")
  .Build();

const limits = mt.limits()
  .Rows(1000)
  .Columns(128)
  .CellChars(6000)
  .TimeoutMs(5000)
  .Build();
```

### `mt.importPolicy()`

- `.AutoConvert(true|false)`
- `.NativeOnly()`
- `.Strict(true|false)`
- `.Lenient()`
- `.Format(format)`
- `.Build()`

### `mt.cache()`

- `.None()`
- `.Memory()`
- `.Disk()`
- `.Auto()`
- `.Dir(path)`
- `.ForceRebuild(true|false)`
- `.Build()`

### `mt.limits()`

- `.Rows(n)`
- `.Columns(n)`
- `.CellChars(n)`
- `.TimeoutMs(n)`
- `.RequireOrderBy(true|false)`
- `.Build()`

## `mt.db()`

`mt.db()` composes source sets and policies into a normalized SQLite DB handle.

Composition methods:

- `.Sources(sourceSet)`
- `.Import(importPolicy)`
- `.Cache(cachePolicy)` or legacy `.Cache("auto")`
- `.Limits(queryLimits)`
- `.Build()`

Convenience methods:

- source: `.File`, `.Archive`, `.Files`, `.Dir`, `.Glob`, `.Content`, `.RuntimeArchives`
- conversion: `.AutoConvert`, `.StrictConversion`, `.Strict`
- cache/storage: `.SQLiteMemory`, `.SQLiteDiskCache`, `.CacheAuto`, `.CacheDir`, `.ForceRebuild`
- limits: `.MaxRows`, `.MaxColumns`, `.MaxCellChars`, `.Timeout`, `.RequireOrderBy`
- presets: `.QueryCommandDefaults()`, `.InteractiveDefaults(cacheDir?)`
- introspection: `.Validate()`, `.CacheKey()`, `.sources()`

### DB handle

`Build()` returns a handle with:

| Method | Description |
|---|---|
| `query(sql, ...args)` | Execute a read-only SQLite query. |
| `queryOne(sql, ...args)` | Return first row. |
| `queryResult(sql, ...args)` | Return result envelope including validation errors. |
| `schema()` / `tables()` / `stats()` | Schema introspection. |
| `sources()` | Sources used to build the handle. |
| `diagnostics()` | Conversion diagnostics. |
| `cacheInfo()` / `CacheInfo()` | Cache metadata. |
| `close()` | Close/release handle. |

Queries are validated as read-only SQL and bounded by row, column, cell, and timeout limits.

### Source events and attachments from JavaScript

Normalized SQLite now includes first-class source lifecycle facts and artifact references:

| Table | What it contains | How to use it from Goja |
|---|---|---|
| `events` | Derived renderable rows for turns/tools/annotations plus explicit source events. | Filter `kind NOT IN ('turn', 'tool_call', 'annotation')` when you want only source lifecycle facts. |
| `attachments` | Durable references to images/files/artifacts associated with sessions, turns, tools, or events. | Join/link by `tool_call_id`, `event_id`, `attachment_id`, or `turn_index`; do not expect blob bytes. |

Important source-event kinds include Pi `compaction`, `model_change`, `thinking_level_change`, Claude Code `mode_change`, `permission_mode_change`, `title_change`, `attachment`, and Codex `image_view`, `subagent_spawn`, `subagent_wait`, and `rate_limits`.

Concrete examples:

```js
// Codex viewed an image. Render this as an image marker or attachment card,
// not as a fake assistant text message.
{
  event_id: "event-image-1",
  kind: "image_view",
  title: "Codex image view",
  summary: "/tmp/screenshots/failure.png",
  tool_call_id: "call-view-image-1",
  attachment_id: "attachment-image-1"
}
{
  attachment_id: "attachment-image-1",
  kind: "image",
  name: "failure.png",
  media_type: "image/png",
  path: "/tmp/screenshots/failure.png",
  tool_call_id: "call-view-image-1",
  event_id: "event-image-1"
}

// Claude Code changed permission mode. Render this as a session/timeline badge
// so later edits can be interpreted with the right autonomy context.
{
  event_id: "event-permission-mode-1",
  kind: "permission_mode_change",
  title: "Permission mode changed",
  summary: "acceptEdits",
  severity: "info"
}

// Pi compacted the conversation. Render this as a context-boundary marker
// instead of scanning turn text for the word "compaction".
{
  event_id: "event-compaction-1",
  kind: "compaction",
  title: "Compaction",
  summary: "Conversation was compacted; prior context summarized.",
  collapsed_by_default: true
}

// Codex spawned a subagent and later waited for it. Render these as
// orchestration facts linked to their tool calls.
{
  event_id: "event-subagent-spawn-1",
  kind: "subagent_spawn",
  title: "Subagent spawned",
  summary: "agent_type=explorer",
  tool_call_id: "call-spawn-agent-1"
}
{
  event_id: "event-subagent-wait-1",
  kind: "subagent_wait",
  title: "Subagent completed",
  summary: "status=completed",
  tool_call_id: "call-wait-agent-1"
}
```

Query them from JavaScript like this:

```js
const session = mt.session()
  .File("/path/to/session.minitrace.json")
  .InteractiveCache(".cache/minitrace")
  .Open();
try {
  const sourceEvents = session.query(`
    SELECT event_id, timestamp, turn_index, kind, title, summary, tool_call_id, attachment_id
    FROM events
    WHERE kind NOT IN ('turn', 'tool_call', 'annotation')
    ORDER BY COALESCE(turn_index, 999999), COALESCE(ordinal, 999999), event_id
  `);
  const attachments = session.query(`
    SELECT attachment_id, timestamp, kind, name, media_type, path, url, tool_call_id, event_id, text_preview
    FROM attachments
    ORDER BY COALESCE(turn_index, 999999), timestamp, attachment_id
  `);
  return { sourceEvents, attachments };
} finally {
  session.close();
}
```

For UI code such as `ClubMedMeetup/minitrace-viz/lib/course-session-data.js`, the recommended pattern is to keep transcript messages derived from `turns` and `tool_calls`, then project source events/attachments into separate annotations, badges, or side panels. This avoids polluting the conversation text while still making compactions, subagents, mode changes, rate limits, and images visible.

## `mt.query()`

`mt.query()` builds SQL recipes. A recipe is a Go-owned object with `name()`, `sql()`, `args()`, `description()`, `output()`, and `toJSON()`.

Recipe selectors:

- `.SessionSummary()`
- `.TurnRows()`
- `.ToolRows()`
- `.EventRows()`
- `.TurnBlockRows()`
- `.TokenUsageRows()`
- `.TranscriptRows()`
- `.TimelineRows()`

Modifiers:

- `.SessionID(id)`
- `.IncludeTools(true|false)`
- `.BySession()`
- `.ByTurn()`
- `.ByRole()`
- `.ByTool()`

Example:

```js
const recipe = mt.query()
  .TranscriptRows()
  .SessionID("sess-123")
  .IncludeTools()
  .Build();
const rows = db.query(recipe.sql(), ...recipe.args());
```

## `mt.view()` and `session.view()`

View builders run query recipes and return plain rows/frames.

Standalone view plans use `.DB(db)`:

```js
const frames = mt.view()
  .DB(db)
  .SessionID(sessionId)
  .TurnFrames()
  .IncludeThinking()
  .IncludeToolResults()
  .Run();
```

Session-bound view plans already know the DB and session id:

```js
const transcript = session.view()
  .Transcript()
  .IncludeTools()
  .Run();
```

Selectors:

- `.Transcript()`
- `.TurnFrames()`
- `.Timeline()`
- `.TokenUsage()`
- `.SessionSummary()`

Modifiers:

- `.IncludeTools(true|false)`
- `.IncludeThinking(true|false)`
- `.IncludeToolResults(true|false)`
- `.CollapseLongTextAt(chars)`
- `.BySession()` / `.ByTurn()` / `.ByRole()` / `.ByTool()`
- `.Plan()` to inspect the underlying query recipe
- `.Run()` to execute

## `mt.session()`

`mt.session()` is the high-level transcript workflow. It applies transcript-friendly defaults and returns a lifecycle-bound session handle.

Builder methods:

- `.Sources(sourceSet)` / `.Source(sourceSet)`
- `.Import(importPolicy)`
- `.Cache(cachePolicy)`
- `.Limits(queryLimits)`
- `.SessionID(id)`
- `.File(path)`
- `.Content(text)`
- `.Name(name)`
- `.InteractiveCache(dir?)`
- `.Strict(true|false)`
- `.Open()`

Session handle methods:

- `id()`
- `summary()`
- `diagnostics()`
- `cacheInfo()`
- `db()`
- `query(sql, ...args)`
- `view()`
- `close()`

Always call `close()` in a `finally` block.

## SQL helpers

`mt.sql` contains string-literal helpers for building safe SQL fragments when query parameters are not ergonomic in generated SQL.

```js
mt.sql.string("O'Reilly")       // => "'O''Reilly'"
mt.sql.stringIn(["codex", "pi"]) // => "'codex', 'pi'"
mt.sql.like("tool fail")        // => "'%tool fail%'"
```

## Runtime metadata

`mt.runtime` includes query-command runtime settings:

| Property | Type | Description |
|---|---|---|
| `archiveGlob` | `string[]` | Runtime archive globs consumed by `.RuntimeArchives()`. |
| `dbPath` | `string` | Legacy/runtime DB path setting. |
| `tableName` | `string` | Legacy DuckDB table name; do not use for new builder workflows. |
| `persistLoaded` | `boolean` | Runtime persistence flag. |
| `commandName` | `string` | Current command name. |

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `mt.query is not a function` | Script follows removed ambient DuckDB API. | Use `const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build()` and query normalized tables. |
| `runtime archive glob is not configured` | `.RuntimeArchives()` used without runtime globs. | Pass `--archive-glob ...`, or use `.File()`, `.Dir()`, `.Glob()`, or `.Content()`. |
| Raw JSONL fails to load | Auto conversion disabled or strict conversion failed. | Use `.AutoConvert()` / `mt.importPolicy().AutoConvert().Build()` and inspect diagnostics. |
| View builder says DB is required | Standalone `mt.view()` was not bound to a DB. | Use `mt.view().DB(db)...` or `session.view()...`. |

## Related help

- `go-minitrace help structured-query-commands` — scanner markers and command metadata.
- `go-minitrace help query` — query subsystem overview.
- `go-minitrace help query-duckdb` — older raw DuckDB workflow for ad hoc exploration.
- `testdata/query-repositories/js-showcase/` — worked JavaScript command examples.
