#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
OUT="${1:-/tmp/go-minitrace-chatgpt-json}"

rm -rf "$OUT"
cd "$ROOT"
go run ./cmd/go-minitrace convert chatgpt-json \
  --source-dir /tmp/chatgpt-exports \
  --output-dir "$OUT" \
  --output json
