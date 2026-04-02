---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: ttmp/2026/04/01/WESEN-OS-001--inspect-wesen-os-deployment-via-go-minitrace-codex-session-analysis/scripts/06-wesen-os-tool-breakdown.sql
      Note: Tool call breakdown for 3 wesen-os sessions
    - Path: ttmp/2026/04/01/WESEN-OS-001--inspect-wesen-os-deployment-via-go-minitrace-codex-session-analysis/scripts/07-assistant-summary-turns.sql
      Note: Assistant turn extractor for session deep-reads
    - Path: ttmp/2026/04/01/WESEN-OS-001--inspect-wesen-os-deployment-via-go-minitrace-codex-session-analysis/scripts/08-all-workdirs.sql
      Note: All working directories and session volumes
    - Path: ttmp/2026/04/01/WESEN-OS-001--inspect-wesen-os-deployment-via-go-minitrace-codex-session-analysis/scripts/09-deploy-timeline.sql
      Note: Deploy+wesen-os+k3s chronological timeline
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# wesen-os Deployment Summary (Last 2 Weeks)

## Executive Summary

Between 2026-03-18 and 2026-04-01, wesen-os underwent two major parallel work streams: (1) a **profile architecture migration** that cleaned up the legacy `geppetto/pkg/profiles` dependency and introduced per-app profile registry chains, and (2) a large **npm package publishing + federation release pipeline** effort aimed at turning the `@go-go-golems/*` / `@hypercard/*` packages into properly published, versioned, and remotely distributable artifacts hosted on Hetzner object storage. A third related stream established a **K3s cluster on Hetzner** as the deployment target that the federation remote URL points to.

All data sourced from Codex session history (3 primary wesen-os sessions + 1 K3s infrastructure session), converted via `go-minitrace` from `~/.codex/sessions/`.

---

## Sessions Analyzed

| Session ID | Date | Hours | Turns | Tools | Topic |
|---|---|---|---|---|---|
| `019d174c` | 2026-03-22 | 24.7h | 315 | 1015 | wesen-os profile migration |
| `019d376d` | 2026-03-28→29 | 87.5h | 1467 | 3307 | NPM publish + federation |
| `019d4a35` | 2026-04-01 | 1.3h | 121 | 300 | SQLITE-FED-001 handoff continuation |
| `019d2f26` | 2026-03-27 | 58.5h | 1807 | 4440 | Hetzner K3s cluster bring-up |

---

## Stream 1: Profile Architecture Migration (019d174c)

### What was done

The `wesen-os` OS chat stack (`go-go-os-chat`, `go-go-app-inventory`, and the main launcher) was migrated away from the legacy `geppetto/pkg/profiles` package. That package used in-memory mixed runtime profiles built directly in the launcher; the new model uses Pinocchio's bootstrap config loading and an external per-app profile registry chain.

**Specific changes:**
- `wesen-os` launcher now loads Pinocchio config and profile files (not wesen-os-owned config).
- Per-app profile registry chains introduced: each app/chat endpoint (`/chat`, `/inventory/chat`, etc.) can stack app-specific profiles on top of the global registry instead of sharing a single flat namespace.
- `@hypercard/kanban-runtime` source alias added to `go-go-app-inventory` — it was missing from `tsconfig.json` and the Vite config, causing module resolution failures.
- Stale CI workflows moved to `.github/workflows-disabled/` with a README explaining why and how to re-enable.

**PRs merged during this session:**
- `go-go-os-frontend#16` — JS runtime manager
- `go-go-os-backend#5` — VM metadata generator

**Final state:**
- Frontend, backend, arc-agi submodule pointers all at `origin/main`.
- `go-go-app-inventory` branch `task/rewrite-runtime` still 37 commits ahead of `origin/main` (open PR pending).
- `wesen-os` superproject still dirty due to updated submodule SHAs not yet committed.

---

## Stream 2: NPM Publish + Federation Release Pipeline (019d376d)

### What was done

This was the longest session in the two-week window (87.5 hours wall clock, 1807 turns in the related K3s session). The overall goal: move `wesen-os` from source-alias-based local workspace imports to *versioned published packages* that can eventually be loaded remotely via module federation.

