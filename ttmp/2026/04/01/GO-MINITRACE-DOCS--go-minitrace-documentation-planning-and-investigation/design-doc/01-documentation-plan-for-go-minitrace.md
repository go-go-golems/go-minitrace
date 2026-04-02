---
Title: Documentation Plan for go-minitrace
Ticket: ""
Status: active
Topics:
    - minitrace
    - documentation
    - glazed
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/go-minitrace/main.go
      Note: CLI entrypoint with HelpSystem setup
    - Path: pkg/doc/doc.go
      Note: Embed wiring for help pages
    - Path: pkg/doc/overview.md
      Note: Existing overview page (to be rewritten)
    - Path: pkg/minitrace/schema.go
      Note: Core minitrace schema types
    - Path: pkg/query/engine.go
      Note: DuckDB query engine
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---






# Documentation Plan for go-minitrace

## Executive Summary

go-minitrace has 5 existing Glazed help pages that provide minimal coverage. This plan proposes **13 new or rewritten help pages** organized into four tiers: concepts, command references, tutorials, and advanced topics. Each page is specified with its Glazed frontmatter, content outline, and target audience. The pages will live in `pkg/doc/` and be embedded into the binary via the existing `go:embed` + `LoadSectionsFromFS` wiring.

## Problem Statement

The current documentation covers:
- A brief overview listing capabilities
- Skeletal descriptions of the convert, discover, and query command groups
- A list of DuckDB presets

What's missing:
- **No conceptual introduction** — A new user cannot learn what minitrace is, what the schema means, or how the discover→convert→query pipeline works
- **No schema reference** — The Session type has 20+ top-level fields; none are documented for end users
- **No per-adapter documentation** — 6 adapter backends exist; only Claude Code and Codex are mentioned in the convert page
- **No workflow tutorials** — No end-to-end examples from raw sessions to insights
- **No query-writing guide** — Users can't learn DuckDB JSON operators or minitrace-specific patterns
- **No troubleshooting** — Common errors (e.g., codex unknown-jsonl) are undocumented

## Current State

### Existing pages in `pkg/doc/`

| File | Slug | SectionType | Content depth |
|------|------|-------------|---------------|
| `overview.md` | `go-minitrace-overview` | GeneralTopic (TopLevel) | Bullet list of capabilities, no explanation |
| `convert.md` | `convert-commands` | GeneralTopic | 2 examples, only Claude Code + Codex |
| `discover.md` | `discover-commands` | Tutorial | 2 examples, no explanation of output |
| `query.md` | `query-commands` | Tutorial | 2 examples, minimal |
| `query-duckdb.md` | `query-duckdb` | Example | Lists 6 presets, no query-writing guidance |

### Architecture (evidence-based)

The tool follows a three-stage pipeline:

```
Discover → Convert → Query
  │          │         │
  │          │         └─ DuckDB presets or custom SQL
  │          └─ Raw sessions → minitrace JSON archives
  └─ Scan native stores, emit session metadata rows
```

**Adapters** (6):
- `claudecode` — JSONL v2 + dir-v1 tool-results + subagents
- `codex` — session JSONL + exec JSONL
- `pi` — JSONL v3
- `claudeai` — privacy export ZIP
- `chatgpt` — data export ZIP
- `chatgpt` (JSON) — per-conversation JSON transcripts

**Discovery** (3): claude-code, codex, pi

**Query** (1): DuckDB with 6 presets, inline SQL, SQL file, configurable table name

### Codebase metrics

- ~10,250 lines of Go across ~40 files
- Largest adapters: codex (877 lines), claudecode (814 lines), turnsdb (786 lines)
- Schema: 250 lines defining ~20 types
- Query engine: 169 lines + 6 embedded SQL presets

### Real-data profile (from investigation)

- 2,502 Claude Code sessions + 710 subagent sessions = 3,212 total
- 1,174 Codex sessions discovered (conversion has edge cases)
- 52 Pi sessions
- 7 turnsdb (Pinocchio) sessions
- Models: predominantly claude-opus-4-6 (2,590 sessions)
- Tool usage: Read (10k), Bash (9.9k), bash (5.4k), Edit (3.3k)

