- [x] Modify goreleaser-darwin job: add needs, download artifact, set SKIP_DAGGER=1, remove Dagger CLI install
ser-linux job: add needs, download artifact, set SKIP_DAGGER=1, remove Dagger CLI install
- [x] Add build-frontend job to release.yaml
- [x] Add SKIP_DAGGER env var check to cmd/build-web/main.go
- [x] Verify .goreleaser.yaml before hooks are safe with SKIP_DAGGER
