# Changelog

## 2026-04-16

- Initial workspace created


## 2026-04-16

Created ticket with full bug report, root cause analysis, and test case. Fix is one-line change in convert.go:175.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/pi/convert.go — Line 175 hardcodes isError=false

## 2026-04-16

Implemented the Go-side Pi adapter fix for message-level `toolResult.isError` handling and added a regression test that covers both successful and failed message-level tool results without creating spurious turns.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/pi/convert.go — Read `msg["isError"]`/`msg["is_error"]` before applying tool results
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/pi/convert_test.go — Added regression coverage for message-level tool results with both success and failure cases

## 2026-04-16

Verified the Go adapter fix against the real Jellyfin Pi session using a ticket-local Go script. The converter now reports 59 failed tool calls, including the previously misclassified `edit` failures with `File not found` errors.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/16/bug-iserror-001--pi-adapter-iserror-not-mapped-to-output-success/scripts/01-verify-real-session.go — Reproducible verification script that converts the real session and counts failed tool calls
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/pi/convert.go — Verified against the real session after the `isError` fix

## 2026-04-16

Migrated `pkg/minitracecmd/repositories.go` off the removed Glazed `ResolveAppConfigPath` helper to the new config plan API (`config.NewPlan` + `SystemAppConfig`/`HomeAppConfig`/`XDGAppConfig`), using Pinocchio's newer bootstrap style as the reference. Added overlay tests for the app-config loader and command-level tests proving `query commands` picks up repository overrides from app config, with XDG config overriding the legacy home-dotdir config.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitracecmd/repositories.go — Replaced legacy config-path resolution with Glazed config plans and layered config loading
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitracecmd/repositories_test.go — Added coverage for multi-file overlay behavior and omitted-field preservation
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/query/commands_test.go — Added app-config-backed repository discovery tests for the `query commands` command tree

