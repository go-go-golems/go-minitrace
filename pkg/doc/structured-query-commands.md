---
Title: Structured Query Commands
Slug: structured-query-commands
Short: Run and author repository-backed sqleton-style query commands for go-minitrace
Topics:
- minitrace
- sqlite
- sqleton
- glazed
Commands:
- query
- query commands
- serve
Flags:
- query-repository
- archive-glob
- db-path
- table-name
- persist-loaded
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

Structured query commands add a metadata layer on top of raw SQL and JavaScript-backed analysis scripts. Instead of remembering a long `--sql` string or maintaining ad hoc files with no parameter schema, you define a scanner-first command source, let go-minitrace turn its fields into real CLI parameters and web-form fields, and then either render read-only SQL or invoke a JS handler that explicitly builds a normalized SQLite database with `mt.db()`.

This matters when a query or analysis script becomes part of your team's repeatable workflow. A structured command gives you a stable name, typed inputs, alias support, discoverability in `go-minitrace query commands --help`, and a matching form in the `/query` web UI.

## What gets loaded

The `query commands` subgroup loads a catalog from two places:

1. the embedded go-minitrace catalog shipped in `pkg/minitracecmd/core/`
2. any external repositories configured through flags, environment, or app config

Today the embedded catalog includes examples such as:

- `overview session-list`
- `overview framework-summary`
- `timing timing-analysis`
- `nightly session-inventory`
- `nightly workspace-summary`
- `overview aliases codex-framework-summary` (an alias)

You can inspect the currently loaded commands with standard Cobra help:

```bash
go-minitrace query commands --help
go-minitrace query commands overview session-list --help
go-minitrace query commands overview framework-summary --help
```

## Running structured query commands from the CLI

The CLI surface is:

```bash
go-minitrace query commands <group...> <command-name> [command flags] [query runtime flags]
```

Repository subdirectories become nested Cobra groups. SQL files map directly to leaf commands, while JS files usually add one extra group level based on the file stem. For example:

- `pkg/minitracecmd/core/overview/session-list.sql` → `go-minitrace query commands overview session-list`
- `pkg/minitracecmd/core/overview/session-tools.js` with `name: session-list` → `go-minitrace query commands overview session-tools session-list`
- `pkg/minitracecmd/core/nightly/session-inventory.sql` → `go-minitrace query commands nightly session-inventory`
- `pkg/minitracecmd/core/overview/aliases/codex-framework-summary.alias.yaml` → `go-minitrace query commands overview aliases codex-framework-summary`

A special case avoids redundant doubled paths for self-named single-verb JS files. If `hardware-research/research-summary.js` defines exactly one verb named `research-summary`, the executable path is collapsed to:

- `go-minitrace query commands hardware-research research-summary`

instead of `hardware-research research-summary research-summary`.

The command-specific flags come from the sqleton-style file metadata. The query-runtime flags control which archives are loaded: `--archive-glob` selects the session files that get materialized into the normalized SQLite database. The legacy DuckDB flags `--db-path`, `--table-name`, and `--persist-loaded` are still accepted but deprecated: SQL commands ignore them (a warning is printed when they are set), and they are only surfaced to JS commands through `mt.runtime`.

List sessions with the embedded command:

```bash
go-minitrace query commands overview session-list \
  --archive-glob './output/active/*/*.minitrace.json'
```

Filter the list by framework and limit:

```bash
go-minitrace query commands overview session-list \
  --archive-glob './output/active/*/*.minitrace.json' \
  --framework codex,pi \
  --limit 25
```

Run a summary command:

```bash
go-minitrace query commands overview framework-summary \
  --archive-glob './output/active/*/*.minitrace.json'
```

Run an alias command with baked-in defaults:

```bash
go-minitrace query commands overview aliases codex-framework-summary \
  --archive-glob './output/active/*/*.minitrace.json'
```

Use `--output json`, `--output csv`, `--fields ...`, and the other standard Glazed output flags exactly as you would with `query run`. The rendered SQL executes through the same sandboxed normalized-SQLite query runner that JS commands use (read-only validation, row/column/cell limits, per-query timeout).

## Using structured commands through serve and the web UI

The same catalog is also available when you run the HTTP server:

```bash
go-minitrace serve --archive-glob './output/active/*/*.minitrace.json'
```

In serve mode, structured commands matter in three ways:

- the `/query` page shows them in the Commands sidebar
- the form is generated from the command's typed flag definitions
- the backend exposes listing and execution endpoints under `/api/v2/query-commands`

That means one command definition can power:

- CLI execution through `go-minitrace query commands ...`
- interactive browser execution in the query editor
- API-driven execution from other clients

The browser UI also exposes raw-template and rendered-SQL debug panels so you can see exactly what the command definition produced.

