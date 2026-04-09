#!/usr/bin/env bash
set -euo pipefail

TICKET_DIR="$(cd "$(dirname "$0")/.." && pwd)"
OUTPUT_DIR="$TICKET_DIR/analysis/archive"

MINIMAX_SESSION='/home/manuel/.pi/agent/sessions/--home-manuel-workspaces-2026-04-08-sqleton-minitrace-minimax--/2026-04-09T00-23-06-562Z_2d525241-fe32-417b-8576-b29ce3b3e47c.jsonl'
GPT_SESSION='/home/manuel/.pi/agent/sessions/--home-manuel-workspaces-2026-04-08-sqleton-minitrace--/2026-04-09T00-13-39-925Z_7f61f412-40f0-417f-ab85-4dffdb9927e5.jsonl'

mkdir -p "$OUTPUT_DIR"

cd /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace

go-minitrace convert pi --source-session "$MINIMAX_SESSION" --output-dir "$OUTPUT_DIR"
go-minitrace convert pi --source-session "$GPT_SESSION" --output-dir "$OUTPUT_DIR"

find "$OUTPUT_DIR" -maxdepth 3 -type f | sort
