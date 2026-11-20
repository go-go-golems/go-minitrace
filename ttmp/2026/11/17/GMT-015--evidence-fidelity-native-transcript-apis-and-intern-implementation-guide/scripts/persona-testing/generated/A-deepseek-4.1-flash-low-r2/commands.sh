#!/bin/sh
# Command receipts (run from the go-minitrace repo worktree)
REPO=/Users/manuel.odendahl/code/go-go-golems/go-minitrace/.worktrees/feat-evidence-fidelity
RAW=/private/tmp/gmt015-persona/corpus/raw
ARCH='/private/tmp/gmt015-persona/corpus/archives/*/active/*/*.minitrace.json'
W=/private/tmp/gmt015-persona/matrix/A-deepseek-4.1-flash-low-r2
cd "$REPO"
go run ./cmd/go-minitrace discover claude-code --source-dir "$RAW/claude-code" --output json > "$W/converted/receipts/discover-claude-code.json"
go run ./cmd/go-minitrace discover codex --source-dir "$RAW/codex" --output json > "$W/converted/receipts/discover-codex.json"
go run ./cmd/go-minitrace discover pi --source-dir "$RAW/pi" --output json > "$W/converted/receipts/discover-pi.json"
go run ./cmd/go-minitrace convert claude-code --source-session "$RAW/claude-code/7dce992c-7793-4e1a-950c-4db92cef2502.jsonl" --output-dir "$W/converted/claude-code" --output json > "$W/converted/receipts/claude-code-convert.json"
go run ./cmd/go-minitrace convert codex --source-session "$RAW/codex/rollout-2026-10-12T14-20-10-01a13157-f631-7cd3-8f84-e2666b24af01.jsonl" --output-dir "$W/converted/codex" --run-record "$W/converted/receipts/codex-run.json" --output json > "$W/converted/receipts/codex-convert.json"
go run ./cmd/go-minitrace convert pi --source-session "$RAW/pi/2026-09-15T15-50-28-655Z_01a0a5c3-342f-7447-be74-b6c35f542ff7.jsonl" --output-dir "$W/converted/pi" --output json > "$W/converted/receipts/pi-convert.json"
for f in claude-code codex pi; do go run ./cmd/go-minitrace validate --path "$W/converted/$f" --archive --recursive --output json > "$W/converted/receipts/$f-validate.json"; done
go run ./cmd/go-minitrace query run --archive-glob "$ARCH" --sql-file "$W/sql/session-counts.sql" --output json > "$W/sql/session-counts.out.json"
go run ./cmd/go-minitrace query run --archive-glob "$ARCH" --sql-file "$W/sql/top-tools.sql" --output json > "$W/sql/top-tools.out.json"
go run ./cmd/go-minitrace query run --archive-glob "$ARCH" --sql-file "$W/sql/date-range.sql" --output json > "$W/sql/date-range.out.json"
go run ./cmd/go-minitrace query commands --query-repository "$W/query-commands" evidence session-counts --archive-glob "$ARCH" --output json > "$W/query-commands/session-counts.out.json"
go run ./cmd/go-minitrace query commands history file-history --archive-glob "$ARCH" --path 'SKILL.md' --limit 20 --output json > "$W/sql/history-file-history.out.json"
go run ./cmd/go-minitrace query run --archive-glob "$ARCH" --sql "SELECT ... failed candidates ..." --output json > "$W/sql/failed-candidates.json"
