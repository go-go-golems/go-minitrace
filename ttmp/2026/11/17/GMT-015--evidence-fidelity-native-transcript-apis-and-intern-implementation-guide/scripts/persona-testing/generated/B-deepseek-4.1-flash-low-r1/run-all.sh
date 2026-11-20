#!/bin/sh
# Rerun all probes. Run from the go-minitrace repository root.
set -eu
R=/private/tmp/gmt015-persona/matrix/B-deepseek-4.1-flash-low-r1
for v in unicode overflow tools tokens manifest; do
  go run ./cmd/go-minitrace query commands --query-repository "$R/qrepo" fidelity main "$v" --output json > "$R/out-$v.json"
done
PI=/private/tmp/gmt015-persona/corpus/archives/pi/active/2026-09/01a0a5c3-342f-7447-be74-b6c35f542ff7.minitrace.json
CX=/private/tmp/gmt015-persona/corpus/archives/codex/active/2026-10/01a13157-f631-7cd3-8f84-e2666b24af01.minitrace.json
CC=/private/tmp/gmt015-persona/corpus/archives/claude-code/active/2026-10/d047a697-35c0-460c-8fda-541799b5b5b0.minitrace.json
go run ./cmd/go-minitrace query commands --query-repository "$R/qrepo" fidelity main real-summary --path "$PI" --output json > "$R/out-real-pi.json"
go run ./cmd/go-minitrace query commands --query-repository "$R/qrepo" fidelity main real-summary --path "$CX" --output json > "$R/out-real-codex.json"
go run ./cmd/go-minitrace query commands --query-repository "$R/qrepo" fidelity main real-summary --path "$CC" --output json > "$R/out-real-claudecode.json"
go run ./cmd/go-minitrace query run --archive-glob /private/tmp/gmt015-persona/inputs/archive.json --cache-mode memory --sql 'SELECT COUNT(*) AS tools FROM tool_calls' --output json > "$R/out-cli-memory.json"
go run ./cmd/go-minitrace query run --archive-glob /private/tmp/gmt015-persona/inputs/archive.json --cache-mode none --sql 'SELECT tool_call_id, source_ordinal FROM tool_calls ORDER BY source_ordinal' --output json > "$R/out-cli-none.json"
