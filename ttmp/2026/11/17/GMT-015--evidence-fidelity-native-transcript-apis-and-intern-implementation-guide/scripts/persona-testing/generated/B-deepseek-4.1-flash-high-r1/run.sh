#!/usr/bin/env bash
# Reproduce every observation in report.md. Read-only against the repo/corpus.
set -u
REPO=/Users/manuel.odendahl/code/go-go-golems/go-minitrace/.worktrees/feat-evidence-fidelity
R=/private/tmp/gmt015-persona/matrix/B-deepseek-4.1-flash-high-r1
FIX=/private/tmp/gmt015-persona/inputs
MT="${MT:-$R/bin/go-minitrace}"   # or: (cd $REPO && go run ./cmd/go-minitrace)
mkdir -p "$R/out"

# --- JS probes (query-command repository) ---------------------------------
for v in utf8 clip strict overflow toolrows tokens manifest envelopes; do
  "$MT" query commands --query-repository "$R/qrepo" probe fidelity "$v" \
    --archive-glob "$FIX/archive.json" --output json \
    > "$R/out/fidelity-$v.json" 2> "$R/out/fidelity-$v.err"
  echo "fidelity $v exit=$?"
done

# --- token/tool summary on one real framework corpus each ------------------
for f in claude-code codex pi; do
  "$MT" query commands --query-repository "$R/qrepo" probe corpus summary \
    --glob "/private/tmp/gmt015-persona/corpus/archives/$f/active/*/*.minitrace.json" \
    --label "$f" --output json > "$R/out/corpus-$f.json" 2> "$R/out/corpus-$f.err"
  echo "corpus $f exit=$?"
done

# --- CLI in-memory / no-disk-cache queries --------------------------------
G=/private/tmp/gmt015-persona/inputs/archive.json
"$MT" query run --archive-glob "$G" --cache-mode memory \
  --sql "SELECT length(CAST('abé' AS BLOB)) AS bytes, length('abé') AS chars" --output json
"$MT" query run --archive-glob "$G" --cache-mode none \
  --sql "SELECT COUNT(*) AS tools FROM tool_calls" --output json
"$MT" query run --archive-glob "$G" --cache-mode memory --max-cell-bytes 3 \
  --sql "SELECT 'abé' AS text" --output json            # emits "ab", NO truncation flag
"$MT" query run --archive-glob "$G" --cache-mode memory --max-cell-bytes 3 --strict-evidence \
  --sql "SELECT 'abé' AS text" --output json; echo "strict exit=$?"   # stderr, exit 1
"$MT" query run --archive-glob "$G" --cache-mode memory --max-rows 2 \
  --sql "SELECT tool_call_id FROM tool_calls ORDER BY tool_call_id" --output json
"$MT" query run --archive-glob "$G" --cache-mode memory \
  --sql "SELECT nope FROM missing_table" --output json; echo "invalid exit=$?"
# cache-mode auto + --cache-dir DOES write a sqlite file (contrast):
"$MT" query run --archive-glob "$G" --cache-mode auto --cache-dir "$R/cacheprobe-auto" \
  --sql "SELECT COUNT(*) AS tools FROM tool_calls" --output json
ls -la "$R/cacheprobe-auto"

# --- status divergence across all nine corpus archives -------------------
"$MT" query run --archive-glob '/private/tmp/gmt015-persona/corpus/archives/*/active/*/*.minitrace.json' \
  --cache-mode memory --output json --sql \
  "SELECT s.agent_framework AS fw, t.output_status, t.outcome_status, COUNT(*) AS n
   FROM tool_calls t JOIN sessions s ON s.session_id=t.session_id GROUP BY 1,2,3 ORDER BY 1,4 DESC"
