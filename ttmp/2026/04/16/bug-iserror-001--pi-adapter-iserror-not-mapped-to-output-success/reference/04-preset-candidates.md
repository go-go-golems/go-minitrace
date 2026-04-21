# Preset Candidates: SQL Queries Worth Integrating

## Assessment Criteria

A query belongs as a preset if it answers a **general question** analysts will ask repeatedly, regardless of which session they're investigating. Session-specific WHERE clauses (like `LIKE '%jellyfin-001%'`) are scaffolding, not presets.

## Existing Presets (6)

| Preset | What It Does |
|--------|-------------|
| `framework-summary` | Sessions, turns, tool_calls by framework/model |
| `session-list` | Per-session overview with timing/metrics |
| `tool-operation-breakdown` | Tool calls by operation type (READ/MODIFY/NEW/EXECUTE) |
| `timing-analysis` | Duration, idle ratio, time-of-day patterns |
| `read-ratio-distribution` | How much agents read before acting |
| `annotations` | Annotation counts by category/framework |

## Existing Recipes (in help docs, not presets)

The `duckdb-query-recipes` help page already has inline recipes for:
- Session overview, model analysis, token analysis, tool analysis (including "Tool success rate"), timing, subagent analysis, annotation analysis, advanced patterns

## Gap Analysis: What's Missing

### Tier 1: Should Be Presets (general-purpose, reusable)

#### `file-operations` — Track all file mutations in a session

This was the most useful query for debugging. General form: show every write/edit/read on files, in order.

```sql
-- file-operations: Track file read/write/edit operations
-- Params: none (works on all sessions)
SELECT
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  json_extract(tc, '$.operation_type') AS operation,
  COALESCE(
    CAST(json_extract(tc, '$.input.file_path') AS VARCHAR),
    LEFT(CAST(json_extract(tc, '$.input.command') AS VARCHAR), 200)
  ) AS target,
  json_extract(tc, '$.output.success') AS success,
  LEFT(CAST(json_extract(tc, '$.output.error') AS VARCHAR), 200) AS error,
  json_extract(tc, '$.timestamp') AS timestamp
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') IN ('write', 'edit', 'read')
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT);
```

**Why:** Every "what happened to my files" investigation starts here. Currently there's no way to see file-level operations without writing this from scratch.

#### `tool-failures` — List all failed tool calls

The existing recipe "Tool success rate" gives an aggregate count. This gives you the actual failures with context.

```sql
-- tool-failures: All failed tool calls with error details
SELECT
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  json_extract(tc, '$.operation_type') AS operation,
  COALESCE(
    CAST(json_extract(tc, '$.input.file_path') AS VARCHAR),
    LEFT(CAST(json_extract(tc, '$.input.command') AS VARCHAR), 200)
  ) AS target,
  LEFT(CAST(json_extract(tc, '$.output.error') AS VARCHAR), 300) AS error,
  json_extract(tc, '$.timestamp') AS timestamp
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE
  json_extract(tc, '$.output.success') = false
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT);
```

**Why:** Essential for debugging. Currently impossible without writing custom SQL. The `output.success` bug in the Pi adapter makes this even more important — once fixed, people will want to see what was previously hidden.

#### `file-timeline` — Chronological operations on a specific file path

Generalized from `query-jellyfin-timeline.sql`. Takes a `{{PATH_FILTER}}` placeholder.

```sql
-- file-timeline: Chronological operations on matching file paths
-- Usage: replace {{PATH_FILTER}} with a LIKE pattern (e.g. '%diary%', '%.go')
SELECT
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  json_extract(tc, '$.operation_type') AS operation,
  COALESCE(
    CAST(json_extract(tc, '$.input.file_path') AS VARCHAR),
    LEFT(CAST(json_extract(tc, '$.input.command') AS VARCHAR), 120)
  ) AS target,
  json_extract(tc, '$.output.success') AS success,
  LEFT(
    COALESCE(
      CAST(json_extract(tc, '$.output.error') AS VARCHAR),
      CAST(json_extract(tc, '$.output.result') AS VARCHAR)
    ), 150
  ) AS result_preview,
  json_extract(tc, '$.timestamp') AS timestamp
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE
  COALESCE(
    CAST(json_extract(tc, '$.input.file_path') AS VARCHAR),
    CAST(json_extract(tc, '$.input.command') AS VARCHAR), ''
  ) LIKE '%{{PATH_FILTER}}%'
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT);
```

