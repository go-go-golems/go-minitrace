#!/usr/bin/env bash
set -euo pipefail

# Convert sampled Copilot session-state directories. The Copilot adapter discovers
# sessions from directories that contain events.jsonl, so this script maps the
# inventory's events.jsonl sample paths back to their parent session dirs.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
TICKET="$ROOT/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts"
SRC="$TICKET/sources/source-shape-inventory"
OUT="$TICKET/sources/converted-corpus"
LOGS="$TICKET/scripts/logs"
LIST="$SRC/copilot-sample-list.txt"
DEST="$OUT/copilot"
LOG="$LOGS/05-convert-copilot.log"

mkdir -p "$DEST" "$LOGS"
cd "$ROOT"

if [[ ! -s "$LIST" ]]; then
  echo "skip copilot: no sampled sources" | tee "$LOG"
  exit 0
fi

rm -rf "$DEST"
mkdir -p "$DEST"
echo "## convert copilot" | tee "$LOG"
status=0
while IFS= read -r source_path; do
  [[ -n "$source_path" ]] || continue
  session_dir="$source_path"
  if [[ "$(basename "$session_dir")" == "events.jsonl" ]]; then
    session_dir="$(dirname "$session_dir")"
  fi
  echo "## source-dir $session_dir" | tee -a "$LOG"
  set +e
  GOWORK=off go run ./cmd/go-minitrace convert copilot \
    --source-dir "$session_dir" \
    --output-dir "$DEST" \
    --output json 2>&1 | tee -a "$LOG"
  cmd_status=${PIPESTATUS[0]}
  set -e
  if [[ "$cmd_status" -ne 0 ]]; then
    status="$cmd_status"
  fi
done < "$LIST"

echo "exit=$status" | tee -a "$LOG"
exit "$status"
