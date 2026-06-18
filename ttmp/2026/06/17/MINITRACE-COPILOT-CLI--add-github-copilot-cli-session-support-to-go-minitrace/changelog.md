# Changelog

## 2026-06-17

- Initial workspace created


## 2026-06-17

Completed evidence gathering and intern-ready Copilot CLI adapter design package; no implementation started.

### Related Files

- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/ttmp/2026/06/17/MINITRACE-COPILOT-CLI--add-github-copilot-cli-session-support-to-go-minitrace/design-doc/01-github-copilot-cli-session-support-design-and-implementation-guide.md — Primary design and implementation guide
- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/ttmp/2026/06/17/MINITRACE-COPILOT-CLI--add-github-copilot-cli-session-support-to-go-minitrace/sources/github-docs-copilot-cli-session-data.md — Official Copilot session data reference
- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/ttmp/2026/06/17/MINITRACE-COPILOT-CLI--add-github-copilot-cli-session-support-to-go-minitrace/sources/local-copilot-session-structural-analysis.md — Local sample evidence


## 2026-06-17

Validated ticket and uploaded bundled design package to reMarkable at /ai/2026/06/17/MINITRACE-COPILOT-CLI.

### Related Files

- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/ttmp/2026/06/17/MINITRACE-COPILOT-CLI--add-github-copilot-cli-session-support-to-go-minitrace/design-doc/01-github-copilot-cli-session-support-design-and-implementation-guide.md — Uploaded primary design guide
- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/ttmp/2026/06/17/MINITRACE-COPILOT-CLI--add-github-copilot-cli-session-support-to-go-minitrace/reference/01-investigation-diary.md — Uploaded investigation diary


## 2026-06-17

Implemented Copilot session-state discovery package and tests.

### Related Files

- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/pkg/adapters/copilot/discover.go — Discovery implementation
- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/pkg/adapters/copilot/discover_test.go — Discovery coverage


## 2026-06-17

Implemented Copilot events.jsonl conversion with turns, tools, permissions, shutdown token metrics, malformed-line annotations, and synthetic tests.

### Related Files

- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/pkg/adapters/copilot/convert.go — Copilot event parsing and minitrace conversion
- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/pkg/adapters/copilot/convert_test.go — Synthetic conversion


## 2026-06-17

Added Copilot discover/convert CLI commands, registered them, and smoke-tested against the local Copilot session in dry-run mode.

### Related Files

- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/cmd/go-minitrace/cmds/convert/copilot.go — New convert copilot command
- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/cmd/go-minitrace/cmds/convert/root.go — Registers convert copilot
- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/cmd/go-minitrace/cmds/discover/copilot.go — New discover copilot command
- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/cmd/go-minitrace/cmds/discover/root.go — Registers discover copilot


## 2026-06-17

Completed validation: local non-dry conversion wrote valid JSON archive, full go test ./... passed, and docmgr doctor passed.

### Related Files

- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/cmd/go-minitrace/cmds/convert/copilot.go — Validated non-dry archive writing path
- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/pkg/adapters/copilot/convert.go — Validated through local conversion and full test suite


## 2026-06-17

Wired Copilot JSONL into minitracedb/minitracejs AutoDetect/Convert and verified xgoja importer verbs against the real local Copilot events.jsonl.

### Related Files

- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/pkg/minitracedb/convert.go — Auto-detects copilot-jsonl and calls the Copilot adapter
- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/pkg/minitracedb/convert_test.go — Autoconvert coverage for Copilot JSONL
- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/pkg/minitracejs/import_builder_test.go — ImportBuilder Convert coverage for Copilot JSONL
- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/pkg/minitracejs/provider/provider_test.go — xgoja importer verb coverage for Copilot JSONL


## 2026-06-17

Addressed PR #18 code review: carry queued permission metadata into later tool starts and use NEW operation type for creation tools.

### Related Files

- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/pkg/adapters/copilot/convert.go — Review fixes for permission metadata and operation vocabulary
- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/pkg/adapters/copilot/convert_test.go — Regression tests for permission-before-tool and NEW classification


## 2026-06-17

Fixed Copilot tool ordering by associating tool starts through parent-event chains instead of globally reused turn IDs; added regression coverage.

### Related Files

- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/pkg/adapters/copilot/convert.go — Parent-chain tool-to-turn association
- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/pkg/adapters/copilot/convert_test.go — Regression test for reused Copilot turn IDs


## 2026-06-17

Attached Copilot permission events to emitting turns via the parent-event chain and added regression coverage.

### Related Files

- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/pkg/adapters/copilot/convert.go — Permission event turn-index propagation
- /home/manuel/workspaces/2026-06-17/minitrace-copilot-cli/go-minitrace/pkg/adapters/copilot/convert_test.go — Permission event turn-index regression test

