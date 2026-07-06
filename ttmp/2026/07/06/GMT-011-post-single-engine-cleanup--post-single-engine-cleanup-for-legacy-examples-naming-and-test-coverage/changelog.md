# Changelog

## 2026-07-06

- Initial workspace created


## 2026-07-06

Implemented post-single-engine cleanup: stale DuckDB examples removed from Makefile/testdata/web stories, README migration checklist added, JS runtime tests restored by renaming _js test file, shared query fixtures consolidated, and Go tests plus glazed-lint pass.

### Related Files

- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/Makefile — Removed deprecated --db-path from dev serve examples
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/README.md — Added user migration checklist
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/cmd/go-minitrace/cmds/query/command_runtime_javascript_test.go — Restored JS runtime test coverage
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/cmd/go-minitrace/cmds/query/sqlite_fixtures_test.go — Shared richer query fixture
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/web/src/components/QueryEditor/stories/QueryEditor.stories.tsx — Normalized Storybook SQL examples

