#!/usr/bin/env bash
set -euo pipefail

# Convert one claude.ai conversation into a temp output archive.

REPO_DIR="/home/manuel/code/wesen/corporate-headquarters/go-minitrace"
SOURCE_ZIP="${1:-$HOME/Downloads/data-2026-03-29-11-53-11-batch-0000.zip}"
UUID_FILTER="${2:-7756135a}"
OUTPUT_DIR="${3:-$(mktemp -d /tmp/go-minitrace-claudeai-XXXXXX)/output}"

cd "$REPO_DIR"
go run ./cmd/go-minitrace convert claude-ai \
  --source "$SOURCE_ZIP" \
  --uuid-filter "$UUID_FILTER" \
  --output-dir "$OUTPUT_DIR" \
  --output json
