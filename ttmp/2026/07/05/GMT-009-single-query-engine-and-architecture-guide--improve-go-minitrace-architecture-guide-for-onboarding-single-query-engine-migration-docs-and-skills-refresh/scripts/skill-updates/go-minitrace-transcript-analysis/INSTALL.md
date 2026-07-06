# Install: go-minitrace-transcript-analysis skill refresh

`~/.claude/skills/` is mounted read-only in the authoring environment, so this
updated skill is staged here. Apply it once that directory is writable.

This refresh reflects the single-query-engine migration: `query run` /
`query commands` on the normalized SQLite engine (DuckDB removed), the builder
JS API (`mt.db().RuntimeArchives().QueryCommandDefaults().Build()` — not
`mt.query()`/`mt.tableName`), first-class `discover --cwd-contains/--since`,
and `convert --source-session/--source-list`. The old grep-narrowing and
staging workflow is gone.

```bash
DST=~/.claude/skills/go-minitrace-transcript-analysis
SRC=<this dir>            # .../GMT-009-.../scripts/skill-updates/go-minitrace-transcript-analysis

cp "$SRC/SKILL.md"                  "$DST/SKILL.md"
cp "$SRC/references/queries.md"     "$DST/references/queries.md"
cp "$SRC/scripts/query_minitrace.sh" "$DST/scripts/query_minitrace.sh"

# Delete scripts made obsolete by native discover filters + --source-session/--source-list:
rm -f "$DST/scripts/discover_codex_by_cwd.sh" \
      "$DST/scripts/discover_pi_by_cwd.sh" \
      "$DST/scripts/stage_codex_by_cwd.sh" \
      "$DST/scripts/audit_manifests.sh"
```

Verify after install: `SKILL.md` teaches `query run` + the builder JS API and
contains no `mt.query(sql)` / `query duckdb` / `sessions_base`-as-table usage.