**Why:** Debugging "what happened to file X" is the #1 use case for tool call analysis. Currently requires custom SQL every time. The `{{PATH_FILTER}}` pattern is the same kind of parameterization that could use `--filter` or similar.

#### `turn-range` — Show all operations between two turns

Generalized from `query-gap-14-48.sql`. Useful for "what happened during this time window."

```sql
-- turn-range: Operations between {{START_TURN}} and {{END_TURN}}
SELECT
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  json_extract(tc, '$.operation_type') AS operation,
  COALESCE(
    CAST(json_extract(tc, '$.input.file_path') AS VARCHAR),
    LEFT(CAST(json_extract(tc, '$.input.command') AS VARCHAR), 120)
  ) AS target,
  json_extract(tc, '$.output.success') AS success,
  LEFT(
    COALESCE(
      CAST(json_extract(tc, '$.output.error') AS VARCHAR),
      CAST(json_extract(tc, '$.output.result') AS VARCHAR)
    ), 150
  ) AS result_preview
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) BETWEEN 0 AND 999999
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT);
```

**Why:** Less universally needed, but the pattern of "show me what happened around turn N" comes up constantly when investigating anomalies.

### Tier 2: Should Be Help Page Recipes (not presets, but documented)

These are more niche but would save people from the common `json_extract` patterns:

#### Schema Discovery
```sql
DESCRIBE sessions_base;
SELECT json_structure(tool_calls[1]) FROM sessions_base LIMIT 1;
SELECT DISTINCT typeof(json_extract(tc, '$.tool_name'))
FROM sessions_base, UNNEST(tool_calls) AS t(tc);
```
**Why:** Every new user hits the "JSON type vs string" issue. Adding a "Schema Discovery" section to `writing-duckdb-queries` would prevent this.

#### Tool Call with Result Content Search
```sql
-- Find tool calls whose result contains specific text
SELECT
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  LEFT(CAST(json_extract(tc, '$.output.result') AS VARCHAR), 200) AS result_preview
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE
  CAST(json_extract(tc, '$.output.result') AS VARCHAR) LIKE '%{{SEARCH}}%';
```
**Why:** People will want to search tool call outputs. This pattern isn't documented anywhere.

### Tier 3: Session-Specific Scaffolding (keep in ticket, not worth integrating)

| Query | Why Not a Preset |
|-------|-----------------|
| `query-docmgr-commands.sql` | Hardcoded `%docmgr%` filter — too specific |
| `query-docmgr-create-results.sql` | Hardcoded `%docmgr ticket create%` filter |
| `query-jellyfin-file-operations.sql` | Session-specific `%jellyfin-001%` filter |
| `query-jellyfin-file-operations-v2.sql` | Debugging dead end |
| `query-deletion-operations.sql` | Hardcoded turn range and filters |
| `query-gap-14-48.sql` | Hardcoded turn range |
| `query-gap-turns-30-50.sql` | Hardcoded turn range |
| `query-git-operations.sql` | Could be generalized into `bash-command-search` |
| `query-first-bash-calls.sql` | Session-specific debugging |
| `query-ttmp-file-operations.sql` | Dead code from initial attempt (wrong UNNEST syntax) |
| `query-write-structure.sql` | Schema exploration, one-time use |
| `query-tool-name-check.sql` | Schema exploration, one-time use |

### Tier 4: Existing Recipes to Verify

The `duckdb-query-recipes` help page already has a "Tool success rate" recipe:
```sql
-- from duckdb-query-recipes
### Tool success rate
```
I couldn't see the full recipe in the help output. Should verify it actually works
with UNNEST syntax (the doc examples use a different pattern than the presets).

## Implementation Plan

### Add to `pkg/query/presets/` and wire in `assets.go`:

| Preset Name | Source | Priority |
|-------------|--------|----------|
| `file-operations` | Generalized from `query-jellyfin-file-operations.sql` | High |
| `tool-failures` | Generalized from `query-all-failures.sql` | High |
| `file-timeline` | Generalized from `query-jellyfin-timeline.sql` | Medium |
| `turn-range` | Generalized from `query-gap-14-48.sql` | Low |

### Add to `duckdb-query-recipes` help page:

- Schema discovery section (DESCRIBE, json_structure)
- Tool result content search pattern
- File path filtering pattern

### Not worth integrating:

The 12 session-specific queries in `scripts/` — they served their purpose for the investigation but are too specific.