**Phase 1 — Package renaming:**
- All frontend packages renamed to `@go-go-golems/*` scope (the guide had already decided this; the session discovered that and aligned the task board to it).
- Previously source-aliased `@hypercard/*` packages given proper build/publish configs.

**Phase 2 — GitHub Packages publishing:**
- Publish scripts wired (`workspace-links/go-go-os-frontend/scripts/packages/publish-github-package.mjs`).
- CI workflow `publish-federation-remote.yml` added, publishing federation bundles to Hetzner object storage.
- Canary publish step included for pre-release validation.

**Phase 3 — Federation / Hetzner distribution:**
- Federation bundle target: Hetzner object storage.
- CI secrets needed:
  - `HETZNER_OBJECT_STORAGE_BUCKET`
  - `HETZNER_OBJECT_STORAGE_ENDPOINT`
  - `HETZNER_OBJECT_STORAGE_REGION`
  - `SQLITE_FEDERATION_PUBLIC_BASE_URL`
- The `federation.registry.json` in the K3s gitops repo (`gitops/kustomize/wesen-os/config/federation.registry.json`) is the runtime target file the cluster uses to resolve remote module URLs.

**Phase 4 — SQLite federation (partial, handed off):**
- `go-go-app-sqlite` integration was started but the multi-repo patch complexity was high.
- Rather than push through with inconsistent state, a clean handoff ticket `SQLITE-FED-001` was created with:
  - Full current-state audit of all repos and branches
  - Intern-facing implementation guide
  - Replay script so the next engineer can resume without this conversation
  - Concrete task list

**Tools used (top):**
- `exec_command`: 2578 calls (code changes, build runs, npm scripts)
- `write_stdin`: 664 calls (file writes)
- `update_plan`: 31 calls
- `mcp__codex_apps__github_create_pull_request`: 3 calls
- `mcp__playwright__browser_navigate`: 6 calls (registry/GitHub navigation)

---

## Stream 3: Hetzner K3s Cluster (019d2f26, related)

### What was done

This 58.5-hour session (workdir `~/code/wesen/2026-03-27--hetzner-k3s`) provisioned the K3s cluster on Hetzner that serves as the deployment target for the wesen-os federation. Key artifacts:
- Gitops repo at `~/code/wesen/2026-03-27--hetzner-k3s/`
- `federation.registry.json` at `gitops/kustomize/wesen-os/config/federation.registry.json`
- The cluster is the destination for services being migrated from Coolify (see also `019d4437`, "Deploy hair-booking by moving it from coolify over to k3s").

---

## Stream 4: SQLITE-FED-001 Continuation (019d4a35)

Picked up the handoff ticket on 2026-04-01. Tasks completed:
- Confirmed GitHub Actions secrets for Hetzner object storage are in place.
- `go-go-app-sqlite` PR merged by user.
- Federation publish workflow validated.

---

## Current Deployment State (as of 2026-04-01)

| Component | Status | Location |
|---|---|---|
| Profile migration | ✅ Complete | `wesen-os` submodules at `origin/main` |
| `@go-go-golems` rename | ✅ Complete | All packages renamed |
| GitHub Packages publish | ✅ Complete | CI wired |
| Hetzner K3s cluster | ✅ Running | `2026-03-27--hetzner-k3s` |
| Federation CI workflow | ✅ Merged | `publish-federation-remote.yml` |
| SQLite federation | ⚠️ In progress | `SQLITE-FED-001` handoff ticket |
| `go-go-app-inventory` PR | ⚠️ Open | 37 commits ahead, `task/rewrite-runtime` |
| `wesen-os` superproject pointer | ⚠️ Dirty | Submodule SHAs updated, not committed |

---

## Open Issues / Next Steps

1. **Merge `go-go-app-inventory` PR** (`task/rewrite-runtime`, 37 commits) — the kanban-runtime alias fix and profile registry rewrite are on this branch.
2. **Commit `wesen-os` superproject** with updated submodule SHAs.
3. **Complete `SQLITE-FED-001`** — federated SQLite remote publication is the last missing piece in the federation pipeline.
4. **CI secrets** — confirm all four Hetzner/SQLite federation secrets are set in every relevant repo (not just `go-go-app-sqlite`).
5. **Re-enable wesen-os CI** — the workflows were moved to `workflows-disabled/` as a temporary measure; revisit when the superproject/submodule pointer situation stabilizes.
