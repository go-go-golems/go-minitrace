# Changelog

## 2026-05-08

- Initial workspace created


## 2026-05-08

Add SKIP_DAGGER env var check to build-web/main.go for graceful Dagger skip

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/build-web/main.go — Added SKIP_DAGGER early exit


## 2026-05-08

Rewrote release.yaml: added build-frontend job, modified linux/darwin jobs to download pre-built frontend and set SKIP_DAGGER=1, removed Dagger CLI installs from release jobs

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/.github/workflows/release.yaml — Separated Dagger frontend build into dedicated job


## 2026-05-08

Add runtime check in embed.go for missing frontend, update README to warn about go install

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/README.md — Added warning about go install missing web UI
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/embed.go — Added init() check for missing index.html

