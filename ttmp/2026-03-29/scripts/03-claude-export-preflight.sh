#!/usr/bin/env bash
set -euo pipefail

# Check whether a claude.ai export ZIP looks like the expected privacy export.
# The expected members are:
#   conversations.json
#   users.json
#   projects.json
#   memories.json

ZIP_PATH="${1:-$HOME/Downloads/data-2026-03-29-11-53-11-batch-0000.zip}"

echo "FILE: $ZIP_PATH"
unzip -l "$ZIP_PATH" | sed -n '1,24p'
