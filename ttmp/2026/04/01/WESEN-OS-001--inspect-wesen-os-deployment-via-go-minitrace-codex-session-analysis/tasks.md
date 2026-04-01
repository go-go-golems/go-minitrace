# Tasks

## TODO

- [x] Add tasks here

- [x] Get go-minitrace help and understand tool structure
- [x] Discover codex sessions from last 2 weeks
- [x] Convert codex sessions to minitrace format
- [x] Understand sessions_base schema
- [x] Write SQL to find wesen-os/deploy sessions
- [x] Deep-read wesen-os sessions content
- [x] Write final deployment summary report
- [x] Document minitrace improvement suggestions
- [x] Phase 1: add serve command package and register go-minitrace serve in the root CLI
- [x] Phase 1: build a startup session index from --archive-glob and thread it into the server
- [x] Phase 1: implement server skeleton plus POST /api/query with structured 200/400 JSON responses
- [x] Phase 2: implement GET /api/sessions using DuckDB plus explicit DTO normalization for frontend types
- [x] Phase 2: implement GET /api/sessions/{id} by loading converted .minitrace.json files from the startup index
- [x] Phase 2: return computed blocks inline from GET /api/sessions/{id} to match the current React app contract
- [x] Phase 3: implement block decomposition with internal raw block types and stable gap calculations
- [x] Phase 3: implement badge and artifact detection before DTO conversion and add optional GET /api/sessions/{id}/blocks
- [ ] Phase 4: implement GET /api/presets by merging built-in presets with --preset-dir using sidebar-friendly folders
- [ ] Phase 4: implement query-dir CRUD with path validation for folder and filename inputs
- [ ] Phase 5: wire embedded frontend assets, SPA fallback, and a build path that does not break plain go build ./...
- [ ] Phase 5: wire Vite-first dev mode with /api proxying to the Go backend
- [ ] Testing: add unit tests for blocks and badges plus integration coverage for /api/query, /api/sessions, and /api/sessions/{id}
- [ ] Validation: run end-to-end manual testing against the real WESEN-OS-001 archive and verify the existing web UI works without mock handlers
