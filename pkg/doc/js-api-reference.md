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

# JavaScript API reference

Load the local module with `require("minitrace")`. A source builder describes input, a database builder opens it, and a query builder describes the work. Call `db.execute(plan)` to run the query. Close the database in `finally`.

```js
const mt = require("minitrace");
const sources = mt.sources().runtimeArchives().build();
const db = mt.db().sources(sources)
  .import({autoConvert: true, strict: true})
  .limits({maxRows: 2000, maxCellBytes: 8192, strict: true})
  .build();
try {
  const sessions = mt.from(db.table("sessions"));
  const tools = mt.from(db.table("tool_calls"));
  const counts = sessions.select(s => ({
    session_id: s.session_id,
    tool_count: tools.where(t => t.session_id.eq(s.session_id)).count(),
  })).orderBy(s => s.session_id.asc());
  const result = db.execute(counts.build());
  if (result.error || !result.complete) throw Error(result.error || "Incomplete result");
  return result.rows;
} finally {
  db.close();
}
```

This count starts from sessions and includes sessions with zero tools. It measures actual tool rows rather than stored summary counts.

## Sources and database policies

Source builders are immutable. Keep the returned value when composing.

| Method | Meaning |
| --- | --- |
| `files(paths)` | Explicit paths to normalized or raw transcript files |
| `directory(path)` | Recursively discover normalized archive files |
| `glob(pattern)` | Match paths using the host filesystem glob |
| `content(text, {name})` | Supply an in-memory transcript with an optional diagnostic name |
| `runtimeArchives()` | Use the command host's `--archive-glob` selection |
| `build()` | Capture an opaque selection for `db().sources(selection)` |

`db().import({autoConvert, strict})` configures input conversion. Raw conversion is opt-in. Strict input handling is on by default. `db().cache({mode, dir, maxBytes, forceRebuild})` uses optional `none`, `memory`, `disk`, or `auto` caching. Use `none` for an uncached in-memory database. `limits({maxRows, maxColumns, maxCellBytes, timeoutMs, strict, requireOrderBy})` configures execution. The cell limit counts UTF-8 bytes. SQL `LIMIT` expresses selection, while `maxRows` detects delivery overflow.

`cacheKey()` fingerprints the builder inputs. A handle provides `cacheInfo()`, `sources()`, `diagnostics()`, `schema()`, `tables()`, and `stats()`. A query built from a catalog table belongs to the handle that supplied the table. SQL-only plans run against the handle passed to execution.

## Fluent expressions and tagged SQL

`mt.from(db.table(name))` begins a fluent query. `where`, `select` and `orderBy` receive symbolic rows when the method is called. Use `.eq(value)`, `.gte(value)`, `.and(expr)`, `.contains(text)`, and `.isNull()` to construct expressions. JS operators and conditionals evaluate immediately and cannot build SQL expressions. `select` establishes new field names. Supply one order array for several keys. A later `orderBy` replaces the previous order.

Use tagged SQL for grouping, window functions, or existing SQL. Interpolate scalar values and named parameters. Keep statement text and identifiers in the template. SQL sources can execute directly with inferred SQLite columns. Declare columns using `.columns({name: mt.type.string()})` when composing with fluent methods, and specify the final order on the outer query.

```js
const framework = mt.param.string("framework").default("pi");
const plan = mt.sql`
  SELECT session_id, title
  FROM sessions
  WHERE agent_framework = ${framework}
  ORDER BY session_id
  LIMIT ${25}
`.build();
const result = db.execute(plan, {params: {framework: "codex"}});
```

Parameters support `string`, `boolean`, `integer`, and `number`. Integer parameters may declare `.range(min, max)`. Use `.ui({widget, label, help})` for optional UI metadata. `plan.parameters()` shows descriptors and `plan.explain()` shows compiled SQL and whether the plan maps rows. Executing a plan with new values does not call its construction callbacks again.

## Fragments and runtime callbacks

`query.use(fn)` runs a local composition helper. `query.extend(mt.fragment(config))` adds a group's methods to one branch. A fragment declares `name`, `target: "query"`, optional `requires` field names, and `methods`. Each method receives its current query first and returns a query derived from that input. Fragment construction takes a snapshot of method and field declarations.

```js
const byFramework = mt.fragment({
  name: "framework-selection", target: "query", requires: ["agent_framework"],
  methods: { forFramework(q, name) { return q.where(s => s.agent_framework.eq(name)); } },
});
const plan = mt.from(db.table("sessions")).extend(byFramework)
  .forFramework("pi")
  .mapRows(row => ({id: row.session_id, title: row.title}), {
    columns: {id: mt.type.string(), title: mt.type.string().nullable()},
  }).build();
```

`mapRows` runs during execution after successful, complete SQL delivery. It produces one row per input row and requires declared scalar output columns. Build and execute the terminal stage to obtain rows. A callback error reports its row index without exposing partial output. Use ordinary JS after checking several complete results for operations such as grouping frames or ranking workspaces.

## Results and evidence

The result contains `columns`, `rows`, `count`, `complete`, and optional `error`, `truncated`, `rowsTruncated`, and `cellDiagnostics`. Check the envelope before reducing results. Permissive clipping reports loss while returning available rows. Strict mode suppresses incomplete rows. Query completeness describes delivery of the selected normalized rows. Native evidence and source coverage are separate.

For native evidence, use `mt.native().file(path).inspect()` or `.forEach(record => ...)`. Set `.maxRecordBytes(n)` or `.maxIndexBytes(n)` to bound native processing. Returning false stops iteration and yields a prefix hash. Inspection reports available physical references, compaction coordinates, and explicit limitations. `mt.corpusManifest(paths)` fingerprints archives, and `mt.utf8ByteLength(text)` counts bytes.

Local conversion is separate from queries. Use `mt.importer().file(path).autoDetect().convert().preview()` to review a raw file. Configure `.into(directory)` before conversion and call `.save()` on the resulting importer to write an archive. Saving respects the explicit overwrite option. Runtime metadata is available through `mt.runtime`; use installed command help to see which flags the host supplies.
