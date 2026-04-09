# Tasks

## TODO

- [x] Review the current embedded-only query-command loading path across serve handlers and `query commands`
- [x] Write the detailed repository config/flag support implementation plan
- [x] Relate the most relevant sqleton and go-minitrace code files to the new design doc
- [x] Add a shared repository-resolution helper for config, environment, and CLI inputs
- [x] Add `GO_MINITRACE_QUERY_REPOSITORIES` support and normalization tests
- [x] Add app-config `queryRepositories` support and parsing tests
- [x] Add repeated `--query-repository` support to `go-minitrace serve`
- [x] Add repeated `--query-repository` support to `go-minitrace query commands`
- [x] Compose embedded + external source roots in explicit precedence order
- [x] Update the CLI command catalog loader to use resolved source roots instead of embedded-only loading
- [x] Update the serve query-command handlers to use resolved source roots instead of embedded-only loading
- [x] Add tests for repository precedence and external-root overrides in CLI/server command loading paths
- [x] Add help/docs/examples for repository config and override behavior
- [x] Run `docmgr doctor --ticket GMT-003 --stale-after 30`
