#!/usr/bin/env bash
set -euo pipefail

# Scan Downloads for likely ChatGPT / claude.ai export or transcript files.

DOWNLOADS_DIR="${1:-$HOME/Downloads}"

echo "== archives =="
find "$DOWNLOADS_DIR" -maxdepth 2 -type f \( -iname '*.zip' -o -iname '*.tar' -o -iname '*.gz' \) | sed -n '1,120p'

echo
echo "== candidate names =="
find "$DOWNLOADS_DIR" -maxdepth 2 -type f | rg 'chatgpt|openai|data export|conversations|claude' -i -n | sed -n '1,120p'
