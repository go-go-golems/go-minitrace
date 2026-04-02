---
title: minitrace Improvement Suggestions
doc-type: analysis
ticket: WESEN-OS-001
status: active
intent: long-term
topics:
  - go-minitrace
  - analysis
---

# go-minitrace Improvement Suggestions

Running notes collected during WESEN-OS-001 investigation. These are direct pain points and friction points encountered doing real workflow work against the tool.

---

## 1. Convert: fails hard on first `unknown-jsonl` session, no skip option

**What happened:**
```
go-minitrace convert codex --source-dir ~/.codex --output-dir /tmp/out
Error: converting Codex session rollout-2025-08-27T08-42-14-...: unsupported Codex format hint: unknown-jsonl
```

**Impact:** High — every user who has any pre-`session-jsonl-v1` sessions in their `~/.codex` directory will hit this on their first convert attempt. The command aborts and writes nothing.

**Suggested fixes (in priority order):**
1. **`--skip-unsupported` flag** (easiest, minimal change): skip sessions with unsupported format hints, emit a warning per skipped session, continue.
2. **`--from-date YYYY-MM-DD` / `--to-date YYYY-MM-DD` flags** (best ergonomics): filter sessions by the date encoded in the filename before attempting to convert. This is a natural fit since the session IDs are timestamped.
3. **Emit a summary** at the end: "Converted: 86, Skipped: 52 (unknown-jsonl), Errors: 0."

**Workaround documented in diary:** Copy the target date-range directories to a scratch location with only supported sessions present.

---

## 2. Discover: does not traverse symlinked directories

**What happened:** Created a temp dir with directory symlinks pointing to `~/.codex/sessions/2026/03` and `04`, then ran `go-minitrace discover codex --source-dir /tmp/codex-symlinked`. Result: zero sessions found. The walk does not follow directory symlinks.

**Impact:** Medium — a natural workaround for issue #1 would be to symlink only the supported date ranges. That workaround doesn't work, forcing actual file copies.

**Suggested fix:** Either follow symlinks during the session walk (with cycle detection), or document the limitation prominently in `discover --help`.

---

## 3. Query: `started_at` is not a top-level column — causes confusing errors

**What happened:** The `session-list` preset output column is named `started_at`, which led me to try:
```sql
SELECT started_at FROM sessions_base
-- Error: Referenced column "started_at" not found
```
The correct form is `timing->>'started_at'`. But the preset output makes it look like a real column.

**Impact:** Low-medium — confusing for anyone who sees the session-list output and tries to write follow-up SQL.

**Suggested fix:**
- Either expose `timing->>'started_at' AS started_at` as a computed column in the table definition (DuckDB supports this via a view).
- Or add a note in the `writing-duckdb-queries` help page: "The columns you see in preset output are aliases; the raw table uses JSON extraction. See the column mapping table."
- **Bonus:** expose a pre-built view `sessions_flat` with all the commonly-used JSON fields already extracted.

---

## 4. Documentation: DuckDB JSON cheatsheet is missing

**What happened:** The `query-duckdb` and `writing-duckdb-queries` help pages show good examples but don't have a compact cheatsheet of:
- 1-indexed array syntax: `turns[1]` (NOT `turns[0]`)
- JSON scalar extraction: `col->>'field'`
- JSON object extraction (for passing to further functions): `col->'field'`
- UNNEST with ordinality: `CROSS JOIN UNNEST(turns) WITH ORDINALITY AS t(turn, idx)`
- NULL-safe cast: `CAST(col->>'field' AS VARCHAR)` (returns NULL, not error, if field absent)

**Impact:** Low — but would save every new user ~30 minutes of trial and error.

**Suggested fix:** Add a "DuckDB JSON quick reference" section at the top of the `writing-duckdb-queries` help page.

---

## 5. Convert output: column truncation in table output makes session paths unreadable

