# Changelog

## 2026-07-20

- Initial workspace created


## 2026-07-20

Embedded file-history, ticket-timeline, and context-window as built-in go-minitrace JS query commands (pkg/minitracecmd/core/history/), confirmed the embedded catalog treats JS identically to SQL, rebuilt+installed the binary, verified against real archives with zero --query-repository config, and removed the now-redundant copies from the go-minitrace-transcript-analysis skill (all three hardlinked mirrors).

### Related Files

- pkg/minitracecmd/assets_test.go — test assertions for the 3 new embedded commands
- pkg/minitracecmd/core/history/context-window.js — embedded context-window verb
- pkg/minitracecmd/core/history/file-history.js — embedded file-history verb
- pkg/minitracecmd/core/history/ticket-timeline.js — embedded ticket-timeline verb

