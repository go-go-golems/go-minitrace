---
title: Analysis & Implementation Guide
doc-type: design-doc
ticket: dagger-frontend-ci
status: active
intent: long-term
topics:
  - ci
  - deployment
  - frontend
created: 2026-05-08
---

# Analysis & Implementation Guide

## Executive Summary

The GoReleaser release workflow fails on macOS runners because `go generate ./...` invokes a Dagger-based frontend build that requires a container runtime (Docker). GitHub's `macos-latest` runners don't have Docker. The fix is to separate the Dagger frontend build into its own CI job on Linux, then share the built assets as artifacts to both the Linux and Darwin release jobs.

## Problem Statement

### Current Flow

```
release.yaml (split build)
├── goreleaser-linux (ubuntu-latest)
│   ├── Install Dagger CLI
│   ├── goreleaser release --split  ← go generate → Dagger build (works, Docker available)
│   └── upload artifact dist-linux
├── goreleaser-darwin (macos-latest)
│   ├── Install Dagger CLI
│   ├── goreleaser release --split  ← go generate → Dagger build (FAILS, no Docker)
│   └── upload artifact dist-darwin
└── goreleaser-merge (ubuntu-latest)
    ├── download dist-linux + dist-darwin
    └── goreleaser continue --merge
```

### Root Cause

- `cmd/go-minitrace/cmds/serve/generate.go` has `//go:generate go run ../../../build-web`
- `cmd/build-web/main.go` uses `dagger.io/dagger` to spin up a Node container
- Dagger needs Docker/container runtime to pull OCI images
- `macos-latest` has no Docker → `driver for scheme "image" was not available`
- The Dagger CLI being installed is irrelevant without a runtime

### Why Other Approaches Don't Work

| Approach | Problem |
|---|---|
| Install Docker on macOS runner | Slow (~2-3min), flaky, Colima often breaks |
| Cross-compile Darwin from Linux with CGO | Requires osxcross + Apple SDK, complex setup |
| Skip Dagger on macOS, use checked-in assets | Frontend can go stale, two sources of truth |
| Make frontend build non-Dagger | Loses reproducibility, pnpm cache, isolated build |

## Proposed Solution

**Separate the Dagger frontend build into a standalone CI job on Linux, then share the built frontend assets as artifacts.**

### New Flow

```
release.yaml (split build)
├── build-frontend (ubuntu-latest)
│   ├── Install Dagger CLI
│   ├── go run ./cmd/build-web    ← Dagger build (works)
│   └── upload artifact frontend-dist (cmd/go-minitrace/cmds/serve/frontend/)
├── goreleaser-linux (ubuntu-latest)
│   ├── needs: build-frontend
│   ├── download artifact frontend-dist → cmd/.../serve/frontend/
│   ├── goreleaser release --split  ← go generate SKIPPED (frontend already built)
│   └── upload artifact dist-linux
├── goreleaser-darwin (macos-latest)
│   ├── needs: build-frontend
│   ├── download artifact frontend-dist → cmd/.../serve/frontend/
│   ├── goreleaser release --split  ← go generate SKIPPED (frontend already built)
│   └── upload artifact dist-darwin
└── goreleaser-merge (ubuntu-latest)
    ├── needs: [goreleaser-linux, goreleaser-darwin]
    ├── download dist-linux + dist-darwin
    └── goreleaser continue --merge
```

### Key Design Decisions

1. **Frontend build is a separate job**, not a workflow_call reusable workflow — keeps it simple, one file
2. **Skip `go generate` in GoReleaser by setting `SKIP_DAGGER=1`** — the `build-web` binary checks this env var and exits 0 when set, so `go generate` succeeds as a no-op
3. **Frontend artifact includes all files from `serve/frontend/`** — both `dist/` output and `.gitkeep`
4. **Both Linux and Darwin release jobs download the same frontend artifact** — single source of truth, no divergent builds

## Implementation Plan

### Task 1: Add `SKIP_DAGGER` env var check to `build-web/main.go`

Add a check at the top of `main()` that exits 0 if `SKIP_DAGGER=1` is set. This lets `go generate` succeed as a no-op when the frontend has already been built by the separate job.

```go
func main() {
    if os.Getenv("SKIP_DAGGER") != "" {
        fmt.Println("SKIP_DAGGER set, skipping Dagger frontend build")
        os.Exit(0)
    }
    // ... existing code
}
```

### Task 2: Add `build-frontend` job to release.yaml

New job at the top of the jobs list:
- Runs on `ubuntu-latest`
- Installs Dagger CLI (same as current Linux job)
- Runs `go run ./cmd/build-web` (not `go generate`, direct)
- Uploads `cmd/go-minitrace/cmds/serve/frontend/` as artifact `frontend-dist`

### Task 3: Modify `goreleaser-linux` job

- Add `needs: build-frontend`
- Add step: download `frontend-dist` artifact into `cmd/go-minitrace/cmds/serve/frontend/`
- Set `SKIP_DAGGER=1` in goreleaser env
- Remove Dagger CLI install step (no longer needed here)

### Task 4: Modify `goreleaser-darwin` job

- Add `needs: build-frontend`
- Add step: download `frontend-dist` artifact into `cmd/go-minitrace/cmds/serve/frontend/`
- Set `SKIP_DAGGER=1` in goreleaser env
- Remove Dagger CLI install step

### Task 5: Verify `.goreleaser.yaml` before hooks are safe

The `go generate ./...` hook in `.goreleaser.yaml` should remain — it will now be a no-op when `SKIP_DAGGER=1` is set (exits 0 immediately). This preserves the ability to run `go generate` locally without the env var.

### Task 6: Test the workflow

Push a test tag or use `workflow_dispatch` to trigger the release and verify:
- `build-frontend` job succeeds
- Both `goreleaser-linux` and `goreleaser-darwin` succeed
- `goreleaser-merge` succeeds
- Released binaries include the embedded frontend

## Alternatives Considered

1. **Remove `go generate` from before hooks entirely, always use artifact** — risks stale frontend in local builds. Better to keep `go generate` as a fallback.
2. **Use `workflow_call` for the frontend build** — adds complexity for no real benefit in a single-repo setup.
3. **Commit built frontend to git** — pollutes history, merge conflicts on generated files.
4. **Install Colima on macOS** — adds 2-3min per run, fragile, not worth it when we can just build on Linux.

## Open Questions

- None currently. The approach is straightforward.
