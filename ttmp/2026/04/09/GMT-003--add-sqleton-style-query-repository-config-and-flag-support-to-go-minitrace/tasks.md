# Tasks

## TODO

- [x] Review the current embedded-only query-command loading path across serve handlers and `query commands`
- [x] Write the detailed repository config/flag support implementation plan
- [x] Relate the most relevant sqleton and go-minitrace code files to the new design doc
- [ ] Add a shared repository-resolution helper for config, environment, and CLI inputs
- [ ] Add `GO_MINITRACE_QUERY_REPOSITORIES` support and normalization tests
- [ ] Add app-config `queryRepositories` support and parsing tests
- [ ] Add repeated `--query-repository` support to `go-minitrace serve`
- [ ] Add repeated `--query-repository` support to `go-minitrace query commands`
- [ ] Compose embedded + external source roots in explicit precedence order
- [ ] Update the CLI command catalog loader to use resolved source roots instead of embedded-only loading
- [ ] Update the serve query-command handlers to use resolved source roots instead of embedded-only loading
- [ ] Add tests for repository precedence and external-root overrides in CLI/server command loading paths
- [ ] Add help/docs/examples for repository config and override behavior
- [ ] Run `docmgr doctor --ticket GMT-003 --stale-after 30`