**What happened:** The default `--output table` for `convert codex` truncates all paths and titles to fit the terminal. The `session_path` column is particularly important (it's the path to the output file for further use) but gets cut to ~30 characters.

**Impact:** Low — but switching to `--output json` just to get readable paths is friction.

**Suggested fix:**
- Add a `--no-truncate` flag to the table formatter.
- Or: the convert command could emit `session_path` last so at least the ID and quality columns are always visible.

---

## 6. No date range filtering in `discover codex`

**What happened:** `go-minitrace discover codex` returns every session ever. There's no `--from-date` or `--since` flag.

**Impact:** Low-medium — for users with years of sessions this list gets very long. More importantly, it makes the convert limitation (issue #1) harder to work around, since you can't even scope discovery to a date range before convert.

**Suggested fix:** Add `--from-date YYYY-MM-DD` / `--to-date YYYY-MM-DD` flags to both `discover codex` and `convert codex`.

---

## 7. `--preset session-list` truncates titles to ~80 chars with no override

**What happened:** The `session-list` preset truncates the title column. For sessions with long prompts (common with wesen-os work), the title is truncated and you can't see what the session is actually about.

**Impact:** Low — but makes the preset less useful for first-pass triage.

**Suggested fix:** Allow `--fields title` to override and show full-length title. Or add a `--full-titles` flag to the preset.

---

## 8. `summary` field is empty in all sessions

**What happened:** Every session row has `summary = NULL`. The schema has this column but it is never populated by the codex converter.

**Impact:** Low for now, but the `summary` field would be the killer feature for doing semantic search / "find sessions about X" without needing to UNNEST all turns and do LIKE on content.

**Suggested fix:**
- For a quick win: populate `summary` with the first 500 chars of the first user turn at convert time (the current title is already doing this, but `summary` could give more space).
- For a better win: the codex JSONL files may already contain a `summary` field — check if it's being dropped during conversion.
- For the ideal win: an LLM-generated summary pass as a post-convert step (`go-minitrace enrich --archive-glob ...`).

---

## 9. `operational_context.working_directory` uses `~` home shorthand, not absolute paths

**What happened:** `working_directory` values look like `~/workspaces/2026-03-02/os-openai-app-server/wesen-os`. For filtering this is fine (LIKE `%wesen-os%` works), but for any tool that tries to open/navigate to the directory, you'd need to expand `~`.

**Impact:** Very low.

**Suggested fix:** Normalize to absolute paths during conversion, or provide both `working_directory` (original) and `working_directory_abs` (expanded).

---

## 10. No cross-session search across turn content (full-text)

**What happened:** To find sessions about wesen-os that don't have "wesen-os" in the title, we had to do `LOWER(CAST(turns[1]->>'content' AS VARCHAR)) LIKE '%wesen-os%'`. This only searches the *first* user turn. Sessions that start with a follow-up question or a `cd` command won't match even if most of their content is about wesen-os.

**Impact:** Medium — for any real "find sessions about X" use case, you want full-text search across all turns.

**Suggested fix:**
- A `go-minitrace query duckdb --preset session-search --search-term "wesen-os"` preset that UNNESTs all turns and does full-text matching, then deduplicates to one row per session.
- Or: a `go-minitrace search` top-level command that uses a trigram/FTS index.

---

## Summary Table

| # | Issue | Impact | Effort | Priority |
|---|---|---|---|---|
| 1 | convert fails on unknown-jsonl, no skip | High | Low | P0 |
| 2 | discover doesn't follow symlinks | Medium | Low | P1 |
| 3 | started_at not a top-level column | Medium | Low | P1 |
| 4 | DuckDB JSON cheatsheet missing | Low | Low | P2 |
| 5 | Table output truncates session paths | Low | Low | P2 |
| 6 | No date range filter in discover | Medium | Low | P1 |
| 7 | session-list truncates titles | Low | Low | P3 |
| 8 | summary field always NULL | Medium | Medium | P1 |
| 9 | workdir uses ~ shorthand | Low | Low | P3 |
| 10 | No full-turn full-text search | Medium | High | P2 |
