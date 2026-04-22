#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="/home/manuel/code/wesen/corporate-headquarters/go-minitrace"
cd "$REPO_ROOT"

printf '\n== runtime settings types ==\n'
rg -n "type (DuckDBQuerySettings|MinitraceQueryRuntimeSettings)|archive-glob|db-path|table-name|persist-loaded" \
  cmd/go-minitrace/cmds/query/{duckdb.go,command_runtime.go,runtime_section.go} -S

printf '\n== commands command builder ==\n'
rg -n "BuildCobraCommandWithShortHelpSections|QueryRuntimeSectionSlug|StringSlice\(minitracecmd.QueryRepositoryFlagName" \
  cmd/go-minitrace/cmds/query/commands.go cmd/go-minitrace/cmds/common/build.go -S

printf '\n== docs that advertise archive-glob on query commands ==\n'
rg -n "query commands .*archive-glob|JS files add one more group level|session-tools session-list" \
  pkg/doc cmd/go-minitrace/cmds/query/commands.go -S
