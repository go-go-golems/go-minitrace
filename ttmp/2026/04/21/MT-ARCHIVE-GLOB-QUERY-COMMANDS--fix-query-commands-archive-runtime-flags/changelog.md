# Changelog

## 2026-04-22

- Investigated the reported `query commands ... --archive-glob` failure and established that runtime flags were already wired correctly on executable leaf commands.
- Reproduced the misleading failure mode for self-named single-verb JS files and documented it with replayable scripts in `scripts/`.
- Implemented the fix in `pkg/minitracecmd/parse_javascript.go` by collapsing redundant self-named single-verb JS command paths during command-spec creation.
- Updated query/serve tests and user-facing docs so the new path rule is explicit and validated.
- Committed the implementation as `e2d6c37b140edcc8a3dd8ccf4557c668de94d2d9` (`query: collapse self-named JS command paths`).
- Uploaded the ticket bundle to reMarkable at `/ai/2026/04/22/MT-ARCHIVE-GLOB-QUERY-COMMANDS/MT-ARCHIVE-GLOB-QUERY-COMMANDS` and verified the remote folder exists.
