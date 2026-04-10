---
Title: Dagger pnpm release-pipeline implementation guide
Ticket: GMT-005
Status: active
Topics:
    - backend
    - frontend
    - configuration
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: .github/workflows/release.yaml
      Note: Release jobs currently provision Go but not Dagger
    - Path: .goreleaser.yaml
      Note: Release before-hooks currently call make frontend
    - Path: Makefile
      Note: Current frontend build recipe to replace
    - Path: cmd/go-minitrace/cmds/serve/embed.go
      Note: Build output must still feed the embed path
    - Path: web/package.json
      Note: Frontend scripts and package-manager metadata target
ExternalSources: []
Summary: Evidence-backed plan for migrating go-minitrace frontend release builds to a Go Dagger pnpm flow.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Dagger pnpm release-pipeline implementation guide

## Executive summary

`go-minitrace` currently builds its embedded web frontend through a Makefile target that shells into `web/`, runs `npm ci && npm run build`, deletes `cmd/go-minitrace/cmds/serve/frontend`, and copies `web/dist` into that embed directory. GoReleaser calls that Makefile target in a `before.hooks` entry, so the release pipeline inherits the same host-level npm assumptions. This ticket should replace that path with a Go command that uses the Dagger Go SDK to run a pnpm-based build inside a Node container, export the built assets into the embed directory, and let both local Make targets and release automation delegate to that one implementation.

The migration should be explicit about reproducibility: commit a pnpm lockfile, set package-manager metadata, pin a Dagger/Node/pnpm baseline, and install Dagger in the GitHub release jobs before GoReleaser runs. The Makefile should then shrink down to orchestration rather than hand-implementing frontend build steps.

## Problem statement and scope

### Current behavior

- `Makefile` defines `frontend` as:
  - `cd web && npm ci && npm run build`
  - `rm -rf cmd/go-minitrace/cmds/serve/frontend`
  - `cp -r web/dist cmd/go-minitrace/cmds/serve/frontend`
- `.goreleaser.yaml` runs `make frontend` in `before.hooks`
- `.github/workflows/release.yaml` invokes GoReleaser on Linux and macOS runners without any Dagger installation step because the current path does not use it
- `web/` currently carries `package-lock.json` but no `pnpm-lock.yaml`

### Desired behavior

- A Go command, e.g. `go run ./cmd/build-web`, uses Dagger to:
  - mount `web/` into a Node container
  - activate a pinned pnpm version via Corepack
  - run `pnpm install --frozen-lockfile`
  - run `pnpm run build`
  - export the resulting `dist/` into `cmd/go-minitrace/cmds/serve/frontend`
- The repository commits pnpm metadata (`pnpm-lock.yaml` and package-manager declaration)
- The Makefile frontend target becomes a thin wrapper around the Go Dagger builder
- GoReleaser uses the Dagger builder path instead of the old npm shell recipe
- GitHub release jobs install Dagger before GoReleaser runs

## Current-state analysis

### The Makefile owns the frontend build recipe today

Evidence: `Makefile:33-40`

Observed behavior:
- `build` depends on `frontend`
- `frontend` shells into `web` with npm and manually copies `web/dist` into the embed directory
- `build-bin` also depends on `frontend`

This is the main cleanup target because it duplicates build logic that a dedicated Go command can encapsulate more cleanly.

### GoReleaser already hooks frontend building before compiling Go binaries

Evidence: `.goreleaser.yaml:4-11`

Observed behavior:
- `before.hooks` includes `go mod tidy`, `make frontend`, and `go generate ./...`
- The release pipeline is therefore already structurally prepared for a frontend build step; we only need to replace the implementation, not redesign the overall sequence

### Release workflow runners do not currently provision Dagger

Evidence: `.github/workflows/release.yaml:1-113`

Observed behavior:
- Linux and macOS release jobs install Go (and on Linux, a cross-compiler)
- Then they call GoReleaser directly
- There is no Dagger CLI setup step today

If the frontend build moves to Dagger, the release jobs need an explicit Dagger install step before the GoReleaser action runs.

### The repo is still npm-lock oriented, not pnpm-lock oriented

Evidence: `web/package.json` plus presence of `web/package-lock.json`

Observed behavior:
- scripts are generic enough to run under pnpm (`build`, `dev`, etc.)
- but reproducible pnpm installs need a committed `pnpm-lock.yaml`
- the repo currently lacks package-manager metadata telling Corepack/pnpm what to activate