## Repository discovery and override rules

External repositories are discovered with this precedence:

1. repeated `--query-repository` flags
2. `GO_MINITRACE_QUERY_REPOSITORIES`
3. `queryRepositories` in app and local config files, resolved from lower to higher layers:
   - `/etc/go-minitrace/config.yaml`
   - `~/.go-minitrace/config.yaml`
   - `$XDG_CONFIG_HOME/go-minitrace/config.yaml`
   - `<git-root>/.go-minitrace.yml`
   - `<git-root>/.go-minitrace.override.yml`
   - `<cwd>/.go-minitrace.yml`
   - `<cwd>/.go-minitrace.override.yml`
4. the embedded catalog last

Higher-precedence repositories are mounted first so they can override embedded commands without changing loader behavior. If multiple config files contain `queryRepositories`, the later/higher layer replaces the earlier config-file list. CLI flags and environment repositories are then prepended ahead of the config-derived repositories.

### CLI flag

Use one or more `--query-repository` flags:

```bash
go-minitrace query commands overview framework-summary \
  --query-repository ./query-commands/team \
  --query-repository ./query-commands/local \
  --archive-glob './output/active/*/*.minitrace.json'
```

The flag uses StringSlice semantics, so comma-separated values work too:

```bash
go-minitrace query commands overview framework-summary \
  --query-repository './query-commands/team,./query-commands/local' \
  --archive-glob './output/active/*/*.minitrace.json'
```

### Environment variable

Use the platform path-list separator (`:` on Unix, `;` on Windows):

```bash
export GO_MINITRACE_QUERY_REPOSITORIES=./query-commands/team:./query-commands/local

go-minitrace query commands overview session-list \
  --archive-glob './output/active/*/*.minitrace.json'
```

### App config and local config

Add repository roots to the go-minitrace app config:

```yaml
queryRepositories:
  - ./query-commands/team
  - ~/.config/go-minitrace/query-commands
```

App config files are read from `/etc/go-minitrace/config.yaml`, `~/.go-minitrace/config.yaml`, and `$XDG_CONFIG_HOME/go-minitrace/config.yaml`. Relative paths inside config files are resolved relative to the config file directory.

For project-local query commands, add a git-root or current-working-directory config file:

```yaml
# .go-minitrace.yml
queryRepositories:
  - ./query-commands
```

Use `.go-minitrace.override.yml` for machine-local or private overrides that should take precedence over `.go-minitrace.yml` in the same git root or current working directory:

```yaml
# .go-minitrace.override.yml
queryRepositories:
  - ./private-query-commands
```

When both git-root and current-working-directory files exist, the current-working-directory files are higher precedence. If the current working directory is also the git root, duplicate paths are deduped.

## Repository layout

A repository is just a directory tree containing scanner-first command files and optional alias YAML files.

A small example looks like this:

```text
query-commands/
├── overview/
│   ├── session-list.sql
│   ├── session-tools.js
│   ├── framework-summary.sql
│   └── aliases/
│       └── codex-framework-summary.alias.yaml
├── timing/
│   └── timing-analysis.sql
└── tools/
    └── tool-failures.sql
```

Those folders become nested CLI groups. The source file extension decides how the command is executed:

- `.sql` -> sqleton-style SQL preamble + SQL template body
- `.js` / `.cjs` -> scanner-first JS metadata (`__verb__`, `__section__`, `__package__`) + JS handler body
- `.alias.yaml` / `.alias.yml` -> alias definition that targets a previously loaded command by name

Those folders become nested CLI groups, and JS file stems usually become one more group level:

```bash
go-minitrace query commands overview session-list
go-minitrace query commands overview session-tools session-list
go-minitrace query commands overview framework-summary
go-minitrace query commands overview aliases codex-framework-summary
go-minitrace query commands timing timing-analysis
go-minitrace query commands tools tool-failures
```

The CLI leaf command name still comes from the file metadata `name:` field, not from the filename alone. For JS sources, the filename usually contributes a group and the scanned verb name contributes the final leaf command. The exception is a self-named single-verb JS file, where go-minitrace collapses the redundant extra path level.

## Writing a sqleton-style SQL command file

A structured SQL command is a `.sql` file whose first block is a `/* sqleton ... */` YAML preamble.

The preamble describes the command name, help text, and typed parameters. The rest of the file is the SQL template that runs against the normalized SQLite schema (`sessions`, `turns`, `tool_calls`, `annotations`, `metrics`, ...).

Here is a minimal but realistic example:

