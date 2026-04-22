#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="/home/manuel/code/wesen/corporate-headquarters/go-minitrace"
TMPDIR_="${1:-$(mktemp -d)}"
CLEANUP=0
if [[ $# -eq 0 ]]; then
  CLEANUP=1
fi
trap '[[ "$CLEANUP" -eq 1 ]] && rm -rf "$TMPDIR_"' EXIT

mkdir -p "$TMPDIR_/hardware-research"
cat > "$TMPDIR_/hardware-research/research-summary.js" <<'EOF'
function researchSummary() {
  const mt = require("minitrace");
  return mt.query(`SELECT 1 AS ok FROM ${mt.tableName} LIMIT 1`);
}

__verb__("researchSummary", {
  name: "research-summary",
  short: "Generate summary"
});
EOF

cd "$REPO_ROOT"

echo "== group help =="
go run ./cmd/go-minitrace query commands --query-repository "$TMPDIR_" hardware-research --help

echo
echo "== js file-stem help =="
go run ./cmd/go-minitrace query commands --query-repository "$TMPDIR_" hardware-research research-summary --help

echo
echo "== executable leaf help =="
go run ./cmd/go-minitrace query commands --query-repository "$TMPDIR_" hardware-research research-summary research-summary --help
