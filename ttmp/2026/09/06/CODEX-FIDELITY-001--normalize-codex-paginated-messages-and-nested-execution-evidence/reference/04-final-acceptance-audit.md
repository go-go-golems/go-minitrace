---
Title: Final acceptance audit
Ticket: CODEX-FIDELITY-001
Status: active
Topics:
    - codex
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/go-minitrace/cmds/serve/unassociated_test.go
      Note: No tool loss or fabricated turn in API
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/scripts/11-final-smoke.sh
      Note: Consolidated final acceptance runner
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/various/final/api-acceptance.json
      Note: Every record retained across seven API sessions
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/various/final/semantic-acceptance.json
      Note: Independent native counts hashes and receipt acceptance
ExternalSources: []
Summary: ""
LastUpdated: 2026-09-06T19:22:55.004145292-04:00
WhatFor: ""
WhenToUse: ""
---


# Final acceptance audit

CODEX-FIDELITY-001 is implemented. P1–P4 implementation receipts and P5 final evidence are retained in this ticket. The consolidated smoke found genuine gaps, which were fixed and revalidated rather than waived.

## Requirement-to-evidence map

| Requirement | Implementation and concrete acceptance evidence |
|---|---|
| Paginated messages, identity mirrors, repeats, explicit/null emitters | messages.go/messages_test.go; various/p3/final-message-coverage.json; various/final/message-audit.log; final-fidelity-sql.json has zero orphan links |
| Native execution identity/lifecycle, duplicate notifications, same-command distinct executions | executions.go, execution_identity_test.go, missing_identity_test.go; final execution-audit.log accounts for all 2,296 executions and 155 failures |
| Typed outputs, early/replayed results, conflicts, truncation | outputs_test.go, output_replay_test.go, legacy_outcomes_test.go; synthetic-oracle.log; proto-roundtrip.log; provenance_test.go |
| Nullable unknown/pending/cancelled/succeeded/failed outcomes across consumers | Go/database/API outcome tests; 78 browser tests; production UI screenshot; final SQL/independent native exit audit |
| Provenance and uncertain parentage preserved through API/UI | strictProtoStruct; provenance_test.go; unassociated_test.go; api-acceptance.json proves every record survives API normalization; UI has a separate unassociated ledger, not invented turns |
| Structural multi-file targets, moves and independent target outcomes | files_test.go; file_content_conflict_test.go; database file_evidence_test.go; file-audit.log matches 472 native events and 995 targets |
| No execution/write inference from opaque JS, false branches, quoted histories, heredocs or dynamic shell targets | paginated-fidelity.jsonl quoted-history and false-branch records; files_test.go; missing_identity_test.go; history.json and semantic-acceptance.json |
| Counting/materialization/presets/history and non-Codex preservation | activity_counts_test.go; SQLite schema v5; private-sql.log; file-activity.json, ticket-timeline.json, context-window.json, non-codex-history.json; structural history acceptance script 10 |
| Native sources unchanged, identity/receipt/archive checks | Six SHA-256 hashes equal original native inventory; semantic-acceptance.json; receipt-summary.json: six inputs, six outputs, no failures, complete; validator.log |
| Full tests/lint/build/generation/local CI checks | make-all.log; final-fix-commit.log (full hook tests/lint after the API correction); race.log; logcopter.log and logcopter-after-fix.log; production-build.log (actual go generate/Dagger plus go build) |
| Frontend and actual browser checks | web-build-after-fix.log, ledger-layout-build.log; web-lint-after-fix.log; browser-after-fix.log: 78/78; production-ui-acceptance.json; final-ui.png; earlier p3/p4 outcome/target screenshots |
| Documentation and skill caveats | pkg/adapters/codex/README.md; pkg/doc/adapter-reference.md; installed transcript-analysis SKILL.md and references/queries.md, review snapshots in various/final; help.log, schema-help.log |
| Focused commits and chronological diary | git log; reference/03 Steps 1–14; latest implementation d8a19a7, b344f44, 78b8100; diary records failures, fixes, commands, budget renewals and review instructions |
| Physical overall/phase slips | various/slips 00 overall, 01/02 P1, 03/04 P2, 05/06 P3, 07/08 P4, 09/10 P5; every required print has an actual positive receipt |
| Final docmgr/diff/hygiene | doctor.log and final doctor output; git diff --check; intentionally staged docs/evidence; native archives remain outside Git under /tmp |

## Final observed totals

- Six unchanged baseline sources, including three legacy controls.
- 2,296 native command executions, including 155 nonzero exits.
- 472 native FileChange events, 995 confirmed native targets including rename destinations.
- Zero orphan non-null tool links; zero malformed map-style output strings.
- Record-kind counters partition normalized records; file targets do not become extra tool records.
- Seven API sessions (six local plus the synthetic fixture) retain every normalized tool record, including those without a proven emitting message.
- 78 browser tests pass after fixing missing story providers. Production UI has no page errors and displays unassociated records separately.

## Failures found and resolved in the final sweep

1. Ticket-timeline SQL had an ambiguous tool_name after its files join. Qualified the column and reran the consumer smoke.
2. Seven browser stories lacked Redux/Router providers. Wired real decorators; all 78 tests pass.
3. API blocks omitted tools with no emitting message. Added an explicit unassociated_tool_calls detail ledger, switched the viewer to complete detail data, and verified all records against archive counts.
4. A JSX closing brace was missing in the new ledger section. The type build caught it; fixed and rebuilt. The ledger is bounded and independently scrollable so large sessions do not compress transcript content.

The full smoke was not repeatedly rerun after every change. Only affected checks/builds were rerun after demonstrated failures; repository commit hooks retained their normal full test/lint behavior.

## Explicit limitations, not waived failures

- The validator reports six informational source-unavailable messages because archive paths contain literal `~`. Independent expanded-path reads and SHA-256 comparisons prove that the sources exist and are unchanged. Validator exit success is not used as semantic proof.
- Shell file inference deliberately supports a narrow bounded grammar. Unsupported forms produce diagnostics; they do not become inferred writes. Targets are syntactic attempts unless native FileChange evidence confirms effects.
- `resolved` means lexical absolute-path resolution, not filesystem/symlink canonicalization. Shared turn/time is not parent proof.
- File-touch counts count evidence rows, not unique paths or authorship. Command-text matches remain candidates; repository attribution requires external verification.
- Thirteen existing frontend lint warnings and bundle-size warnings remain; there are no lint errors. No requirement asked to eliminate all pre-existing warnings.
- Installed skill files are outside this Git repository. Their updated text is retained in committed snapshots; no private transcript bodies are included.

## Review and reproduction

Run scripts/11-final-smoke.sh from the repository root with the saved source list available. Its original browser failure is retained as historical evidence; browser-after-fix.log records the passing correction. Native archives are read-only inputs; output archives and API private bodies live under /tmp, not in Git. The production server/browser sessions used for verification were stopped.
