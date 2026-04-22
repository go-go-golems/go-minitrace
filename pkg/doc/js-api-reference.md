---
Title: JS API Reference
Slug: js-api-reference
Short: JavaScript runtime API available to minitrace JS command handlers
Topics:
- minitrace
- javascript
- js-commands
- js-api
- duckdb
- goja
Commands:
- query commands
Flags:
- archive-glob
- db-path
- table-name
- persist-loaded
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: GeneralTopic
---

JavaScript command handlers run in a Goja-powered JS runtime with access to DuckDB through the `minitrace` module, the Goja NodeJS stdlib, and several other built-in modules. This page is a complete reference for what each module exposes.

## require("minitrace")

The `minitrace` module is the primary API for querying and working with the loaded DuckDB session table.

```js
const mt = require("minitrace");
```

### mt.query(sql, ...args)

Executes a read-only SQL query against the loaded DuckDB table and returns all rows as an array of objects. Arguments are flattened and passed as query parameters.

```js
const rows = mt.query(`
  SELECT id, title, environment->>'agent_framework' AS framework
  FROM ${mt.tableName}
  WHERE timing->>'started_at' > '2026-01-01'
  LIMIT 100
`);
```

- **sql**: A SQL string. Use template literals to embed `mt.tableName`.
- **args**: Optional positional arguments that are flattened and forwarded to DuckDB.
- **Returns**: `Array<Record<string, any>>` — one map per row.
- **Throws**: If the query is not read-only (only `SELECT` and read-only functions are allowed).
- **See also**: `mt.queryOne()` for single-row queries.

### mt.queryOne(sql, ...args)

Same as `mt.query()`, but returns only the first row or `null` if the result set is empty.

```js
const session = mt.queryOne(`
  SELECT * FROM ${mt.tableName}
  WHERE id = $1
`, sessionId);
```

- **Returns**: `Record<string, any> | null`

### mt.tableName

The name of the DuckDB table the archive was loaded into. Use this in SQL FROM clauses inside template literals — never hardcode a table name.

```js
`FROM ${mt.tableName} WHERE ...`
```

- **Type**: `string`

### mt.sql

A collection of helper functions that generate safely escaped SQL literals. Always use these when user-supplied values become part of predicates; never concatenate raw strings.

#### mt.sql.string(value)

Returns a single-quoted, single-quote-escaped SQL string literal.

```js
mt.sql.string("O'Reilly")  // => "'O''Reilly'"
```

- **value**: `string`
- **Returns**: `string` — a single-quoted SQL literal

#### mt.sql.stringIn(value)

Returns a comma-separated, single-quoted list of SQL string literals. Handles `string[]`, `Array<any>`, and bare `string` (returns a single quoted string).

```js
mt.sql.stringIn(["codex", "pi"])  // => "'codex', 'pi'"
mt.sql.stringIn("codex")         // => "'codex'"
```

- **value**: `string | string[] | any[]`
- **Returns**: `string` — a comma-separated, single-quoted list

#### mt.sql.like(value)

Returns a `LIKE` pattern with `%` wildcards on both sides. Handles single-quote escaping.

```js
mt.sql.like("tool fail")  // => "'%tool fail%'"
```

- **value**: `string`
- **Returns**: `string` — a single-quoted `LIKE` pattern

### mt.runtime

A read-only object with information about the current command invocation context.

| Property | Type | Description |
|----------|------|-------------|
| `mt.runtime.tableName` | `string` | Same as `mt.tableName` |
| `mt.runtime.dbPath` | `string` | Path to the loaded DuckDB database |
| `mt.runtime.archiveGlob` | `string[]` | Archive glob patterns used |
| `mt.runtime.persistLoaded` | `boolean` | Whether the table was persisted |
| `mt.runtime.commandName` | `string` | The current command name |

```js
if (mt.runtime.persistLoaded) {
  // use persisted table hints
}
```

## require("timer")

The NodeJS `timer` module for scheduling and async control.

```js
const { setTimeout, setInterval, clearInterval } = require("timer");
```

- `setTimeout(fn, ms)` — run `fn` after `ms` milliseconds
- `setInterval(fn, ms)` — run `fn` every `ms` milliseconds; returns an interval ID
- `clearInterval(id)` — cancel an interval

```js
// Async command using the timer module
function slowAnalysis(filters) {
  return new Promise((resolve) => {
    setTimeout(() => {
      const mt = require("minitrace");
      resolve(mt.query(`SELECT COUNT(*) AS n FROM ${mt.tableName}`));
    }, 1000);
  });
}
```

## require("database")

DuckDB access from go-go-goja. Registered in the default module set.

```js
const db = require("database");
```

Provides direct DuckDB connection access. For most use cases `mt.query()` is simpler; use `database` when you need connection-level control or transactions.

## require("fs")

The NodeJS `fs` module for file system operations.

```js
const fs = require("fs");
```

- `fs.readFileSync(path)` — read a file as a string
- `fs.readFile(path, callback)` — async read
- `fs.writeFileSync(path, data)` — write a string to a file
- `fs.statSync(path)` — get file stats

