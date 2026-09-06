---
Title: Investigation and delivery diary
Ticket: CODEX-FIDELITY-001
Status: active
Topics:
    - codex
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Source inspection, concrete evidence, design decisions, and document delivery.
LastUpdated: 2026-09-06
WhatFor: Preserve investigation context for implementation.
WhenToUse: Before resuming this ticket.
---

# Investigation and delivery diary

## Step 1: Trace observed conversion loss to source

The user requested a docmgr ticket, detailed intern analysis/design/implementation guide, and reMarkable delivery after seeing a concrete nested Codex exec example. Read docmgr and remarkable-upload skills and docmgr's workflow reference. Created this ticket and two documents through docmgr.

Inspected `parseSessionJSONL`, tool construction, output decoding, command/path classification, shared output schema, SQLite schema, and consumer locations at repository HEAD a6acfcc9130f88fd8a6237494c4664d3acb440bb. The installed dev binary's precise build commit is unknown.

Findings: persisted item_completed executions and response messages are not handled by the current dispatch; custom output arrays are stringified through fmt.Sprint; success defaults true before any output is known. Therefore the earlier conversational claim that success proves outer-wrapper completion was too strong. The guide corrects it.

Design: prefer authoritative structured execution events over parsing JavaScript; preserve orchestration separately, deduplicate native representations, retain uncertain associations, and make nullable/explicit outcome semantics a schema review gate. No transcript code will be evaluated. No production implementation was changed.

A guessed materialization glob returned no matching import files; listing the package identified materialize.go and convert.go. Documentation and delivery are the current scope; adapter implementation remains open.

## Step 2: Author implementation guide

Wrote an intern guide covering native versus normalized identities, pipeline architecture, exact failing example, code navigation, pseudocode, text diagrams, output and file evidence contracts, phase gates, SQL assertions, negative tests, and downstream schema risks. It explicitly distinguishes an attempted shell write from verified per-statement success and excludes unrelated missing-side-session recovery from this fix.

At this checkpoint, validation and delivery were pending. Synthetic fixture implementation remains part of the open implementation tasks.

## Step 3: Resolve TeX prerequisites and deliver

Initial rendering failed because DejaVu fonts were missing; the fallback then exposed missing TeX PATH tools and enumitem.sty. The user upgraded BasicTeX and installed the recommended dependencies. Verified XeTeX/TeX Live 2026, tlmgr revision 79639, enumitem.sty under the 2026 installation, and extractbb availability.

With `/Library/TeX/texbin` prepended to PATH, standard `remarquee upload md --pdf-only` succeeded without the fallback wrapper or font overrides. Rendered all eight pages using Ghostscript and inspected both contact sheets. The PDF is readable; some diagrams/pseudocode continue across page boundaries. Uploaded using the same default rendering settings.

Receipt: `various/remarkable-upload-tex2026.log` reports `OK: uploaded CODEX-FIDELITY-001_Intern_Guide.pdf -> /ai/2026/09/06/CODEX-FIDELITY-001`. Local PDF is in `various/pdf/`; review images are in `various/pdf-review/`. No remote listing or overwrite was needed. Documentation delivery is complete; adapter implementation remains open.
