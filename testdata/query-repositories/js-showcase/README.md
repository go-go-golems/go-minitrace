# JS showcase query repository

See also: `../README.md` for a comparison of all checked-in showcase repositories.


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

### `analysis phase3-cookbook ...`
Defined in `analysis/phase3-cookbook.js`.

- `context-inventory`
  - joins sessions, metrics, annotations, handovers, and spawned-agent aggregates
  - demonstrates the Phase 3 operational-context and usage/token columns
- `annotation-risk-matrix`
  - groups normalized annotation rows by category, classification, and scope
- `handover-queue`
  - lists received and produced handover documents
- `spawned-agent-audit`
  - lists tool calls that spawned subagents and their outcomes

### `analysis report-cookbook ...`
Defined in `analysis/report-cookbook.js`.

- `session-inventory`
  - reports session-level metrics, annotations, handovers, tokens, cost, model, and workspace
- `tool-risk-matrix`
  - groups tool calls by tool/operation with failure, annotation, spawned-agent, duration, and payload-size signals
- `file-heatmap`
  - groups normalized file touches by path and operation type
- `prompt-instruction-audit`
  - audits system prompt coverage with simple raw-SQL heuristics
- `turn-timeline`
  - reads the normalized `events` table as a turn/tool/annotation timeline

## Patterns demonstrated

This showcase now covers all of these patterns:

- multi-verb files
- JS aliases that target JS-backed analysis commands
- relative helper modules
- pure synthetic row generation with no DB query
- async commands using `require("timer")`
- `queryOne(...)`
- query results post-processed in JavaScript before emission
- multiple SQL queries combined in JS
- JS-side joins across independently queried aggregates
- JS-side scoring/classification logic
- per-session tool co-occurrence analysis in JS
- Phase 3 normalized schema cookbook queries over annotations, handovers, usage tokens, operational context, and spawned agents
- raw SQL report cookbook examples for tool risk, file heatmaps, prompt audits, timelines, and session inventories

## Alias examples

Additional alias examples live under:

- `analysis/aliases/focus-top-workspaces.alias.yaml`
- `analysis/aliases/core-tool-pairs.alias.yaml`
- `analysis/aliases/heavy-session-shapes.alias.yaml`

These show that aliases can target advanced JS-backed commands just like they already target SQL commands.

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

go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/js-showcase \
  analysis phase3-cookbook context-inventory \
  --archive-glob './output/active/*/*.minitrace.json' \
  --output json

go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/js-showcase \
  analysis report-cookbook tool-risk-matrix \
  --archive-glob './output/active/*/*.minitrace.json' \
  --output json

go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/js-showcase \
  analysis aliases focus-top-workspaces \
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
