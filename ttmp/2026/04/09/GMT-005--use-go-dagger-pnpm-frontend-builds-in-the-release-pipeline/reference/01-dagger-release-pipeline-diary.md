---
Title: Dagger release-pipeline diary
Ticket: GMT-005
Status: active
Topics:
    - backend
    - frontend
    - configuration
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: .github/workflows/release.yaml
      Note: Step 1 workflow inventory
    - Path: .goreleaser.yaml
      Note: Step 1 before-hook inventory
    - Path: Makefile
      Note: Step 1 release-path inventory
    - Path: web/package.json
      Note: Step 1 pnpm migration input
ExternalSources: []
Summary: Chronological diary for migrating go-minitrace release frontend builds to Go Dagger + pnpm.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Capture the step-by-step work required to move go-minitrace's release/frontend build path from ad hoc npm Makefile commands to a Go Dagger command that runs pnpm inside the release pipeline.

## Step 1: Map the current build and release plumbing before changing it

Before adding Dagger, I verified the exact places where the current npm-based frontend build is baked into the repository. That avoided guessing about whether the release workflow, GoReleaser, or the Makefile was the real source of truth.

The code path is simple but spread across multiple files: the Makefile owns the actual frontend recipe, GoReleaser calls that target in a before-hook, and the GitHub release jobs currently know nothing about Dagger because they only need Go plus GoReleaser. That means the migration can stay focused on replacing the frontend build implementation and provisioning Dagger in the release jobs rather than redesigning the whole release workflow.

### Prompt Context

**User prompt (verbatim):** "Use go dagger to build pnpm build this in the release pipeline (look up skills). Clean up our makefile after that."

**Assistant interpretation:** Replace the current npm-based frontend build path with a Go Dagger command that performs a pnpm build, wire that into the release path, and simplify the Makefile accordingly.

**Inferred user intent:** Make frontend release builds more reproducible and less host-dependent while reducing build logic duplication in the Makefile.

### What I did
- Read the `go-go-golems-project-setup` skill to orient on the repo's release-plumbing conventions.
- Inspected `Makefile`, `.goreleaser.yaml`, and `.github/workflows/release.yaml`.
- Confirmed that `web/package.json` scripts are compatible with pnpm.
- Generated a first `pnpm-lock.yaml` from the existing npm lock so the migration has a reproducible pnpm baseline.
- Wrote the ticket docs, implementation guide, and task plan.

### Why
- The release path currently depends on `npm ci && npm run build`; the user wants Go Dagger + pnpm instead.
- Dagger changes need to be justified against the actual release path, not an assumed one.
- A committed pnpm lockfile is a prerequisite for `pnpm install --frozen-lockfile`.

### What worked
- The current release flow is already centralized enough that the migration can be done without touching unrelated CI jobs.
- `pnpm import` succeeded from the existing `package-lock.json`, so the repo can move to pnpm without reconstructing dependency resolution by hand.
- The web package's scripts already use generic names (`build`, `dev`), making the package-manager swap straightforward.

### What didn't work
- N/A so far.

### What I learned
- The migration is mostly plumbing, not frontend code: one new builder command, one lockfile migration, one Makefile cleanup, and one workflow install step should cover most of the work.

### What was tricky to build
- The main planning challenge was avoiding a half-migration. If the repo switches to pnpm in Dagger but leaves npm assumptions in the Makefile or release jobs, the end state becomes harder to reason about than the starting point. The task plan therefore treats pnpm metadata, Makefile cleanup, and workflow installation as one cohesive change.

### What warrants a second pair of eyes
- The exact release-path handoff between `.goreleaser.yaml` and `.github/workflows/release.yaml` once Dagger is introduced.
- Whether the repo should keep or delete `package-lock.json` after the pnpm lockfile is committed.

### What should be done in the future
- After the migration lands, consider whether local dev docs should mention pnpm explicitly instead of npm.

### Code review instructions
- Start with `Makefile`, then `.goreleaser.yaml`, then `.github/workflows/release.yaml`.
- Confirm the docs explain why the Dagger builder replaces the old npm recipe rather than layering on top of it.

### Technical details
- Commands run:
  - `command -v dagger`
  - `dagger version`
  - `go list -m -versions dagger.io/dagger`
  - `cd go-minitrace/web && pnpm import`
  - `cd go-minitrace/web && pnpm install --frozen-lockfile && pnpm run build`
