# JS showcase query repository

This testdata repository demonstrates scanner-first JavaScript query commands that go well beyond wrapping a single SQL statement. It is intended as both:

- a copy-paste starting point for authors, and
- a reusable test/smoke fixture for `go-minitrace`.

## Included command groups

## Overview examples

### `overview session-tools ...`
Defined in `overview/session-tools.js`.

- `session-list`
  - basic JS-backed query command
  - uses section-bound typed inputs
  - uses `mt.sql.stringIn(...)`
- `framework-share`
  - queries grouped framework counts
  - uses a relative helper module (`./lib/transforms`)
  - computes `share_percent` and `rank` in JS after the query

### `overview runtime-playground ...`
Defined in `overview/runtime-playground.js`.

- `show-context`
  - returns a row built entirely from runtime metadata
  - no DuckDB query needed inside the handler
- `build-synthetic-rows`
  - returns multiple rows generated entirely from typed inputs
  - showcases pure JS row generation

### `overview async-tools ...`
Defined in `overview/async-tools.js`.

- `delayed-summary`
  - async command using `require("timer")`
  - uses `mt.queryOne(...)`
  - returns a single synthesized summary row
- `top-session-cards`
  - async command returning multiple rows
  - reshapes queried rows into richer card-like objects in JS

## Analysis examples

### `analysis workspace-lab ...`
Defined in `analysis/workspace-lab.js`.

- `workspace-scoreboard`
  - runs multiple queries over the same archive
  - joins workspace stats, highlight sessions, and tool usage in JS
  - computes a derived `focus_score`
- `workspace-session-highlights`
  - turns workspace aggregates into card-like rows
  - combines top titles, models, and dominant tools

### `analysis tool-intelligence ...`
Defined in `analysis/tool-intelligence.js`.

- `toolbox-overview`
  - joins overall tool counts, dominant operation type, and dominant workspace
  - computes JS-side reuse density metrics
- `tool-pair-matrix`
  - loads distinct session/tool rows and computes co-occurring tool pairs in JS

### `analysis session-architectures ...`
Defined in `analysis/session-architectures.js`.

- `session-shape-ranker`
  - joins session metrics, per-role turn counts, and unique tool counts
  - classifies sessions into labels like `tool-orchestrator` or `balanced-builder`
- `session-spotlights`
  - produces spotlight rows combining title, dominant tools, and role mix

## Patterns demonstrated

This showcase now covers all of these patterns:

- multi-verb files
- relative helper modules
- pure synthetic row generation with no DB query
- async commands using `require("timer")`
- `queryOne(...)`
- query results post-processed in JavaScript before emission
- multiple SQL queries combined in JS
- JS-side joins across independently queried aggregates
- JS-side scoring/classification logic
- per-session tool co-occurrence analysis in JS

## Suggested smoke commands

```bash
go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/js-showcase \
  overview --help

go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/js-showcase \
  analysis workspace-lab workspace-scoreboard \
  --archive-glob './output/active/*/*.minitrace.json' \
  --output json

go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/js-showcase \
  analysis tool-intelligence toolbox-overview \
  --archive-glob './output/active/*/*.minitrace.json' \
  --output json

go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/js-showcase \
  analysis session-architectures session-shape-ranker \
  --archive-glob './output/active/*/*.minitrace.json' \
  --output json
```

## Validate against real Pi traces

You can validate these examples against real local Pi sessions without committing any private data:

```bash
go run ./cmd/go-minitrace convert pi \
  --source-dir ~/.pi/agent/sessions \
  --output-dir /tmp/pi-minitrace-showcase

go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/js-showcase \
  analysis workspace-lab workspace-scoreboard \
  --archive-glob '/tmp/pi-minitrace-showcase/active/*/*.minitrace.json' \
  --output json
```

That is the recommended way to see the more advanced examples produce interesting, non-synthetic rows on real session data.