## Proposed Documentation Set

### Tier 1: Concepts (GeneralTopic, IsTopLevel: true)

These pages explain what minitrace is and how the tool works conceptually. They should be the first thing a new user reads.

---

#### Page 1: `what-is-minitrace.md` (REWRITE of overview.md)

**Slug:** `what-is-minitrace`
**SectionType:** GeneralTopic
**IsTopLevel:** true
**ShowPerDefault:** true

**Content outline:**
1. What is minitrace — a structured format for recording AI agent sessions across frameworks
2. The minitrace schema — sessions contain turns, tool calls, metrics, annotations, provenance
3. The three-stage pipeline — discover → convert → query
4. Supported source formats — table of 6 adapters with what each one reads
5. Output format — `active/YYYY-MM/<id>.minitrace.json` + `manifest.json`
6. Quick start — 3-command workflow (discover, convert, query)
7. Where to read next — cross-references to other help pages

**Why this matters:** The current overview is a bullet list. A new user has no idea what minitrace is or why they'd use it.

---

#### Page 2: `minitrace-schema.md`

**Slug:** `minitrace-schema`
**SectionType:** GeneralTopic
**IsTopLevel:** true
**ShowPerDefault:** true

**Content outline:**
1. Schema version — currently `minitrace-v0.2.0`
2. Session — top-level container with id, title, classification, quality grade (A/B/C)
3. Provenance — source_format, source_path, converter_version, original_session_id
4. Environment — model, agent_framework, platform_type, tools_enabled
5. Timing — started_at, ended_at, duration_seconds, active_duration_seconds, privacy_level
6. Turns — index, role (user/assistant/system), content, usage (token counts), thinking
7. Tool calls — tool_name, operation_type (READ/MODIFY/NEW/EXECUTE/DELEGATE/OTHER), input/output, spawned_agent
8. Metrics — turn_count, tool_call_count, read_ratio, time_to_first_action, token totals, subagent counts
9. Annotations — annotator, scope, content (category/tags/title/detail), taxonomy mappings
10. Coordination — project_id, predecessor_session, human_attention
11. Flags — for_research, needs_cleaning, contains_error, contains_pii
12. Field reference table — all top-level fields with types and descriptions

**Why this matters:** The schema is the core concept. Every query, every analysis, every extension depends on understanding it. The 250-line Go struct is not documentation.

---

### Tier 2: Command References (GeneralTopic)

These pages document each command group in detail. They replace the existing skeletal pages.

---

#### Page 3: `convert-commands.md` (REWRITE)

**Slug:** `convert-commands`
**SectionType:** GeneralTopic
**ShowPerDefault:** true
**Commands:** convert, convert claude-code, convert codex, convert pi, convert claude-ai, convert chatgpt, convert chatgpt-json, convert turnsdb

**Content outline:**
1. What the convert group does — reads native session stores, writes minitrace JSON archives
2. Output directory structure — `output/active/YYYY-MM/<id>.minitrace.json` + `manifest.json`
3. Common flags — `--output-dir`, `--dry-run`, `--output`
4. Per-subcommand reference:
   - `convert claude-code` — source: `~/.claude/projects`, formats: JSONL v2, dir-v1 tool-results, subagents with parent backlinking
   - `convert codex` — source: `~/.codex`, formats: session JSONL, exec JSONL
   - `convert pi` — source: `~/.pi/agent/sessions`, format: JSONL v3
   - `convert claude-ai` — source: privacy export ZIP, notes on how to get the export
   - `convert chatgpt` — source: data export ZIP from ChatGPT settings
   - `convert chatgpt-json` — source: per-conversation JSON files with tool-call extraction
   - `convert turnsdb` — source: SQLite turns.db, `--conv-id` for single conversation, snapshot diffing algorithm
5. Quality grades — how A/B/C quality is assigned
6. Manifest format
7. Troubleshooting table — unknown format hints, empty sessions, permission errors

**Why this matters:** The current page mentions only 2 of 7 subcommands and has no explanation of output format or per-adapter behavior.

---

