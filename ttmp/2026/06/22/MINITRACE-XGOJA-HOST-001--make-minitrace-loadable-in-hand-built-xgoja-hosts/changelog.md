# Changelog

## 2026-06-22

- Initial workspace created


## 2026-06-22

Created intern-oriented design package for issue #20: evidence-backed architecture analysis, default-registry implementation guide, reproduction scripts, diary, and future implementation checklist.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/22/MINITRACE-XGOJA-HOST-001--make-minitrace-loadable-in-hand-built-xgoja-hosts/design-doc/01-hand-built-xgoja-host-module-loading-design-and-implementation-guide.md — Primary design and implementation guide
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/22/MINITRACE-XGOJA-HOST-001--make-minitrace-loadable-in-hand-built-xgoja-hosts/reference/01-investigation-diary.md — Chronological investigation diary


## 2026-06-22

Validated ticket with docmgr doctor and uploaded the documentation bundle to reMarkable at /ai/2026/06/22/MINITRACE-XGOJA-HOST-001/MINITRACE XGOJA HOST 001 Design.pdf.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/22/MINITRACE-XGOJA-HOST-001--make-minitrace-loadable-in-hand-built-xgoja-hosts/design-doc/01-hand-built-xgoja-host-module-loading-design-and-implementation-guide.md — Included in uploaded design bundle
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/22/MINITRACE-XGOJA-HOST-001--make-minitrace-loadable-in-hand-built-xgoja-hosts/reference/01-investigation-diary.md — Included in uploaded design bundle


## 2026-06-22

Step 2: committed initial ticket documentation before code changes (commit 41ae31bf6ec0d92116e9e9a4ccb140011d22a267).

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/22/MINITRACE-XGOJA-HOST-001--make-minitrace-loadable-in-hand-built-xgoja-hosts/reference/01-investigation-diary.md — Diary to update with commit checkpoint


## 2026-06-22

Step 3: implemented default-registry minitrace module adapter and integration test; fixed command runtime duplicate-loader ordering so RuntimeArchives keeps command settings (commit 0836dda).

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go — Exclude default minitrace when command-scoped loader is registered
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/default_module.go — Default adapter
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/default_module_test.go — Plain builder require smoke test


## 2026-06-22

Step 4: documented hand-built embedded minitrace module usage in README, including blank-import registration and runtime-settings caveats (commit 5e7d52a).

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/README.md — Embedded minitrace documentation


## 2026-06-22

Step 5: migrated the minitrace xgoja command-provider example to xgoja/v2 and validated make smoke successfully (commit 4c8fb8d).

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/examples/xgoja/minitrace-command-provider/xgoja.yaml — xgoja v2 spec
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/22/MINITRACE-XGOJA-HOST-001--make-minitrace-loadable-in-hand-built-xgoja-hosts/sources/05-xgoja-example-smoke-after-migration.txt — Smoke output


## 2026-06-22

Step 6: final validation passed with go test ./..., GOWORK=off go test ./..., and docmgr doctor; all implementation tasks checked.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/22/MINITRACE-XGOJA-HOST-001--make-minitrace-loadable-in-hand-built-xgoja-hosts/sources/06-final-validation.txt — Final validation evidence

