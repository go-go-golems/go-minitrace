# Query repository showcase guide

This directory contains checked-in example repositories for `go-minitrace query commands`.

Use this file as the starting point when you want to understand which showcase to copy from.

## At a glance

| Repository | Best for | Highlights |
| --- | --- | --- |
| `js-showcase/` | Learning JS-backed command authoring | Multi-verb JS files, helper modules, async verbs, advanced analysis commands, JS aliases |
| `mixed-sql-js-showcase/` | Designing a realistic mixed repository | SQL leaf commands, JS file-group commands, SQL aliases, JS aliases in one tree |

## Recommended reading order

### 1. Start with `js-showcase/`

Path:

- `testdata/query-repositories/js-showcase/`

Use this when you want to learn the JS command model itself.

It demonstrates:

- scanner-first JS discovery,
- file-stem command grouping (`overview/session-tools.js` -> `overview session-tools ...`),
- multiple verbs per JS file,
- helper modules hidden from command discovery,
- `query(...)` and `queryOne(...)`,
- JS-side reshaping/scoring/classification,
- async commands using `require("timer")`,
- advanced analysis examples,
- aliases that target JS-backed commands.

Representative commands:

- `overview session-tools session-list`
- `overview async-tools delayed-summary`
- `analysis workspace-lab workspace-scoreboard`
- `analysis tool-intelligence tool-pair-matrix`
- `analysis aliases focus-top-workspaces`

### 2. Then read `mixed-sql-js-showcase/`

Path:

- `testdata/query-repositories/mixed-sql-js-showcase/`

Use this when you want to see how a production-style repository might mix direct SQL for simple queries and JS for richer orchestration.

It demonstrates:

- SQL leaf commands alongside JS file-group commands,
- SQL aliases and JS aliases side by side,
- the same grouping/help model across both runtimes,
- when a query can remain a plain `.sql` file versus when it benefits from JS post-processing.

Representative commands:

- `overview framework-summary`
- `overview aliases codex-framework-summary`
- `overview session-tools framework-share`
- `analysis raw-workspace-stats`
- `analysis workspace-lab workspace-scoreboard`
- `analysis aliases top-workspaces`

## Which showcase should I copy from?

### Copy from `js-showcase/` if:

- you are authoring your first JS verbs,
- you need multiple queries combined in one command,
- you want JS-side joins, scoring, ranking, or classification,
- you want examples of helper modules and async verbs,
- you want aliases over JS-backed commands.

### Copy from `mixed-sql-js-showcase/` if:

- you already know the basics,
- your repo will contain both SQL and JS commands,
- some commands are simple enough to stay SQL,
- other commands need richer orchestration in JS,
- you want an example of a repository users can browse in help and immediately understand.

## Suggested smoke commands

### JS showcase

```bash
go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/js-showcase \
  analysis aliases focus-top-workspaces \
  --archive-glob './output/active/*/*.minitrace.json' \
  --output json
```

### Mixed SQL + JS showcase

```bash
go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/mixed-sql-js-showcase \
  overview framework-summary \
  --archive-glob './output/active/*/*.minitrace.json' \
  --output json

go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/mixed-sql-js-showcase \
  analysis aliases top-workspaces \
  --archive-glob './output/active/*/*.minitrace.json' \
  --output json
```

## Rule of thumb

- If one SQL query is enough, start with `.sql`.
- If you need multiple queries, joins in host code, scoring, ranking, or nontrivial shaping, move to `.js`.
- If you want a stable shortcut for either kind, add an alias.