## require("exec")

Run external shell commands.

```js
const { execSync, exec } = require("exec");
```

- `execSync(cmd)` — run a command synchronously, return stdout
- `exec(cmd, callback)` — run a command asynchronously

## require("path")

The NodeJS `path` module for path manipulation.

```js
const path = require("path");
```

- `path.join(...parts)`
- `path.basename(p)`, `path.dirname(p)`, `path.extname(p)`
- `path.resolve(...parts)`

## require("console")

The NodeJS `console` module. Logs go to the server log or CLI output.

```js
const console = require("console");
console.log("debug info:", someValue);
```

## Other built-in modules

These modules are registered by go-go-goja's default registry but are less commonly needed in command handlers:

| Module | Contents |
|--------|----------|
| `buffer` | NodeJS `Buffer` API (`Buffer.from(...)`, etc.) |
| `events` | NodeJS `EventEmitter` (`new events.EventEmitter()`) |
| `os` | NodeJS `os` module (`os.EOL`, `os.platform()`, etc.) |
| `assert` | NodeJS assertion library |
| `util` | NodeJS utilities (`util.inspect(...)`, `util.format(...)`) |
| `url` | NodeJS `URL` parser |
| `querystring` | NodeJS query string utilities |

## require() of local helper files

JS command files can `require()` sibling or relative helper modules. Those files are resolved relative to the command file location.

```js
// overview/helpers.js — relative helper module
function classifyFramework(framework) {
  if (framework === "codex") return "Codex";
  if (framework === "pi") return "Pi";
  return "Other";
}
module.exports = { classifyFramework };
```

```js
// overview/session-analysis.js — consuming the helper
const { classifyFramework } = require("./helpers");
```

## Output modes

JS command handlers can declare an output mode in `__verb__`:

- **Glazed (default / implied)**: Return row-shaped data — a single object, an array of objects, or a Goja iterator — and it flows through Glazed's processor pipeline with full output formatting (`--output json`, `--output csv`, etc.).
- **Text (not yet supported in `query commands`)**: Returning plain strings or unstructured data is deferred. Do not rely on text-mode output in `go-minitrace query commands` yet.

```js
__verb__("myCommand", {
  name: "my-command",
  short: "My command",
  output: "glaze",   // default — row-shaped data
});
```

## Read-only query validation

Every `mt.query()` and `mt.queryOne()` call is validated before execution. Only `SELECT` statements and read-only functions are allowed. Any write operation (`INSERT`, `UPDATE`, `DELETE`, `COPY`, etc.) will be rejected with an error.

## Scanner marker reference

These static markers are scanned by go-minitrace at load time to build the CLI command surface. They must appear at the top level of the JS file and cannot be inside functions.

### __verb__(name, spec)

Declares a CLI-executable command verb.

```js
__verb__("sessionList", {
  name: "session-list",        // CLI command name (required)
  short: "List sessions",     // Short help text (required)
  long: "Optional longer help text",
  output: "glaze",             // Output mode: "glaze" or "text"
  fields: {                    // Bind flag sections
    filters: { bind: "filters" }
  },
  tags: ["overview", "sessions"],
  metadata: {                  // Free-form metadata
    since: "1.2.0"
  }
});
```

### __section__(name, spec)

Declares a named form section for grouping fields in the CLI flag schema and web UI form.

```js
__section__("filters", {
  title: "Filters",
  fields: {
    framework: {
      type: "stringList",
      default: [],
      help: "Filter by agent framework",
    },
    limit: {
      type: "int",
      default: 25,
      help: "Maximum number of rows",
    },
  },
});
```

### __package__(name, spec)

Package-level metadata for the file.

```js
__package__("session-tools", {
  short: "Session analysis tools",
  long: "Set of commands for analyzing minitrace sessions",
});
```

## Supported field types

Fields declared in `__section__` use Glazed field definitions. The following types are fully supported in both CLI and web UI:

| Type | CLI flag example | Notes |
|------|-------------------|-------|
| `string` | `--my-field value` | |
| `bool` | `--my-field` | Defaults to `false` when optional |
| `int` | `--my-field 42` | |
| `float` | `--my-field 3.14` | |
| `date` | `--my-field 2026-01-01` | ISO 8601 date |
| `choice` | `--my-field opt1` | Restricted to defined choices |
| `stringList` | `--my-field a,b,c` | Comma-separated values |
| `intList` | `--my-field 1,2,3` | Comma-separated integers |
| `floatList` | `--my-field 1.0,2.5` | Comma-separated floats |
| `choiceList` | `--my-field opt1,opt2` | Multiple restricted choices |

Types outside this set may work on the CLI but the web form will not render them correctly yet.

## See also

- `go-minitrace help structured-query-commands` — authoring guide for JS command files
- `go-minitrace help writing-duckdb-queries` — SQL patterns for the `sessions_base` schema
- `testdata/query-repositories/js-showcase/` — worked examples
- `testdata/query-repositories/mixed-sql-js-showcase/` — mixed SQL and JS repositories