```sql
/* sqleton
name: session-list
short: List minitrace sessions
flags:
  - name: framework
    type: stringList
    help: Filter by agent framework
  - name: title_like
    type: string
    help: Filter titles with LIKE
  - name: limit
    type: int
    default: 100
    help: Limit the number of rows returned
*/
SELECT
  session_id AS id,
  agent_framework AS framework,
  model,
  title,
  turn_count AS turns,
  tool_call_count AS tools
FROM sessions
WHERE 1=1
{{ if .framework -}}
AND agent_framework IN ({{ .framework | sqlStringIn }})
{{ end -}}
{{ if .title_like -}}
AND title LIKE {{ .title_like | sqlLike }}
{{ end -}}
ORDER BY started_at DESC
LIMIT {{ .limit }};
```

### Required fields

At minimum, a command file needs:

- `name`: the CLI-visible command name
- `short`: a short help description
- a non-empty SQL body after the preamble

### Optional fields

You can also include:

- `long`: longer help text
- `flags`: Glazed field definitions for named parameters
- `arguments`: positional parameters
- `layout`: Glazed layout metadata for form organization
- `tags`: extra metadata for organization/search
- `metadata`: free-form command metadata

## Supported parameter types

Structured commands reuse Glazed field definitions. That means the CLI side gets real typed flags, and the web UI can render controls based on the field type.

The current web form supports these practical field types:

- `string`
- `bool`
- `choice`
- `int`
- `float`
- `date`
- `stringList`
- `intList`
- `floatList`
- `choiceList`

If you choose a field type outside that currently supported UI set, the CLI may still be able to expose it as a flag, but the browser form may not render it the way you expect yet.

## Template variables and helper functions

The SQL body is rendered locally by go-minitrace before execution. Two inputs matter most:

- `{{TABLE_NAME}}` expands to the `sessions_base` compatibility view, which reconstructs the legacy session-level JSON-blob columns from the normalized `sessions` table. New command files should reference the normalized tables (`sessions`, `tool_calls`, ...) directly instead.
- `{{ .flag_name }}` accesses one of your structured flag values

The current helper functions are:

- `sqlString`
- `sqlStringIn`
- `sqlIntIn`
- `sqlLike`

Use these helpers whenever user input becomes part of a predicate. They handle the quoting and escaping rules that the command renderer expects.

For example:

```sql
WHERE title LIKE {{ .title_like | sqlLike }}
```

or:

```sql
AND id IN ({{ .session_ids | sqlStringIn }})
```

## Writing a scanner-first JS command file

A structured JS command is a `.js` or `.cjs` file that exposes command metadata through static scanner markers and executes only when the command is invoked.

A minimal example looks like this:

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
      default: 25,
      help: "Maximum number of rows",
    },
  },
});

function sessionList(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    return db.query(`
      SELECT
        session_id,
        title,
        agent_framework AS framework
      FROM sessions
      WHERE 1=1
      ${filters.framework?.length ? `AND agent_framework IN (${mt.sql.stringIn(filters.framework)})` : ""}
      ORDER BY started_at DESC
      LIMIT ${filters.limit}
    `);
  } finally {
    db.close();
  }
}

__verb__("sessionList", {
  name: "session-list",
  short: "List minitrace sessions",
  fields: {
    filters: { bind: "filters" }
  }
});
```

If that file is stored as `overview/session-tools.js`, the resulting CLI path is:

```bash
go-minitrace query commands overview session-tools session-list
```

Important rules for JS command files:

- metadata must stay static and scanner-friendly,
- `__verb__` must point at a top-level function,
- one file may define multiple verbs,
- the JS file stem usually becomes a command group and each scanned verb name becomes a leaf command,
- if a JS file defines exactly one verb and that verb has the same name as the file stem, go-minitrace collapses the redundant extra path level,
- helper modules can still be loaded with relative `require()` calls,
- and text-output JS commands are currently deferred in `go-minitrace query commands`.

For more realistic examples, start with the checked-in showcase guide:

- `testdata/query-repositories/README.md`

Then inspect the repositories under:

- `testdata/query-repositories/js-showcase/`
- `testdata/query-repositories/mixed-sql-js-showcase/`

Those testdata repositories demonstrate:

- multi-verb files,
- JS aliases targeting JS-backed commands,
- relative helper modules,
- pure synthetic row generation with no DB query,
- async commands using `require("timer")`,
- `queryOne(...)` plus JS-side reshaping,
- query results post-processed in JavaScript before emission,
- multiple SQL queries joined together in JS,
- JS-side scoring and session classification logic,
- side-by-side SQL leaves and JS file-group commands in one repository,
- and real-data validation workflows using `go-minitrace convert pi --source-dir ~/.pi/agent/sessions ...`.

## Writing alias files

Alias files let you publish a shortcut command that points at another command and supplies default values.

A minimal alias looks like this:

```yaml
name: codex-framework-summary
short: Summarize only codex sessions
aliasFor: framework-summary
flags:
  framework:
    - codex
