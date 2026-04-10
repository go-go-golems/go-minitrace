# Changelog

## 2026-04-09

- Initial workspace created
- Added the Dagger/pnpm release-pipeline task breakdown and implementation-plan deliverables
- Step 2: added a Go Dagger frontend builder, switched release/local frontend orchestration to pnpm + Dagger, provisioned Dagger in the release workflow, removed the old npm lockfile, and validated the path with `go run ./cmd/build-web`, `pnpm run build`, `go test ./...`, `golangci-lint run -v`, and `goreleaser release --skip=sign --snapshot --clean --single-target` (commit `e96f4bc`)
- Step 3: ran `docmgr doctor`, uploaded the GMT-005 bundle to reMarkable, and verified the remote entry after correcting an initial `cloud ls` path mismatch

## 2026-04-09

Ticket closed

