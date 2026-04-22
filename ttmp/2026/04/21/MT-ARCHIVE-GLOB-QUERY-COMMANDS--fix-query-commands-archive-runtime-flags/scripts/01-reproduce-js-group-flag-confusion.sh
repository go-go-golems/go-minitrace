#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="/home/manuel/code/wesen/corporate-headquarters/go-minitrace"
TMPDIR_="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_"' EXIT

mkdir -p "$TMPDIR_/hardware-research"
cat > "$TMPDIR_/hardware-research/research-summary.js" <<'EOF'
__section__("filters", {
  fields: {
    limit: { type: "int", default: 5, help: "limit" }
  }
});

function researchSummary(filters) {
  const mt = require("minitrace");
  return mt.query(`SELECT id FROM ${mt.tableName} LIMIT ${filters.limit}`);
}

__verb__("researchSummary", {
  name: "research-summary",
  short: "Generate summary",
  fields: {
    filters: { bind: "filters" }
  }
});
EOF

echo "== Repository help =="
(
  cd "$REPO_ROOT"
  go run ./cmd/go-minitrace query commands --query-repository "$TMPDIR_" --help | rg 'hardware-research|research-summary' -n -S || true
)

echo
echo "== WRONG invocation: stops on JS file-stem group =="
(
  cd "$REPO_ROOT"
  set +e
  go run ./cmd/go-minitrace query commands --query-repository "$TMPDIR_" \
    hardware-research research-summary \
    --archive-glob './output/active/*/*.minitrace.json'
  status=$?
  set -e
  echo "exit_status=$status"
)

echo
echo "== CORRECT invocation: include JS file-stem group AND leaf verb =="
(
  cd "$REPO_ROOT"
  go run ./cmd/go-minitrace query commands --query-repository "$TMPDIR_" \
    hardware-research research-summary research-summary \
    --archive-glob './output/active/*/*.minitrace.json' \
    --limit 1
)
