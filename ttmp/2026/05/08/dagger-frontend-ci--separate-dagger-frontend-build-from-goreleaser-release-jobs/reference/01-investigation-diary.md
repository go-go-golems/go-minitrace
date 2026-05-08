---
title: Investigation Diary
doc-type: reference
ticket: dagger-frontend-ci
status: active
intent: long-term
topics:
  - ci
  - deployment
  - frontend
created: 2026-05-08
---

# Investigation Diary

## 2026-05-08 — Session Start

### Context
User reported GoReleaser release failure on macOS runner for v0.0.10 tag.
Error: `driver for scheme "image" was not available` from Dagger engine.

### Analysis
- Traced the failure: `go generate ./...` → `cmd/go-minitrace/cmds/serve/generate.go` → `go run ../../../build-web` → `cmd/build-web/main.go` → Dagger.Connect()
- macOS runners have no Docker/container runtime → Dagger can't start
- Dagger CLI is installed on the macOS runner but is useless without a runtime
- Linux runner works fine (Docker available on ubuntu-latest)

### Decision: Option 4 — Separate frontend build job
- Create a dedicated `build-frontend` job on Linux with Dagger
- Share frontend artifacts to both release jobs
- Add `SKIP_DAGGER` env var to `build-web/main.go` for graceful skip
- Both release jobs set `SKIP_DAGGER=1` and download pre-built frontend

### Work Log

- [x] Diagnosed root cause
- [x] Created docmgr ticket `dagger-frontend-ci`
- [x] Created analysis & implementation guide
- [x] Task 1: Add `SKIP_DAGGER` env var to `build-web/main.go` (commit 36ddb7d)
- [x] Task 2: Add `build-frontend` job to `release.yaml` (commit 62096ef)
- [x] Task 3: Modify `goreleaser-linux` job (commit 62096ef)
- [x] Task 4: Modify `goreleaser-darwin` job (commit 62096ef)
- [x] Task 5: Verify `.goreleaser.yaml` before hooks — safe, go generate exits 0 with SKIP_DAGGER=1
- [ ] Task 6: Test the workflow (needs tag push or workflow_dispatch)

### 2026-05-08 — Implementation Complete

All code changes done. Two commits:
1. `36ddb7d` — feat(build-web): add SKIP_DAGGER env var to skip Dagger frontend build
2. `62096ef` — fix(ci): separate Dagger frontend build from GoReleaser release jobs

The `.goreleaser.yaml` before hooks remain unchanged — `go generate ./...` will hit the SKIP_DAGGER check and exit 0 in CI, while still working fully for local builds.

Remaining: user needs to push and test via workflow_dispatch or a new tag.

### 2026-05-08 — go install missing frontend

User reported that `go install` produces a binary with no embedded frontend.

Root cause: `go install` does NOT run `go generate`. The `frontend/` directory
is gitignored (except `.gitkeep`), so the embed picks up an empty dir.

Fixes applied:
1. Added `init()` check in `embed.go` that logs a clear error at startup if
   `index.html` is missing from the embedded FS — so `go install` users see a
   helpful message instead of a mystery broken site.
2. Updated README.md to warn that `go install` produces a binary without the
   web UI, and recommend Homebrew or building from source.

These are UX guardrails — the real fix for `go install` would require either
committing built frontend assets or a different distribution approach.
