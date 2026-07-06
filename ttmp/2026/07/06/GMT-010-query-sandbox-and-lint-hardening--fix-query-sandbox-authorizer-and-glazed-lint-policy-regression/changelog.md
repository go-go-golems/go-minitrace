# Changelog

## 2026-07-06

- Initial workspace created


## 2026-07-06

Implemented lint and query-sandbox hardening: cache-size lookup is explicit on DBBuilder, attached SQLite reads are schema-aware, regression tests added, and full go test plus glazed-lint pass.

### Related Files

- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace/pkg/minitracedb/query.go — Schema-aware SQLite authorizer
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace/pkg/minitracedb/query_test.go — Attached-schema regression test
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace/pkg/minitracedb/schema.go — Explicit anno.* allowlist entries
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace/pkg/minitracejs/db_builder.go — Glazed lint fix removes direct env lookup
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace/pkg/minitracejs/db_builder_cache_test.go — Cache-size explicit override tests


## 2026-07-06

Committed GMT-010 hardening implementation as 4065f0c: schema-aware SQLite authorizer, explicit cache-size builder setting, regression tests, and passing go test/glazed-lint.

### Related Files

- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/minitracedb/query.go — Committed schema-aware authorizer
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/minitracejs/db_builder.go — Committed lint fix

