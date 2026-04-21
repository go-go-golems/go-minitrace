# Mixed SQL + JS showcase query repository

This repository demonstrates how plain SQL command files, JS-backed command groups, and aliases can coexist in a single structured query-command repository.

## Layout

- SQL leaf commands:
  - `overview/framework-summary.sql`
  - `analysis/raw-workspace-stats.sql`
- JS file-group commands:
  - `overview/session-tools.js` -> `overview session-tools ...`
  - `analysis/workspace-lab.js` -> `analysis workspace-lab ...`
- Aliases:
  - SQL alias in `overview/aliases/`
  - JS alias in `analysis/aliases/`

## Suggested smoke commands

```bash
go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/mixed-sql-js-showcase \
  overview --help

go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/mixed-sql-js-showcase \
  overview framework-summary \
  --archive-glob './output/active/*/*.minitrace.json' \
  --output json

go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/mixed-sql-js-showcase \
  overview session-tools framework-share \
  --archive-glob './output/active/*/*.minitrace.json' \
  --output json

go run ./cmd/go-minitrace query commands \
  --query-repository ./testdata/query-repositories/mixed-sql-js-showcase \
  analysis aliases top-workspaces \
  --archive-glob './output/active/*/*.minitrace.json' \
  --output json
```
