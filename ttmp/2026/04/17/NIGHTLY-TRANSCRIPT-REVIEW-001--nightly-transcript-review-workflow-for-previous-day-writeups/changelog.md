# Changelog

## 2026-04-17

- Initial workspace created
- Switched nightly structured query rendering to reuse clay's SQL date helpers (`sqlDate`, `sqlDateTime`, `sqliteDate`, `sqliteDateTime`) while preserving local safe string escaping, and restored the nightly query catalog to date-typed `day` parameters.
- Migrated the nightly review SQL into embedded commands under `pkg/minitracecmd/core/nightly/` and exposed them as the `go-minitrace query commands nightly ...` subverb, while keeping the ticket-local SQL bundle as historical reference.