#### Page 4: `discover-commands.md` (REWRITE)

**Slug:** `discover-commands`
**SectionType:** GeneralTopic
**ShowPerDefault:** true
**Commands:** discover, discover claude-code, discover codex, discover pi

**Content outline:**
1. What discovery does — scans native stores, emits one row per source with id, format_hint, source_path
2. When to use it — before conversion, to count sessions, to check source directories, to filter what to convert
3. Per-subcommand:
   - `discover claude-code` — scans `~/.claude/projects/`, finds JSONL + dir-v1 sessions
   - `discover codex` — scans `~/.codex/`, finds session + exec JSONL
   - `discover pi` — scans `~/.pi/agent/sessions/`, finds JSONL v3 files
4. Output format — JSON array of `{id, format_hint, source_path}`
5. Using discovery output — piping to jq, counting sessions, filtering by format
6. Why some converters don't have discover — claude.ai, chatgpt, turnsdb take explicit source files

**Why this matters:** Discovery is the first step in the pipeline. Users need to understand what it shows them and how to use it.

---

#### Page 5: `query-commands.md` (REWRITE)

**Slug:** `query-commands`
**SectionType:** GeneralTopic
**ShowPerDefault:** true
**Commands:** query, query duckdb

**Content outline:**
1. What the query group does — loads minitrace archives into an analysis backend and runs queries
2. The DuckDB backend — currently the only backend
3. How loading works — `read_json()` with explicit column schema, `ignore_errors=true`
4. The `sessions_base` table — what columns are available, how nested JSON is accessed
5. Query modes — preset, inline SQL, SQL file, load-only
6. Flags reference table — archive-glob, db-path, table-name, preset, sql, sql-file, load-only, persist-loaded
7. Available presets — table with name + what each one shows
8. Using Glazed output formatting — `--output json`, `--output yaml`, `--output csv`, `--fields`
9. Combining with external tools — piping JSON to jq, Python, etc.

**Why this matters:** The query command is the primary analysis interface. It needs comprehensive documentation of its options and modes.

---

#### Page 6: `validate-command.md` (NEW)

**Slug:** `validate-command`
**SectionType:** GeneralTopic
**ShowPerDefault:** true
**Commands:** validate

**Content outline:**
1. What validate does — checks JSON syntax of minitrace files
2. Current scope — JSON syntax validation only (schema validation is planned)
3. Flags — `--path`, `--recursive`
4. Output fields — path, valid_json, error
5. Validating a converted archive
6. Future: full schema validation

**Why this matters:** Validate has no help page at all currently.

---

### Tier 3: Tutorials (Tutorial)

Step-by-step guides for common workflows.

---

#### Page 7: `getting-started.md` (NEW)

**Slug:** `getting-started`
**SectionType:** Tutorial
**IsTopLevel:** true
**ShowPerDefault:** true

**Content outline:**
1. Install go-minitrace (Homebrew, go install)
2. Discover what sessions you have — run discover commands for each source you have
3. Convert your sessions — pick one adapter, convert to a local directory
4. Run your first query — `query duckdb --preset session-list`
5. Explore with presets — framework-summary, tool-operation-breakdown
6. Write a custom query — simple COUNT, then a GROUP BY model
7. Next steps — links to schema reference, query writing guide, per-adapter details

**Why this matters:** New users need a single "do this first" path, not a reference dump.

---

#### Page 8: `end-to-end-analysis.md` (NEW)

**Slug:** `end-to-end-analysis`
**SectionType:** Application
**ShowPerDefault:** true

**Content outline:**
1. Scenario — "I want to understand my Claude Code usage over the past month"
2. Discover sessions — count, inspect format hints
3. Convert to archive — full conversion with default output directory
4. Verify conversion — validate JSON, check manifest
5. Aggregate analysis — framework-summary, timing-analysis presets
6. Focused analysis — custom SQL for model usage, token costs, tool frequency
7. Cross-framework comparison — convert Pi sessions too, compare
8. Saving results — `--output csv` for spreadsheets, `--output json` for further processing
9. Using external DuckDB — `duckdb analysis.duckdb -init queries/load.sql` for interactive exploration

