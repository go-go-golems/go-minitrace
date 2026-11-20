#!/usr/bin/env bash
# Reproducible checks for the good/bad JS query-command repository.
set -uo pipefail
REPO=/Users/manuel.odendahl/code/go-go-golems/go-minitrace/.worktrees/feat-evidence-fidelity
HERE=/private/tmp/gmt015-persona/matrix/D-deepseek-4.1-flash-low-r2
CMD="go run ./cmd/go-minitrace"
A_CLAUDE='/private/tmp/gmt015-persona/corpus/archives/claude-code/active/*/*.minitrace.json'
A_CODEX='/private/tmp/gmt015-persona/corpus/archives/codex/active/*/*.minitrace.json'
A_PI='/private/tmp/gmt015-persona/corpus/archives/pi/active/*/*.minitrace.json'
cd "$REPO" || exit 1

# 1. Ordinary help: required source flag + runtime archive flags.
$CMD query commands good --query-repository "$HERE/commands" --help \
  > "$HERE/good-help.txt" 2>&1

# 2. Catalog diagnostic for the intentional signed-default sibling.
$CMD query commands --query-repository "$HERE/commands" --help \
  > "$HERE/catalog-help.txt" 2>&1

# 3. good command over one real archive per Claude/Codex/Pi.
$CMD query commands good --query-repository "$HERE/commands" \
  --archive-glob "$A_CLAUDE" --archive-glob "$A_CODEX" --archive-glob "$A_PI" \
  --source corpus-mixed --output json > "$HERE/good.json" 2> "$HERE/good.err"

# 4. Contrast array-return formatting vs a raw JS object value.
for v in array-return raw-value; do
  $CMD query commands contrast "$v" --query-repository "$HERE/commands" \
    --archive-glob "$A_PI" --output json > "$HERE/contrast-$v.json" 2>&1
  $CMD query commands contrast "$v" --query-repository "$HERE/commands" \
    --archive-glob "$A_PI" --output table > "$HERE/contrast-$v.txt" 2>&1
done

# 5. CLI cache-mode memory + run receipt.
$CMD query run --archive-glob "$A_CLAUDE" --cache-mode memory \
  --run-record "$HERE/run-receipt.json" \
  --sql "SELECT session_id, COUNT(*) AS tool_count FROM tool_calls GROUP BY session_id ORDER BY session_id" \
  --output json > "$HERE/memory-run.json" 2> "$HERE/memory-run.err"

# 6. CLI strict-evidence: pass and deliberately-failing cell limit.
$CMD query run --archive-glob "$A_CLAUDE" --cache-mode memory --strict-evidence \
  --sql "SELECT COUNT(*) AS n FROM sessions" --output json \
  > "$HERE/strict-pass.json" 2>&1
$CMD query run --archive-glob "$A_CLAUDE" --cache-mode memory --strict-evidence \
  --max-cell-bytes 8 --sql "SELECT session_id, 'abé' AS text FROM sessions LIMIT 1" \
  --output json > "$HERE/strict-fail.txt" 2>&1

# 7. Intentional disallowed SQL introspection.
$CMD query run --archive-glob "$A_CLAUDE" --cache-mode memory \
  --sql "SELECT name FROM sqlite_master" --output json \
  > "$HERE/disallowed.txt" 2>&1

# 8. Required flag omission.
$CMD query commands good --query-repository "$HERE/commands" \
  --archive-glob "$A_PI" --output json > "$HERE/missing-source.txt" 2>&1
