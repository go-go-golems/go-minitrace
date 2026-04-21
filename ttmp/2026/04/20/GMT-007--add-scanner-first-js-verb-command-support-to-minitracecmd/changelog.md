# Changelog

## 2026-04-20

- Initial workspace created for GMT-007
- Added the primary design doc and investigation diary
- Gathered evidence from the current SQL catalog/runtime path and the existing jsverbs scanner/runtime path
- Drafted the detailed analysis, design, and implementation guide for scanner-first JS command support
- Ran `docmgr doctor --ticket GMT-007 --stale-after 30` successfully
- Performed a dry-run bundled reMarkable upload, uploaded the final bundle, and verified the remote entry after correcting an initial `cloud ls` path mismatch
- Broke the three remaining implementation tasks into a detailed engineering checklist covering catalog/model changes, JS runtime execution, and mixed SQL/JS validation work
- Step 1 implementation landed in commit `bf7a787` (`Add scanner-first JS command catalog support`), covering JS source detection, command-model generalization, JS scan/adapter code, duplicate logical-path checks, and initial catalog tests
- Step 2 implementation landed in commits `6d935a5` (`Add JS command runtime execution support`) and `e9db41e` (`Harden JS command runtime error handling`), covering CLI/serve JS execution, alias-default propagation, Promise support, explicit text-mode deferral, and runtime failure-path tests
- Step 3 validation and docs work added mixed SQL/JS help coverage, JS command documentation, focused `go-go-goja` regression coverage, successful CLI smoke runs for both JS-backed and SQL-backed commands from the same repository, and a final clean `docmgr doctor` pass after the implementation docs were updated
- Uploaded a refreshed final reMarkable bundle named `GMT-007 scanner-first JS verb commands complete` and verified it under `/ai/2026/04/20/GMT-007/`
- Adjusted JS command registration so each JS file becomes a command group and each scanned verb becomes a leaf command, updating parser behavior, CLI/serve tests, and user docs to prefer paths like `overview session-tools session-list`
- Added a checked-in `testdata/query-repositories/js-showcase/` repository with realistic JS examples covering helper modules, multi-verb files, synthetic row generation, async commands, `queryOne`, and JS-side post-processing, plus tests and smoke validation proving the showcase runs end-to-end