**Why this matters:** Shows the complete pipeline in a realistic scenario with real questions and real answers.

---

### Tier 4: Advanced Topics (GeneralTopic or Example)

Deep dives into specific capabilities.

---

#### Page 9: `writing-duckdb-queries.md` (NEW)

**Slug:** `writing-duckdb-queries`
**SectionType:** Tutorial
**ShowPerDefault:** true

**Content outline:**
1. DuckDB JSON access syntax — `->>'field'` for string extraction, `->` for JSON sub-objects
2. The sessions_base schema — what each column contains and its DuckDB type
3. Accessing nested fields — `environment->>'model'`, `metrics->>'turn_count'`
4. Casting — `CAST(metrics->>'turn_count' AS INT)`, `ROUND(CAST(... AS DOUBLE), 2)`
5. Working with arrays — `UNNEST(tool_calls)`, `UNNEST(turns)`, `UNNEST(annotations)`
6. Common query patterns:
   - Sessions by model
   - Tool frequency
   - Token usage
   - Time-based analysis (hour_of_day, day_of_week)
   - Subagent vs. main session filtering
7. Using `--sql-file` for reusable queries
8. Interactive DuckDB workflows — `--db-path`, `--persist-loaded`, external duckdb CLI
9. Performance tips — `ignore_errors=true`, column pruning with explicit `read_json` columns
10. Example queries gallery — 8-10 ready-to-use queries

**Why this matters:** DuckDB's JSON syntax is non-obvious. Users need to learn the `->>'field'` operator and UNNEST patterns to write useful queries.

---

#### Page 10: `adapter-reference.md` (NEW)

**Slug:** `adapter-reference`
**SectionType:** GeneralTopic
**ShowPerDefault:** true

**Content outline:**
1. What adapters do — translate native session formats into minitrace schema
2. Per-adapter deep dive:
   - **Claude Code**: JSONL v2 format, dir-v1 tool-results, subagent detection and parent backlinking, what gets mapped where
   - **Codex**: session JSONL vs exec JSONL, how format_hint discovery works, known limitations (unknown-jsonl)
   - **Pi**: JSONL v3 format, session directory naming convention, token mapping
   - **claude.ai**: ZIP export structure, conversation → session mapping, model detection
   - **ChatGPT (ZIP)**: Export structure, conversation parts → turns, tool mapping
   - **ChatGPT (JSON)**: Per-file transcript format, tool-call extraction
   - **turnsdb**: SQLite schema, snapshot-based diffing algorithm, conversation isolation with --conv-id
3. Fidelity notes — what gets lost in conversion, what gets synthesized
4. Quality grading — how each adapter assigns A/B/C quality

**Why this matters:** Users converting data from a specific tool need to know what to expect and what's preserved.

---

#### Page 11: `duckdb-query-recipes.md` (NEW)

**Slug:** `duckdb-query-recipes`
**SectionType:** Example
**ShowPerDefault:** true

**Content outline:**
1. Using the `queries/` directory with external duckdb
2. Loading workflow — `duckdb analysis.duckdb -init queries/load.sql`
3. Recipe: Session list with key metadata
4. Recipe: Framework comparison
5. Recipe: Model usage breakdown
6. Recipe: Token cost estimation
7. Recipe: Tool usage heatmap (tool × framework)
8. Recipe: Temporal analysis (sessions by hour, by day)
9. Recipe: Subagent analysis (parent/child relationships)
10. Recipe: Finding long/expensive sessions
11. Recipe: Annotation extraction
12. Writing your own recipes — conventions, table name, the `queries/` directory pattern

**Why this matters:** The `queries/` directory exists but has minimal documentation. Providing a recipe book makes the tool immediately useful for analysis.

---

#### Page 12: `output-formats-and-pipelines.md` (NEW)

**Slug:** `output-formats-and-pipelines`
**SectionType:** Example
**ShowPerDefault:** true

