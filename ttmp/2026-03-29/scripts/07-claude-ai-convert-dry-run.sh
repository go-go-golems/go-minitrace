#!/usr/bin/env bash
set -euo pipefail

# Run the claude.ai export converter in dry-run mode against the real export ZIP.

REPO_DIR="/home/manuel/code/wesen/corporate-headquarters/go-minitrace"
SOURCE_ZIP="${1:-$HOME/Downloads/data-2026-03-29-11-53-11-batch-0000.zip}"

cd "$REPO_DIR"
go run ./cmd/go-minitrace convert claude-ai \
  --source "$SOURCE_ZIP" \
  --dry-run \
  --output json
