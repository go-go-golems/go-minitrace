#!/usr/bin/env bash
set -euo pipefail

# Convert the real Pi session that was used for the first smoke test.
# Output is written into a fresh temp directory unless an explicit target is
# passed as the second argument.

REPO_DIR="/home/manuel/code/wesen/corporate-headquarters/go-minitrace"
SOURCE_SESSION="${1:-$HOME/.pi/agent/sessions/--home-manuel-code-others-llms-minitrace--/2026-03-28T21-19-08-451Z_bda24bdb-9762-4e1e-b749-f29dbe2dd0b8.jsonl}"
OUTPUT_DIR="${2:-$(mktemp -d /tmp/go-minitrace-pi-XXXXXX)/output}"

cd "$REPO_DIR"
go run ./cmd/go-minitrace convert pi \
  --source-session "$SOURCE_SESSION" \
  --output-dir "$OUTPUT_DIR" \
  --output json
