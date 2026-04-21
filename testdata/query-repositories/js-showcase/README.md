# JS showcase query repository

This testdata repository demonstrates scanner-first JavaScript query commands that do more than just wrap a single SQL statement.

## Included command groups

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

## Suggested smoke commands

```bash
go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/js-showcase \
  overview --help

go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/js-showcase \
  overview session-tools framework-share \
  --archive-glob './output/active/*/*.minitrace.json' \
  --output json

go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/js-showcase \
  overview runtime-playground build-synthetic-rows \
  --archive-glob './output/active/*/*.minitrace.json' \
  --prefix demo \
  --tags alpha,beta,gamma \
  --output json
```
