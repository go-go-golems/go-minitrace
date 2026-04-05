#!/usr/bin/env bash
# Build a deterministic UI smoke fixture from one real Codex session.
set -euo pipefail

REPO="$(cd "$(dirname "$0")/../../../../../../.." && pwd)"
cd "$REPO"

SMOKE_ROOT=tmp/ui-smoke
RAW="$SMOKE_ROOT/raw-codex"
OUT="$SMOKE_ROOT/output"
SESSION_SRC="${SESSION_SRC:-$HOME/.codex/sessions/2026/01/12/rollout-2026-01-12T15-46-57-019bb3f6-3c71-7013-b585-4f16d9bdceb6.jsonl}"
SESSION_ID="019bb3f6-3c71-7013-b585-4f16d9bdceb6"
TURN_ID="0"
TOOL_ID="call_Y70XEopD3Ef1mGctwTXG2CEq"

rm -rf "$SMOKE_ROOT"
mkdir -p "$RAW/sessions/2026/01/12" "$OUT"
cp "$SESSION_SRC" "$RAW/sessions/2026/01/12/"

./go-minitrace convert codex --source-dir "$RAW" --output-dir "$OUT"
rm -f "$OUT/annotations.db"* "$OUT/analysis.ui.db"*

./go-minitrace annotate add \
  --output-dir "$OUT" \
  --session "$SESSION_ID" \
  --category ai-failure \
  --title "Session scoped smoke" \
  --detail "session scope" \
  --annotator smoke

./go-minitrace annotate add \
  --output-dir "$OUT" \
  --session "$SESSION_ID" \
  --scope turn \
  --target-id "$TURN_ID" \
  --category question \
  --title "Turn scoped smoke" \
  --detail "turn scope" \
  --annotator smoke

./go-minitrace annotate add \
  --output-dir "$OUT" \
  --session "$SESSION_ID" \
  --scope tool_call \
  --target-id "$TOOL_ID" \
  --category observation \
  --title "Tool scoped smoke" \
  --detail "tool scope" \
  --annotator smoke

echo "Fixture ready: $OUT"
find "$OUT" -name '*.minitrace.json' -print
sqlite3 "$OUT/annotations.db" "select scope_type,target_id,title from annotations order by created_at;"
