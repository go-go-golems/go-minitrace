---
Title: Local source validation baseline
Ticket: CODEX-FIDELITY-001
Status: active
Topics:
    - codex
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/scripts/02-local-fidelity.sql
      Note: Normalized fidelity and orphan-link baseline
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/scripts/03-native-inventory.py
      Note: Read-only native event inventory and fingerprints
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/various/local-validation/native-inventory.json
      Note: Measured source counts and anchors
    - Path: repo://ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence/various/local-validation/sql-results.json
      Note: Checkout-built converter results
ExternalSources: []
Summary: ""
LastUpdated: 2026-09-06T16:12:56.48777247-04:00
WhatFor: ""
WhenToUse: ""
---


# Local source validation baseline

## Conclusion

Confirmed on this Linux machine with a binary built from checkout `28cd1c2ad215d1f31d54abd45eb439588c8aae12`. Three local Codex CLI 0.153.3 sessions declare `history_mode: paginated` and reproduce missing messages, discarded structured executions, malformed typed outputs, and orphan tool-turn references. This is independent reproduction, not the guide's original macOS/0.153.4 source.

## Measured baseline

Native counts below are item-completed event counts, not deduplicated conversation counts. CommandExecution IDs are unique within each source. Mirrored response messages must not be added to item-message counts.

| Session ID | Native UserMessage / AgentMessage | Native executions (nonzero exit) | Normalized turns | Tools / exec OTHER | Orphan links | map outputs |
|---|---:|---:|---:|---:|---:|---:|
| 01a06de0-5c5b-74e1-acc0-ed790e41732b | 50 / 188 | 886 (77) | 0 | 629 / 622 | 629 | 622 |
| 01a06de6-34fd-7930-891b-a492ab38d918 | 45 / 254 | 1104 (60) | 0 | 927 / 870 | 927 | 878 |
| 01a06dea-5dc2-7fb3-992f-dad6143c5d34 | 19 / 76 | 306 (18) | 0 | 275 / 272 | 275 | 270 |

All three archives have zero normalized commands and exit codes. Every retained tool has success=true; that does not establish successful subprocess execution. Native execution outcomes are absent as first-class tools. A map-output count is the SQL substring diagnostic, not the number of array-valued native outputs.

Three legacy-mode controls from the same discovery convert to 573, 640, and 178 turns respectively. This strengthens the format-specific diagnosis; it does not establish complete fidelity for legacy sources (one control also has five malformed exec outputs).

## Concrete source anchors

Full native paths, SHA-256 fingerprints, IDs, and anchors are saved in `../various/local-validation/native-inventory.json`. All six sources are under `~/.codex/sessions/2026/09/04/`.

In the agent-forum source (`01a06de0...`), native lines 10 and 11 contain completed user and assistant messages, yet the archive has no turns. Line 15 contains a CommandExecution. Line 73 contains failed execution `exec-98014f6b-4215-46f0-9ca2-5db59e135c53` with exit_code=1; its ID is absent from normalized tools. Line 17 is array-valued output for `call_RkK6nyth18RIwKlNfYRVjEXR`; the same archive call contains Go `map[text:... type:input_text]` formatting instead of correctly decoded content blocks. No JavaScript or historical commands were evaluated.

The gatemate and plot-editor sources independently contain failed execution events at lines 74 and 76, respectively. Their IDs and outcomes are saved in the inventory. Do not infer wrapper-child parentage merely from nearby lines.

## Reproduction and artifacts

Checked installed discover/convert/query help and schema documentation first. Discovery selected sessions **started** since 2026-09-01; this is a reproducible shortlist, not a complete activity census. Converted all six discovered sources to include controls.

```sh
go build -o /tmp/codex-fidelity-001-local/go-minitrace ./cmd/go-minitrace
/tmp/codex-fidelity-001-local/go-minitrace convert codex \
  --source-list /tmp/codex-fidelity-001-local/sources.txt \
  --output-dir /tmp/codex-fidelity-001-local/archives \
  --run-record /tmp/codex-fidelity-001-local/conversion.json
```

Run `scripts/03-native-inventory.py` against the source list and `query run --sql-file scripts/02-local-fidelity.sql --output json` against the fresh archive glob. Paths above are repository-relative to this ticket except the explicitly absolute temporary paths.

Tracked evidence in `various/local-validation/`: discovery, source list, native inventory with fingerprints, conversion receipt, SQL results, and validation output. Full converted transcripts and binary remain outside the repository at `/tmp/codex-fidelity-001-local/`; native sources remain untouched. Recreate temporary archives when needed.

Six distinct native IDs produced six archives without collision errors. Archive validation exited zero but emitted six informational `source-unavailable` diagnostics because it attempted to open literal `~/.codex/...` paths. Sources were independently read and fingerprinted successfully. This is the known tilde-expansion issue, not a fidelity pass; the validator did not flag missing messages or orphan links.

## Next implementation step

Create small synthetic fixtures for these shapes, then assert the failures above. Keep the private sources out of regression fixtures. Existing implementation tasks remain open; this checkpoint changes no adapter behavior.
