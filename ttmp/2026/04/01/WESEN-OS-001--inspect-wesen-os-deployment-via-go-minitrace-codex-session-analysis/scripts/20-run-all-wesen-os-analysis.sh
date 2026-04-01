#!/usr/bin/env bash
# 20-run-all-wesen-os-analysis.sh
#
# Master runner: executes all wesen-os analysis scripts in order
# and writes results to /tmp/wesen-os-analysis/.
#
# Prerequisites:
#   - go-minitrace converted archive at /tmp/minitrace-output/active/*/*.minitrace.json
#   - python3 available
#
# Usage:
#   cd ttmp/2026/04/01/WESEN-OS-001--inspect-wesen-os-deployment-via-go-minitrace-codex-session-analysis/scripts
#   bash 20-run-all-wesen-os-analysis.sh

set -euo pipefail

GLOB='/tmp/minitrace-output/active/*/*.minitrace.json'
OUT=/tmp/wesen-os-analysis
SCRIPTS="$(cd "$(dirname "$0")" && pwd)"

mkdir -p "$OUT"

echo "=== 13: wesen-os active vs wall-clock ==="
go-minitrace query duckdb --archive-glob "$GLOB" \
  --sql-file "$SCRIPTS/13-wesen-os-active-vs-wall.sql" 2>&1 | tee "$OUT/13-active-vs-wall.txt"
echo

echo "=== 10+14: human blocks for 019d174c (profile migration) ==="
go-minitrace query duckdb --archive-glob "$GLOB" \
  --sql "$(sed 's/SESSION_ID/019d174c-fc68-7c00-8f1b-7fcc067c1fd6/g' "$SCRIPTS/10-human-blocks.sql")" \
  --output json > "$OUT/blocks-174c.json" 2>&1
python3 "$SCRIPTS/14-autopilot-blocks.py" "$OUT/blocks-174c.json" | tee "$OUT/14-autopilot-174c.txt"
echo

echo "=== 10+14: human blocks for 019d376d (NPM publish + federation) ==="
go-minitrace query duckdb --archive-glob "$GLOB" \
  --sql "$(sed 's/SESSION_ID/019d376d-0103-7dc3-a96d-650c7c2e1cf7/g' "$SCRIPTS/10-human-blocks.sql")" \
  --output json > "$OUT/blocks-376d.json" 2>&1
python3 "$SCRIPTS/14-autopilot-blocks.py" "$OUT/blocks-376d.json" | tee "$OUT/14-autopilot-376d.txt"
echo

echo "=== 10+14: human blocks for 019d4a35 (sqlite handoff) ==="
go-minitrace query duckdb --archive-glob "$GLOB" \
  --sql "$(sed 's/SESSION_ID/019d4a35-9c8d-7f10-8fef-ef0650432725/g' "$SCRIPTS/10-human-blocks.sql")" \
  --output json > "$OUT/blocks-4a35.json" 2>&1
python3 "$SCRIPTS/14-autopilot-blocks.py" "$OUT/blocks-4a35.json" | tee "$OUT/14-autopilot-4a35.txt"
echo

echo "=== 15+16: docmgr and ttmp ops ==="
go-minitrace query duckdb --archive-glob "$GLOB" \
  --sql-file "$SCRIPTS/15-docmgr-and-ttmp-ops.sql" \
  --output json > "$OUT/docmgr-calls.json" 2>&1
python3 "$SCRIPTS/16-classify-docmgr-ttmp.py" "$OUT/docmgr-calls.json" | tee "$OUT/16-docmgr-ttmp.txt"
echo

echo "=== 17+18: user input gaps ==="
go-minitrace query duckdb --archive-glob "$GLOB" \
  --sql-file "$SCRIPTS/17-user-input-gaps.sql" \
  --output json > "$OUT/gaps.json" 2>&1
python3 "$SCRIPTS/18-analyze-gaps.py" "$OUT/gaps.json" | tee "$OUT/18-gaps.txt"
echo

echo "=== 19: ticket creation timeline ==="
go-minitrace query duckdb --archive-glob "$GLOB" \
  --sql-file "$SCRIPTS/19-ticket-creation-timeline.sql" 2>&1 | tee "$OUT/19-tickets.txt"
echo

echo "All results in $OUT/"
ls -la "$OUT/"
