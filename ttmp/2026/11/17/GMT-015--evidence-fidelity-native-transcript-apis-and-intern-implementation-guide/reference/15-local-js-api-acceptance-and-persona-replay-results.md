---
Title: Local JS API acceptance and persona replay results
Ticket: GMT-015
Status: active
Topics:
    - analysis
    - minitrace
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/persona-local-api/run.py
      Note: Pinned synthetic corpus and protected host replay
    - Path: repo://ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/sources/local-api-persona-matrix.json
      Note: Compact original and follow-up persona grades
ExternalSources: []
Summary: ""
LastUpdated: 2026-11-18T16:18:18.60142-05:00
WhatFor: ""
WhenToUse: ""
---

# Local JS API acceptance and persona replay results


The clean-cut JavaScript API builds and executes local plans with fluent queries, tagged SQL, parameters, scoped fragments and terminal mapping. Deterministic package and CLI fixtures cover the behavior. A separate persona matrix tests whether a model can write a working command using the published reference.

## Verified implementation

| Boundary | Evidence |
| --- | --- |
| Immutable branches, callback timing, scope and stable sort | `TestBranchAndOrderValidation`, `TestCompiledQueryAgainstSQLite`, `TestRelationalBindingThroughRequire` |
| Actual per-session counts including zero, file and token provenance | `TestSessionToolCountsGoja`, `TestAnalysisTokenOriginsAndSourceProjection`, `TestGojaFidelityViews` |
| Scalar binding, parameters, fragments and declared SQL | `TestTaggedSQLBinding`, `TestParametersAndFragments`, `TestSQLDeclarations` |
| Runtime mapping, limits, cancellation and error envelopes | `TestMapRows`, `TestJSQueryFidelityBuildersAndExecute`, `TestJSNegativeResultContractsAndDiagnostics` |
| Commands and public entry points | Embedded and docmetrics CLI fixtures, JS and mixed showcases, both xgoja smoke targets |
| Repository gates | Unfiltered `go test ./...`, build, vet, repository lint and complete embedded help passed at the implementation checkpoint |
| Declaration contracts | Strict TypeScript 5.9.3 checked the direct and generated provider surfaces |
| Maintained guidance | Repository and installed transcript analysis skill match. Current examples and help use the single execution surface |

## Persona matrix

The pinned binary was built at `a3b805e`. The corpus has three synthetic sessions. One has no tools. The two tasks cover fluent actual counts with fragments, and declared tagged SQL with named bindings and runtime mapping. Pi saw the published API reference and a task. It had no tools. Each generated command replayed under fixed host argv, with exact row comparison against its oracle. The replay allowed no network and was checked for protected reads and writes. The run harness and compact receipts are kept in the ticket.

The original pilot received two scripts from GLM at low thinking. The initial sandbox aborted the binary before query startup, including when invoked with `--version`. Both scripts passed when replayed unchanged under the corrected path restrictions. Original failure receipts are separate from the recovery grades.

| Cohort | Passed | Attempted |
| --- | ---: | ---: |
| Pilot, original sandbox | 0 | 2 |
| Pilot, replayed unchanged | 2 | 2 |
| Fixed matrix | 6 | 8 |
| Targeted GLM report retest with revised reference | 2 | 2 |

All eight matrix generations completed. Both models passed inventory at both thinking levels. DeepSeek passed the mapped report at both levels. GLM's report failed at low because it passed a built SQL plan to `mt.from()`. Its high-thinking report tried to call `orderBy()` on the mapped stage. The low output also included the same ordering mistake after its failing `from()` call. Both are caller errors and show where the guide could give a more complete composition example.

The follow-up clarifies that typed SQL already returns a query builder and that sorting precedes mapping. The runtime now reports `E_QUERY_SOURCE` with a specific hint when a builder or plan is passed to `from()`. The original cohort remains frozen. The targeted retest pins commit `d0989b3` and the revised API reference. Both GLM runs placed ordering before mapping, bound the parameter and returned the exact ordered rows. This demonstrates that the revised guidance resolved the two observed cases. It does not establish a general model comparison or substitute for the original matrix.

[Synthetic harness](../scripts/persona-local-api/00-reproduction.md) · [Compact receipts](../sources/local-api-persona-matrix.json) · [Implementation diary](01-implementation-diary.md)