```

This matters when you want a stable, memorable command for a common filter without copying the SQL template.

A few rules are worth remembering:

- aliases are stored in `.alias.yaml` or `.alias.yml` files
- `aliasFor` points at the target command name
- caller-supplied values override alias defaults
- aliases are first-class entries in the loaded catalog and in the UI sidebar

## Authoring workflow

A practical authoring loop looks like this:

1. create a repository directory
2. add one sqleton-style `.sql` command file or one scanner-first `.js` command file
3. run `go-minitrace query commands <groups...> <leaf> --help` to verify the parameter schema
4. run the command against a real archive glob
5. optionally open `/query` in serve mode and verify the form/debug panels
6. add alias files for repeated filters only after the base command works

A concrete example:

```bash
mkdir -p ./query-commands/overview/aliases
$EDITOR ./query-commands/overview/session-tools.js

go-minitrace query commands overview session-tools session-list \
  --query-repository ./query-commands \
  --archive-glob './output/active/*/*.minitrace.json' \
  --help

go-minitrace query commands overview session-tools session-list \
  --query-repository ./query-commands \
  --archive-glob './output/active/*/*.minitrace.json' \
  --framework codex

If your JS file is self-named and defines exactly one same-named verb, use the collapsed path instead. For example:

go-minitrace query commands hardware-research research-summary \
  --query-repository ./query-commands \
  --archive-glob './output/active/*/*.minitrace.json'
```

## When to use structured commands vs raw SQL

Use `query commands` when:

- the query should be discoverable by name
- the query has reusable typed parameters
- you want the same definition to work in CLI, API, and browser UI
- you want aliases for common defaults

Use `query run --sql` or `--sql-file` when:

- you are doing one-off exploration
- the query is still changing rapidly
- the query does not need a typed parameter schema yet

The two approaches are complementary. Raw `query run` is the fastest ad hoc path over the normalized SQLite tables. Structured SQL commands make that style reusable, while structured JS commands can post-process results in JavaScript when the analysis needs multiple queries, reshaping, or table-level schema discovery.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| `unknown command` under `query commands` | The repository root was not loaded, or the command `name:` is different from the filename you expected | Run `go-minitrace query commands --help`, then check `--query-repository`, env/config sources, and the file's `name:` field |
| A repository command does not override the embedded one | The higher-precedence repository was not mounted first, or the path/config source was wrong | Prefer explicit `--query-repository` during debugging and confirm the directory exists |
| The browser form is missing a field | The field type is not currently rendered by the UI | Use a currently supported practical type or run the command through the CLI instead |
| The rendered SQL fails read-only validation | The command template produced non-read-only SQL | Keep the command limited to `SELECT`-style analysis queries |
| A JS command fails during execution | The handler threw, rejected a Promise, or called the host API with invalid SQL | Re-run through the CLI first, then check the exact JS runtime error and the handler source file |
| `unknown flag: --archive-glob` on a JS command that should support runtime flags | You probably stopped at an intermediate JS group instead of the executable leaf, or you assumed a doubled path for a self-named single-verb JS file | Run `... --help` on the exact path you are invoking. Multi-verb JS files keep the extra file-stem group; self-named single-verb JS files collapse it |
| A JS command with `output: "text"` does not run | Text-output JS commands are currently deferred in `go-minitrace query commands` | Return row-shaped data for now, or implement writer-mode support in a follow-up slice |
| An alias does not behave as expected | The target command name is wrong, or the alias default shape does not match the target flag type | Verify `aliasFor`, then compare the alias `flags:` values to the target command's definitions |
| `--query-repository` with commas behaves strangely | Shell quoting changed the argument before Cobra saw it | Wrap the comma-separated value in quotes or use repeated flags |

## See also

- `go-minitrace help js-api-reference` — complete JS runtime API: `mt.db()`, `db.query()`, normalized SQLite tables, cache modes, `mt.sql.*`, `mt.runtime`, built-in modules, scanner markers, field types
- `go-minitrace help analysis-guide` — end-to-end workflow with JS command guidance, authoring loop, and when to use JS vs SQL
- `go-minitrace help query-recipes` — ready-to-use SQL examples that can be promoted into structured commands
- `go-minitrace help writing-queries` — SQL patterns for the normalized schema: joins, JSON access, filtering
- `go-minitrace help query-duckdb` — migrating saved DuckDB-era SQL to the normalized engine
- `go-minitrace help query-commands` — the `query run` command, presets, and custom SQL modes
- `go-minitrace help getting-started` — end-to-end first-run tutorial
- `README.md` — project overview and quick-start commands
