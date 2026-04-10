# Tasks

## TODO

- [x] Inspect the current frontend build path across `Makefile`, `.goreleaser.yaml`, and `.github/workflows/release.yaml`
- [x] Write the Dagger/pnpm implementation guide and relate the key build/release files
- [x] Add a Go-based Dagger frontend build command that runs `pnpm install --frozen-lockfile` and `pnpm run build`
- [x] Add and commit the pnpm lockfile / package-manager metadata needed for reproducible pnpm installs
- [x] Update the Makefile frontend-related targets to call the new Dagger builder and remove the old `npm ci && npm run build` flow
- [x] Update `.goreleaser.yaml` so release builds use the Dagger-based frontend build step
- [x] Update `.github/workflows/release.yaml` so the release jobs install Dagger before invoking GoReleaser
- [x] Run focused validation for the Dagger frontend build path and full repo validation
- [x] Update the diary/changelog/tasks, run `docmgr doctor`, and upload the ticket bundle to reMarkable
