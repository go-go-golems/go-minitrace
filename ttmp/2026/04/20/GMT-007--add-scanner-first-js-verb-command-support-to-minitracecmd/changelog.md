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

