#!/usr/bin/env bash
set -euo pipefail

# Convert the sampled JSONL-based adapters into ticket-local output directories.
# Raw source transcripts are not copied into the ticket; sample-list files hold
# local paths only and should be reviewed before committing if privacy matters.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
TICKET="$ROOT/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts"
SRC="$TICKET/sources/source-shape-inventory"
OUT="$TICKET/sources/converted-corpus"
LOGS="$TICKET/scripts/logs"
mkdir -p "$OUT" "$LOGS"
cd "$ROOT"

run_convert() {
  local adapter="$1"
  local subcmd="$2"
  local list="$SRC/${adapter}-sample-list.txt"
  local dest="$OUT/${adapter}"
  local log="$LOGS/02-convert-${adapter}.log"
  if [[ ! -s "$list" ]]; then
    echo "skip ${adapter}: no sampled sources" | tee "$log"
    return 0
  fi
  rm -rf "$dest"
  mkdir -p "$dest"
  echo "## convert ${adapter}" | tee "$log"
  set +e
  GOWORK=off go run ./cmd/go-minitrace convert "$subcmd" \
    --source-list "$list" \
    --output-dir "$dest" \
    --output json 2>&1 | tee -a "$log"
  local status=${PIPESTATUS[0]}
  set -e
  echo "exit=$status" | tee -a "$log"
  return "$status"
}

run_convert pi pi || true
run_convert codex codex || true
run_convert claude-code claude-code || true

echo "Converted archives under $OUT"
