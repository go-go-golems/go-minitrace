---
Title: Make minitrace loadable in hand-built xgoja hosts
Ticket: MINITRACE-XGOJA-HOST-001
Status: active
Topics:
    - minitrace
    - xgoja
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - "https://github.com/go-go-golems/go-minitrace/issues/20"
Summary: "Ticket workspace for analyzing and implementing require(\"minitrace\") support in hand-built go-go-goja/xgoja hosts."
LastUpdated: 2026-06-22T17:15:00-04:00
WhatFor: "Use this ticket to implement GitHub issue #20 and understand the related minitracejs, go-go-goja module registry, and xgoja provider-example architecture."
WhenToUse: "Before changing pkg/minitracejs module registration or refreshing examples/xgoja/minitrace-command-provider."
---

# Make minitrace loadable in hand-built xgoja hosts

## Overview

This ticket turns GitHub issue #20 into an implementation-ready design package. The issue asks for `require("minitrace")` to work in a hand-built go-go-goja host alongside linked default modules such as `fs` and `template`, and for the stale xgoja command-provider example to be refreshed.

The analysis concludes that the go-minitrace-owned fix is to add a default-registry `modules.NativeModule` adapter in `pkg/minitracejs` that delegates to the existing `NewLoader`. This preserves the generated xgoja provider path while making the hand-built host path consistent with goja-text's `template` module.

## Key links

- Design guide: [design-doc/01-hand-built-xgoja-host-module-loading-design-and-implementation-guide.md](./design-doc/01-hand-built-xgoja-host-module-loading-design-and-implementation-guide.md)
- Investigation diary: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)
- GitHub issue capture: [sources/01-github-issue-20.md](./sources/01-github-issue-20.md)
- Module-loading probe: [scripts/01-probe-module-loading.sh](./scripts/01-probe-module-loading.sh)
- xgoja example check: [scripts/02-check-xgoja-example.sh](./scripts/02-check-xgoja-example.sh)

## Status

Current status: **active**. The research/design work is complete. Implementation remains future work.

## Topics

- minitrace
- xgoja
- architecture

## Tasks

See [tasks.md](./tasks.md) for completed documentation tasks and future implementation tasks.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- `design-doc/` — Architecture and implementation guidance.
- `reference/` — Investigation diary.
- `sources/` — Captured issue body and command outputs.
- `scripts/` — Reproduction scripts used by the investigation.
