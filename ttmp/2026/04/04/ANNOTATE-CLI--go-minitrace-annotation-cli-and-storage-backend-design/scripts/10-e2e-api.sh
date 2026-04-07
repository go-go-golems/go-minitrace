#!/usr/bin/env bash
# scripts/10-e2e-api.sh
# End-to-end test: HTTP API for annotations.
# Starts serve, exercises all annotation endpoints with curl.
set -euo pipefail

BIN="${BIN:-./go-minitrace}"
DIR=$(mktemp -d)
DUCKDB_FILE="$DIR/analysis.duckdb"
PORT=18766

cleanup() {
  kill "${SERVER_PID:-}" 2>/dev/null || true
  rm -rf "$DIR"
}
trap cleanup EXIT
mkdir -p "$DIR/active/2026-04"

echo "=== E2E: Annotation HTTP API ==="
echo "DIR: $DIR"

# Create a minimal session file in the expected path.
cat > "$DIR/active/2026-04/sess-api-001.minitrace.json" << 'EOF'
{
  "id": "sess-api-001",
  "schema_version": "0.2.0",
  "profile": "organic",
  "title": "API E2E test session",
  "classification": "internal",
  "provenance": {"source_format": "e2e", "source_path": "e2e"},
  "flags": {},
  "environment": {"agent_framework": "claude-code", "model": "claude-opus-4"},
  "operational_context": {"working_directory": "/tmp"},
  "timing": {"started_at": "2026-04-04T00:00:00Z"},
  "turns": [], "tool_calls": [],
  "annotations": []
}
EOF

echo "--- Building ---"
go build -o "$BIN" ./cmd/go-minitrace/ 2>/dev/null || \
  [ -x "$BIN" ] || { echo "BIN not found: $BIN"; exit 1; }

echo "--- Starting serve ---"
$BIN serve \
  --archive-glob "$DIR/active/2026-04/*.minitrace.json" \
  --db-path "$DUCKDB_FILE" \
  --port "$PORT" \
  > /dev/null 2>&1 &
SERVER_PID=$!
sleep 3

BASE="http://localhost:$PORT/api"
fail() { echo "FAIL: $1"; kill "$SERVER_PID" 2>/dev/null; exit 1; }

# Capture response body + HTTP code into separate variables.
# Uses __SEP__ as sentinel to split body from status code.
capture() {
  local path="$1"; shift
  local body="${1:-}"
  _BODY=$(curl -s -X "${method:-GET}" \
    -H "Content-Type: application/json" \
    ${body:+-d "$body"} \
    -w "\n__HTTP__:%{http_code}" \
    "$BASE$path")
  _CODE=$(echo "$_BODY" | grep "^__HTTP__:" | cut -d: -f2)
  _BODY=$(echo "$_BODY" | sed '/^__HTTP__:/d')
}
post() { local method="POST"; capture "$@"; }
put()  { local method="PUT";  capture "$@"; }
del()  { local method="DELETE"; capture "$@"; }

echo ""
echo "--- GET /api/sessions/sess-api-001/annotations (empty) ---"
capture "/sessions/sess-api-001/annotations"
echo "HTTP $_CODE"
echo "$_BODY" | python3 -m json.tool 2>/dev/null || echo "$_BODY"
[ "$_CODE" = "200" ] || fail "GET annotations failed: HTTP $_CODE"

echo ""
echo "--- POST /api/sessions/sess-api-001/annotations ---"
post "/sessions/sess-api-001/annotations" '{"category":"ai-failure","title":"API E2E test","detail":"Created via HTTP","annotator":"e2e-test","tags":["auth","e2e"],"scope_type":"session"}'
echo "HTTP $_CODE"
echo "$_BODY" | python3 -m json.tool 2>/dev/null || echo "$_BODY"
[ "$_CODE" = "201" ] || fail "POST annotation failed: HTTP $_CODE"

ANN_ID=$(echo "$_BODY" | python3 -c "import json,sys; print(json.load(sys.stdin)['id'])" 2>/dev/null)
echo "Created annotation ID: $ANN_ID"
[ -n "$ANN_ID" ] || fail "Could not extract annotation ID"

echo ""
echo "--- GET /api/sessions/sess-api-001/annotations (should have 1) ---"
capture "/sessions/sess-api-001/annotations"
echo "HTTP $_CODE"
COUNT=$(echo "$_BODY" | python3 -c "import json,sys; print(json.load(sys.stdin)['count'])" 2>/dev/null)
echo "count=$COUNT"
[ "$_CODE" = "200" ] || fail "GET annotations failed: HTTP $_CODE"
[ "$COUNT" = "1" ] || fail "Expected count=1, got $COUNT"

echo ""
echo "--- PUT /api/annotations/{id} (patch) ---"
put "/annotations/$ANN_ID" '{"title":"Updated via E2E","detail":"Updated detail"}'
echo "HTTP $_CODE"
echo "$_BODY"
[ "$_CODE" = "200" ] || fail "PUT annotation failed: HTTP $_CODE"

echo ""
echo "--- DELETE /api/annotations/{id} ---"
del "/annotations/$ANN_ID"
echo "HTTP $_CODE"
[ "$_CODE" = "204" ] || fail "DELETE annotation failed: HTTP $_CODE (body: $_BODY)"

echo ""
echo "--- GET after delete (should be empty) ---"
capture "/sessions/sess-api-001/annotations"
echo "HTTP $_CODE"
COUNT=$(echo "$_BODY" | python3 -c "import json,sys; print(json.load(sys.stdin)['count'])" 2>/dev/null)
echo "count=$COUNT"
[ "$_CODE" = "200" ] || fail "GET after delete failed: HTTP $_CODE"
[ "$COUNT" = "0" ] || fail "Expected count=0 after delete, got $COUNT"

echo ""
echo "--- POST /api/annotations/sync (dry-run) ---"
post "/annotations/sync" '{"dry_run":true}'
echo "HTTP $_CODE"
echo "$_BODY" | python3 -m json.tool 2>/dev/null || echo "$_BODY"
[ "$_CODE" = "200" ] || fail "POST sync failed: HTTP $_CODE"

echo ""
echo "--- 404 on unknown annotation ---"
del "/annotations/does-not-exist-0000"
echo "HTTP $_CODE"
echo "$_BODY"
[ "$_CODE" = "404" ] || fail "Expected 404 for unknown annotation, got HTTP $_CODE (body: $_BODY)"

echo ""
echo "--- 400 on missing required fields ---"
post "/sessions/sess-api-001/annotations" '{"title":""}'
echo "HTTP $_CODE"
[ "$_CODE" = "400" ] || fail "Expected 400 for missing category, got HTTP $_CODE"

echo ""
echo "--- 503 when store unavailable (no output-dir) ---"
# Skip: would need a serve instance without annotations.db
echo "(Skipped: no-output-dir check)"

kill "$SERVER_PID" 2>/dev/null
wait "$SERVER_PID" 2>/dev/null || true

echo ""
echo "=== ALL HTTP API TESTS PASSED ==="