## Proposed solution

### Add a dedicated Go Dagger builder

Create a new command, likely under `cmd/build-web/main.go`, that:
1. finds the repo root by walking upward to `go.mod`
2. computes:
   - source dir: `web/`
   - export dir: `cmd/go-minitrace/cmds/serve/frontend`
3. removes any stale exported assets before writing new ones
4. connects to Dagger with log output enabled for CI visibility
5. runs a Node container with Corepack + a pinned pnpm version
6. runs `pnpm install --frozen-lockfile`
7. runs `pnpm run build`
8. exports `/src/web/dist` into the embed directory

Recommended defaults:
- Node image: `node:22-bookworm`
- pnpm version: pin explicitly, e.g. `10.15.0`
- allow override by env vars like `WEB_BUILDER_IMAGE` and `WEB_PNPM_VERSION` if desired

### Make pnpm reproducible

1. Generate and commit `web/pnpm-lock.yaml`
2. Add `packageManager` to `web/package.json`, e.g. `pnpm@10.15.0`
3. Remove `web/package-lock.json` if the repo is fully cutting over to pnpm

### Simplify Makefile orchestration

Replace the current `frontend` recipe with something like:

```make
frontend:
	go run ./cmd/build-web
```

Then clean up any Makefile dev commands that still use npm where pnpm is now the chosen package manager, for example the Vite dev pane command.

### Wire release automation

#### GoReleaser

Update `.goreleaser.yaml` so `before.hooks` uses the Dagger builder directly or indirectly through the cleaned Make target.

Either of these are acceptable:

```yaml
before:
  hooks:
    - go mod tidy
    - go run ./cmd/build-web
    - go generate ./...
```

or keep:

```yaml
before:
  hooks:
    - go mod tidy
    - make frontend
    - go generate ./...
```

provided `make frontend` is now just the Dagger wrapper.

#### GitHub Actions release workflow

Add a Dagger install step to the Linux and macOS release jobs before the GoReleaser action runs.

A portable pattern is:

```yaml
- name: Install Dagger CLI
  run: |
    curl -fsSL https://dl.dagger.io/dagger/install.sh | BIN_DIR="$HOME/.local/bin" DAGGER_VERSION=0.20.5 sh
    echo "$HOME/.local/bin" >> "$GITHUB_PATH"
    "$HOME/.local/bin/dagger" version
```

Pin the version deliberately so the workflow and the Go SDK stay aligned.

## Implementation plan

### Phase 1 — Add the Dagger builder and pnpm metadata

Files:
- `cmd/build-web/main.go`
- `go.mod`
- `go.sum`
- `web/package.json`
- `web/pnpm-lock.yaml`
- remove `web/package-lock.json` if cutting over fully

### Phase 2 — Switch local build orchestration

Files:
- `Makefile`

Actions:
- replace the inline npm build/copy recipe with `go run ./cmd/build-web`
- update any `npm run dev` references to pnpm if the repo is standardizing on pnpm
- keep targets focused and avoid duplicate frontend build logic

### Phase 3 — Switch release orchestration

Files:
- `.goreleaser.yaml`
- `.github/workflows/release.yaml`

Actions:
- point GoReleaser at the Dagger builder path
- provision Dagger in release jobs before GoReleaser runs

## Testing and validation strategy

Run at minimum:

```bash
cd go-minitrace

go run ./cmd/build-web
go test ./... -count=1
golangci-lint run -v

cd web
pnpm install --frozen-lockfile
pnpm run build
```

And validate docs/bookkeeping with:

```bash
cd go-minitrace
docmgr doctor --ticket GMT-005 --stale-after 30
```

## Risks and alternatives

### Risks

1. Dagger connectivity/setup differences between local machines and GitHub runners
2. pnpm lockfile churn if the chosen version is not pinned clearly enough
3. Build output path mistakes that break `//go:embed` at release time

### Alternatives considered

1. Keep npm in Makefile and only containerize later — rejected because it preserves the host-tooling assumption the user asked to remove
2. Use shell-script Dagger invocations instead of a Go command — rejected because the user explicitly asked for Go Dagger and a Go command keeps the logic reviewable in-repo

## References

- `Makefile`
- `.goreleaser.yaml`
- `.github/workflows/release.yaml`
- `web/package.json`
- `cmd/go-minitrace/cmds/serve/embed.go`
