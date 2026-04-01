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
- [x] Phase 4: implement GET /api/presets by merging built-in presets with --preset-dir using sidebar-friendly folders
- [x] Phase 4: implement query-dir CRUD with path validation for folder and filename inputs
- [x] Phase 5: wire embedded frontend assets, SPA fallback, and a build path that does not break plain go build ./...
- [x] Phase 5: wire Vite-first dev mode with /api proxying to the Go backend
- [x] Testing: add unit tests for blocks and badges plus integration coverage for /api/query, /api/sessions, and /api/sessions/{id}
- [x] Validation: run end-to-end manual testing against the real WESEN-OS-001 archive and verify the existing web UI works without mock handlers
- [x] Serve follow-up: make --query-dir a string-list flag, load all configured query roots, and keep CRUD behavior deterministic
- [x] Serve follow-up: make --preset-dir a string-list flag and load all configured preset roots
- [x] Serve follow-up: add regression tests for multi-root preset/query loading
- [x] Serve follow-up: make --archive-glob a string-list flag and load all configured archive globs into DuckDB
- [x] Serve follow-up: update session indexing and query-engine loading to support multiple archive globs deterministically
- [x] Serve follow-up: add regression tests for multi-glob archive loading and indexing
- [x] Serve follow-up: confirm preset-dir/query-dir already reload on fresh API fetch and document the remaining frontend gap
- [x] Serve follow-up: poll presets and saved queries in the query editor so disk changes show up without a page reload
- [x] Serve follow-up: track the editor source file and auto-refresh the editor when the backing file changes and the buffer is still clean
- [x] Serve follow-up: warn instead of overwriting when the backing file changes while the editor has unsaved local edits
- [x] Serve follow-up: refresh the embedded frontend bundle after the hot-reload UI changes
- [x] Frontend follow-up: align Storybook test helpers with the Storybook 10.3 stack so npm ci and make frontend work again