**Content outline:**
1. Glazed output formats — json, yaml, csv, table, markdown
2. Field selection — `--fields`
3. Piping to jq — filtering, transforming query results
4. Piping to Python — analysis scripts
5. CSV to spreadsheet workflows
6. Combining discover + convert — scripting full pipeline
7. The manifest file — what it contains, how to use it

**Why this matters:** The tool produces structured output in multiple formats. Users need to know how to use these in real workflows.

---

#### Page 13: `troubleshooting.md` (NEW)

**Slug:** `troubleshooting`
**SectionType:** GeneralTopic
**ShowPerDefault:** true
**IsTopLevel:** true

**Content outline:**

| Problem | Cause | Solution |
|---------|-------|----------|
| `unsupported Codex format hint: unknown-jsonl` | Older Codex session with unrecognized format | Skip or update the adapter |
| Empty convert output | Source directory doesn't contain expected file patterns | Run discover first to verify sessions exist |
| DuckDB query returns no rows | archive-glob doesn't match any files | Check the glob path, use `--load-only` to verify loading |
| Large output from presets | Some presets return per-session rows | Pipe to `head` or add LIMIT in custom SQL |
| JSON parse errors during loading | Malformed minitrace files | Run validate first, check for truncated files |
| Permission denied on source | Session files owned by different user | Check file permissions |
| Slow conversion | Large number of sessions | Convert in batches by source directory |

**Why this matters:** Users hit errors. A troubleshooting page saves time and reduces frustration.

---

## Implementation Plan

### Phase 1: Core concepts (Pages 1, 2, 7)
Rewrite the overview, add schema reference, add getting-started tutorial.
These three pages make the tool usable for a new user.

### Phase 2: Command references (Pages 3, 4, 5, 6)
Rewrite convert, discover, query pages; add validate page.
These make every command well-documented.

### Phase 3: Tutorials and recipes (Pages 8, 9, 11)
Add end-to-end analysis tutorial, query writing guide, recipe book.
These make the tool productive for real analysis.

### Phase 4: Advanced topics (Pages 10, 12, 13)
Add adapter reference, output format guide, troubleshooting.
These cover the long tail of user needs.

### File placement

All files go in `pkg/doc/` and are automatically embedded via the existing `go:embed *.md` directive in `pkg/doc/doc.go`.

### Naming convention

Current files use short names (`overview.md`, `convert.md`). New files should follow the same pattern:
- `what-is-minitrace.md` (replaces `overview.md`)
- `minitrace-schema.md`
- `getting-started.md`
- `convert.md` (rewrite in place)
- `discover.md` (rewrite in place)
- `query.md` (rewrite in place)
- `query-duckdb.md` (rewrite in place → merged into query.md or kept as separate advanced page)
- `validate.md`
- `end-to-end-analysis.md`
- `writing-duckdb-queries.md`
- `duckdb-query-recipes.md`
- `adapter-reference.md`
- `output-formats-and-pipelines.md`
- `troubleshooting.md`

## Risks and Open Questions

1. **Page count**: 13 pages is substantial. Could be phased to avoid scope creep.
2. **Schema stability**: The schema is v0.2.0. If it changes, documentation needs updating. Mitigate by linking to the Go types as ground truth.
3. **Adapter edge cases**: Some adapters have undocumented behavior. Writing the adapter reference may expose bugs that need fixing first.
4. **DuckDB version coupling**: DuckDB JSON syntax may evolve. Document the tested version.
5. **Should `query-duckdb.md` stay separate from `query.md`?**: Currently separate. Could merge since DuckDB is the only backend. Recommend keeping separate for now since more backends may come.

## References

### Key source files
- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/doc/doc.go` — embed wiring
- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/doc/*.md` — existing help pages
- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitrace/schema.go` — schema types
- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/query/engine.go` — query engine
- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/query/assets.go` — preset registry
- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/main.go` — CLI entrypoint
- `/home/manuel/code/wesen/corporate-headquarters/go-minitrace/queries/` — standalone DuckDB recipes

### Investigation artifacts
- `scripts/explore-sessions.sql` — exploratory SQL queries
- `scripts/adhoc-queries.sql` — custom queries run during investigation
- `scripts/full-workflow-demo.sh` — end-to-end workflow demo script
