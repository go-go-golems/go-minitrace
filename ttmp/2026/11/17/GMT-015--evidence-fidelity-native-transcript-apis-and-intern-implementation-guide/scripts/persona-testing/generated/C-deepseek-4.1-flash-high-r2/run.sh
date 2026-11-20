#!/usr/bin/env bash
# Reproduce every command used for this review. Run from the repo worktree.
set -euo pipefail
REPO=/Users/manuel.odendahl/code/go-go-golems/go-minitrace/.worktrees/feat-evidence-fidelity
W=/private/tmp/gmt015-persona/matrix/C-deepseek-4.1-flash-high-r2
ARCH='/private/tmp/gmt015-persona/corpus/archives/*/active/*/*.minitrace.json'
RAW=$(find /private/tmp/gmt015-persona/corpus/raw -type f | sort | paste -sd, -)
cd "$REPO"
go run ./cmd/go-minitrace query commands native native-scan scan-files \
  --query-repository "$W/qrepo" --files "$RAW" --output json > "$W/out/real-scan.json"
go run ./cmd/go-minitrace query commands native native-detail detail-files \
  --query-repository "$W/qrepo" \
  --files /private/tmp/gmt015-persona/corpus/raw/claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl,/private/tmp/gmt015-persona/corpus/raw/claude-code/7dce992c-7793-4e1a-950c-4db92cef2502.jsonl \
  --sample 8 --output json > "$W/out/real-detail.json"
go run ./cmd/go-minitrace query commands native native-inspect inspect-files \
  --query-repository "$W/qrepo" --files /private/tmp/gmt015-persona/inputs/native.jsonl \
  --show-requests --output json > "$W/out/synth-inspect.json"
go run ./cmd/go-minitrace query commands native native-bounds bounds-demo \
  --query-repository "$W/qrepo" --file /private/tmp/gmt015-persona/inputs/native.jsonl \
  --output json > "$W/out/bounds-full.json"
go run ./cmd/go-minitrace query commands native native-bounds bounds-demo \
  --query-repository "$W/qrepo" --file /private/tmp/gmt015-persona/inputs/native.jsonl \
  --stop-after 4 --output json > "$W/out/bounds-stop4.json"
go run ./cmd/go-minitrace query commands native native-bounds bounds-demo \
  --query-repository "$W/qrepo" --file /private/tmp/gmt015-persona/inputs/native.jsonl \
  --max-record-bytes 60 --output json > "$W/out/bounds-oversize60.json"
go run ./cmd/go-minitrace query run --archive-glob "$ARCH" --sql-file "$W/sql/01-instruction-file-read-candidates.sql" \
  --max-rows 200 --output json > "$W/out/sql-01-candidates.json"
go run ./cmd/go-minitrace query run --archive-glob "$ARCH" --sql-file "$W/sql/02a-read-evidence.sql" \
  --output json > "$W/out/sql-02a-read-evidence.json"
go run ./cmd/go-minitrace query run --archive-glob "$ARCH" --sql-file "$W/sql/02b-mention-search.sql" \
  --output json > "$W/out/sql-02b-mention-search.json"
go run ./cmd/go-minitrace query run --archive-glob "$ARCH" --sql-file "$W/sql/03-coverage.sql" \
  --output json > "$W/out/sql-03-coverage.json"
go run ./cmd/go-minitrace query commands history context-window \
  --archive-glob '/private/tmp/gmt015-persona/corpus/archives/pi/active/*/*.minitrace.json' \
  --session 01a0b061-0cdd-7557-a80a-96802d3d02c5 --turn 4 --output json > "$W/out/history-context-window.json"
