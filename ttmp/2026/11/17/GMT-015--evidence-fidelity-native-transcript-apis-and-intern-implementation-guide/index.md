---
Title: Evidence fidelity native transcript APIs and intern implementation guide
Ticket: GMT-015
Status: active
Topics: [analysis, minitrace]
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/go-minitrace/cmds/query/run.go
      Note: CLI evidence and cache controls
    - Path: repo://pkg/minitracecmd/catalog.go
      Note: Catalog diagnostics and isolation
    - Path: repo://pkg/minitracedb/query.go
      Note: Loss diagnostics and strict query behavior
    - Path: repo://pkg/minitracejs/typescript.go
      Note: Public JavaScript declaration contracts
    - Path: repo://pkg/nativeevidence/inspect.go
      Note: Native request and compaction inspection
    - Path: repo://pkg/nativeevidence/reader.go
      Note: Bounded streaming and exact source coordinates
ExternalSources: []
Summary: ""
LastUpdated: 2026-11-17T14:01:35.276716-05:00
WhatFor: ""
WhenToUse: ""
---

# Evidence fidelity and native transcript APIs

Implementation and intern onboarding for [upstream issue 34](https://github.com/go-go-golems/go-minitrace/issues/34). Branch `feat/evidence-fidelity`, based on `19d18b9`.

- [Intern architecture and implementation guide](design-doc/01-trustworthy-transcript-analysis-architecture-implementation-and-intern-guide.md)
- [Detailed diary](reference/01-implementation-diary.md)
- [Query fidelity](reference/02-query-fidelity-notes.md)
- [Provenance](reference/03-provenance-notes.md)
- [Native evidence](reference/04-native-evidence-notes.md)

Reproduction scripts are in `scripts/`; validation receipts are in `sources/`. The guide explicitly separates delivered fixes from deferred scanner and replay work. Earlier configuration test failures were reproduced and fixed in the platform test fixtures. The final full suite passes.

Planned delivery: `/ai/2026/11/17/GMT-015`, document `GMT 015 Evidence Fidelity and Intern Guide`. Upload success will be reported in the final session receipt; all commits precede upload at the user’s request.

## Persona testing follow-up

The user requested actual Pi/Lunaroute role-based runs after the original implementation delivery. The fixed matrix uses nine historical Claude/Codex/Pi transcripts, synthetic edge cases, two models and two requested thinking levels.

- [Consolidated findings and improvement plan](reference/10-persona-testing-findings-and-improvement-plan.md)
- [Testing guide and question bank](design-doc/02-persona-based-agent-testing-guide.md)
- [Corpus selection](reference/06-persona-corpus-selection.md)
- [Independent results and adjudications](reference/07-persona-test-results.md)
- [Observed API friction and command replays](reference/08-persona-friction-review.md)
- [Analysis of the generated Pi sessions](reference/09-persona-session-self-analysis.md)
- [Harness reproduction instructions](scripts/persona-testing/00-reproduction.md)

The machine-readable result file owns the current completion count. Intermediate snapshots are explicitly partial; correct answer fields are distinct from complete task fulfillment.

Final persona result: 32/32 runs with answers and reports, zero timeouts, 1,671 tool calls. All eight command replays and all 32 session-normalization checks are complete. See the consolidated review for confirmed bugs, scope adjudications and workspace-instruction violations. Delivery target: `/ai/2026/11/17/GMT-015`, document `GMT 015 Persona Testing Review and Guide`.

## Review-driven implementation

The follow-up design is [design03](design-doc/03-persona-review-followups-result-contracts-discoverability-and-enforced-test-isolation.md). Product changes are committed in acac40b; the separate enforced v2 harness is a709fe7. See [v2 reproduction](scripts/persona-testing-v2/00-reproduction.md) for frozen-binary execution, task contracts and isolation limits. Historical v1 results above describe the original implementation and are not revised by this new experiment.


Final v2 implementation results: [current review](reference/13-review-followup-implementation-and-v2-validation-results.md) and [independent transcript review](reference/14-v2-persona-transcript-review.md). The 32-slot matrix produced 31 final answers, 818 passing executed checks and 14 unassessed checks after one timeout. Separate zero-tool probes pass 4/7 finalized generated commands. Post-cohort host-output read hardening is locally tested and explicitly distinguished from the frozen experiment. The pedagogical report is published in playback-vault, commit ccac298.


## Composable API replacement design

[Design04: composable JavaScript query grammar and Go architecture](design-doc/04-composable-javascript-query-grammar-architecture-api-and-intern-implementation-guide.md) specifies the next clean-cut API. It deliberately removes legacy exports and aliases, replaces domain preset switches with a small typed grammar, and describes Go package restructuring, callback phases, local execution, scoped fragments and staged implementation. This is proposed work; the persona-tested implementation above remains the current code.

Design04 now includes a before/after review of all 28 maintained JS scripts and an index of all 267 discovered JS/MJS/CJS files. The local-execution revision retains fragments and ordinary callbacks; it excludes plan transport and function registries. Reproduce the inventory with `scripts/04-inventory-js-api-migration.py`; hashes and lexical method evidence are in `sources/js-api-migration-inventory.json`.

## Local JS API implementation

Design04 is implemented on this branch. The local API supports fluent queries, tagged SQL, named parameters, scoped fragments and terminal mapping through a single execution method. Published docs, examples and the installed transcript analysis skill use the new API. See [acceptance and persona replays](reference/15-local-js-api-acceptance-and-persona-replay-results.md) for the verified checks, the frozen original matrix and the separate retest.

For the next authoring cohort, see [open-ended persona briefs](reference/16-open-ended-persona-briefs-for-real-transcript-analysis.md).
