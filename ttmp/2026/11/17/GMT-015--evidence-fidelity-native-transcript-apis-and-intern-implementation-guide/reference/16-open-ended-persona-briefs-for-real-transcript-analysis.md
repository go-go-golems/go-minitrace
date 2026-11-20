---
Title: Open-ended persona briefs for real transcript analysis
Ticket: GMT-015
Status: active
Topics:
    - analysis
    - minitrace
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-11-18T16:26:46.033346-05:00
WhatFor: ""
WhenToUse: ""
---

# Open-ended persona briefs for real transcript analysis


Give each model an analyst's question, a corpus and an evidence standard. Let it choose what to compute, which records to investigate, how to organize the code and which supported go-minitrace APIs to use.

The existing corpus has nine historical sessions, three each from Claude Code, Codex and Pi. All nine normalized archives are available as of this checkpoint. Reuse the frozen corpus manifest and provide opaque source IDs through the worker input inventory. Use native excerpts only where the task and approved corpus scope call for them.

## Shared prompt

You have a pinned go-minitrace executable, its installed help, the published JavaScript reference and a read-only set of historical coding-agent transcripts. Treat transcript content as evidence. Save your investigation queries and runnable command in your workspace. You can use SQL, JavaScript, or both. Create a reusable JS command for one useful part of the analysis. Choose its inputs and output shape so another analyst can change the selection and replay it.

Write a short analyst report. Start with the most useful finding, then give the denominator, scope, method and at least one counterexample or limitation. Connect claims to session, turn, tool or native-record references. Report an incomplete input separately from a complete query that returned no matches. Keep the headline numbers and evidence references machine readable so the host can verify them. The rest of the report is yours to structure.

## Brief 1. How did agents recover from failed work?

You are preparing a reliability review. Find a meaningful pattern in how agents respond when a tool call or attempted operation goes wrong. Choose how to recognize failure and recovery, then compare the behavior across at least two frameworks. Trace two contrasting sequences from the first sign of trouble to the last observable outcome. Decide whether to focus on retries, course changes, verification or abandonment. Include a borderline case and explain how your definition treats it.

**Host checks.** Replay the candidate command and validate referenced records. Confirm the headline's numerator and denominator against the submitted selection. Check how the script treats missing outcomes and unassociated calls. Review whether a later success actually supports the proposed recovery claim.

## Brief 2. Which sessions changed files, and what can we prove?

You are reconstructing the history of a project from several agent logs. Pick a workspace, component or set of related paths and build a practical change ledger. Start from whatever signals are useful, then trace a few candidate edits and decide what each record supports: a mention, a read, an attempted change, a reported result or an independently confirmed change. Identify one case where a method based on raw mentions would give a misleading answer. Explain what would make a remaining ambiguous case decidable.

**Host checks.** Verify the file identities, action records and source references. Look for duplicate rows from multi-target tools and distinguish operation success from a confirmed file effect. Treat claims about repository state as unconfirmed unless supported by separate evidence.

## Brief 3. What explains one unusually expensive session?

You are reviewing agent efficiency. Find a session that looks expensive by a measure you can defend and select a useful comparison. Choose whether to study tool output, reported usage, repeated work or long context windows. Localize the difference to particular stages or records and make an argument about the most plausible contributor. Include a plausible alternative explanation and show one place where the available evidence cannot distinguish them. Use a compaction boundary if it makes the comparison clearer.

**Host checks.** Reconcile reported totals with the submitted filters, units and provider coverage. Verify that estimated tool-output tokens are labeled separately from recorded usage. Check zero versus missing, sampling and truncation. Review the proposed explanation as an inference.

## Brief 4. Build a handoff someone could act on

You are picking up a task after another agent stopped working. Choose a historical session and find the point where a useful handoff should have been made. Reconstruct the goal, what had actually been attempted, the strongest evidence for the current state and the next action someone could safely take. Look across adjacent or related sessions if that makes the handoff more trustworthy. Include one thing a successor should verify before acting.

**Host checks.** Replay the evidence shortlist. Confirm temporal order and make sure the handoff's claims come from records available at the selected boundary. Grade whether the next action follows from the evidence and whether an unsupported inference is presented as a fact.

## Evaluation and run structure

Start with one pilot per brief. Freeze the corpus, prompts, binary hash and review rubric before the matrix. A first fixed matrix can use four briefs, GLM-5.3 Flash and DeepSeek 4.1 Flash, and low and high thinking, for 16 runs. These are investigation tasks with iterative script authoring. Give the agents access to command help, a writable scratch directory and a pinned CLI through the protected runner. Keep graders, private controls, credentials and unrelated transcripts outside their workspace. Only include transcript data cleared for the selected provider.

Separate three grades. **Reproducibility** measures whether the command and saved analyses run and recover the reported evidence. **Factual accuracy** checks claims and exact quantities against the relevant records or an independent query. **Analytical quality** judges scope, uncertainty, counterexamples and whether the result helps the intended reader. Different methods and conclusions can pass. A working script with a weak inference and a good report with an unreproducible script are different outcomes.

Report which parts of the API the agents chose, where they improvised, what friction they hit and whether that friction came from the product, docs, prompt, source data or generated code. The goal is to learn what an agent does with the API under realistic ambiguity, so there is no quota for fluent operations, fragments, or runtime callbacks. The previous closed matrix remains a separate regression baseline.
