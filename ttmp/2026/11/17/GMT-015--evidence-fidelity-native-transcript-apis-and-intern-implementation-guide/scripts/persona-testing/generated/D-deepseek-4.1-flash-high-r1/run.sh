#!/usr/bin/env bash
# Reproduce the GMT-015 query-command checks.
set -u
ROOT="${ROOT:-/private/tmp/gmt015-persona/matrix/D-deepseek-4.1-flash-high-r1}"
REPO="${REPO:-/Users/manuel.odendahl/code/go-go-golems/go-minitrace/.worktrees/feat-evidence-fidelity}"
CORPUS="${CORPUS:-/private/tmp/gmt015-persona/corpus/archives}"
CLAUDE="$CORPUS/claude-code/active/2026-10/7dce992c-7793-4e1a-950c-4db92cef2502.minitrace.json"
CODEX="$CORPUS/codex/active/2026-10/01a13157-f631-7cd3-8f84-e2666b24af01.minitrace.json"
PI="$CORPUS/pi/active/2026-09/01a0a5c3-342f-7447-be74-b6c35f542ff7.minitrace.json"
mkdir -p "$ROOT/captures"
cd "$REPO" || exit 1
mt() { go run ./cmd/go-minitrace "$@"; }

mt query commands --query-repository "$ROOT/commands" --help \
  > "$ROOT/captures/commands-help.txt" 2> "$ROOT/captures/commands-help.stderr.txt"
mt query commands good --query-repository "$ROOT/commands" --help \
  > "$ROOT/captures/good-help.txt" 2>/dev/null

for pair in "claude-code|$CLAUDE" "codex|$CODEX" "pi|$PI"; do
  src="${pair%%|*}"; arch="${pair#*|}"
  mt query commands good --query-repository "$ROOT/commands" \
    --archive-glob "$arch" --source "$src" --output json \
    > "$ROOT/captures/good-$src.json" 2> "$ROOT/captures/good-$src.stderr.txt"
done

mt query commands good --query-repository "$ROOT/commands" \
  --archive-glob "$CLAUDE" --output json \
  > "$ROOT/captures/good-missing-source.txt" 2>&1 || true

mt query commands probe --query-repository "$ROOT/commands" \
  --archive-glob "$CLAUDE" --source claude-code --output json \
  > "$ROOT/captures/probe.json" 2> "$ROOT/captures/probe.stderr.txt"

mt query commands raw --query-repository "$ROOT/commands" \
  --archive-glob "$CLAUDE" --source demo --output json \
  > "$ROOT/captures/raw.json" 2>/dev/null
mt query commands raw --query-repository "$ROOT/commands" \
  --archive-glob "$CLAUDE" --source demo --output table \
  > "$ROOT/captures/raw-table.txt" 2>/dev/null

mt query run --archive-glob "$CLAUDE" --archive-glob "$CODEX" --archive-glob "$PI" \
  --cache-mode memory --strict-evidence --max-rows 10 \
  --run-record "$ROOT/captures/query-run-receipt.json" \
  --sql "SELECT s.agent_framework AS source, t.session_id, COUNT(*) AS tool_count FROM tool_calls t JOIN sessions s USING (session_id) GROUP BY 1,2 ORDER BY tool_count DESC" \
  --output json > "$ROOT/captures/query-run-strict.json" 2> "$ROOT/captures/query-run-strict.stderr.txt"

mt query run --archive-glob "$CLAUDE" --cache-mode memory --strict-evidence \
  --max-cell-bytes 3 --run-record "$ROOT/captures/query-run-strict-fail-receipt.json" \
  --sql "SELECT 'abé' AS text" --output json \
  > "$ROOT/captures/query-run-strict-fail.json" 2> "$ROOT/captures/query-run-strict-fail.stderr.txt" || true

mt query run --archive-glob "$CLAUDE" --cache-mode memory --max-cell-bytes 3 \
  --sql "SELECT 'abé' AS text" --output json \
  > "$ROOT/captures/query-run-nonstrict-truncated.json" 2>&1 || true
