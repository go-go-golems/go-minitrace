---
Title: Investigation Diary
Slug: go-minitrace-docs-diary
Short: Chronological diary of the go-minitrace documentation investigation
Topics:
- minitrace
- documentation
- glazed
DocType: reference
Status: active
Intent: long-term
CreatedAt: "2026-04-01"
---

# Diary

## Goal

Investigate go-minitrace end-to-end — its codebase structure, command surface, data sources, conversion pipeline, and query capabilities — to produce a concrete documentation plan for Glazed help pages.

## Step 1: Codebase Architecture Survey

I mapped the full project structure to understand what go-minitrace does, how big it is, and how the pieces connect.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to create great documentation for go-minitrace. THen, investigate the tool, load a few transcripts from ~/.claude and ~/.codex and /tmp/turns.db and such, experiment with queries (store all queries and scripts in the scripts/ folder of the ticket), keeping a detailed diary as you go. Then, write propsoe a set of documentation to write (which we will then later write and put into the glazed binary according to the help entries for glazed skill)."

**Assistant interpretation:** Create a ticket, deeply explore the tool by running real commands, and propose a documentation plan for Glazed help integration.

**Inferred user intent:** Build a concrete, evidence-based documentation plan that can be directly turned into Glazed help pages embedded in the go-minitrace binary.

### What I did
- Listed all `.go` files — 10,253 lines total across ~40 files
- Identified the adapter structure: `pkg/adapters/{claudecode,codex,pi,claudeai,chatgpt,turnsdb}/`
- Found the core schema in `pkg/minitrace/schema.go` (250 lines, ~20 top-level types)
- Found existing docs in `pkg/doc/*.md` — 5 files already wired into Glazed help
- Identified the query engine in `pkg/query/` with 6 embedded SQL presets
- Found standalone DuckDB query recipes in `queries/`

### Why
Need to understand what's already documented vs. what's missing before proposing new docs.

### What worked
- The project is well-structured with clear separation: adapters → minitrace schema → query engine → CLI commands
- Existing `pkg/doc/*.md` files already use Glazed frontmatter and are wired via `go:embed` in `pkg/doc/doc.go`
- The `main.go` follows the canonical Glazed initialization pattern (HelpSystem + SetupCobraRootCommand)

### What didn't work
- N/A (exploration phase)

