#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
OUT="${1:-/tmp/go-minitrace-turnsdb}"

rm -rf "$OUT"
cd "$ROOT"
go run ./cmd/go-minitrace convert turnsdb \
  --source /tmp/turns.db \
  --output-dir "$OUT" \
  --output json
