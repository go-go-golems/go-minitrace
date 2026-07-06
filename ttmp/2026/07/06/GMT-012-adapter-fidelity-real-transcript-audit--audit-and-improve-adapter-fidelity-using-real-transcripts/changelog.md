# Changelog

## 2026-07-06

- Initial workspace created


## 2026-07-06

Added intern implementation guide and first evidence pass scripts for real-transcript adapter fidelity audit; inventoried local sources, converted sampled Pi/Codex/Claude Code sessions, and queried initial fidelity metrics.

### Related Files

- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts/design-doc/02-adapter-fidelity-intern-implementation-guide.md — Intern-facing adapter fidelity guide
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts/scripts/01-inventory-source-shapes.py — Inventory script
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts/scripts/03-query-converted-fidelity.sh — Fidelity query script


## 2026-07-06

Uploaded GMT-012 adapter fidelity guide bundle to reMarkable at /ai/2026/07/06/GMT-012/GMT-012 Adapter Fidelity Intern Guide.pdf.

### Related Files

- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts/design-doc/02-adapter-fidelity-intern-implementation-guide.md — Uploaded intern guide


## 2026-07-06

Added source-vs-archive coverage profiler, investigation guide, and missing functionality report; created GitHub issue #23 for JS-first documentation cleanup.

### Related Files

- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts/analysis/01-source-vs-archive-coverage-investigation-guide.md — Coverage workflow guide
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts/analysis/02-missing-adapter-functionality-report.md — Missing functionality report
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts/scripts/04-profile-source-vs-archive-coverage.py — Coverage profiler


## 2026-07-06

Uploaded GMT-012 coverage investigation and missing functionality report bundle to reMarkable at /ai/2026/07/06/GMT-012.

### Related Files

- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts/analysis/02-missing-adapter-functionality-report.md — Uploaded missing functionality report


## 2026-07-06

Implemented lineage preservation for Pi parentSession, Codex parent_thread_id, and Claude Code subagent parent sessions; added regression tests and updated adapter reference.

### Related Files

- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/adapters/claudecode/convert.go — Promotes Claude subagent parent lineage
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/adapters/codex/convert.go — Promotes Codex parent_thread_id lineage
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/adapters/pi/convert.go — Promotes Pi parentSession lineage
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/doc/adapter-reference.md — Documents normalized predecessor lineage


## 2026-07-06

Added Codex legacy rollout JSONL conversion support; all 12 sampled Codex files now convert, and adapter docs/profile report were updated.

### Related Files

- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/adapters/codex/convert.go — Legacy rollout JSONL parser
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/adapters/codex/convert_test.go — Legacy rollout regression test
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/doc/adapter-reference.md — Documents legacy Codex support


## 2026-07-06

Validated GMT-012 lineage and legacy Codex changes with full Go tests and docmgr doctor; committed checkpoints 6909205, 4c89999, and 0de7545.

### Related Files

- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/adapters/codex/convert.go — Validated in full test suite
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts/reference/01-diary.md — Validation record


## 2026-07-06

Mapped Pi image content blocks to bounded attachments with turn/tool-call links and sanitized raw metadata; reran Pi sampled conversion and coverage profile.

### Related Files

- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/adapters/pi/convert.go — Pi image attachment mapping
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/adapters/pi/convert_test.go — Image attachment regression tests
- /home/manuel/workspaces/2026-07-05/improve-docmgr/go-minitrace-pr/pkg/doc/adapter-reference.md — Documents Pi image attachments