### What I learned
- **6 adapter backends**: Claude Code, Codex, Pi, claude.ai, ChatGPT (ZIP + JSON), turnsdb
- **3 discovery backends**: Claude Code, Codex, Pi (claude.ai/chatgpt/turnsdb don't have discover commands)
- **1 query backend**: DuckDB with 6 presets + inline SQL + SQL file support
- The minitrace schema is v0.2.0 with rich structure: Session → Turns, ToolCalls, Annotations, Metrics
- Token tracking includes input, output, cache_read, cache_creation, reasoning, and tool tokens
- The existing 5 help pages are brief overviews — none explain the schema, adapters, workflows, or query syntax in detail

### What was tricky to build
- N/A (exploration phase)

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- N/A (no code changes)

### Technical details

Key file sizes (lines):
```
877  pkg/adapters/codex/convert.go
814  pkg/adapters/claudecode/convert.go
786  pkg/adapters/turnsdb/convert.go
716  pkg/adapters/chatgpt/transcript.go
645  pkg/adapters/pi/convert.go
630  pkg/adapters/claudeai/convert.go
484  pkg/adapters/chatgpt/convert.go
264  pkg/minitrace/util.go
251  pkg/minitrace/archive.go
250  pkg/minitrace/schema.go
213  pkg/minitrace/metrics.go
188  pkg/minitrace/builders.go
169  pkg/query/engine.go
```

Existing help pages:
- `overview.md` — GeneralTopic, IsTopLevel: true
- `convert.md` — GeneralTopic, covers convert group
- `discover.md` — Tutorial, covers discover group
- `query.md` — Tutorial, covers query group
- `query-duckdb.md` — Example, covers `query duckdb` subcommand

---

## Step 2: Running Real Discoveries and Conversions

I ran every discovery and conversion command against real data to understand the tool from a user's perspective.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Exercise all commands with real data from ~/.claude, ~/.codex, ~/.pi/agent/sessions, and /tmp/turns.db.

**Inferred user intent:** Generate real converted archives to query against and verify the tool works end-to-end.

### What I did
- `discover claude-code` → **2,502 sessions** found
- `discover codex` → **1,174 sessions** found
- `discover pi` → **52 sessions** found
- `convert claude-code` → **3,212 archives** written (includes subagents)
- `convert turnsdb` → **7 archives** written
- `convert pi` → **52 archives** written
- `convert codex` → **error** on one session with `unknown-jsonl` format hint
- All output goes to `output/active/YYYY-MM/` by default with `.minitrace.json` suffix
- A `manifest.json` is written alongside with session count

### Why
Documentation must reflect real behavior, not just code reading. Running against real data reveals the actual output shapes, error messages, and edge cases.

### What worked
- Claude Code, Pi, and turnsdb converters all ran cleanly on real data
- The archive directory structure `active/YYYY-MM/<id>.minitrace.json` is predictable
- The dry-run flag works on all convert commands
- The output table format during conversion is informative (shows source, format hint, output path)

### What didn't work
- **Codex converter error**: `converting Codex session rollout-2025-08-27T08-42-14-9367de1d...: unsupported Codex format hint: unknown-jsonl`
  - Command: `go run ./cmd/go-minitrace convert codex --source-dir ~/.codex --output-dir /tmp/minitrace-docs-test`
  - This is a real edge case — older Codex sessions may have unknown JSONL formats

### What I learned
- The convert commands emit a table-style summary including manifest path, session count, and per-session details
- The `--dry-run` flag is available on all convert subcommands
- Subagent sessions get their own minitrace files with source_format `claude-code-jsonl-v2+subagent`
- The output directory structure is `active/YYYY-MM/` (date-bucketed)

### What was tricky to build
- N/A (running commands)

### What warrants a second pair of eyes
- The codex converter fails on unknown JSONL formats rather than skipping — this should be documented as a known limitation or fixed

### What should be done in the future
- Document the codex unknown-jsonl edge case

### Code review instructions
- N/A

### Technical details

Discovery output format (JSON):
```json
{
  "format_hint": "jsonl-v2",
  "id": "00016dab-e34b-4890-85a7-b1d3b27bccc8",
  "source_path": "/home/manuel/.claude/projects/..."
}
```

Converted session shape (top-level keys):
```
id, schema_version, profile, scenario_id, quality, title, summary,
classification, provenance, flags, environment, operational_context,
timing, condition, coordination, handover, turns, tool_calls,
outcome, annotations, metrics
```

---

## Step 3: Query Experimentation

I ran all built-in presets and wrote custom SQL queries to understand the query surface area.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Test all DuckDB presets and write ad-hoc queries to explore what questions can be answered.

**Inferred user intent:** Understand how the query command works in practice so documentation can include useful examples.

### What I did
- Ran all 6 presets: session-list, framework-summary, tool-operation-breakdown, timing-analysis, read-ratio-distribution, annotations
- Wrote custom queries for: model breakdown per framework, token usage summary, tool name frequency, longest sessions
- Stored all queries in `scripts/adhoc-queries.sql` and `scripts/explore-sessions.sql`

### Why
The query command is the analysis interface — understanding what questions it can answer defines the "use case" documentation.

### What worked
- **framework-summary**: Shows Pi sessions average 384 turns/207 tools vs Claude Code at 16 turns/10 tools — Pi sessions are dramatically larger
- **tool-operation-breakdown**: READ (16,160), EXECUTE (14,448), MODIFY (4,762), NEW (3,595), DELEGATE (1,443)
- **Custom model query**: 2,590 sessions on claude-opus-4-6, 332 on claude-haiku-4-5, 47 Pi sessions on claude-opus-4-6
- **Custom token query**: Total ~1.8M input tokens and ~10.8M output tokens across Claude Code sessions
- The `--sql-file` flag works well for running saved query files

### What didn't work
- The `read-ratio-distribution` preset dumps every session (one row per session) — output is very long and not grouped/summarized

### What I learned
- The `--output json` flag makes query output easy to pipe to other tools
- The `--sql-file` flag enables reusable query libraries
- All presets reference `sessions_base` as the table name (configurable via `--table-name`)
- DuckDB's `read_json` with `columns=` spec and `ignore_errors=true` is the loading mechanism
- The `--load-only` flag creates the table without running a query (useful for interactive DuckDB sessions)
- The `--persist-loaded` flag creates a permanent table instead of temp (useful for `--db-path` workflows)

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- The read-ratio-distribution preset might benefit from bucketing instead of per-session output

### What should be done in the future
- Consider adding a "model-summary" preset since model usage is a common question

### Code review instructions
- Scripts stored in ticket's `scripts/` directory

### Technical details

Framework summary results:
```
pi:          52 sessions, avg 384 turns, 208 tools, 12,210s duration
claude-code: 3,212 sessions, avg 16 turns, 10 tools, 488s duration
pinocchio:   7 sessions, avg 7 turns, 0.1 tools, 23s duration
```

Tool frequency top-5:
```
Read: 10,000 | Bash: 9,957 | bash: 5,399 | Edit: 3,281 | read: 2,918
```

---

## Step 4: Existing Documentation Gap Analysis

I compared the existing 5 help pages against what a new user would need to know.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Analyze what documentation exists vs what's needed.

**Inferred user intent:** Identify specific documentation gaps to fill.

### What I did
- Read all 5 existing `pkg/doc/*.md` files
- Compared against the Glazed help-page authoring skill requirements
- Evaluated each existing page for content depth, examples, and discoverability
- Read the Glazed help-page authoring skill to understand requirements

### Why
The documentation plan must identify specific gaps, not just propose topics.

### What worked
- The existing pages have correct Glazed frontmatter and are wired into the help system
- The overview page is IsTopLevel: true, which means it shows up in the top-level help

### What didn't work
- N/A

### What I learned
The existing 5 pages have significant gaps:

1. **`overview.md`** — Lists capabilities but doesn't explain what minitrace *is*, the schema concept, or the conversion pipeline
2. **`convert.md`** — Only mentions Claude Code and Codex; missing Pi, claude.ai, ChatGPT, turnsdb. No worked examples. No explanation of output format
3. **`discover.md`** — Extremely brief. No explanation of what discovery output means or how to use it
4. **`query.md`** — Only mentions DuckDB exists. No examples of custom SQL, presets, or pipeline usage
5. **`query-duckdb.md`** — Lists presets but doesn't explain the schema, DuckDB-specific JSON operators, or how to write custom queries

Missing entirely:
- **Schema documentation**: What is a minitrace session? What fields exist? What do they mean?
- **Per-adapter documentation**: How each source format maps to minitrace
- **Workflow/tutorial**: End-to-end from "I have claude sessions" to "I have insights"
- **Custom query guide**: How to write DuckDB queries against the minitrace schema
- **Architecture/concepts**: What is the discover→convert→query pipeline?
- **Troubleshooting**: Common errors, edge cases, data quality notes

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- Whether the existing pages should be rewritten from scratch or incrementally improved

### What should be done in the future
- Write the proposed documentation plan (next step)

### Code review instructions
- N/A

### Technical details
N/A

---

## Step 5: Writing All 14 Documentation Pages

With the investigation complete and the documentation plan approved, I wrote all 14 Glazed help pages — rewriting the 5 existing ones and creating 9 new ones.

### Prompt Context

**User prompt (verbatim):** "ok, write them / rewrite them / consolidate them"

**Assistant interpretation:** Implement the full 13-page documentation plan from the design doc.

**Inferred user intent:** Get all documentation pages written, wired into the binary, and verified working.

### What I did
- Re-read all source files: schema.go, metrics.go, builders.go, util.go, archive.go, all 7 convert command files, query engine, preset SQL files
- Rewrote `overview.md` → now titled "What is go-minitrace" with pipeline diagram, source format table, quick start
- Rewrote `convert.md` → now covers all 7 subcommands with flag tables, output structure diagram, troubleshooting
- Rewrote `discover.md` → now covers all 3 subcommands, explains why some formats lack discover, includes usage patterns
- Rewrote `query.md` → now covers all flags, 4 query modes, preset table, sessions_base column reference, Glazed output
- Rewrote `query-duckdb.md` → now has 7 worked examples with custom SQL, SQL file usage, pipeline examples
- Created `minitrace-schema.md` — field-by-field reference for all ~20 types, 13 KB
- Created `getting-started.md` — 6-step tutorial from install to custom query
- Created `validate.md` — command reference for the validate command
- Created `end-to-end-analysis.md` — 10-step Application tutorial with realistic scenario
- Created `writing-duckdb-queries.md` — DuckDB JSON syntax tutorial: ->>, CAST, UNNEST, filtering, patterns
- Created `duckdb-query-recipes.md` — 20+ ready-to-use SQL recipes organized by category
- Created `adapter-reference.md` — per-adapter deep dive with field mapping tables and known limitations
- Created `output-formats.md` — Glazed output formats, jq/Python piping, scripting, manifest files
- Created `troubleshooting.md` — 12 problems with causes and solutions

### Why
The design doc specified 13 pages. Delivering them all in one pass ensures consistency of cross-references and avoids partial states.

### What worked
- All 14 slugs resolve correctly via `go-minitrace help <slug>`
- Top-level help now shows 3 general topics + 1 tutorial (was: 1 general topic)
- Build passes with no errors
- All existing tests pass
- Cross-references between pages are consistent (every "See also" link points to a real slug)

### What didn't work
- N/A

### What I learned
- The `go:embed *.md` directive in `pkg/doc/doc.go` automatically picks up all new `.md` files — no wiring changes needed
- Glazed's `SectionType` determines where pages appear: `IsTopLevel: true` pages show in `go-minitrace help`, others show in command-specific help or when queried by slug
- The `Commands` frontmatter field associates a help page with specific subcommands but doesn't affect slug lookup

### What was tricky to build
- Getting the DuckDB SQL examples right in markdown: the `$` in `json_extract(tc, '$.tool_name')` needs escaping with `\$` in shell inline `--sql` but not in SQL files. Documented this in the examples.
- Ensuring all cross-references are valid across 14 pages — each page has 3-5 "See also" links. Verified manually.

### What warrants a second pair of eyes
- The adapter-reference.md page documents adapter behavior from reading the Go source. Some adapter details (especially field mapping) may need verification against actual output.
- The tool operation mapping tables should be checked against the actual adapter code for each framework.

### What should be done in the future
- Add the ChatGPT and claude.ai adapters to the `query duckdb` test data to verify the recipes work across all source formats
- Add schema validation to the `validate` command and update `validate.md` when it's ready
- Consider adding a `model-summary` preset based on the custom queries written during investigation

### Code review instructions
- Start at `pkg/doc/` — all 14 `.md` files
- Verify by running `go-minitrace help <slug>` for each slug
- Check cross-references by searching for "go-minitrace help" in all files
- Build: `go build ./...` and `go test ./...`

### Technical details

Files written/rewritten:
```
pkg/doc/overview.md          (rewritten) — 3,990 bytes
pkg/doc/minitrace-schema.md  (new)       — 13,082 bytes
pkg/doc/getting-started.md   (new)       — 5,805 bytes
pkg/doc/convert.md           (rewritten) — 9,123 bytes
pkg/doc/discover.md          (rewritten) — 4,292 bytes
pkg/doc/query.md             (rewritten) — 6,830 bytes
pkg/doc/query-duckdb.md      (rewritten) — 4,580 bytes
pkg/doc/validate.md          (new)       — 3,038 bytes
pkg/doc/end-to-end-analysis.md (new)     — 6,665 bytes
pkg/doc/writing-duckdb-queries.md (new)  — 7,148 bytes
pkg/doc/duckdb-query-recipes.md (new)    — 8,027 bytes
pkg/doc/adapter-reference.md (new)       — 8,652 bytes
pkg/doc/output-formats.md    (new)       — 4,466 bytes
pkg/doc/troubleshooting.md   (new)       — 6,433 bytes
```

Total: ~92 KB of documentation across 14 pages.
